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

	eval, err := e.stock.Eval()
	if !IsNil(err) {
		return nil, 0, err
	}

	return nil, eval, NilError
}

type BasicEvaluator struct {
}

var _ Evaluator = (*BasicEvaluator)(nil)

func (e BasicEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error) {
	// NEXT: maybe try stockfish NNUE evaluation so I can just focus on alpha beta
	return nil, Evaluate(helper.Bitboards, player), NilError
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

	quiescenceDepth := helper.IterativeDeepeningDepth * 8
	// quiescenceDepth := 10

	moves, score, err := helper.alphaBeta(alpha, beta, currentDepth,
		quiescenceDepth,
		pastMoves,
	)
	return moves, score, err

	/*
	   if helper.WithoutIterativeDeepeningInQuiescence {
	   }

	   principleVariations := []Pair[int, []SearchMove]{}
	   mode := OnlyCaptures

	   unregisterCounter, counter := NewMoveCounter(helper.GameState)
	   defer unregisterCounter()

	   lastCount := Empty[int]()

	   cleanup, result, moves, err := helper.MoveGen.generateMoves(mode)
	   defer cleanup()

	   	if result != SomeLegalMoves {
	   		return nil, alpha, Errorf("quiescence should only search captures")
	   	}

	   	if err.HasError() {
	   		return nil, alpha, err
	   	}

	   // Loop through & perform first generated moves

	   	for depthRemaining := quiescenceDepth; depthRemaining <= quiescenceDepth; depthRemaining += 1 {
	   		if lastCount.HasValue() {
	   			if counter.NumMoves() == lastCount.Value() {
	   				break
	   			}
	   		}

	   		err = helper.MoveSorter.sortMoves(moves)
	   		if err.HasError() {
	   			return nil, alpha, err
	   		}

	   		for _, move := range *moves {
	   			undo, legal, err := performMoveAndReturnLegality(helper.GameState, helper.Bitboards, move)
	   			if err.HasError() {
	   				return nil, alpha, err
	   			}

	   			if legal {
	   				// Traverse past the first generated move
	   				variation, enemyScore, err := helper.alphaBeta(
	   					alpha, beta,
	   					currentDepth,
	   					depthRemaining-1,
	   					pastMoves,
	   				)

	   				if err.HasError() {
	   					return nil, alpha, err
	   				}

	   				score := -enemyScore
	   				principleVariations = append(principleVariations, Pair[int, []SearchMove]{
	   					First: score, Second: append([]SearchMove{{move, false}}, variation...)})
	   			}

	   			err = undo()
	   			if err.HasError() {
	   				return nil, alpha, err
	   			}
	   		}

	   		if err.HasError() {
	   			return nil, 0, err
	   		}

	   		if len(principleVariations) == 0 {
	   			return helper.Evaluator.evaluate(helper, player, alpha, beta, currentDepth, pastMoves)
	   		}

	   		SortMaxFirst(&principleVariations, func(t Pair[int, []SearchMove]) int {
	   			return t.First
	   		})

	   		// Prioritize the newly discovered principle variations first
	   		helper.MoveSorter.reset(principleVariations)

	   		lastCount = Some(counter.NumMoves())
	   		counter.Reset()
	   	}

	   bestMove := principleVariations[0]
	   return bestMove.Second, bestMove.First, NilError
	*/
}
