package chess

import (
	"fmt"
	"strings"
)

type Runner struct {
	g *GameState
	b *Bitboards
}

func (r *Runner) HandleInputAndReturnDone(input string) bool {
	if input == "uci" {
		fmt.Println("name chess-go")
		fmt.Println("id author Kenrick Rilee")
		fmt.Println("uciok")
	} else if input == "isready" {
		fmt.Println("readyok")
	} else if strings.HasPrefix(input, "position fen ") {
		s := strings.TrimPrefix(input, "position fen ")
		game, err := GamestateFromFenString(s)
		if err != nil {
			panic(fmt.Errorf("couldn't create game from %v", s))
		}
		r.g = &game

		bitboards := SetupBitboards(r.g)
		r.b = &bitboards
	} else if strings.HasPrefix(input, "go") {
		move := Search(r.g, r.b, 6)
		if move.IsEmpty() {
			panic(fmt.Errorf("failed to find move for %v ", r.g.Board.String()))
		}
		fmt.Printf("bestmove %v\n", move.Value().String())
	} else if input == "quit" {
		return true
	}
	return false
}
