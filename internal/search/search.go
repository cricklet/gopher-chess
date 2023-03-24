package search

import (
	"fmt"
	"sort"
	"strings"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/dustin/go-humanize"
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
	CurrentPath  []string
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
		}
		// else {
		// result += fmt.Sprintf("%v%v (%v %v)\n",
		// 	strings.Repeat(" ", line.Depth),
		// 	line.DebugString,
		// 	line.Alpha,
		// 	line.Beta)
		// }
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

func (s *debugSearchTree) MovePop(move Move, player Player, alpha int, beta int, result int, legal bool) {
	s.CurrentDepth -= 1
	s.Result = append(s.Result, debugSearchLine{
		DebugString: fmt.Sprintf("$ %v (%v)", strings.Join(s.CurrentPath, " "), player),
		Depth:       s.CurrentDepth,
		Alpha:       alpha,
		Beta:        beta,
		Score:       Some(result),
		Legal:       Some(legal),
		IsCapture:   move.MoveType == CaptureMove || move.MoveType == EnPassantMove,
	})
	s.CurrentPath = s.CurrentPath[:len(s.CurrentPath)-1]
}

func (s *debugSearchTree) MovePush(move Move, player Player, alpha int, beta int) {
	s.CurrentPath = append(s.CurrentPath, move.String())
	s.Result = append(s.Result, debugSearchLine{
		DebugString: fmt.Sprintf("> %v (%v)", strings.Join(s.CurrentPath, " "), player),
		Depth:       s.CurrentDepth,
		Alpha:       alpha,
		Beta:        beta,
		IsCapture:   move.MoveType == CaptureMove || move.MoveType == EnPassantMove,
	})
	s.CurrentDepth += 1
}

type SearcherV2 struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

	hasIncrementedDepthForCheck bool

	options SearcherOptions

	DebugTotalEvaluations    int
	DebugTotalMovesPerformed int
	DebugDepthIteration      int
	DebugMovesToConsider     int
	DebugMovesConsidered     int
	DebugCapturesSearched    int
	DebugCapturesSkipped     int
}

type SearcherOptions struct {
	evaluationOptions      []EvaluationOption
	skipTranspositionTable bool

	debugSearchTree *debugSearchTree
	maxDepth        Optional[int]
}

var DefaultSearchOptions = SearcherOptions{}

var AllSearchOptions = []string{
	"skipTranspositionTable",
}

var DisallowedSearchOptionCombinations = [][]string{}

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
		if strings.HasPrefix(arg, "debugSearchTree") {
			options.debugSearchTree = &debugSearchTree{}
		} else if strings.HasPrefix(arg, "skipTranspositionTable") {
			options.skipTranspositionTable = true
		} else if arg == "" {
		} else {
			return options, Errorf("unknown option: '%s'", arg)
		}
	}

	return options, NilError
}

func NewSearcherV2(logger Logger, game *GameState, bitboards *Bitboards, options SearcherOptions) *SearcherV2 {
	return &SearcherV2{
		Logger:    logger,
		OutOfTime: false,
		Game:      game,
		Bitboards: bitboards,
		options:   options,
	}
}

func (s *SearcherV2) SortMoves(moves *[]Move, evals map[Move]int) {
	for i := range *moves {
		evals[(*moves)[i]] = EvaluateMove(&(*moves)[i], s.Game)
	}
	sort.SliceStable(*moves, func(i, j int) bool {
		return evals[(*moves)[i]] > evals[(*moves)[j]]
	})
}

func (s *SearcherV2) PerformMoveAndReturnLegality(move Move, update *BoardUpdate) (bool, Error) {
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

func (s *SearcherV2) EvaluatePosition(player Player) int {
	return Evaluate(s.Bitboards, player, s.options.evaluationOptions...)
}

func (s *SearcherV2) evaluateCapturesForPlayer(player Player, alpha int, beta int) (int, Error) {
	var returnScore int
	var returnError Error

	if s.OutOfTime {
		return returnScore, returnError
	}

	if player != s.Game.Player {
		returnError = Errorf("player != s.Game.Player")
		return returnScore, returnError
	}

	// if s.options.transpositionTable != nil {
	// 	if entry := s.options.transpositionTable.Get(s.Game.ZobristHash(), 0); entry.HasValue() {
	// 		returnScore := entry.Value().Score
	// 		return returnScore, returnErrors
	// 	}
	// }

	standPat := s.EvaluatePosition(player)

	if standPat >= beta {
		returnScore := beta
		return returnScore, returnError
	} else if standPat > alpha {
		alpha = standPat
	}

	returnScore = alpha

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	evals := make(map[Move]int)
	GeneratePseudoCaptures(s.Bitboards, s.Game, moves)
	s.SortMoves(moves, evals)

	if len(*moves) == 0 {
		returnScore = s.EvaluatePosition(player)
		s.DebugTotalEvaluations++
		return returnScore, returnError
	}

	for i := range *moves {
		if eval, ok := evals[(*moves)[i]]; ok {
			if eval <= 0 {
				s.DebugCapturesSkipped++
				break
			}
		}

		s.DebugCapturesSearched++

		score, legality, childError := s.evaluateCaptureForPlayer(player, (*moves)[i], alpha, beta)

		if !IsNil(childError) {
			returnError = Join(returnError, childError)
			return returnScore, returnError
		}

		if !legality {
			continue
		}

		if score >= beta {
			// The enemy will avoid this line
			returnScore = beta
			break
		} else if score > alpha {
			// This is our best choice of move
			alpha = score
			returnScore = score
		}
	}

	// if s.options.transpositionTable != nil {
	// 	hash := s.Game.ZobristHash()
	// 	s.options.transpositionTable.Put(hash, 0, returnScore)
	// }
	return returnScore, returnError
}

func (s *SearcherV2) evaluateCaptureForPlayer(player Player, move Move, alpha int, beta int) (int, bool, Error) {
	var returnScore int
	var returnLegality bool
	var returnError Error

	if s.OutOfTime {
		return returnScore, returnLegality, returnError
	}

	if player != s.Game.Player {
		returnError = Errorf("player != s.Game.Player")
		return returnScore, returnLegality, returnError
	}

	enemy := player.Other()

	if s.options.debugSearchTree != nil {
		s.options.debugSearchTree.MovePush(
			move,
			player, alpha, beta)
		defer func() {
			s.options.debugSearchTree.MovePop(
				move, player,
				alpha, beta, returnScore, returnLegality)
		}()
	}

	var update BoardUpdate
	var err Error
	returnLegality, err = s.PerformMoveAndReturnLegality(move, &update)
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if !IsNil(err) {
			returnError = Join(returnError, err)
		}
	}()
	if !IsNil(err) {
		returnError = Join(returnError, err)
		return returnScore, returnLegality, returnError
	}
	if !returnLegality {
		return returnScore, returnLegality, returnError
	}

	var enemyScore int
	enemyScore, returnError = s.evaluateCapturesForPlayer(enemy, -beta, -alpha)
	returnScore = -enemyScore
	return returnScore, returnLegality, returnError
}

func (s *SearcherV2) evaluatePositionForPlayer(player Player, alpha int, beta int, depth int) (int, Error) {
	if s.OutOfTime {
		return 0, NilError
	}

	if player != s.Game.Player {
		return 0, Errorf("player != s.Game.Player")
	}

	if !s.options.skipTranspositionTable {
		if entry := DefaultTranspositionTable().Get(s.Game.ZobristHash(), depth); entry.HasValue() {
			score := entry.Value().Score
			scoreType := entry.Value().ScoreType
			if scoreType == Exact {
				if score >= beta {
					// The enemy will avoid this line
					return beta, NilError
				} else if score > alpha {
					return score, NilError
				} else {
					return alpha, NilError
				}
			} else if scoreType == AlphaFailUpperBound {
				if score <= alpha {
					// There isn't a better result in this subtree
					return alpha, NilError
				}
			} else if scoreType == BetaFailLowerBound {
				if score >= beta {
					// The enemy will avoid this line
					return beta, NilError
				}
			}
		}
	}

	returnScore := alpha
	returnScoreType := AlphaFailUpperBound

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	evals := make(map[Move]int)
	GeneratePseudoMoves(s.Bitboards, s.Game, moves)
	s.SortMoves(moves, evals)

	hasLegalMove := false

	for i := range *moves {
		moveScore, moveLegality, err := s.evaluateMoveForPlayer(player, (*moves)[i], alpha, beta, depth)

		if !IsNil(err) {
			return moveScore, err
		}

		if !moveLegality {
			continue
		} else {
			hasLegalMove = true
		}

		if moveScore >= beta {
			// The enemy will avoid this line
			returnScore = beta
			returnScoreType = BetaFailLowerBound
			break
		} else if moveScore > alpha {
			// This is our best choice of move
			alpha = moveScore
			returnScore = moveScore
			returnScoreType = Exact
		}
	}

	if !hasLegalMove {
		if KingIsInCheck(s.Bitboards, s.Game.Player) {
			returnScore = -Inf
			returnScoreType = Exact
		} else {
			returnScore = 0
			returnScoreType = Exact
		}
	}

	if !s.options.skipTranspositionTable {
		// This always clobbers the existing value in the transposition table. TODO: should we be smarter?
		// eg only clobber if we have an exact score or if the depth increased?
		hash := s.Game.ZobristHash()
		DefaultTranspositionTable().Put(hash, depth, returnScore, returnScoreType)
	}
	return returnScore, NilError
}

func (s *SearcherV2) evaluateMoveForTests(player Player, move Move, depth int) (int, bool, Error) {
	if player == s.Game.Player {
		score, legality, err := s.evaluateMoveForPlayer(player, move, -Inf, Inf, depth)
		return score, legality, err
	} else {
		score, legality, err := s.evaluateMoveForPlayer(player.Other(), move, -Inf, Inf, depth)
		return -score, legality, err
	}
}

func (s *SearcherV2) evaluatePositionForTests(player Player, depth int) (int, Error) {
	if player == s.Game.Player {
		score, err := s.evaluatePositionForPlayer(player, -Inf, Inf, depth)
		return score, err
	} else {
		score, err := s.evaluatePositionForPlayer(player.Other(), -Inf, Inf, depth)
		return -score, err
	}
}

func (s *SearcherV2) evaluateMoveForPlayer(player Player, move Move, alpha int, beta int, depth int) (int, bool, Error) {
	var returnScore int
	var returnLegality bool
	var returnError Error

	if s.OutOfTime {
		return returnScore, returnLegality, returnError
	}

	if player != s.Game.Player {
		returnError = Errorf("player != s.Game.Player")
		return returnScore, returnLegality, returnError
	}
	enemy := player.Other()

	if s.options.debugSearchTree != nil {
		s.options.debugSearchTree.MovePush(
			move,
			player, alpha, beta)
		defer func() {
			s.options.debugSearchTree.MovePop(
				move, player,
				alpha, beta, returnScore, returnLegality)
		}()
	}

	var update BoardUpdate
	var err Error
	returnLegality, err = s.PerformMoveAndReturnLegality(move, &update)

	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if !IsNil(err) {
			returnError = Join(returnError, err)
		}
	}()

	if !IsNil(err) {
		returnError = err
		return returnScore, returnLegality, returnError
	}
	if !returnLegality {
		return returnScore, returnLegality, returnError
	}
	if depth <= 1 {
		if !s.OutOfTime && !s.hasIncrementedDepthForCheck {
			if KingIsInCheck(s.Bitboards, enemy) {
				depth += 2
				s.hasIncrementedDepthForCheck = true
				defer func() {
					s.hasIncrementedDepthForCheck = false
				}()
			}
		}
	}

	if depth == 0 {
		var enemyScore int
		enemyScore, returnError = s.evaluateCapturesForPlayer(enemy, -beta, -alpha)
		returnScore = -enemyScore
	} else {
		var enemyScore int
		enemyScore, returnError = s.evaluatePositionForPlayer(enemy, -beta, -alpha, depth-1)
		returnScore = -enemyScore
	}

	return returnScore, returnLegality, returnError
}

func (s *SearcherV2) DebugStats() string {
	result := fmt.Sprintf("depth: %v, %v / %v, evals %v, moves %v, quiescence %v, skipped %v",
		humanize.Comma(int64(s.DebugDepthIteration)),
		humanize.Comma(int64(s.DebugMovesConsidered)), humanize.Comma(int64(s.DebugMovesToConsider)),
		humanize.Comma(int64(s.DebugTotalEvaluations)), humanize.Comma(int64(s.DebugTotalMovesPerformed)),
		humanize.Comma(int64(s.DebugCapturesSearched)), humanize.Comma(int64(s.DebugCapturesSkipped)))
	if !s.options.skipTranspositionTable {
		result += fmt.Sprintf(", %v", DefaultTranspositionTable().Stats())
	}
	if s.options.debugSearchTree != nil {
		result += fmt.Sprintf(", stack: %v", strings.Join(s.options.debugSearchTree.CurrentPath, ","))
	}
	return result
}

type MoveKey int

func MoveToMoveKey(move Move) MoveKey {
	key := move.StartIndex*64 + move.EndIndex
	if move.PromotionPiece.HasValue() {
		key += int(move.PromotionPiece.Value()) * 4096
	}
	return MoveKey(key)
}

func (s *SearcherV2) Search() (Optional[Move], Error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	evals := make(map[Move]int)
	GeneratePseudoMoves(s.Bitboards, s.Game, moves)
	s.SortMoves(moves, evals)

	maxDepth := 20
	if s.options.maxDepth.HasValue() {
		maxDepth = s.options.maxDepth.Value()
	}

	evaluationsAtDepth := make(map[int]map[MoveKey]int)
	getEvalAtDepth := func(depth int, move Move) int {
		if eval, ok := evaluationsAtDepth[depth][MoveToMoveKey(move)]; ok {
			return eval
		} else {
			return -Inf
		}
	}

	depthForSorting := 1

	for depth := 1; depth <= maxDepth; depth++ {
		s.DebugDepthIteration = depth
		s.DebugMovesToConsider = len(*moves)
		s.DebugMovesConsidered = 0
		evaluationsAtDepth[depth] = make(map[MoveKey]int)

		err := func() Error {
			if s.options.debugSearchTree != nil {
				s.options.debugSearchTree.DepthPush(fmt.Sprintf("depth %d", depth))
				defer func() {
					s.options.debugSearchTree.DepthPop(fmt.Sprintf("depth %d", depth), getEvalAtDepth(depth, (*moves)[0]))
				}()
			}

			for i := range *moves {
				score, legality, err := s.evaluateMoveForPlayer(s.Game.Player, (*moves)[i], -Inf, Inf, depth)
				if !IsNil(err) {
					return err
				}

				if s.OutOfTime {
					// We just ran out of time. It's likely we didn't evaluate this move fully
					break
				}

				// s.Logger.Println("considering move", (*moves)[i].String(),
				// 	"at depth", depth, "with legality ", legality, "and score", score)
				s.DebugMovesConsidered++

				moveKey := MoveToMoveKey((*moves)[i])
				if !legality {
					evaluationsAtDepth[depth][moveKey] = -Inf
					continue
				} else {
					evaluationsAtDepth[depth][moveKey] = score
				}
			}

			SortMaxFirst(moves, func(m Move) int {
				return getEvalAtDepth(depth, m)
			})

			s.Logger.Println(fmt.Sprintf("info move: %v %v", (*moves)[0].String(), getEvalAtDepth(depth, (*moves)[0])), s.DebugStats())

			if !s.OutOfTime || len(evaluationsAtDepth[depth]) >= 6 {
				depthForSorting = depth
			}

			return NilError
		}()

		if !IsNil(err) {
			return Empty[Move](), err
		}

		if s.OutOfTime {
			break
		}
	}

	if len(*moves) == 0 {
		return Empty[Move](), NilError // forfeit / stalemate
	}

	bestIndex := IndexOfMax(*moves, func(m Move) int {
		return getEvalAtDepth(depthForSorting, m)
	})
	bestMove := (*moves)[bestIndex]
	bestEval := getEvalAtDepth(depthForSorting, bestMove)

	s.Logger.Printf("info using evaluation from depth %v => %v %v\n", depthForSorting, bestMove.String(), bestEval)

	if bestEval == -Inf {
		return Empty[Move](), NilError // forfeit / stalemate
	}

	// fmt.Println(s.DebugTree.Sprint(2))
	s.OutOfTime = false
	return Some((*moves)[0]), NilError
}
