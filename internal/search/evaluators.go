package search

import (
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type Evaluator interface {
	evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error)
}

type BasicEvaluator struct {
}

var _ Evaluator = (*BasicEvaluator)(nil)

var CreateBasicEvaluator EvaluatorConstructor = func(game *game.GameState) (func(), Evaluator) {
	return func() {}, BasicEvaluator{}
}

func (e BasicEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error) {
	return nil, Evaluate(helper.GameState.Bitboards, player), NilError
}

type QuiescenceEvaluator struct {
}

var _ Evaluator = (*QuiescenceEvaluator)(nil)

func (e QuiescenceEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error) {
	if len(pastMoves) > 0 {
		lastMove := pastMoves[len(pastMoves)-1]
		if !lastMove.MoveType.Captures() {
			return nil, Evaluate(helper.GameState.Bitboards, player), NilError
		}
	}

	prevInQuiescence := helper.InQuiescence
	prevEvaluator := helper.Evaluator
	prevSorter := helper.MoveSorter

	helper.InQuiescence = true
	helper.Evaluator = BasicEvaluator{}
	helper.MoveSorter = helper.MoveSorter.copy()

	defer func() {
		helper.InQuiescence = prevInQuiescence
		helper.Evaluator = prevEvaluator
		helper.MoveSorter = prevSorter
	}()

	quiescenceDepth := helper.MaxDepth.ValueOr(defaultMaxDepth) * 8
	// quiescenceDepth := 10

	moves, score, err := helper.alphaBeta(alpha, beta, currentDepth,
		quiescenceDepth,
		pastMoves,
	)
	return moves, score, err
}
