package search

import (
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/stockfish"
)

type Evaluator interface {
	evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error)
}

type StockfishEvaluator struct {
	stock *stockfish.StockfishRunner
}

var _ Evaluator = (*StockfishEvaluator)(nil)

var CreateStockfishEvaluator EvaluatorConstructor = func(game *game.GameState) (func(), Evaluator) {
	return func() {}, StockfishEvaluator{}
}

func (e StockfishEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error) {
	var err Error
	if e.stock == nil {
		e.stock, err = stockfish.NewStockfishRunner(
			stockfish.WithLogger(&SilentLogger),
		)
		if !IsNil(err) {
			return nil, 0, err
		}
	}

	fen := game.FenStringForGame(helper.GameState)
	err = e.stock.SetupPosition(Position{Fen: fen, Moves: nil})
	if !IsNil(err) {
		return nil, 0, err
	}

	eval, err := e.stock.Eval(helper.GameState.Player)
	if !IsNil(err) {
		return nil, 0, err
	}

	return nil, eval.PlayerScore, NilError
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
