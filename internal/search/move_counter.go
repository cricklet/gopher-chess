package search

import (
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type MoveCounter struct {
	movesSearched int
	noCopy        NoCopy
}

var _ game.MoveListener = (*MoveCounter)(nil)

func NewMoveCounter(
	g *game.GameState,
) (func(), *MoveCounter) {
	gen := &MoveCounter{}

	unregister := g.RegisterListener(gen)
	return unregister, gen
}

func (gen *MoveCounter) Reset() {
	gen.movesSearched = 0
}

func (gen *MoveCounter) AfterMove(move Move) {
	gen.movesSearched++
}

func (gen *MoveCounter) AfterUndo() {
}

func (gen *MoveCounter) NumMoves() int {
	return gen.movesSearched
}
