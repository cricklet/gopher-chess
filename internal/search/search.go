package search

import (
	"fmt"

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

func performMoveAndReturnLegality(g *GameState, move Move) (func() Error, bool, Error) {
	var update BoardUpdate
	err := g.PerformMove(move, &update)
	if !err.IsNil() {
		return func() Error { return NilError }, false, err
	}

	undo := func() Error {
		return g.UndoUpdate(&update)
	}

	if !KingIsInCheck(g.Bitboards, g.Enemy()) {
		return undo, true, NilError
	}

	return undo, false, NilError
}

type MoveGenerationMode int

const (
	AllMoves MoveGenerationMode = iota
	OnlyCaptures
)

type MoveGenerationResult int

const (
	AllLegalMoves MoveGenerationResult = iota
	SomeLegalMoves
)

type MoveGen interface {
	generateMoves(mode MoveGenerationMode) (func(), MoveGenerationResult, *[]Move, Error)
}

type MoveSorter interface {
	sortMoves(moves *[]Move) Error
	reset(variations []Pair[int, []SearchMove])
	copy() MoveSorter
}

type NoOpMoveSorter struct {
	noCopy NoCopy
}

var _ MoveSorter = (*NoOpMoveSorter)(nil)

func (s *NoOpMoveSorter) sortMoves(moves *[]Move) Error {
	return NilError
}

func (s *NoOpMoveSorter) reset(variations []Pair[int, []SearchMove]) {
}

func (s *NoOpMoveSorter) copy() MoveSorter {
	return &NoOpMoveSorter{}
}

type SearchHelper struct {
	MoveGen      MoveGen
	MoveSorter   MoveSorter
	Evaluator    Evaluator
	GameState    *GameState
	OutOfTime    *bool
	InQuiescence bool
	Logger
	Debug                     Logger
	IterativeDeepeningDepth   int
	WithoutIterativeDeepening bool
	WithoutCheckStandPat      bool

	noCopy NoCopy
}

func (helper *SearchHelper) String() string {
	return helper.GameState.Board.String()
}

func (helper *SearchHelper) inCheck() bool {
	return KingIsInCheck(helper.GameState.Bitboards, helper.GameState.Player)
}

var LABEL_COL int = 7
var SCORE_COL int = 5

type SearchMove struct {
	Move
	inQuiescence bool
}

func (move SearchMove) String() string {
	return move.DebugString()
}

func VariationDebugString(searchMoves []SearchMove, currentIndex int, label string, score Optional[int]) string {
	allMoves := ""
	inQuiescence := false
	for i, move := range searchMoves {
		prefix := ""
		postfix := ""

		if move.inQuiescence && !inQuiescence {
			prefix += "<"
			inQuiescence = true
		}
		if i == currentIndex {
			prefix += "["
			postfix += "]"
		}
		if i == len(searchMoves)-1 && move.inQuiescence {
			postfix += ">"
		}

		allMoves += fmt.Sprintf("%2s%-7s", prefix, fmt.Sprint(move.DebugString(), postfix))
	}

	scoreLabel := ""
	if score.HasValue() {
		scoreLabel = ScoreString(score.Value())
	}

	return PrintColumns(
		[]string{
			label,
			scoreLabel,
			allMoves,
		},
		[]int{LABEL_COL, SCORE_COL},
		"",
	)
}

func (helper *SearchHelper) PrintlnVariation(logger Logger,
	past []SearchMove,
	current Optional[SearchMove],
	future []SearchMove,
	label string,
	score Optional[int],
) {
	if logger == &SilentLogger {
		return
	}

	fullVariation := []SearchMove{}
	fullVariation = append(fullVariation, past...)
	if current.HasValue() {
		fullVariation = append(fullVariation, current.Value())
	}
	fullVariation = append(fullVariation, future...)

	currentIndex := Inf
	if len(future) > 0 || current.HasValue() {
		currentIndex = len(past)
	}

	result := VariationDebugString(fullVariation, currentIndex, label, score)
	if current.IsEmpty() {
		result += " [****]"
	}
	logger.Println(result)
}

func (helper *SearchHelper) alphaBeta(alpha int, beta int, currentDepth int, depthRemaining int, past []SearchMove) ([]SearchMove, int, Error) {
	if helper.OutOfTime != nil && *helper.OutOfTime {
		return nil, Evaluate(helper.GameState.Bitboards, helper.GameState.Player), NilError
	}

	if depthRemaining <= 0 {
		future, score, err := helper.Evaluator.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth, past)
		// helper.PrintlnVariation(helper.Debug, past, Empty[SearchMove](), future, "eval", Some(score))
		return future, score, err
	}

	if helper.InQuiescence && !helper.WithoutCheckStandPat {
		// if we decide not to not take (eg make a neutral move / stand-pat)
		// and that's really good for us (eg other player will have prevented this path)
		//   we can return early
		// if it's good for us but not so good the other player can prevent this path
		//   we need to search captures
		//   but we can also update alpha
		//   because the future capture must beat standing pat in order for us to choose it
		// if it's bad for us, we need to search captures
		_, standPat, err := BasicEvaluator{}.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth, past)
		if err.HasError() {
			return nil, alpha, err
		}

		if standPat >= beta {
			helper.PrintlnVariation(helper.Debug, past, Empty[SearchMove](), nil, "sp-b-cut", Some(standPat))
			return nil, beta, NilError
		} else if standPat > alpha {
			helper.PrintlnVariation(helper.Debug, past, Empty[SearchMove](), nil, "sp-alpha", Some(standPat))
			alpha = standPat
		}
	}

	var principleVariation []SearchMove = nil

	mode := AllMoves
	if helper.InQuiescence {
		mode = OnlyCaptures
	}

	foundMove := false

	cleanup, result, moves, err := helper.MoveGen.generateMoves(mode)
	defer cleanup()

	if err.HasError() {
		return nil, alpha, err
	}

	err = helper.MoveSorter.sortMoves(moves)
	if err.HasError() {
		return nil, alpha, err
	}

	for _, move := range *moves {
		betaCutoff := false
		searchMove := SearchMove{move, helper.InQuiescence}

		helper.PrintlnVariation(helper.Debug, past, Some(searchMove), nil, "???", Empty[int]())

		undo, legal, err := performMoveAndReturnLegality(helper.GameState, move)
		if err.HasError() {
			return nil, alpha, err
		}

		if legal {
			foundMove = true
			future, enemyScore, err := helper.alphaBeta(-beta, -alpha, currentDepth+1, depthRemaining-1, append(past, searchMove))

			if err.HasError() {
				return nil, alpha, err
			}

			if IsMate(enemyScore) {
				enemyScore, err = IncrementMate(enemyScore)
				if err.HasError() {
					return nil, alpha, err
				}
			}

			score := -enemyScore
			if score >= beta {
				alpha = beta // fail hard beta-cutoff
				betaCutoff = true
				helper.PrintlnVariation(helper.Debug, past, Some(searchMove), future, "b-cut", Some(score))
			} else if score > alpha {
				alpha = score
				helper.PrintlnVariation(helper.Debug, past, Some(searchMove), future, "pv", Some(score))
				principleVariation = append([]SearchMove{searchMove}, future...)
			} else {
				helper.PrintlnVariation(helper.Debug, past, Some(searchMove), future, "a-skip", Some(score))
			}
		}

		err = undo()
		if err.HasError() {
			return nil, alpha, err
		}

		if betaCutoff {
			break
		}
	}

	if err.HasError() {
		return nil, alpha, err
	}

	if !foundMove {
		if result == AllLegalMoves {
			if helper.InQuiescence {
				return nil, alpha, Errorf("quiescence should only search captures")
			}
			// If no legal moves exist, we're in stalemate or checkmate
			if helper.inCheck() {
				alpha = MateNegative()
			} else {
				alpha = 0
			}
		} else {
			return helper.Evaluator.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth, past)
		}
	}

	return principleVariation, alpha, NilError
}

type SearchResult int

const (
	OutOfTime SearchResult = iota
	Failed
	Completed
)

func (helper *SearchHelper) SearchUpToDepth(
	depthRemaining int,
	moves *[]Move,
) ([]Pair[int, []SearchMove], SearchResult, Error) {
	var err Error

	// The next set of principle variations will go here
	nextVariations := []Pair[int, []SearchMove]{}

	if err.HasError() {
		return nextVariations, Failed, err
	}

	err = helper.MoveSorter.sortMoves(moves)
	if err.HasError() {
		return nextVariations, Failed, err
	}

	for _, move := range *moves {
		if helper.OutOfTime != nil && *helper.OutOfTime {
			return nextVariations, OutOfTime, NilError
		}

		undo, legal, err := performMoveAndReturnLegality(helper.GameState, move)
		if err.HasError() {
			return nextVariations, Failed, err
		}

		if legal {
			// Traverse past the first generated move
			variation, enemyScore, err := helper.alphaBeta(-InitialBounds(), InitialBounds(),
				// current depth is 1 (0 would be before we applied `move`)
				1,
				// we've already searched one move, so decrement depth remaining
				depthRemaining-1,
				[]SearchMove{{move, false}})

			if err.HasError() {
				return nextVariations, Failed, err
			}

			if IsMate(enemyScore) {
				enemyScore, err = IncrementMate(enemyScore)
				if err.HasError() {
					return nextVariations, Failed, err
				}
			}

			score := -enemyScore
			nextVariations = append(nextVariations, Pair[int, []SearchMove]{
				First: score, Second: append([]SearchMove{{move, false}}, variation...)})
		}

		err = undo()
		if err.HasError() {
			return nextVariations, Failed, err
		}
	}

	return nextVariations, Completed, NilError
}

func (helper *SearchHelper) Search() ([]Move, int, int, Error) {
	knownVariations := []Pair[int, []SearchMove]{}

	depthIncrement := 1

	startDepthRemaining := 1
	if helper.WithoutIterativeDeepening {
		startDepthRemaining = helper.IterativeDeepeningDepth
	}

	cleanup, _, moves, err := helper.MoveGen.generateMoves(AllMoves)
	defer cleanup()

	searchedDepth := 0

	if err.HasError() {
		return nil, 0, searchedDepth, err
	}

	doneEarly := false

	for depthRemaining := startDepthRemaining; !doneEarly && depthRemaining <= helper.IterativeDeepeningDepth; depthRemaining += depthIncrement {
		// The generator will prioritize trying the principle variations first
		helper.MoveSorter.reset(knownVariations)

		nextVariations, searchResult, err := helper.SearchUpToDepth(depthRemaining, moves)

		if err.HasError() {
			return nil, 0, searchedDepth, err
		}

		if searchResult == OutOfTime && len(knownVariations) > 0 {
			break
		}

		SortMaxFirst(&nextVariations, func(t Pair[int, []SearchMove]) int {
			return t.First
		})

		for i, move := range nextVariations {
			if i > 5 {
				break
			}

			label := ""
			if i == 0 {
				label = fmt.Sprint("best(", depthRemaining, ")")
			}
			helper.Logger.Println(
				VariationDebugString(
					move.Second,
					0,
					label,
					Some(move.First),
				))
		}
		helper.Debug.Println()

		knownVariations = nextVariations

		if err.HasError() {
			return nil, 0, searchedDepth, err
		}

		searchedDepth = depthRemaining
	}

	if len(knownVariations) == 0 {
		return nil, 0, searchedDepth, NilError
	}

	bestMove := knownVariations[0]
	return MapSlice(bestMove.Second, func(m SearchMove) Move {
		return m.Move
	}), bestMove.First, searchedDepth, NilError
}

type SearchOptions struct {
	// NEXT: use this style of options for searching
	// change interfaces of generator/evaluator etc so that dependencies
	// must be passed in
	Logger      Optional[Logger]
	DebugLogger Optional[Logger]
}

type SearchOption interface {
	apply(helper *SearchHelper) Optional[func()]
}

type WithDebugLogging struct {
}

func (o WithDebugLogging) apply(helper *SearchHelper) Optional[func()] {
	if helper.Logger == &SilentLogger {
		helper.Logger = &DefaultLogger
	}
	helper.Debug = &DefaultLogger
	return Empty[func()]()
}

type WithLogger struct {
	Logger Logger
}

func (o WithLogger) apply(helper *SearchHelper) Optional[func()] {
	helper.Logger = o.Logger
	return Empty[func()]()
}

type WithoutQuiescence struct {
}

func (o WithoutQuiescence) apply(helper *SearchHelper) Optional[func()] {
	helper.Evaluator = BasicEvaluator{}
	return Empty[func()]()
}

type WithMaxDepth struct {
	MaxDepth int
}

func (o WithMaxDepth) apply(helper *SearchHelper) Optional[func()] {
	helper.IterativeDeepeningDepth = o.MaxDepth
	return Empty[func()]()
}

type WithoutIterativeDeepening struct {
}

func (o WithoutIterativeDeepening) apply(helper *SearchHelper) Optional[func()] {
	helper.WithoutIterativeDeepening = true
	return Empty[func()]()
}

type WithoutCheckStandPat struct {
}

func (o WithoutCheckStandPat) apply(helper *SearchHelper) Optional[func()] {
	helper.WithoutCheckStandPat = true
	return Empty[func()]()
}

type WithSearch struct {
	search SearchTree
}

func (o WithSearch) apply(helper *SearchHelper) Optional[func()] {
	unregister, gen := NewSearchTreeMoveGenerator(o.search, helper.GameState)
	helper.MoveGen = gen
	return Some(unregister)
}

type WithTimer struct {
	OutOfTime *bool
}

func (o WithTimer) apply(helper *SearchHelper) Optional[func()] {
	helper.OutOfTime = o.OutOfTime
	return Empty[func()]()
}

func NewSearchHelper(game *GameState, opts ...SearchOption) (func(), *SearchHelper) {
	unregisterCallbacks := []func(){}

	helper := SearchHelper{
		GameState:               game,
		OutOfTime:               nil,
		Logger:                  &SilentLogger,
		Debug:                   &SilentLogger,
		IterativeDeepeningDepth: 3,
	}

	for _, opt := range opts {
		unregister := opt.apply(&helper)
		if unregister.HasValue() {
			unregisterCallbacks = append(unregisterCallbacks, unregister.Value())
		}
	}

	if helper.Evaluator == nil {
		helper.Evaluator = QuiescenceEvaluator{}
	}

	if helper.MoveSorter == nil {
		unregisterSorter, sorter := NewVariationMovePrioritizer(game)
		unregisterCallbacks = append(unregisterCallbacks, unregisterSorter)

		helper.MoveSorter = sorter
	}

	if helper.MoveGen == nil {
		defaultMoveGenerator := DefaultMoveGenerator{
			GameState: game,
		}
		helper.MoveGen = &defaultMoveGenerator
	}

	return func() {
		for _, unregister := range unregisterCallbacks {
			unregister()
		}
	}, &helper
}

func Search(fen string, opts ...SearchOption) ([]Move, int, Error) {
	game, err := GamestateFromFenString(fen)
	if !err.IsNil() {
		return []Move{}, 0, err
	}

	unregister, helper := NewSearchHelper(game, opts...)
	defer unregister()

	pv, score, _, err := helper.Search()
	return pv, score, err
}
