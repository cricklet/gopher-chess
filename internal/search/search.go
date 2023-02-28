package search

import (
	"fmt"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

var Inf int = 999999

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

type searcher struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

	MaximizingPlayer Player

	DebugTotalEvaluations int
	DebugTree             SearchDebugTree
}

func NewSearcher(logger Logger, game *GameState, bitboards *Bitboards) searcher {
	return searcher{
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

func (s *searcher) PerformMoveAndReturnLegality(move Move, update *BoardUpdate) (bool, Error) {
	err := s.Game.PerformMove(move, update, s.Bitboards)
	if !IsNil(err) {
		return false, err
	}

	if KingIsInCheck(s.Bitboards, s.Game.Enemy()) {
		return false, NilError
	}

	return true, NilError
}

func (s *searcher) scoreDirectionForPlayer(player Player) int {
	if player == s.MaximizingPlayer {
		return 1
	} else {
		return -1
	}
}

func (s *searcher) EvaluatePosition() int {
	return Evaluate(s.Bitboards, s.MaximizingPlayer)
}

func (s *searcher) evaluateCapturesInner(alpha int, beta int) (int, []SearchDebugNode, []Error) {
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

func (s *searcher) evaluateCapture(move Move, alpha int, beta int) (int, *SearchDebugNode, []Error) {
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
func (s *searcher) evaluateCaptures(alpha int, beta int) (int, []SearchDebugNode, []Error) {
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

func (s *searcher) evaluateSubtree(alpha int, beta int, depth int) (int, []SearchDebugNode, []Error) {
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

func (s *searcher) evaluateMove(move Move, alpha int, beta int, depth int) (int, *SearchDebugNode, []Error) {
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

func (s *searcher) Search() (Optional[Move], []Error) {
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

		s.Logger.Println("evaluated ",
			"to depth", depth,
			"- total evals", s.DebugTotalEvaluations,
			"- best move", (*moves)[0].String(),
			"- score", (*moves)[0].Evaluation.Value())

		if s.OutOfTime {
			break
		}
	}

	if len(*moves) == 0 {
		return Empty[Move](), nil
	}

	// fmt.Println(s.DebugTree.Sprint(2))

	return Some((*moves)[0]), nil
}

func evaluateCapturesInner(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, Error) {
	if KingIsInCheck(b, g.Enemy()) {
		return SearchResult{Inf, 1, 1}, NilError
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoCaptures(b, g, moves)

	if len(*moves) == 0 {
		score := Evaluate(b, g.Player)
		return SearchResult{score, 1, 1}, NilError
	}

	totalSearched := 0

	for _, move := range *moves {
		if move.Evaluation.Value() < 100 {
			continue
		}

		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("perform evaluateCapturesInner %v: %w", move.String(), err)
		}

		result, err := evaluateCapturesInner(g, b,
			-enemyCanForceScore,
			-playerCanForceScore)
		if !IsNil(err) {
			return SearchResult{}, Errorf("recurse evaluateCapturesInner %v: %w", move.String(), err)
		}
		enemyScore := result.Score
		totalSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("undo evaluateCapturesInner %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, totalSearched}, NilError
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, totalSearched}, NilError
}

func evaluateCaptures(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, Error) {
	standPat := Evaluate(b, g.Player)
	if standPat > enemyCanForceScore {
		return SearchResult{enemyCanForceScore, 1, 1}, NilError
	} else if standPat > playerCanForceScore {
		playerCanForceScore = standPat
	}

	return evaluateCapturesInner(g, b, playerCanForceScore, enemyCanForceScore)
}

type SearchResult struct {
	Score              int
	TotalSearched      int
	QuiescenceSearched int
}

func evaluateSearch(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int, depth int) (SearchResult, Error) {
	if KingIsInCheck(b, g.Enemy()) {
		return SearchResult{Inf, 1, 0}, NilError
	}

	if depth == 0 {
		score := Evaluate(b, g.Player)
		return SearchResult{score, 1, 0}, NilError
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(b, g, moves)

	totalSearched := 0
	quiescenceSearched := 0

	for _, move := range *moves {
		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("perform evaluateSearch %v: %w", move.String(), err)
		}

		var result SearchResult
		if depth == 1 && move.MoveType == CaptureMove {
			result, err = evaluateCaptures(g, b,
				-enemyCanForceScore,
				-playerCanForceScore)
		} else {
			result, err = evaluateSearch(g, b,
				-enemyCanForceScore,
				-playerCanForceScore,
				depth-1)
		}
		if !IsNil(err) {
			return SearchResult{}, Errorf("%v %v: %w", move.String(), depth, err)
		}

		enemyScore := result.Score
		totalSearched += result.TotalSearched
		quiescenceSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("undo evaluateSearch %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, quiescenceSearched}, NilError
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, quiescenceSearched}, NilError
}

func Search(g *GameState, b *Bitboards, depth int, logger Logger) (Optional[Move], Error) {
	// defer profile.Start(profile.ProfilePath("../data/Search")).Stop()

	moves := GetMovesBuffer()
	GenerateSortedPseudoMoves(b, g, moves)

	bestMoveSoFar := Empty[Move]()
	bestScoreSoFar := -Inf

	quiescenceSearched := 0
	totalSearched := 0

	startTime := time.Now()

	for i, move := range *moves {
		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if !IsNil(err) {
			return Empty[Move](), Errorf("perform Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		result, err := evaluateSearch(g, b,
			-Inf, Inf, depth)
		if !IsNil(err) {
			return Empty[Move](), Errorf("evaluate Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		enemyScore := result.Score
		totalSearched += result.TotalSearched
		quiescenceSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if !IsNil(err) {
			return Empty[Move](), Errorf("undo Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		currentScore := -enemyScore
		logger.Println(i, "/", len(*moves), move, "searched", result.TotalSearched, "with initial search", result.TotalSearched-result.QuiescenceSearched, "and ending captures", result.QuiescenceSearched, "under", move.String(), "and found score", currentScore)

		if currentScore > bestScoreSoFar {
			bestScoreSoFar = currentScore
			bestMoveSoFar = Some(move)
		}
	}

	PLY_COUNTS := []int{
		1,
		20,
		400,
		8902,
		197281,
		4865609,
		119060324,
		3195901860,
	}

	for i := 0; i < len(PLY_COUNTS); i++ {
		if totalSearched < PLY_COUNTS[i] {
			logger.Println("searched", totalSearched,
				"with initial search", totalSearched-quiescenceSearched, "and ending captures", quiescenceSearched,
				"nodes in", time.Since(startTime), ", ~ perft of ply", i, "(", PLY_COUNTS[i], ")")
			break
		}
	}

	logger.Println(bestMoveSoFar, bestScoreSoFar)

	return bestMoveSoFar, NilError
}
