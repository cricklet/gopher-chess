package search

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type searcherV2 struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

	MaximizingPlayer Player

	extendingSearchDepth      int
	limitExtendingSearchDepth int

	DebugTotalEvaluations int
}

type SearcherOptions struct {
	incDepthForCheck Optional[int]
}

var DefaultSearchOptions = SearcherOptions{
	incDepthForCheck: Empty[int](),
}

func (s SearcherOptions) String() string {
	result := ""
	if s.incDepthForCheck.HasValue() {
		result += fmt.Sprintf("incDepthForCheck=%d", s.incDepthForCheck.Value())
	}
	return "SearcherOptions<" + result + ">"
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
				options.incDepthForCheck = Some(int(n))
			} else {
				options.incDepthForCheck = Some(3)
			}
		} else {
			return options, Errorf("unknwon option: %s", arg)
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
	}
	if options.incDepthForCheck.HasValue() {
		s.limitExtendingSearchDepth = options.incDepthForCheck.Value()
	}
	return s
}

func (s *searcherV2) PerformMoveAndReturnLegality(move Move, update *BoardUpdate) (bool, Error) {
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
	return Evaluate(s.Bitboards, s.MaximizingPlayer)
}

func (s *searcherV2) evaluateCapturesInner(alpha int, beta int) (int, []Error) {
	var returnScore int
	var returnErrors []Error

	player := s.Game.Player

	if s.MaximizingPlayer == player {
		returnScore = alpha
	} else {
		returnScore = beta
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoCaptures(s.Bitboards, s.Game, moves)
	if len(*moves) == 0 {
		returnScore = s.EvaluatePosition()
		s.DebugTotalEvaluations++
		return returnScore, returnErrors
	}

	for i := range *moves {
		score, childErrors := s.evaluateCapture((*moves)[i], alpha, beta)

		if len(childErrors) > 0 {
			returnErrors = append(returnErrors, childErrors...)
			return returnScore, returnErrors
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

	return returnScore, returnErrors
}

func (s *searcherV2) evaluateCapture(move Move, alpha int, beta int) (int, []Error) {
	var returnScore int
	var returnErrors []Error

	player := s.Game.Player

	var update BoardUpdate
	legal, err := s.PerformMoveAndReturnLegality(move, &update)
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if !IsNil(err) {
			returnErrors = append(returnErrors, err)
		}
	}()
	if !IsNil(err) {
		returnErrors = append(returnErrors, err)
		return returnScore, returnErrors
	}
	if !legal {
		returnScore = -Inf * s.scoreDirectionForPlayer(player)
		return returnScore, returnErrors
	}

	returnScore, returnErrors = s.evaluateCaptures(alpha, beta)

	return returnScore, returnErrors
}
func (s *searcherV2) evaluateCaptures(alpha int, beta int) (int, []Error) {
	standPat := s.EvaluatePosition()
	player := s.Game.Player

	if player == s.MaximizingPlayer {
		if standPat > beta {
			return standPat, nil
		} else if standPat > alpha {
			alpha = standPat
		}
	} else {
		if standPat < alpha {
			return standPat, nil
		} else if standPat < beta {
			beta = standPat
		}
	}

	return s.evaluateCapturesInner(alpha, beta)
}

func (s *searcherV2) evaluateSubtree(alpha int, beta int, depth int) (int, []Error) {
	var returnScore int
	var returnErrors []Error

	player := s.Game.Player

	if s.MaximizingPlayer == player {
		returnScore = alpha
	} else {
		returnScore = beta
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for i := range *moves {
		score, childErrors := s.evaluateMove((*moves)[i], alpha, beta, depth)

		if len(childErrors) > 0 {
			returnErrors = append(returnErrors, childErrors...)
			return returnScore, returnErrors
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

	return returnScore, returnErrors
}

func (s *searcherV2) evaluateMove(move Move, alpha int, beta int, depth int) (int, []Error) {
	var returnScore int
	var returnErrors []Error

	player := s.Game.Player
	playerHint := FenStringForPlayer(player)
	if player == s.MaximizingPlayer {
		playerHint += "-max"
	} else {
		playerHint += "-min"
	}

	var update BoardUpdate
	legal, err := s.PerformMoveAndReturnLegality(move, &update)
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if !IsNil(err) {
			returnErrors = append(returnErrors, err)
		}
	}()

	if !IsNil(err) {
		returnErrors = append(returnErrors, err)
		return returnScore, returnErrors
	}
	if !legal {
		returnScore = -Inf * s.scoreDirectionForPlayer(player)
		return returnScore, returnErrors
	}

	if depth <= 1 && !s.OutOfTime &&
		s.extendingSearchDepth < s.limitExtendingSearchDepth {
		if KingIsInCheck(s.Bitboards, player.Other()) {
			depth += 1
			s.extendingSearchDepth += 1
			defer func() {
				s.extendingSearchDepth -= 1
			}()
		}
	}

	if depth == 0 {
		if move.MoveType == CaptureMove || move.MoveType == EnPassantMove {
			returnScore, returnErrors = s.evaluateCaptures(alpha, beta)
		} else {
			s.DebugTotalEvaluations++
			returnScore = s.EvaluatePosition()
		}
	} else {
		returnScore, returnErrors = s.evaluateSubtree(alpha, beta, depth-1)
	}

	return returnScore, returnErrors
}

func (s *searcherV2) Search() (Optional[Move], []Error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for depth := 2; ; depth += 2 {
		alpha := -Inf
		for i := range *moves {
			score, errs := s.evaluateMove((*moves)[i], alpha, Inf, depth)
			if len(errs) > 0 {
				return Empty[Move](), errs
			}

			if s.OutOfTime {
				break
			}

			// if score > alpha {
			// 	alpha = score
			// }

			(*moves)[i].Evaluation = Some(score)
		}

		SortMaxFirst(moves, func(m Move) int {
			return m.Evaluation.Value()
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

		if s.OutOfTime {
			break
		}
	}

	if len(*moves) == 0 || (*moves)[0].Evaluation.Value() == -Inf {
		return Empty[Move](), nil // forfeit
	}

	// fmt.Println(s.DebugTree.Sprint(2))

	return Some((*moves)[0]), nil
}
