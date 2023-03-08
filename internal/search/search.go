package search

import (
	"fmt"
	"strings"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

// type MoveEvaluation int

// const (
// 	IllegalMove MoveEvaluation = iota
// 	AllMove
// 	BestMove
// 	RefutationMove
// )

type SearchDebugTree struct {
	FenString        string
	Player           Player
	SearchIterations []SearchIteration
}
type SearchIteration struct {
	Depth       int
	SearchRoots []SearchDebugNode
}
type SearchDebugNode struct {
	Move          string
	PlayerHint    string
	StartingAlpha int
	StartingBeta  int
	EndingAlpha   int
	EndingBeta    int
	ReturnedScore int
	Children      []SearchDebugNode
}

func (tree *SearchDebugTree) addIteration(depth int) *SearchIteration {
	tree.SearchIterations = append(tree.SearchIterations, SearchIteration{Depth: depth})
	return &tree.SearchIterations[len(tree.SearchIterations)-1]
}

func (it *SearchIteration) addRoot(node *SearchDebugNode) {
	it.SearchRoots = append(it.SearchRoots, *node)
}

func createNode(move Move, playerHint string, alpha int, beta int) *SearchDebugNode {
	return &SearchDebugNode{
		Move:          move.String(),
		PlayerHint:    playerHint,
		StartingAlpha: alpha,
		StartingBeta:  beta,
		Children:      []SearchDebugNode{},
	}
}

func (child *SearchDebugNode) finalize(alpha, beta, score int) {
	child.EndingAlpha = alpha
	child.EndingBeta = beta
	child.ReturnedScore = score
}

func (node *SearchDebugNode) Sprint(indent int, prefix string, depth int) string {
	if depth == 0 {
		return ""
	}
	result := ""
	for i := 0; i < indent; i++ {
		result += " "
	}
	result += fmt.Sprintf("%v %v %v [%v %v] => [%v %v] %v", prefix, node.Move, node.PlayerHint, node.StartingAlpha, node.StartingBeta, node.EndingAlpha, node.EndingBeta, node.ReturnedScore)
	result += "\n"

	numChildren := len(node.Children)

	for i, child := range node.Children {
		childPrefix := fmt.Sprintf("(%v/%v)", i+1, numChildren)
		result += child.Sprint(indent+2, childPrefix, depth-1)
		break
	}
	return result
}

func (node *SearchDebugTree) Sprint(depth int) string {
	result := node.FenString + "\n"
	playerStr := "white"
	if node.Player == Black {
		playerStr = "black"
	}
	urlString := "http://0.0.0.0:8002/" + playerStr + "/fen/" + node.FenString
	result += strings.ReplaceAll(urlString, " ", "%20") + "\n"

	numIterations := len(node.SearchIterations)

	f := func(i int) {
		searchIt := node.SearchIterations[i]
		searchRoots := searchIt.SearchRoots
		numSearchRoots := len(searchRoots)
		result += fmt.Sprintf("  search (%v/%v) %v moves w/ depth %v\n", i+1, numIterations, numSearchRoots, searchIt.Depth)
		for j, root := range searchRoots {
			result += root.Sprint(4, fmt.Sprintf("(%v/%v)", j, numSearchRoots), depth)
		}
	}

	f(len(node.SearchIterations) - 2)
	f(len(node.SearchIterations) - 1)

	return result
}

type searcherV2 struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

	MaximizingPlayer Player

	DebugTotalEvaluations int
	DebugTree             SearchDebugTree
}

func NewSearcherV2(logger Logger, game *GameState, bitboards *Bitboards) searcherV2 {
	return searcherV2{
		Logger:           logger,
		OutOfTime:        false,
		Game:             game,
		Bitboards:        bitboards,
		MaximizingPlayer: game.Player,
		DebugTree: SearchDebugTree{
			Player:    game.Player,
			FenString: FenStringForGame(game),
		},
	}
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

func (s *searcherV2) evaluateCapturesInner(alpha int, beta int) (int, []SearchDebugNode, []Error) {
	var returnScore int
	var returnDebug []SearchDebugNode
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
		return returnScore, returnDebug, returnErrors
	}

	for i := range *moves {
		score, debugChild, childErrors := s.evaluateCapture((*moves)[i], alpha, beta)
		returnDebug = append(returnDebug, *debugChild)

		if len(childErrors) > 0 {
			returnErrors = append(returnErrors, childErrors...)
			return returnScore, returnDebug, returnErrors
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

	if s.MaximizingPlayer == player {
		SortMaxFirst(&returnDebug, func(n SearchDebugNode) int {
			return n.ReturnedScore
		})
	} else {
		SortMinFirst(&returnDebug, func(n SearchDebugNode) int {
			return n.ReturnedScore
		})
	}

	return returnScore, returnDebug, returnErrors
}

func (s *searcherV2) evaluateCapture(move Move, alpha int, beta int) (int, *SearchDebugNode, []Error) {
	var returnScore int
	var returnErrors []Error

	player := s.Game.Player
	playerHint := FenStringForPlayer(player)
	if player == s.MaximizingPlayer {
		playerHint += "-max"
	} else {
		playerHint += "-min"
	}

	returnNode := createNode(move, playerHint, alpha, beta)
	defer func() { returnNode.finalize(alpha, beta, returnScore) }()

	var update BoardUpdate
	legal, err := s.PerformMoveAndReturnLegality(move, &update)
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		returnErrors = append(returnErrors, err)
	}()
	if !IsNil(err) {
		returnErrors = append(returnErrors, err)
		return returnScore, returnNode, returnErrors
	}
	if !legal {
		returnScore = -Inf * s.scoreDirectionForPlayer(player)
		return returnScore, returnNode, returnErrors
	}

	returnScore, returnNode.Children, returnErrors = s.evaluateCaptures(alpha, beta)

	return returnScore, returnNode, returnErrors
}
func (s *searcherV2) evaluateCaptures(alpha int, beta int) (int, []SearchDebugNode, []Error) {
	standPat := s.EvaluatePosition()
	player := s.Game.Player

	if player == s.MaximizingPlayer {
		if standPat > beta {
			return standPat, nil, nil
		} else if standPat > alpha {
			alpha = standPat
		}
	} else {
		if standPat < alpha {
			return standPat, nil, nil
		} else if standPat < beta {
			beta = standPat
		}
	}

	return s.evaluateCapturesInner(alpha, beta)
}

func (s *searcherV2) evaluateSubtree(alpha int, beta int, depth int) (int, []SearchDebugNode, []Error) {
	var returnDebug []SearchDebugNode
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
		score, debugChild, childErrors := s.evaluateMove((*moves)[i], alpha, beta, depth)
		returnDebug = append(returnDebug, *debugChild)

		if len(childErrors) > 0 {
			returnErrors = append(returnErrors, childErrors...)
			return returnScore, returnDebug, returnErrors
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

	if s.MaximizingPlayer == player {
		SortMaxFirst(&returnDebug, func(n SearchDebugNode) int {
			return n.ReturnedScore
		})
	} else {
		SortMinFirst(&returnDebug, func(n SearchDebugNode) int {
			return n.ReturnedScore
		})
	}

	return returnScore, returnDebug, returnErrors
}

func (s *searcherV2) evaluateMove(move Move, alpha int, beta int, depth int) (int, *SearchDebugNode, []Error) {
	var returnScore int
	var returnErrors []Error

	player := s.Game.Player
	playerHint := FenStringForPlayer(player)
	if player == s.MaximizingPlayer {
		playerHint += "-max"
	} else {
		playerHint += "-min"
	}

	returnNode := createNode(move, playerHint, alpha, beta)
	defer func() { returnNode.finalize(alpha, beta, returnScore) }()

	var update BoardUpdate
	legal, err := s.PerformMoveAndReturnLegality(move, &update)
	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		returnErrors = append(returnErrors, err)
	}()

	if !IsNil(err) {
		returnErrors = append(returnErrors, err)
		return returnScore, returnNode, returnErrors
	}
	if !legal {
		returnScore = -Inf * s.scoreDirectionForPlayer(player)
		return returnScore, returnNode, returnErrors
	}

	if depth <= 1 && !s.OutOfTime {
		if KingIsInCheck(s.Bitboards, player.Other()) {
			depth = 1
		}
	}

	if depth == 0 {
		if move.MoveType == CaptureMove || move.MoveType == EnPassantMove {
			returnScore, returnNode.Children, returnErrors = s.evaluateCaptures(alpha, beta)
		} else {
			s.DebugTotalEvaluations++
			returnScore = s.EvaluatePosition()
		}
	} else {
		returnScore, returnNode.Children, returnErrors = s.evaluateSubtree(alpha, beta, depth-1)
	}

	return returnScore, returnNode, returnErrors
}

func (s *searcherV2) Search() (Optional[Move], []Error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for depth := 2; ; depth += 2 {
		debugSearches := s.DebugTree.addIteration(depth)

		alpha := -Inf
		for i := range *moves {
			score, debugNode, errs := s.evaluateMove((*moves)[i], alpha, Inf, depth)
			if len(errs) > 0 {
				return Empty[Move](), errs
			}

			if s.OutOfTime {
				break
			}

			debugSearches.addRoot(debugNode)

			// if score > alpha {
			// 	alpha = score
			// }

			(*moves)[i].Evaluation = Some(score)
		}

		SortMaxFirst(moves, func(m Move) int {
			return m.Evaluation.Value()
		})
		SortMaxFirst(&debugSearches.SearchRoots, func(n SearchDebugNode) int {
			return n.ReturnedScore
		})

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
