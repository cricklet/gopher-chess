package search

import (
	"strings"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
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

func performMoveAndCall[T any](g *GameState, b *Bitboards, move Move, callback func(move Move) (T, Error)) (T, Error) {
	var result T
	var update BoardUpdate
	err := g.PerformMove(move, &update, b)
	if !err.IsNil() {
		return result, err
	}

	if !KingIsInCheck(b, g.Enemy()) {
		result, err = callback(move) // move is legal, run the callback
	}

	if !err.IsNil() {
		return result, err
	}

	err = g.UndoUpdate(&update, b)
	return result, err
}

type LoopResult int

const (
	LoopContinue LoopResult = iota
	LoopBreak
)

type MoveGen interface {
	performEachMoveAndCall(callback func(move Move) (LoopResult, Error)) Error
	searchingAllLegalMoves() bool
}

type DefaultMoveGenerator struct {
	*GameState
	*Bitboards
	onlyCaptures bool

	sortedVariations [][]Move
	currentVariation []Move
	currentDepth     int
}

var _ MoveGen = (*DefaultMoveGenerator)(nil)

func (gen DefaultMoveGenerator) searchingAllLegalMoves() bool {
	if gen.onlyCaptures {
		return false
	} else {
		return true
	}
}

func (gen DefaultMoveGenerator) performEachMoveAndCall(callback func(move Move) (LoopResult, Error)) Error {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	if gen.onlyCaptures {
		GeneratePseudoCaptures(func(m Move) {
			*moves = append(*moves, m)
		}, gen.Bitboards, gen.GameState)
	} else {
		GeneratePseudoMoves(func(m Move) {
			*moves = append(*moves, m)
		}, gen.Bitboards, gen.GameState)
	}

	if gen.currentDepth == 0 {
		// sort moves in the order of sortedVariations
	} else if gen.currentDepth < len(gen.currentVariation) {
		// we are currently in a variation -- prioritize the next move in it
	} else {
	}

	// NEXT

	for _, move := range *moves {
		gen.currentDepth += 1
		result, err := performMoveAndCall(gen.GameState, gen.Bitboards, move, callback)
		gen.currentDepth -= 1

		if !err.IsNil() {
			return err
		}
		if result == LoopBreak {
			break
		}
	}

	return NilError
}

type SearchTreeMoveGenerator struct {
	SearchTree
	*GameState
	*Bitboards
	currentlySearching *SearchTree
}

var _ MoveGen = (*SearchTreeMoveGenerator)(nil)

func (gen *SearchTreeMoveGenerator) searchingAllLegalMoves() bool {
	if gen.currentlySearching.continueSearching {
		return true
	} else {
		return false
	}
}

func (gen *SearchTreeMoveGenerator) performEachMoveAndCall(callback func(move Move) (LoopResult, Error)) Error {
	if gen.currentlySearching == nil {
		gen.currentlySearching = &gen.SearchTree
	}

	if gen.currentlySearching.continueSearching {
		return DefaultMoveGenerator{gen.GameState, gen.Bitboards, false}.performEachMoveAndCall(callback)
	}

	prevSearchTree := gen.currentlySearching
	for nextMoveStr, nextSearchTree := range gen.currentlySearching.moves {
		gen.currentlySearching = nextSearchTree

		nextMove := gen.GameState.MoveFromString(nextMoveStr)
		result, err := performMoveAndCall(gen.GameState, gen.Bitboards, nextMove, callback)

		gen.currentlySearching = prevSearchTree

		if !err.IsNil() {
			return err
		}
		if result == LoopBreak {
			break
		}
	}

	return NilError
}

type Evaluator interface {
	evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int) ([]Move, int, Error)
}

type BasicEvaluator struct {
}

var _ Evaluator = (*BasicEvaluator)(nil)

func (e BasicEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int) ([]Move, int, Error) {
	return []Move{}, Evaluate(helper.Bitboards, player), NilError
}

type QuiescenceEvaluator struct {
}

var _ Evaluator = (*QuiescenceEvaluator)(nil)

func (e QuiescenceEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int) ([]Move, int, Error) {
	captureGenerator := DefaultMoveGenerator{helper.GameState, helper.Bitboards, true /*onlyCaptures*/}
	quiescenceHelper := SearchHelper{
		captureGenerator,
		BasicEvaluator{},
		helper.GameState,
		helper.Bitboards,
		nil,  // helper.OutOfTime,
		true, // helper.CheckStandPat
		helper.Logger,
		&SilentLogger,
		currentDepth + 6, // allow deep capture searching
	}

	moves, score, err := quiescenceHelper.alphaBeta(alpha, beta, currentDepth)
	return moves, score, err
}

type SearchHelper struct {
	MoveGen MoveGen
	// MoveSorter MoveSorter
	Evaluator     Evaluator
	GameState     *GameState
	Bitboards     *Bitboards
	OutOfTime     *bool
	CheckStandPat bool
	Logger
	Debug    Logger
	MaxDepth int
}

func (helper SearchHelper) String() string {
	return helper.GameState.Board.String()
}

func (helper SearchHelper) inCheck() bool {
	return KingIsInCheck(helper.Bitboards, helper.GameState.Player)
}

func (helper *SearchHelper) alphaBeta(alpha int, beta int, currentDepth int) ([]Move, int, Error) {
	if currentDepth >= helper.MaxDepth {
		return helper.Evaluator.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth)
	}

	if helper.CheckStandPat {
		// if we decide not to not take (eg make a neutral move / stand-pat)
		// and that's really good for us (eg other player will have prevented this path)
		//   we can return early
		// if it's good for us but not so good the other player can prevent this path
		//   we need to search captures
		//   but we can also update alpha
		//   because the future capture must beat standing pat in order for us to choose it
		// if it's bad for us, we need to search captures
		_, standPat, err := BasicEvaluator{}.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth)
		if err.HasError() {
			return []Move{}, alpha, err
		}

		if standPat >= beta {
			return []Move{}, beta, NilError
		} else if standPat > alpha {
			alpha = standPat
		}
	}

	principleVariation := []Move{}

	foundMove := false

	err := helper.MoveGen.performEachMoveAndCall(func(move Move) (LoopResult, Error) {
		foundMove = true
		helper.Debug.Println(strings.Repeat(" ", currentDepth), "?", move.DebugString())

		variation, enemyScore, err := helper.alphaBeta(-beta, -alpha, currentDepth+1)
		if err.HasError() {
			return LoopBreak, err
		}

		score := -enemyScore
		if score >= beta {
			alpha = beta // fail hard beta-cutoff
			helper.Debug.Println(strings.Repeat(" ", currentDepth), ">", score, move.DebugString(), "beta cutoff")
			return LoopBreak, NilError
		} else if score > alpha {
			alpha = score
			principleVariation = append([]Move{move}, variation...)
			helper.Debug.Println(strings.Repeat(" ", currentDepth), ">", score, move.DebugString(), "principle variation", principleVariation[1:])
		} else {
			helper.Debug.Println(strings.Repeat(" ", currentDepth), ">", score, move.DebugString(), "skip")
		}
		return LoopContinue, NilError
	})

	if err.HasError() {
		return []Move{}, alpha, err
	}

	if !foundMove {
		if helper.MoveGen.searchingAllLegalMoves() {
			// If no legal moves exist, we're in stalemate or checkmate
			if helper.inCheck() {
				alpha = -Inf
			} else {
				alpha = 0
			}
		} else {
			return helper.Evaluator.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth)
		}
	}

	return principleVariation, alpha, NilError
}

func (helper *SearchHelper) Search() ([]Move, int, Error) {
	availableMoves := []Pair[int, []Move]{}

	// NEXT: iterative search, searching the best variations first
	// NEXT: split the generator from the move sorter. give move sort information about previous principle variations
	// NEXT: to do this, give the generator more info

	/*
		record scores for all variations

		at depth 1
		for each generated move
			find score / variation

		for each next depth
			generate moves
			sort based on best previous variations
			find score / variation
			update recorded score for variations

		there's a problem though...
			the move generator applies the moves directly
			it would be nice if that could be separated from the move ordering...

			MoveGen.generateMoves(moves)
			MoveSort.sortMoves(moves)
			and performEachMove(moves)

		in order for the SearchTreeGenerator to work
			we need to know where we are in the search tree
			we need to know the previously searched moves

		in order for MoveSort to correctly sort the moves based on the previous principle variations...
			we similarly need to know where we are in the search history
			we need info about the previous principle variations to be passed in

		hmm, it is nice to have the move generator apply the moves directly
			that makes it easy for it to store some stateful information about where we are in the traversal

		i can get that too if you can pass a sorting function into the generator
	*/

	// moves := GetMovesBuffer()
	// defer ReleaseMovesBuffer(moves)
	// helper.MoveGen.generateMoves([]Move{}, moves)

	err := helper.MoveGen.performEachMoveAndCall(func(move Move) (LoopResult, Error) {
		if helper.OutOfTime != nil && *helper.OutOfTime {
			return LoopBreak, NilError
		}

		variation, enemyScore, err := findPrincipleVariation(
			*helper,
			// current depth is 1 (0 would be before we applied `move`)
			1)
		if err.HasError() {
			return LoopBreak, err
		}

		score := -enemyScore
		availableMoves = append(availableMoves, Pair[int, []Move]{
			First: score, Second: append([]Move{move}, variation...)})

		return LoopContinue, NilError
	})

	if err.HasError() {
		return []Move{}, 0, err
	}

	SortMaxFirst(&availableMoves, func(t Pair[int, []Move]) int {
		return t.First
	})

	for _, move := range availableMoves {
		helper.Println(">", move.First, move.Second)
	}

	if len(availableMoves) == 0 {
		return []Move{}, 0, NilError
	}

	bestMove := availableMoves[0]
	return bestMove.Second, bestMove.First, NilError
}

func findPrincipleVariation(helper SearchHelper, currentDepth int) ([]Move, int, Error) {
	return helper.alphaBeta(-Inf-1, Inf+1, currentDepth)
}

type SearchOption interface {
	apply(helper *SearchHelper)
}

type WithDebugLogging struct {
}

func (o WithDebugLogging) apply(helper *SearchHelper) {
	if helper.Logger == &SilentLogger {
		helper.Logger = &DefaultLogger
	}
	helper.Debug = &DefaultLogger
}

type WithLogger struct {
	Logger Logger
}

func (o WithLogger) apply(helper *SearchHelper) {
	helper.Logger = o.Logger
}

type WithoutQuiescence struct {
}

func (o WithoutQuiescence) apply(helper *SearchHelper) {
	helper.Evaluator = BasicEvaluator{}
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

func Searcher(game *GameState, b *Bitboards, opts ...SearchOption) *SearchHelper {
	defaultMoveGenerator := DefaultMoveGenerator{
		game,
		b,
		false,
	}
	quiescenceEvaluator := QuiescenceEvaluator{}
	helper := SearchHelper{
		MoveGen:       defaultMoveGenerator,
		Evaluator:     quiescenceEvaluator,
		GameState:     game,
		Bitboards:     b,
		OutOfTime:     nil,
		CheckStandPat: false,
		Logger:        &SilentLogger,
		Debug:         &SilentLogger,
		MaxDepth:      4,
	}

	for _, opt := range opts {
		opt.apply(&helper)
	}
	return &helper
}

func Search(fen string, opts ...SearchOption) ([]Move, int, Error) {
	game, err := GamestateFromFenString(fen)
	if !err.IsNil() {
		return []Move{}, 0, err
	}

	bitboards := game.CreateBitboards()
	helper := Searcher(&game, &bitboards, opts...)

	return helper.Search()
}
