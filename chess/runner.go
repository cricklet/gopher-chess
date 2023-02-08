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
func (r *Runner) Rewind(historyLength int) {
	for len(r.history) > historyLength {
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

func (r *Runner) PerformMoves(startPos string, moves []string) {
	if r.startPos != startPos {
		panic("please use ucinewgame")
	}

	startIndex := 0
	for i := 0; i < len(moves); i++ {
		if r.history[i].move.String() != moves[i] {
			r.Rewind(i)
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
		panic(fmt.Errorf("couldn't create game from %v", position))
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
		for _, f := range fields {
			result = append(result, f)
		}
	}
	return result
}

func parsePosition(input string) Position {
	return Position{parseFen(input), parseMoves(input)}
}

func (r *Runner) HandleInputAndReturnDone(input string) bool {
	if input == "uci" {
		fmt.Println("id name chessgo 1")
		fmt.Println("id author Kenrick Rilee")
		fmt.Println("uciok")
	} else if input == "ucinewgame" {
		r.g = nil
		r.b = nil
		r.startPos = ""
		r.history = []HistoryValue{}
	} else if input == "isready" {
		fmt.Println("readyok")
	} else if strings.HasPrefix(input, "position ") {
		position := parsePosition(input)
		if r.IsNew() {
			r.SetupPosition(position)
		} else {
			r.PerformMoves(position.fen, position.moves)
		}
	} else if strings.HasPrefix(input, "go") {
		move := Search(r.g, r.b, 4)
		if move.IsEmpty() {
			panic(fmt.Errorf("failed to find move for %v ", r.g.Board.String()))
		}
		fmt.Printf("bestmove %v\n", move.Value().String())
	} else if input == "quit" {
		return true
	}
	return false
}
