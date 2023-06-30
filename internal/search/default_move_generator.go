package search

import (
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type DefaultMoveGenerator struct {
	*game.GameState
}

var _ MoveGen = (*DefaultMoveGenerator)(nil)

func (gen *DefaultMoveGenerator) generateMoves(mode MoveGenerationMode) (func(), MoveGenerationResult, *[]Move, Error) {

	moves := GetMovesBuffer()
	cleanup := func() { ReleaseMovesBuffer(moves) }

	result := AllLegalMoves

	if mode == OnlyCaptures {
		result = SomeLegalMoves
		GeneratePseudoCaptures(func(m Move) {
			*moves = append(*moves, m)
		}, gen.GameState)
	} else {
		GeneratePseudoMoves(func(m Move) {
			*moves = append(*moves, m)
		}, gen.GameState)
	}

	return cleanup, result, moves, NilError
}
