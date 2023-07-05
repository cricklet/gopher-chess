package search

import (
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type DefaultMoveGenerator struct {
}

var _ MoveGen = (*DefaultMoveGenerator)(nil)

func (gen *DefaultMoveGenerator) generateMoves(g *game.GameState, mode MoveGenerationMode) (func(), MoveGenerationResult, *[]Move, Error) {

	moves := GetMovesBuffer()
	cleanup := func() { ReleaseMovesBuffer(moves) }

	result := AllLegalMoves

	if mode == OnlyCaptures {
		result = SomeLegalMoves
		GeneratePseudoCaptures(func(m Move) {
			*moves = append(*moves, m)
		}, g)
	} else {
		GeneratePseudoMoves(func(m Move) {
			*moves = append(*moves, m)
		}, g)
	}

	return cleanup, result, moves, NilError
}
