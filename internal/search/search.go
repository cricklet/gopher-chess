package search

import (
	"fmt"
	"strconv"

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
	updatePrincipleVariations(variations []Pair[int, []SearchMove])

	getGenerationMode() MoveGenerationMode
	setGenerationMode(mode MoveGenerationMode)
}

type MoveGenerationMode int

const (
	AllMoves MoveGenerationMode = iota
	OnlyCaptures
)

type DefaultMoveGenerator struct {
	*GameState
	*Bitboards
	mode MoveGenerationMode

	sortedVariations [][]SearchMove
	currentVariation []SearchMove
	inVariation      bool
}

func NewDefaultMoveGenerator(g *GameState, b *Bitboards, mode MoveGenerationMode) DefaultMoveGenerator {
	return DefaultMoveGenerator{
		GameState: g,
		Bitboards: b,
		mode:      mode,
	}
}

var _ MoveGen = (*DefaultMoveGenerator)(nil)

func (gen *DefaultMoveGenerator) getGenerationMode() MoveGenerationMode {
	return gen.mode
}

func (gen *DefaultMoveGenerator) setGenerationMode(mode MoveGenerationMode) {
	gen.mode = mode
}

func (gen *DefaultMoveGenerator) updatePrincipleVariations(variations []Pair[int, []SearchMove]) {
	gen.sortedVariations = [][]SearchMove{}

	SortMaxFirst(&variations, func(t Pair[int, []SearchMove]) int {
		return t.First
	})

	for _, variation := range variations {
		gen.sortedVariations = append(gen.sortedVariations, variation.Second)
	}

	gen.currentVariation = nil
	gen.inVariation = false
}

func (gen *DefaultMoveGenerator) searchingAllLegalMoves() bool {
	if gen.mode == OnlyCaptures {
		return false
	} else {
		return true
	}
}

func (gen *DefaultMoveGenerator) performEachMoveAndCall(callback func(move Move) (LoopResult, Error)) Error {
	if len(gen.sortedVariations) > 0 && !gen.inVariation {
		for _, variation := range gen.sortedVariations {
			if len(variation) == 0 {
				return Errorf("variation has no moves")
			}

			gen.currentVariation = variation[1:]

			gen.inVariation = true
			result, err := performMoveAndCall(gen.GameState, gen.Bitboards, variation[0].Move, callback)
			gen.inVariation = false

			if !err.IsNil() {
				return err
			}
			if result == LoopBreak {
				break
			}
		}
		return NilError
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	if gen.mode == OnlyCaptures {
		GeneratePseudoCaptures(func(m Move) {
			*moves = append(*moves, m)
		}, gen.Bitboards, gen.GameState)
	} else {
		GeneratePseudoMoves(func(m Move) {
			*moves = append(*moves, m)
		}, gen.Bitboards, gen.GameState)
	}

	if gen.currentVariation != nil && len(gen.currentVariation) > 0 {
		// Move the previously calculated best move to the front
		for i, move := range *moves {
			if move == gen.currentVariation[0].Move {
				(*moves)[i] = (*moves)[0]
				(*moves)[0] = move
				break
			}
		}

		previousCurrentVariation := gen.currentVariation
		gen.currentVariation = gen.currentVariation[1:]
		defer func() {
			gen.currentVariation = previousCurrentVariation
		}()
	}

	for _, move := range *moves {
		result, err := performMoveAndCall(gen.GameState, gen.Bitboards, move, callback)

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

	mode MoveGenerationMode
}

var _ MoveGen = (*SearchTreeMoveGenerator)(nil)

func (gen *SearchTreeMoveGenerator) updatePrincipleVariations(variations []Pair[int, []SearchMove]) {
}

func (gen *SearchTreeMoveGenerator) getGenerationMode() MoveGenerationMode {
	return gen.mode
}
func (gen *SearchTreeMoveGenerator) setGenerationMode(mode MoveGenerationMode) {
	gen.mode = mode
}

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
		continueGen := NewDefaultMoveGenerator(gen.GameState, gen.Bitboards, gen.mode)
		return (&continueGen).performEachMoveAndCall(callback)
	}

	prevSearchTree := gen.currentlySearching
	for nextMoveStr, nextSearchTree := range gen.currentlySearching.moves {
		gen.currentlySearching = nextSearchTree

		nextMove := gen.GameState.MoveFromString(nextMoveStr)
		if gen.mode == OnlyCaptures && !nextMove.MoveType.Captures() {
			// If we're in quiescence, don't search non-capture moves.
			// Note, this isn't an error because we could be hitting
			// quiescence early due to iterative deepening (eg searching
			// depth 1 or 2)
			continue
		}

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
	evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error)
}

type BasicEvaluator struct {
}

var _ Evaluator = (*BasicEvaluator)(nil)

func (e BasicEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error) {
	return nil, Evaluate(helper.Bitboards, player), NilError
}

type QuiescenceEvaluator struct {
}

var _ Evaluator = (*QuiescenceEvaluator)(nil)

func (e QuiescenceEvaluator) evaluate(helper *SearchHelper, player Player, alpha int, beta int, currentDepth int, pastMoves []SearchMove) ([]SearchMove, int, Error) {
	// captureGenerator := NewDefaultMoveGenerator(helper.GameState, helper.Bitboards, OnlyCaptures)

	prevGenerationMode := helper.MoveGen.getGenerationMode()
	helper.MoveGen.setGenerationMode(OnlyCaptures)

	defer func() {
		helper.MoveGen.setGenerationMode(prevGenerationMode)
	}()

	quiescenceHelper := SearchHelper{
		MoveGen:                   helper.MoveGen,
		Evaluator:                 BasicEvaluator{},
		GameState:                 helper.GameState,
		Bitboards:                 helper.Bitboards,
		OutOfTime:                 nil,
		InQuiescence:              true,
		Logger:                    helper.Logger,
		Debug:                     helper.Debug,
		IterativeDeepeningDepth:   helper.IterativeDeepeningDepth,
		WithoutIterativeDeepening: helper.WithoutIterativeDeepening,
		WithoutCheckStandPat:      helper.WithoutCheckStandPat,
	}

	// NEXT: include past history for nicer printing

	moves, score, err := quiescenceHelper.alphaBeta(alpha, beta, currentDepth,
		// Search up to 10 more moves
		10,
		pastMoves,
	)
	return moves, score, err

	// prevGenerationMode := helper.MoveGen.getGenerationMode()
	// prevCheckStandPat := helper.CheckStandPat
	// prevDebugLogger := helper.Debug
	// prevEvaluator := helper.Evaluator

	// helper.MoveGen.setGenerationMode(OnlyCaptures)
	// helper.CheckStandPat = true
	// helper.Debug = &SilentLogger
	// helper.Evaluator = BasicEvaluator{}

	// defer func() {
	// 	helper.MoveGen.setGenerationMode(prevGenerationMode)
	// 	helper.CheckStandPat = prevCheckStandPat
	// 	helper.Debug = prevDebugLogger
	// 	helper.Evaluator = prevEvaluator
	// }()

	// return helper.alphaBeta(alpha, beta, currentDepth,
	// 	// Search up to 10 more moves
	// 	10)
}

type SearchHelper struct {
	MoveGen MoveGen
	// MoveSorter MoveSorter
	Evaluator    Evaluator
	GameState    *GameState
	Bitboards    *Bitboards
	OutOfTime    *bool
	InQuiescence bool
	Logger
	Debug                     Logger
	IterativeDeepeningDepth   int
	WithoutIterativeDeepening bool
	WithoutCheckStandPat      bool
}

func (helper SearchHelper) String() string {
	return helper.GameState.Board.String()
}

func (helper SearchHelper) inCheck() bool {
	return KingIsInCheck(helper.Bitboards, helper.GameState.Player)
}

// NEXT: also include if we're in quiescence

var LABEL_COL int = 7
var SCORE_COL int = 5

type SearchMove struct {
	Move
	inQuiescence bool
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

	return PrintColumns(
		[]string{
			label,
			MapOptional(score, func(s int) string {
				if s < 0 {
					return strconv.Itoa(s)
				} else {
					return " " + strconv.Itoa(s)
				}
			}).ValueOr(""),
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

	currentIndex := Inf
	if current.HasValue() {
		currentIndex = len(fullVariation)
		fullVariation = append(fullVariation, current.Value())
	}
	fullVariation = append(fullVariation, future...)

	result := VariationDebugString(fullVariation, currentIndex, label, score)
	if current.IsEmpty() {
		result += " [****]"
	}
	logger.Println(result)
}

func (helper *SearchHelper) alphaBeta(alpha int, beta int, currentDepth int, depthRemaining int, past []SearchMove) ([]SearchMove, int, Error) {
	if depthRemaining <= 0 {
		future, score, err := helper.Evaluator.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth, past)
		helper.PrintlnVariation(helper.Debug, past, Empty[SearchMove](), future, "eval", Some(score))
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

	foundMove := false

	err := helper.MoveGen.performEachMoveAndCall(func(move Move) (LoopResult, Error) {
		foundMove = true

		searchMove := SearchMove{move, helper.InQuiescence}

		helper.PrintlnVariation(helper.Debug, past, Some(searchMove), nil, "???", Empty[int]())

		future, enemyScore, err := helper.alphaBeta(-beta, -alpha, currentDepth+1, depthRemaining-1, append(past, searchMove))
		if err.HasError() {
			return LoopBreak, err
		}

		score := -enemyScore
		if score >= beta {
			alpha = beta // fail hard beta-cutoff
			helper.PrintlnVariation(helper.Debug, past, Some(searchMove), future, "b-cut", Some(score))
			return LoopBreak, NilError
		} else if score > alpha {
			alpha = score
			helper.PrintlnVariation(helper.Debug, past, Some(searchMove), future, "pv", Some(score))
			principleVariation = append([]SearchMove{searchMove}, future...)
		} else {
			helper.PrintlnVariation(helper.Debug, past, Some(searchMove), future, "a-skip", Some(score))
		}
		return LoopContinue, NilError
	})

	if err.HasError() {
		return nil, alpha, err
	}

	if !foundMove {
		if helper.MoveGen.searchingAllLegalMoves() && helper.InQuiescence {
			// If no legal moves exist, we're in stalemate or checkmate
			if helper.inCheck() {
				alpha = -Inf
			} else {
				alpha = 0
			}
		} else {
			return helper.Evaluator.evaluate(helper, helper.GameState.Player, alpha, beta, currentDepth, past)
		}
	}

	return principleVariation, alpha, NilError
}

func (helper *SearchHelper) Search() ([]Move, int, Error) {
	principleVariations := []Pair[int, []SearchMove]{}

	depthIncrement := 1

	startDepthRemaining := 1
	if helper.WithoutIterativeDeepening {
		startDepthRemaining = helper.IterativeDeepeningDepth
	}

	for depthRemaining := startDepthRemaining; depthRemaining <= helper.IterativeDeepeningDepth; depthRemaining += depthIncrement {
		// The generator will prioritize trying the principle variations first
		helper.MoveGen.updatePrincipleVariations(principleVariations)

		// The next set of principle variations will go here
		principleVariations = []Pair[int, []SearchMove]{}

		// Loop through & perform the first generated moves
		err := helper.MoveGen.performEachMoveAndCall(func(move Move) (LoopResult, Error) {
			if helper.OutOfTime != nil && *helper.OutOfTime {
				return LoopBreak, NilError
			}

			// Traverse past the first generated move
			variation, enemyScore, err := helper.alphaBeta(-Inf-1, Inf+1,
				// current depth is 1 (0 would be before we applied `move`)
				1,
				// we've already searched one move, so decrement depth remaining
				depthRemaining-1,
				[]SearchMove{{move, false}})

			if err.HasError() {
				return LoopBreak, err
			}

			score := -enemyScore
			principleVariations = append(principleVariations, Pair[int, []SearchMove]{
				First: score, Second: append([]SearchMove{{move, false}}, variation...)})

			return LoopContinue, NilError
		})

		if err.HasError() {
			return nil, 0, err
		}

		SortMaxFirst(&principleVariations, func(t Pair[int, []SearchMove]) int {
			return t.First
		})

		for _, move := range principleVariations {
			helper.Logger.Println(
				VariationDebugString(
					move.Second,
					0,
					fmt.Sprint("done(", depthRemaining, ")"),
					Some(move.First),
				))
			helper.Debug.Println()
		}
	}

	if len(principleVariations) == 0 {
		return nil, 0, NilError
	}

	bestMove := principleVariations[0]
	return MapSlice(bestMove.Second, func(m SearchMove) Move {
		return m.Move
	}), bestMove.First, NilError
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
	helper.IterativeDeepeningDepth = o.MaxDepth
}

type WithoutIterativeDeepening struct {
}

func (o WithoutIterativeDeepening) apply(helper *SearchHelper) {
	helper.WithoutIterativeDeepening = true
}

type WithoutCheckStandPat struct {
}

func (o WithoutCheckStandPat) apply(helper *SearchHelper) {
	helper.WithoutCheckStandPat = true
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
		AllMoves,
	}
}

type WithOutOfTime struct {
	OutOfTime *bool
}

func (o WithOutOfTime) apply(helper *SearchHelper) {
	helper.OutOfTime = o.OutOfTime
}

func Searcher(game *GameState, b *Bitboards, opts ...SearchOption) *SearchHelper {
	defaultMoveGenerator := NewDefaultMoveGenerator(
		game,
		b,
		AllMoves)
	quiescenceEvaluator := QuiescenceEvaluator{}
	helper := SearchHelper{
		MoveGen:                 &defaultMoveGenerator,
		Evaluator:               quiescenceEvaluator,
		GameState:               game,
		Bitboards:               b,
		OutOfTime:               nil,
		Logger:                  &SilentLogger,
		Debug:                   &SilentLogger,
		IterativeDeepeningDepth: 3,
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
