package chess

import (
	"errors"
	"fmt"
	"strings"
)

type Runner struct {
	Logger Logger

	g *GameState
	b *Bitboards

	startPos string
	history  []HistoryValue
}

type HistoryValue struct {
	move     Move
	update   BoardUpdate
	previous OldGameState
}

func (r *Runner) IsNew() bool {
	return r.g == nil || r.b == nil || len(r.history) == 0
}

func (r *Runner) LastMove() Optional[Move] {
	if len(r.history) > 0 {
		return Some(r.LastHistory().move)
	}
	return Empty[Move]()
}

func (r *Runner) LastHistory() *HistoryValue {
	return &r.history[len(r.history)-1]
}
func (r *Runner) Rewind(num int) {
	for i := 0; i < MinInt(num, len(r.history)); i++ {
		h := r.history[len(r.history)-1]
		r.b.undoUpdate(h.update)
		r.g.undoUpdate(h.previous, h.update)
		r.history = r.history[:len(r.history)-1]
	}
}

func (r *Runner) PerformMove(move Move) error {
	r.history = append(r.history, HistoryValue{})

	h := r.LastHistory()
	h.move = move

	err := SetupBoardUpdate(r.g, move, &h.update)
	if err != nil {
		return fmt.Errorf("PerformMove: %w", err)
	}
	RecordCurrentState(r.g, &h.previous)

	err = r.b.performMove(r.g, move)
	if err != nil {
		return fmt.Errorf("PerformMove: %w", err)
	}

	r.g.performMove(move, h.update)

	return nil
}

func (r *Runner) PerformMoveFromString(s string) error {
	return r.PerformMove(r.g.moveFromString(s))
}

func (r *Runner) PerformMoves(startPos string, moves []string) error {
	if r.startPos != startPos {
		panic("please use ucinewgame")
	}

	startIndex := 0
	for i := 0; i < len(moves); i++ {
		if r.history[i].move.String() != moves[i] {
			r.Rewind(len(moves) - i)
			startIndex = 0
			break
		}
	}

	for i := startIndex; i < len(moves); i++ {
		err := r.PerformMove(r.g.moveFromString(moves[i]))
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) SetupPosition(position Position) error {
	if r.Logger == nil {
		r.Logger = &DEFAULT_LOGGER
	}
	if !r.IsNew() {
		return errors.New("please use ucinewgame")
	}

	game, err := GamestateFromFenString(position.fen)
	if err != nil {
		return fmt.Errorf("couldn't create game from %v, %w", position, err)
	}
	r.g = &game

	bitboards := SetupBitboards(r.g)
	r.b = &bitboards

	r.startPos = position.fen

	for _, m := range position.moves {
		err := r.PerformMove(r.g.moveFromString(m))
		if err != nil {
			return err
		}
	}

	return nil
}

type Position struct {
	fen   string
	moves []string
}

func parseFen(input string) string {
	s := strings.TrimPrefix(input, "position ")

	if strings.HasPrefix(s, "fen ") {
		s = strings.TrimPrefix(s, "fen ")
		return strings.Split(s, " moves ")[0]
	} else if strings.HasPrefix(s, "startpos") {
		return "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}

	panic(fmt.Errorf("couldn't parse '%v'", s))
}
func parseMoves(input string) []string {
	result := []string{}
	if strings.Contains(input, " moves ") {
		fields := strings.Fields(strings.SplitN(input, " moves ", 2)[1])
		result = append(result, fields...)
	}
	return result
}

func parsePosition(input string) Position {
	return Position{parseFen(input), parseMoves(input)}
}

func (r *Runner) HandleInput(input string) ([]string, error) {
	result := []string{}
	if input == "uci" {
		result = append(result, "id name chessgo 1")
		result = append(result, "id author Kenrick Rilee")
		result = append(result, "uciok")
	} else if input == "ucinewgame" {
		r.g = nil
		r.b = nil
		r.startPos = ""
		r.history = []HistoryValue{}
	} else if input == "isready" {
		result = append(result, "readyok")
	} else if strings.HasPrefix(input, "position ") {
		position := parsePosition(input)
		if r.IsNew() {
			err := r.SetupPosition(position)
			if err != nil {
				return result, err
			}
		} else {
			err := r.PerformMoves(position.fen, position.moves)
			if err != nil {
				return result, err
			}
		}
	} else if strings.HasPrefix(input, "go") {
		move, err := Search(r.g, r.b, 4, r.Logger)
		if err != nil {
			return result, err
		}
		if move.IsEmpty() {
			return result, errors.New("no legal moves")
		}
		result = append(result, fmt.Sprintf("bestmove %v", move.Value().String()))
	}
	return result, nil
}

func (r *Runner) MovesForSelection(selection string) ([]FileRank, error) {
	selectionFileRank, err := FileRankFromString(selection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selection %w", err)
	}
	selectionIndex := IndexFromFileRank(selectionFileRank)

	legalMoves := []Move{}
	err = r.b.generateLegalMoves(r.g, &legalMoves)
	if err != nil {
		return nil, err
	}

	moves := FilterSlice(legalMoves, func(m Move) bool {
		return m.startIndex == selectionIndex
	})
	return MapSlice(moves, func(m Move) FileRank {
		return FileRankFromIndex(m.endIndex)
	}), nil
}

func (r *Runner) FenString() string {
	return r.g.fenString()
}

func (r *Runner) Player() Player {
	return r.g.player
}
