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
		} else {
			// result += fmt.Sprintf("%v%v (%v %v)\n",
			// 	strings.Repeat(" ", line.Depth),
			// 	line.DebugString,
			// 	line.Alpha,
			// 	line.Beta)
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

type searcherV2 struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

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

	debugSearchTree *debugSearchTree
	maxDepth        Optional[int]
}

var DefaultSearchOptions = SearcherOptions{
	incDepthForCheck:   incDepthForCheck{},
	evaluationOptions:  []EvaluationOption{},
	handleLegality:     false,
	transpositionTable: nil,
	debugSearchTree:    nil,
	maxDepth:           Empty[int](),
}

var AllSearchOptions = []string{
	"incDepthForCheck",
	"endgamePushEnemyKing",
	"handleLegality",
	"transpositionTable",
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
		Logger:    logger,
		OutOfTime: false,
		Game:      game,
		Bitboards: bitboards,
		options:   options,
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

func (s *searcherV2) EvaluatePosition(player Player) int {
	return Evaluate(s.Bitboards, player, s.options.evaluationOptions...)
}

func (s *searcherV2) evaluateCapturesForPlayer(player Player, alpha int, beta int) (int, Error) {
	var returnScore int
	var returnError Error

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

	s.GenerateSortedPseudoCaptures(moves)
	if len(*moves) == 0 {
		returnScore = s.EvaluatePosition(player)
		s.DebugTotalEvaluations++
		return returnScore, returnError
	}

	for i := range *moves {
		if (*moves)[i].Evaluation.Value() <= 0 {
			s.DebugCapturesSkipped++
			break
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

func (s *searcherV2) evaluateCaptureForPlayer(player Player, move Move, alpha int, beta int) (int, bool, Error) {
	var returnScore int
	var returnLegality bool
	var returnError Error

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

func (s *searcherV2) evaluatePositionForPlayer(player Player, alpha int, beta int, depth int) (int, Error) {
	var returnScore int
	var returnError Error

	if player != s.Game.Player {
		returnError = Errorf("player != s.Game.Player")
		return returnScore, returnError
	}

	if s.options.transpositionTable != nil {
		if entry := s.options.transpositionTable.Get(s.Game.ZobristHash(), depth); entry.HasValue() {
			returnScore := entry.Value().Score
			return returnScore, returnError
		}
	}

	returnScore = alpha

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	s.GenerateSortedPseudoMoves(moves)

	hasLegalMove := false

	for i := range *moves {
		score, legality, childError := s.evaluateMoveForPlayer(player, (*moves)[i], alpha, beta, depth)

		if !IsNil(childError) {
			returnError = Join(returnError, childError)
			return returnScore, returnError
		}

		if !legality {
			continue
		} else {
			hasLegalMove = true
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

		if s.OutOfTime {
			break
		}
	}

	if !hasLegalMove && s.options.handleLegality {
		if KingIsInCheck(s.Bitboards, s.Game.Player) {
			returnScore = -Inf
		} else {
			returnScore = 0
		}
	}

	if s.options.transpositionTable != nil {
		hash := s.Game.ZobristHash()
		s.options.transpositionTable.Put(hash, depth, returnScore)
	}
	return returnScore, returnError
}

func (s *searcherV2) evaluateMoveForTests(player Player, move Move, depth int) (int, bool, Error) {
	if player == s.Game.Player {
		return s.evaluateMoveForPlayer(player, move, -Inf, Inf, depth)
	} else {
		score, legal, err := s.evaluateMoveForPlayer(player.Other(), move, -Inf, Inf, depth)
		return -score, legal, err
	}
}

func (s *searcherV2) evaluatePositionForTests(player Player, depth int) (int, Error) {
	if player == s.Game.Player {
		return s.evaluatePositionForPlayer(player, -Inf, Inf, depth)
	} else {
		score, err := s.evaluatePositionForPlayer(player.Other(), -Inf, Inf, depth)
		return -score, err
	}
}

func (s *searcherV2) evaluateMoveForPlayer(player Player, move Move, alpha int, beta int, depth int) (int, bool, Error) {
	var returnScore int
	var returnLegality bool
	var returnError Error

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
	if depth <= 1 {
		if !s.OutOfTime && s.options.incDepthForCheck.currentDepth < s.options.incDepthForCheck.depthLimit {
			if KingIsInCheck(s.Bitboards, enemy) {
				depth += 1
				s.options.incDepthForCheck.currentDepth += 1
				defer func() {
					s.options.incDepthForCheck.currentDepth -= 1
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

func (s *searcherV2) DebugStats() string {
	result := fmt.Sprintf("depth: %v, %v / %v, evals %v, moves %v, quiescence %v, skipped %v",
		s.DebugDepthIteration,
		s.DebugMovesConsidered, s.DebugMovesToConsider,
		s.DebugTotalEvaluations, s.DebugTotalMovesPerformed,
		s.DebugCapturesSearched, s.DebugCapturesSkipped)
	if s.options.transpositionTable != nil {
		result += fmt.Sprintf(", %v", s.options.transpositionTable.Stats())
	}
	if s.options.debugSearchTree != nil {
		result += fmt.Sprintf(", stack: %v", strings.Join(s.options.debugSearchTree.CurrentPath, ","))
	}
	return result
}

func (s *searcherV2) Search() (Optional[Move], Error) {
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
		err := func() Error {
			if s.options.debugSearchTree != nil {
				s.options.debugSearchTree.DepthPush(fmt.Sprintf("depth %d", depth))
				defer func() {
					s.options.debugSearchTree.DepthPop(fmt.Sprintf("depth %d", depth), (*moves)[0].Evaluation.Value())
				}()
			}

			for i := range *moves {
				score, legality, err := s.evaluateMoveForPlayer(s.Game.Player, (*moves)[i], -Inf, Inf, depth)
				if !IsNil(err) {
					return err
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
				Indent("search results\n"+strings.Join(MapSlice((*moves)[:6], func(m Move) string {
					return m.String() + " " + strconv.Itoa(m.Evaluation.Value())
				}), "\n"), ". "))

			s.Logger.Println(s.DebugStats())

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

	bestMove := (*moves)[0]
	if !bestMove.Evaluation.HasValue() || bestMove.Evaluation.Value() == -Inf {
		return Empty[Move](), NilError // forfeit / stalemate
	}

	// fmt.Println(s.DebugTree.Sprint(2))
	return Some((*moves)[0]), NilError
}
