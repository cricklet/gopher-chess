package searchv3

import (
	"strings"

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

type MoveGen interface {
	forEachMove(errs ErrorRef, callback func(move Move) LoopResult)
}

type DefaultMoveGenerator struct {
	*GameState
	*Bitboards
}

var _ MoveGen = (*DefaultMoveGenerator)(nil)

func (gen DefaultMoveGenerator) forEachMove(errs ErrorRef, callback func(move Move) LoopResult) {
	if errs.HasError() {
		return
	}

	moves := search.GetMovesBuffer()
	defer search.ReleaseMovesBuffer(moves)

	search.GeneratePseudoMoves(func(m Move) {
		*moves = append(*moves, m)
	}, gen.Bitboards, gen.GameState)

	for _, move := range *moves {
		result := LoopContinue
		func() {
			var update BoardUpdate
			errs.Add(
				gen.GameState.PerformMove(move, &update, gen.Bitboards))

			defer func() {
				errs.Add(
					gen.GameState.UndoUpdate(&update, gen.Bitboards))
			}()

			if errs.HasError() {
				result = LoopBreak
			} else if !search.KingIsInCheck(gen.Bitboards, gen.GameState.Enemy()) {
				result = callback(move) // move is legal
			}
		}()
		if result == LoopBreak {
			break
		}
	}

	return
}

type SearchTreeMoveGenerator struct {
	SearchTree
	*GameState
	*Bitboards
	currentlySearching *SearchTree
}

var _ MoveGen = (*SearchTreeMoveGenerator)(nil)

func (gen *SearchTreeMoveGenerator) forEachMove(errs ErrorRef, callback func(move Move) LoopResult) {
	if errs.HasError() {
		return
	}

	if gen.currentlySearching == nil {
		gen.currentlySearching = &gen.SearchTree
	}

	if gen.currentlySearching.continueSearching {
		DefaultMoveGenerator{gen.GameState, gen.Bitboards}.forEachMove(errs, callback)
		return
	}

	prevSearchTree := gen.currentlySearching
	for nextMove, nextSearchTree := range gen.currentlySearching.moves {
		result := LoopContinue
		func() {
			gen.currentlySearching = nextSearchTree
			move := gen.GameState.MoveFromString(nextMove)

			var update BoardUpdate
			errs.Add(
				gen.GameState.PerformMove(move, &update, gen.Bitboards))

			defer func() {
				gen.currentlySearching = prevSearchTree
				errs.Add(
					gen.GameState.UndoUpdate(&update, gen.Bitboards))
			}()

			if errs.HasError() {
				result = LoopBreak
			} else if !search.KingIsInCheck(gen.Bitboards, gen.GameState.Enemy()) {
				result = callback(move) // move is legal
			}
		}()
		if result == LoopBreak {
			break
		}
	}
	return
}

type SearchHelper struct {
	MoveGen
	*GameState
	*Bitboards
	OutOfTime *bool
	Logger
	MaxDepth int
}

func (helper SearchHelper) evaluateWhite() int {
	return search.Evaluate(helper.Bitboards, White)
}
func (helper SearchHelper) evaluateCurrentPlayer() int {
	return search.Evaluate(helper.Bitboards, helper.GameState.Player)
}

func (helper SearchHelper) String() string {
	return helper.GameState.Board.String()
}

func (helper SearchHelper) inCheck() bool {
	return search.KingIsInCheck(helper.Bitboards, helper.GameState.Player)
}

func alphaBeta(errs ErrorRef, helper SearchHelper, alpha int, beta int, currentDepth int, maxDepth int) ([]Move, int) {
	if currentDepth >= maxDepth {
		return []Move{}, helper.evaluateCurrentPlayer()
	}

	if errs.HasError() {
		return []Move{}, alpha
	}

	principleVariation := []Move{}

	foundMove := false

	helper.forEachMove(errs, func(move Move) LoopResult {
		foundMove = true
		helper.Println(strings.Repeat(" ", currentDepth), "?", move.String())

		variation, enemyScore := alphaBeta(errs, helper, -beta, -alpha, currentDepth+1, maxDepth)

		score := -enemyScore
		if score >= beta {
			alpha = beta // fail hard beta-cutoff
			helper.Println(strings.Repeat(" ", currentDepth), ">", score, move.String(), "beta cutoff")
			return LoopBreak
		} else if score > alpha {
			alpha = score
			principleVariation = append([]Move{move}, variation...)
			helper.Println(strings.Repeat(" ", currentDepth), ">", score, move.String(), "principle variation", principleVariation[1:])
		} else {
			helper.Println(strings.Repeat(" ", currentDepth), ">", score, move.String(), "skip")
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

func findPrincipleVariation(errRef ErrorRef, helper SearchHelper, currentDepth int, maxDepth int) ([]Move, int) {
	// player := helper.GameState.Player
	// if player == White {
	// 	return alphaBetaMax(errRef, helper, -100000, 100000, maxDepth)
	// } else {
	// 	variation, score := alphaBetaMin(errRef, helper, -100000, 100000, maxDepth)
	// 	return variation, -score
	// }

	return alphaBeta(errRef, helper, -search.Inf-1, search.Inf+1, currentDepth, maxDepth)
}

type SearchOption interface {
	apply(helper *SearchHelper)
}

type WithDebugLogging struct {
}

func (o WithDebugLogging) apply(helper *SearchHelper) {
	helper.Logger = &DefaultLogger
}

type WithQuiescence struct {
}

func (o WithQuiescence) apply(helper *SearchHelper) {
}

type WithMaxDepth struct {
	MaxDepth int
}

func (o WithMaxDepth) apply(helper *SearchHelper) {
	helper.MaxDepth = o.MaxDepth
}

type SearchTree struct {
	moves             map[string]*SearchTree
	continueSearching bool
}

func SearchTreeFromLines(
	startingFen string,
	lines [][]string,
	continueSearchingPastLines bool,
) (SearchTree, Error) {
	result := SearchTree{
		moves:             map[string]*SearchTree{},
		continueSearching: false,
	}

	for _, line := range lines {
		currentTree := &result
		for _, move := range line {
			if nextTree, contains := currentTree.moves[move]; contains {
				currentTree = nextTree
			} else {
				currentTree.moves[move] = &SearchTree{
					moves:             map[string]*SearchTree{},
					continueSearching: false,
				}
				currentTree = currentTree.moves[move]
			}
		}

		if continueSearchingPastLines {
			currentTree.continueSearching = true
		}
	}

	return result, Error{}
}

type WithSearch struct {
	search SearchTree
}

func (o WithSearch) apply(helper *SearchHelper) {
	helper.MoveGen = &SearchTreeMoveGenerator{
		o.search,
		helper.GameState,
		helper.Bitboards,
		nil,
	}
}

type WithOutOfTime struct {
	OutOfTime *bool
}

func (o WithOutOfTime) apply(helper *SearchHelper) {
	helper.OutOfTime = o.OutOfTime
}

func Search(fen string, opts ...SearchOption) ([]Move, int, Error) {
	game, err := GamestateFromFenString(fen)
	if !err.IsNil() {
		return []Move{}, 0, err
	}

	bitboards := game.CreateBitboards()

	errRef := ErrorRef{}
	helper := SearchHelper{
		MoveGen: DefaultMoveGenerator{
			&game,
			&bitboards,
		},
		MaxDepth:  3,
		GameState: &game,
		Bitboards: &bitboards,
		Logger:    &SilentLogger,
	}

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
		helper.Println("!", move.String())

		variation, enemyScore := findPrincipleVariation(
			errRef,
			helper,
			// current depth is 1 (0 would be before we applied `move`)
			1,
			helper.MaxDepth)
		if errRef.HasError() {
			return LoopBreak
		}

		score := -enemyScore
		helper.Println(">", score, move, "principle variation", variation)
		if score > bestScore {
			bestScore = score
			principleVariation = append([]Move{move}, variation...)
		}

		return LoopContinue
	})

	return principleVariation, bestScore, errRef.Error()
}
