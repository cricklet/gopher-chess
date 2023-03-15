package search

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bluele/psort"
	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/zobrist"
)

type debugSearchLine struct {
	DebugString string
	Depth       int
	Alpha       int
	Beta        int
	IsCapture   bool
	Score       Optional[int]
	Legal       Optional[bool]
}

type debugSearchTree struct {
	CurrentDepth int
	Result       []debugSearchLine
}

func (s *debugSearchTree) DebugString(depth int) string {
	result := ""
	for i := range s.Result {
		// line := s.Result[len(s.Result)-i-1]
		line := s.Result[i]
		if line.Depth >= depth {
			continue
		}
		if line.Score.HasValue() {
			scoreString := fmt.Sprint(line.Score.Value())
			if line.Legal.HasValue() && !line.Legal.Value() {
				scoreString = "illegal"
			}
			captureString := ""
			if line.IsCapture {
				captureString = " x"
			}
			result += fmt.Sprintf("%v%v%v (%v %v) %v\n",
				strings.Repeat(" ", line.Depth),
				line.DebugString,
				captureString,
				line.Alpha,
				line.Beta,
				scoreString)
		} else {
			result += fmt.Sprintf("%v%v (%v %v)\n",
				strings.Repeat(" ", line.Depth),
				line.DebugString,
				line.Alpha,
				line.Beta)
		}
	}
	return result
}

func (s *debugSearchTree) DepthPush(label string) {
	s.Result = append(s.Result, debugSearchLine{
		DebugString: "> " + label,
		Depth:       s.CurrentDepth,
	})
	s.CurrentDepth += 1
}

func (s *debugSearchTree) DepthPop(label string, result int) {
	s.CurrentDepth -= 1
	s.Result = append(s.Result, debugSearchLine{
		DebugString: "$ " + label,
		Depth:       s.CurrentDepth,
		Score:       Some(result),
	})
}

func (s *debugSearchTree) MovePop(move string, isMaximizing bool, alpha int, beta int, result int, legal bool) {
	s.CurrentDepth -= 1
	playerString := "min"
	if isMaximizing {
		playerString = "max"
	}
	s.Result = append(s.Result, debugSearchLine{
		DebugString: fmt.Sprintf("$ %v (%v)", playerString, move),
		Depth:       s.CurrentDepth,
		Alpha:       alpha,
		Beta:        beta,
		Score:       Some(result),
		Legal:       Some(legal),
	})
}

func (s *debugSearchTree) MovePush(move string, isMaximizing bool, alpha int, beta int) {
	playerString := "min"
	if isMaximizing {
		playerString = "max"
	}
	s.Result = append(s.Result, debugSearchLine{
		DebugString: fmt.Sprintf("> %v (%v)", playerString, move),
		Depth:       s.CurrentDepth,
		Alpha:       alpha,
		Beta:        beta,
	})
	s.CurrentDepth += 1
}

func (s *debugSearchTree) CapturePush(move string, isMaximizing bool, alpha int, beta int) {
	playerString := "min"
	if isMaximizing {
		playerString = "max"
	}
	s.Result = append(s.Result, debugSearchLine{
		DebugString: fmt.Sprintf("> %v (%v)", playerString, move),
		Depth:       s.CurrentDepth,
		Alpha:       alpha,
		Beta:        beta,
		IsCapture:   true,
	})
	s.CurrentDepth += 1
}

func (s *debugSearchTree) CapturePop(move string, isMaximizing bool, alpha int, beta int, result int, legal bool) {
	s.CurrentDepth -= 1
	playerString := "min"
	if isMaximizing {
		playerString = "max"
	}
	s.Result = append(s.Result, debugSearchLine{
		DebugString: fmt.Sprintf("$ %v (%v)", playerString, move),
		Depth:       s.CurrentDepth,
		Alpha:       alpha,
		Beta:        beta,
		Score:       Some(result),
		Legal:       Some(legal),
		IsCapture:   true,
	})
}

type searcherV2 struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

	MaximizingPlayer Player

	options SearcherOptions

	DebugTotalEvaluations    int
	DebugTotalMovesPerformed int
	DebugDepthIteration      int
	DebugMovesToConsider     int
	DebugMovesConsidered     int
	DebugCapturesSearched    int
	DebugCapturesSkipped     int
}

type incDepthForCheck struct {
	depthLimit   int
	currentDepth int
}

type SearcherOptions struct {
	incDepthForCheck   incDepthForCheck
	evaluationOptions  []EvaluationOption
	handleLegality     bool
	sortPartial        Optional[int]
	transpositionTable *zobrist.TranspositionTable

	debugSearchTree  *debugSearchTree
	debugSearchStack *[]string
	maxDepth         Optional[int]
}

var DefaultSearchOptions = SearcherOptions{
	incDepthForCheck:   incDepthForCheck{},
	evaluationOptions:  []EvaluationOption{},
	handleLegality:     false,
	transpositionTable: nil,
	debugSearchTree:    nil,
	debugSearchStack:   nil,
	maxDepth:           Empty[int](),
}

var AllSearchOptions = []string{
	"incDepthForCheck",
	"endgamePushEnemyKing",
	"handleLegality",
	"sortPartial",
	"sortPartial=0",
	"sortPartial=10",
}

var DisallowedSearchOptionCombinations = [][]string{
	{"sortPartial", "sortPartial"},
}

func RemoveFirstPrefixMatch(slice []string, prefix string) ([]string, bool) {
	for i, item := range slice {
		if strings.HasPrefix(item, prefix) {
			return append(slice[:i], slice[i+1:]...), true
		}
	}
	return slice, false
}

func FilterDisallowedSearchOptions(allOptions [][]string) [][]string {
	return FilterSlice(allOptions, func(options []string) bool {
		for _, disallowedOptions := range DisallowedSearchOptionCombinations {
			disallowed := true

			for _, disallowedOption := range disallowedOptions {
				var foundDisallowedOption bool
				options, foundDisallowedOption = RemoveFirstPrefixMatch(Clone(options), disallowedOption)
				if !foundDisallowedOption {
					disallowed = false
					break
				}
			}
			if disallowed {
				return false
			}
		}
		return true
	})
}

func SearcherOptionsFromArgs(args ...string) (SearcherOptions, Error) {
	options := SearcherOptions{}

	for _, arg := range args {
		if strings.HasPrefix(arg, "incDepthForCheck") {
			if strings.Contains(arg, "=") {
				n, err := strconv.ParseInt(strings.Split(arg, "=")[1], 10, 64)
				if err != nil {
					return options, Wrap(err)
				}
				options.incDepthForCheck = incDepthForCheck{
					depthLimit: int(n),
				}
			} else {
				options.incDepthForCheck = incDepthForCheck{
					depthLimit: 3,
				}
			}
		} else if strings.HasPrefix(arg, "sortPartial") {
			if strings.Contains(arg, "=") {
				n, err := strconv.ParseInt(strings.Split(arg, "=")[1], 10, 64)
				if err != nil {
					return options, Wrap(err)
				}
				options.sortPartial = Some(int(n))
			} else {
				options.sortPartial = Some(3)
			}
		} else if strings.HasPrefix(arg, "endgamePushEnemyKing") {
			options.evaluationOptions = append(options.evaluationOptions, EndgamePushEnemyKing)
		} else if strings.HasPrefix(arg, "handleLegality") {
			options.handleLegality = true
		} else if strings.HasPrefix(arg, "debugSearchTree") {
			options.debugSearchTree = &debugSearchTree{}
		} else if strings.HasPrefix(arg, "transpositionTable") {
			options.transpositionTable = zobrist.NewTranspositionTable(zobrist.DefaultTranspositionTableSize)
		} else {
			return options, Errorf("unknown option: %s", arg)
		}
	}

	return options, NilError
}

func NewSearcherV2(logger Logger, game *GameState, bitboards *Bitboards, options SearcherOptions) searcherV2 {
	s := searcherV2{
		Logger:           logger,
		OutOfTime:        false,
		Game:             game,
		Bitboards:        bitboards,
		MaximizingPlayer: game.Player,
		options:          options,
	}
	return s
}

func (s *searcherV2) basicMoveEvaluation(moves *[]Move) {
	for i := range *moves {
		(*moves)[i].Evaluation = Some(EvaluateMove(&(*moves)[i], s.Game))
	}
}

func (s *searcherV2) SortMoves(moves *[]Move) {
	if s.options.sortPartial.HasValue() {
		n := s.options.sortPartial.Value()
		if n == 0 {
			return
		} else {
			s.basicMoveEvaluation(moves)
			psort.Slice(*moves, func(i, j int) bool {
				return (*moves)[i].Evaluation.Value() > (*moves)[j].Evaluation.Value()
			}, n)
			return
		}
	}
	s.basicMoveEvaluation(moves)
	sort.SliceStable(*moves, func(i, j int) bool {
		return (*moves)[i].Evaluation.Value() > (*moves)[j].Evaluation.Value()
	})
}

func (s *searcherV2) GenerateSortedPseudoMoves(moves *[]Move) {
	GeneratePseudoMoves(s.Bitboards, s.Game, moves)
	s.SortMoves(moves)
}

func (s *searcherV2) GenerateSortedPseudoCaptures(moves *[]Move) {
	GeneratePseudoCaptures(s.Bitboards, s.Game, moves)
	s.SortMoves(moves)
}

func (s *searcherV2) PerformMoveAndReturnLegality(move Move, update *BoardUpdate) (bool, Error) {
	s.DebugTotalMovesPerformed++
	err := s.Game.PerformMove(move, update, s.Bitboards)
	if !IsNil(err) {
		return false, err
	}

	if KingIsInCheck(s.Bitboards, s.Game.Enemy()) {
		return false, NilError
	}

	return true, NilError
}

func (s *searcherV2) scoreDirectionForPlayer(player Player) int {
	if player == s.MaximizingPlayer {
		return 1
	} else {
		return -1
	}
}

func (s *searcherV2) EvaluatePosition() int {
	return Evaluate(s.Bitboards, s.MaximizingPlayer, s.options.evaluationOptions...)
}

func (s *searcherV2) evaluateCaptures(alpha int, beta int) (int, []Error) {
	var returnScore int
	var returnErrors []Error

	if s.options.transpositionTable != nil {
		if entry := s.options.transpositionTable.Get(s.Game.ZobristHash(), 0); entry.HasValue() {
			returnScore := entry.Value().Score
			return returnScore, returnErrors
		}
	}

	standPat := s.EvaluatePosition()
	player := s.Game.Player

	if player == s.MaximizingPlayer {
		if standPat >= beta {
			returnScore := beta
			return returnScore, returnErrors
		} else if standPat > alpha {
			alpha = standPat
		}
	} else {
		if standPat <= alpha {
			returnScore := alpha
			return returnScore, returnErrors
		} else if standPat < beta {
			beta = standPat
		}
	}

	if s.MaximizingPlayer == player {
		returnScore = alpha
	} else {
		returnScore = beta
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	s.GenerateSortedPseudoCaptures(moves)
	if len(*moves) == 0 {
		returnScore = s.EvaluatePosition()
		s.DebugTotalEvaluations++
		return returnScore, returnErrors
	}

	for i := range *moves {
		if (*moves)[i].Evaluation.Value() <= 0 {
			s.DebugCapturesSkipped++
			break
		}

		s.DebugCapturesSearched++

		score, legality, childErrors := s.evaluateCapture((*moves)[i], alpha, beta)

		if len(childErrors) > 0 {
			returnErrors = append(returnErrors, childErrors...)
			return returnScore, returnErrors
		}

		if !legality {
			continue
		}

		if s.MaximizingPlayer == player {
			if score >= beta {
				// The enemy will avoid this line
				returnScore = beta
				break
			} else if score > alpha {
				// This is our best choice of move
				alpha = score
				returnScore = score
			}
		} else {
			if score <= alpha {
				returnScore = alpha
				break
			} else if score < beta {
				beta = score
				returnScore = score
			}
		}
	}

	if s.options.transpositionTable != nil {
		hash := s.Game.ZobristHash()
		s.options.transpositionTable.Put(hash, 0, returnScore)
	}
	return returnScore, returnErrors
}

func (s *searcherV2) evaluateCapture(move Move, alpha int, beta int) (int, bool, []Error) {
	var returnScore int
	var returnLegality bool
	var returnErrors []Error

	player := s.Game.Player

	if s.options.debugSearchTree != nil {
		s.options.debugSearchTree.CapturePush(
			move.String(),
			player == s.MaximizingPlayer, alpha, beta)
		defer func() {
			s.options.debugSearchTree.CapturePop(
				move.String(), player == s.MaximizingPlayer,
				alpha, beta, returnScore, returnLegality)
		}()
	}

	if s.options.debugSearchStack != nil {
		*s.options.debugSearchStack = append(*s.options.debugSearchStack, move.String())
		defer func() {
			*s.options.debugSearchStack = (*s.options.debugSearchStack)[:len(*s.options.debugSearchStack)-1]
		}()
	}

	var update BoardUpdate
	var err Error
	returnLegality, err = s.PerformMoveAndReturnLegality(move, &update)
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if !IsNil(err) {
			returnErrors = append(returnErrors, err)
		}
	}()
	if !IsNil(err) {
		returnErrors = append(returnErrors, err)
		return returnScore, returnLegality, returnErrors
	}
	if !returnLegality {
		return returnScore, returnLegality, returnErrors
	}

	returnScore, returnErrors = s.evaluateCaptures(alpha, beta)

	return returnScore, returnLegality, returnErrors
}

func (s *searcherV2) evaluateSubtree(alpha int, beta int, depth int) (int, []Error) {
	var returnScore int
	var returnErrors []Error

	if s.options.transpositionTable != nil {
		if entry := s.options.transpositionTable.Get(s.Game.ZobristHash(), depth); entry.HasValue() {
			returnScore := entry.Value().Score
			return returnScore, returnErrors
		}
	}

	player := s.Game.Player

	if s.MaximizingPlayer == player {
		returnScore = alpha
	} else {
		returnScore = beta
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	s.GenerateSortedPseudoMoves(moves)

	hasLegalMove := false

	for i := range *moves {
		score, legality, childErrors := s.evaluateMove((*moves)[i], alpha, beta, depth)

		if len(childErrors) > 0 {
			returnErrors = append(returnErrors, childErrors...)
			return returnScore, returnErrors
		}

		if !legality {
			continue
		} else {
			hasLegalMove = true
		}

		if s.MaximizingPlayer == player {
			if score >= beta {
				// The enemy will avoid this line
				returnScore = beta
				break
			} else if score > alpha {
				// This is our best choice of move
				alpha = score
				returnScore = score
			}
		} else {
			if score <= alpha {
				returnScore = alpha
				break
			} else if score < beta {
				beta = score
				returnScore = score
			}
		}

		if s.OutOfTime {
			break
		}
	}

	if !hasLegalMove && s.options.handleLegality {
		if KingIsInCheck(s.Bitboards, s.Game.Player) {
			returnScore = -Inf * s.scoreDirectionForPlayer(player)
		} else {
			returnScore = 0
		}
	}

	if s.options.transpositionTable != nil {
		hash := s.Game.ZobristHash()
		s.options.transpositionTable.Put(hash, depth, returnScore)
	}
	return returnScore, returnErrors
}

func (s *searcherV2) evaluateMove(move Move, alpha int, beta int, depth int) (int, bool, []Error) {
	var returnScore int
	var returnLegality bool
	var returnErrors []Error

	player := s.Game.Player

	if s.options.debugSearchTree != nil {
		s.options.debugSearchTree.MovePush(
			move.String(),
			player == s.MaximizingPlayer, alpha, beta)
		defer func() {
			s.options.debugSearchTree.MovePop(
				move.String(), player == s.MaximizingPlayer,
				alpha, beta, returnScore, returnLegality)
		}()
	}

	if s.options.debugSearchStack != nil {
		*s.options.debugSearchStack = append(*s.options.debugSearchStack, move.String())
		defer func() {
			*s.options.debugSearchStack = (*s.options.debugSearchStack)[:len(*s.options.debugSearchStack)-1]
		}()

		movesStackString := strings.Join(*s.options.debugSearchStack, ",")
		if movesStackString == "c8e6,d4d5,b4c3,b2c3,g7g6" {
			fmt.Println(s.Game.Board.Unicode())
		}
	}

	var update BoardUpdate
	var err Error
	returnLegality, err = s.PerformMoveAndReturnLegality(move, &update)

	if s.options.debugSearchStack != nil {
		movesStackString := strings.Join(*s.options.debugSearchStack, ",")
		if movesStackString == "c8e6,d4d5,b4c3,b2c3,g7g6" {
			fmt.Println(s.Game.Board.Unicode())
		}
	}
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if !IsNil(err) {
			returnErrors = append(returnErrors, err)
		}
	}()

	if !IsNil(err) {
		returnErrors = append(returnErrors, err)
		return returnScore, returnLegality, returnErrors
	}
	if !returnLegality {
		return returnScore, returnLegality, returnErrors
	}

	if depth <= 1 && !s.OutOfTime &&
		s.options.incDepthForCheck.currentDepth < s.options.incDepthForCheck.depthLimit {
		if KingIsInCheck(s.Bitboards, player.Other()) {
			depth += 1
			s.options.incDepthForCheck.currentDepth += 1
			defer func() {
				s.options.incDepthForCheck.currentDepth -= 1
			}()
		}
	}

	if depth == 0 {
		returnScore, returnErrors = s.evaluateCaptures(alpha, beta)
	} else {
		returnScore, returnErrors = s.evaluateSubtree(alpha, beta, depth-1)
	}

	return returnScore, returnLegality, returnErrors
}

func (s *searcherV2) DebugStats() string {
	result := fmt.Sprintf("depth: %v, %v / %v, evals %v, moves %v, quiescence %v, skipped %v",
		s.DebugDepthIteration,
		s.DebugMovesConsidered, s.DebugMovesToConsider,
		s.DebugTotalEvaluations, s.DebugTotalMovesPerformed,
		s.DebugCapturesSearched, s.DebugCapturesSkipped)
	if s.options.transpositionTable != nil {
		result += fmt.Sprintf(", %v", s.options.transpositionTable.Stats())
	}
	if s.options.debugSearchStack != nil {
		result += fmt.Sprintf(", stack: %v", strings.Join(*s.options.debugSearchStack, ","))
	}
	return result
}

func (s *searcherV2) Search() (Optional[Move], []Error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	s.GenerateSortedPseudoMoves(moves)

	maxDepth := 20
	if s.options.maxDepth.HasValue() {
		maxDepth = s.options.maxDepth.Value()
	}

	for depth := 2; depth <= maxDepth; depth += 1 {
		s.DebugDepthIteration = depth
		s.DebugMovesToConsider = len(*moves)
		s.DebugMovesConsidered = 0
		errs := func() []Error {
			if s.options.debugSearchTree != nil {
				s.options.debugSearchTree.DepthPush(fmt.Sprintf("depth %d", depth))
				defer func() {
					s.options.debugSearchTree.DepthPop(fmt.Sprintf("depth %d", depth), (*moves)[0].Evaluation.Value())
				}()
			}

			for i := range *moves {
				score, legality, errs := s.evaluateMove((*moves)[i], -Inf, Inf, depth)
				if len(errs) > 0 {
					return errs
				}

				s.DebugMovesConsidered++

				if !legality {
					(*moves)[i].Evaluation = Empty[int]()
					continue
				} else {
					(*moves)[i].Evaluation = Some(score)
				}

				if s.OutOfTime {
					break
				}
			}

			SortMaxFirst(moves, func(m Move) int {
				if m.Evaluation.HasValue() {
					return m.Evaluation.Value()
				} else {
					return -Inf
				}
			})

			s.Logger.Println(
				strings.Join(MapSlice(*moves, func(m Move) string {
					return m.String() + " " + strconv.Itoa(m.Evaluation.Value())
				}), " "))

			s.Logger.Println("evaluated",
				"to depth", depth,
				"- total evals", s.DebugTotalEvaluations,
				"- best move", (*moves)[0].String(),
				"- score", (*moves)[0].Evaluation.Value())

			return nil
		}()

		if len(errs) > 0 {
			return Empty[Move](), errs
		}

		if s.OutOfTime {
			break
		}
	}

	if len(*moves) == 0 {
		return Empty[Move](), nil // forfeit / stalemate
	}

	bestMove := (*moves)[0]
	if !bestMove.Evaluation.HasValue() || bestMove.Evaluation.Value() == -Inf {
		return Empty[Move](), nil // forfeit / stalemate
	}

	// fmt.Println(s.DebugTree.Sprint(2))
	return Some((*moves)[0]), nil
}
