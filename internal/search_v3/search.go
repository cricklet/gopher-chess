package searchv3

import (
	"fmt"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/search"
)

/*
alpha/beta w/ caching from the ground up

           a
       /        \     <-- white moves
      b          c
   /   \       /   \   <-- black moves
  d     e     f     g
 / \   / \   / \   / \  <-- white move
h   i j   k l   m n   o

eval(x) = evaluation function for white
maximize(x, i) = search from x, choosing the move that maximizes the evaluation
minimize(y, j) = search from y, choosing the move that minimizes the evaluation

maximize(a, 2) = search from a, with 2 ply
  => white-move a=>b
    minimize(b, 1)
      => black-move b=>d -> maximize(d, 0) -> eval(d)
      => black-move b=>e -> maximize(e, 0) -> eval(e)
  => white-move a=>c
    minimize(c, 1)
      => black-move c=>f -> maximize(f, 0) -> eval(f)
      => black-move c=>g -> maximize(g, 0) -> eval(g)

by the time we're investigating a=>c, we already know the expected result of a=>b
  (eg white eval lower bound, eg score white can force via a=>b)

if black's c=>f move is better for black than the expected result white's a=>c
  this is a refutation move
  in this case, white won't play a=>c and we can ignore this whole branch
  ^ this is the only pruning we're allowed to do!

by the time we're investigating black's c=>g move, we know:
  the best score white can force via a=>b (alpha, eg white eval lower bound)
  the best score black can force via c=>f (beta, eg white eval upper bound)
  this means that
    when we're minimizing (eg minimize(c, j))
      we can early exit if we find a black move that results in a alpha cutoff (eg worse for white than alpha)
      we can ignore results worse for black than beta

similarly, if we're investigating white's future f=>m move, we know:
  the best score white can previously force (either via a=>b or via a=>c=>f=>l) (alpha, eg white eval lower bound)
  the best score black can force (beta, eg white eval upper bound)
  this means that
    when we're maximizing (eg maximize(f, i))
      we can early exit if we find a white move that results in an beta cut-off
      we can ignore results worse for white than alpha
*/

/*
maximize(a, i)
  we find the best move for white and return it
	minimize(b, i - 1)
      we find the best move for black and return it

maximize(board, depth) -> principle-variation, score
minimize(board, depth) -> principle-variation, score
*/

type LoopResult int

const (
	LoopContinue LoopResult = iota
	LoopBreak
)

type SearchHelper interface {
	evaluateWhite() int
	evaluateCurrentPlayer() int
	forEachMove(errs ErrorRef, callback func(move Move) LoopResult)
	inCheck() bool
}

type SearchHelperImpl struct {
	Game      *GameState
	Bitboards *Bitboards
	OutOfTime *bool
	MaxDepth  Optional[int]
}

var _ SearchHelper = (*SearchHelperImpl)(nil)

func (helper SearchHelperImpl) evaluateWhite() int {
	return search.Evaluate(helper.Bitboards, White)
}
func (helper SearchHelperImpl) evaluateCurrentPlayer() int {
	return search.Evaluate(helper.Bitboards, helper.Game.Player)
}

func (helper SearchHelperImpl) String() string {
	return helper.Game.Board.String()
}

func (helper SearchHelperImpl) inCheck() bool {
	return search.KingIsInCheck(helper.Bitboards, helper.Game.Player)
}

func (helper SearchHelperImpl) forEachMove(errs ErrorRef, callback func(move Move) LoopResult) {
	if errs.HasError() {
		return
	}

	moves := search.GetMovesBuffer()
	defer search.ReleaseMovesBuffer(moves)

	search.GeneratePseudoMoves(func(m Move) {
		*moves = append(*moves, m)
	}, helper.Bitboards, helper.Game)

	for _, move := range *moves {
		result := LoopContinue
		func() {
			var update BoardUpdate
			errs.Add(
				helper.Game.PerformMove(move, &update, helper.Bitboards))

			defer func() {
				errs.Add(
					helper.Game.UndoUpdate(&update, helper.Bitboards))
			}()

			if errs.HasError() {
				result = LoopBreak
			} else if !search.KingIsInCheck(helper.Bitboards, helper.Game.Enemy()) {
				result = callback(move) // move is legal
			}
		}()
		if result == LoopBreak {
			break
		}
	}

	return
}

func alphaBeta(errs ErrorRef, helper SearchHelper, alpha int, beta int, depthleft int) ([]Move, int) {
	if depthleft == 0 {
		return []Move{}, helper.evaluateCurrentPlayer()
	}

	if errs.HasError() {
		return []Move{}, alpha
	}

	principleVariation := []Move{}

	foundMove := false

	helper.forEachMove(errs, func(move Move) LoopResult {
		foundMove = true

		variation, enemyScore := alphaBeta(errs, helper, -beta, -alpha, depthleft-1)
		score := -enemyScore
		if score >= beta {
			alpha = beta // fail hard beta-cutoff
			return LoopBreak
		} else if score > alpha {
			alpha = score
			principleVariation = append([]Move{move}, variation...)
		}
		return LoopContinue
	})

	if !foundMove {
		if helper.inCheck() {
			alpha = -search.Inf
		} else {
			alpha = 0
		}
	}

	return principleVariation, alpha
}

// func alphaBetaMax(errs ErrorRef, helper SearchHelper, alpha int, beta int, depthleft int) ([]Move, int) {
// 	if depthleft == 0 {
// 		return []Move{}, helper.evaluateWhite()
// 	}

// 	if errs.HasError() {
// 		return []Move{}, alpha
// 	}

// 	principleVariation := []Move{}

// 	helper.forEachMove(errs, func(move Move) LoopResult {
// 		variation, score := alphaBetaMin(errs, helper, alpha, beta, depthleft-1)
// 		if score >= beta {
// 			alpha = beta // fail hard beta-cutoff
// 			return LoopBreak
// 		}
// 		if score > alpha {
// 			alpha = score // alpha acts like max in MiniMax
// 			principleVariation = append([]Move{move}, variation...)
// 		}
// 		return LoopContinue
// 	})

// 	return principleVariation, alpha
// }

// func alphaBetaMin(errs ErrorRef, helper SearchHelper, alpha int, beta int, depthleft int) ([]Move, int) {
// 	if depthleft == 0 {
// 		return []Move{}, helper.evaluateWhite()
// 	}

// 	if errs.HasError() {
// 		return []Move{}, alpha
// 	}

// 	principleVariation := []Move{}

// 	helper.forEachMove(errs, func(move Move) LoopResult {
// 		variation, score := alphaBetaMax(errs, helper, alpha, beta, depthleft-1)
// 		if score <= alpha {
// 			beta = alpha // fail hard alpha-cutoff
// 			return LoopBreak
// 		}
// 		if score < beta {
// 			beta = score // beta acts like min in MiniMax
// 			principleVariation = append([]Move{move}, variation...)
// 		}

// 		return LoopContinue
// 	})

// 	return principleVariation, beta
// }

func findPrincipleVariation(errRef ErrorRef, helper SearchHelperImpl) ([]Move, int) {
	// player := helper.Game.Player
	// if player == White {
	// 	return alphaBetaMax(errRef, helper, -100000, 100000, helper.MaxDepth.ValueOr(3))
	// } else {
	// 	variation, score := alphaBetaMin(errRef, helper, -100000, 100000, helper.MaxDepth.ValueOr(3))
	// 	return variation, -score
	// }

	return alphaBeta(errRef, helper, -search.Inf, search.Inf, helper.MaxDepth.ValueOr(3))
}

type SearchOption interface {
	apply(helper *SearchHelperImpl)
}

type WithMaxDepth struct {
	MaxDepth int
}

func (o WithMaxDepth) apply(helper *SearchHelperImpl) {
	helper.MaxDepth = Some(o.MaxDepth)
}

type WithOutOfTime struct {
	OutOfTime *bool
}

func (o WithOutOfTime) apply(helper *SearchHelperImpl) {
	helper.OutOfTime = o.OutOfTime
}

func Search(fen string, opts ...SearchOption) ([]Move, int, Error) {
	game, err := GamestateFromFenString(fen)
	if !err.IsNil() {
		return []Move{}, 0, err
	}

	bitboards := game.CreateBitboards()

	errRef := ErrorRef{}
	helper := SearchHelperImpl{Game: &game, Bitboards: &bitboards}

	for _, opt := range opts {
		opt.apply(&helper)
	}

	bestScore := -search.Inf

	principleVariation := []Move{}

	helper.forEachMove(errRef, func(move Move) LoopResult {
		if helper.OutOfTime != nil && *helper.OutOfTime {
			return LoopBreak
		}

		if errRef.HasError() {
			return LoopBreak
		}

		variation, enemyScore := findPrincipleVariation(errRef, helper)
		if errRef.HasError() {
			return LoopBreak
		}

		score := -enemyScore
		fmt.Println(score, move, variation)
		if score > bestScore {
			bestScore = score
			principleVariation = append([]Move{move}, variation...)
		}

		return LoopContinue
	})

	return principleVariation, bestScore, errRef.Error()
}
