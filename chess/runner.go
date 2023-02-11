package chess

import (
	"fmt"
	"strings"
)

type Runner struct {
	g *GameState
	b *Bitboards

	startPos string
	history  []HistoryValue

	Logger Logger
}

type HistoryValue struct {
	move     Move
	update   BoardUpdate
	previous OldGameState
}

func (r *Runner) IsNew() bool {
	return r.g == nil || r.b == nil || len(r.history) == 0
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

func (r *Runner) PerformMove(move Move) {
	r.history = append(r.history, HistoryValue{})

	h := r.LastHistory()

	SetupBoardUpdate(r.g, move, &h.update)
	RecordCurrentState(r.g, &h.previous)

	r.b.performMove(r.g, move)
	r.g.performMove(move, h.update)
}

func (r *Runner) PerformMoveFromString(s string) {
	r.PerformMove(r.g.moveFromString(s))
}

func (r *Runner) PerformMoves(startPos string, moves []string) {
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
		r.PerformMove(r.g.moveFromString(moves[i]))
	}
}

func (r *Runner) SetupPosition(position Position) {
	if !r.IsNew() {
		panic("please use ucinewgame")
	}

	game, err := GamestateFromFenString(position.fen)
	if err != nil {
		panic(fmt.Errorf("couldn't create game from %v, %v", position, err))
	}
	r.g = &game

	bitboards := SetupBitboards(r.g)
	r.b = &bitboards

	r.startPos = position.fen

	for _, m := range position.moves {
		r.PerformMove(r.g.moveFromString(m))
	}
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

func (r *Runner) HandleInput(input string) []string {
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
			r.SetupPosition(position)
		} else {
			r.PerformMoves(position.fen, position.moves)
		}
	} else if strings.HasPrefix(input, "go") {
		move := Search(r.g, r.b, 4, r.Logger)
		if move.IsEmpty() {
			panic(fmt.Errorf("failed to find move for %v ", r.g.Board.String()))
		}
		result = append(result, fmt.Sprintf("bestmove %v", move.Value().String()))
	}
	return result
}

func (r *Runner) MovesForSelection(selection string) []FileRank {
	selectionFileRank, err := FileRankFromString(selection)
	if err != nil {
		panic(fmt.Errorf("failed to parse selection %v", err))
	}
	selectionIndex := IndexFromFileRank(selectionFileRank)

	legalMoves := []Move{}
	r.b.generateLegalMoves(r.g, &legalMoves)

	moves := FilterSlice(legalMoves, func(m Move) bool {
		return m.startIndex == selectionIndex
	})
	return MapSlice(moves, func(m Move) FileRank {
		return FileRankFromIndex(m.endIndex)
	})
}

func (r *Runner) FenString() string {
	return r.g.fenString()
}

func (r *Runner) Player() Player {
	return r.g.player
}
