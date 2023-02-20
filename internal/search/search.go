package search

import (
	"fmt"
	"sort"
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/fen"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/pkg/profile"
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
	SearchIterations [][]SearchDebugNode
}
type SearchDebugNode struct {
	Move          string
	StartingAlpha int
	StartingBeta  int
	EndingAlpha   int
	EndingBeta    int
	ReturnedScore int
	Children      []SearchDebugNode
}

func (tree *SearchDebugTree) addIteration() *[]SearchDebugNode {
	tree.SearchIterations = append(tree.SearchIterations, []SearchDebugNode{})
	return &tree.SearchIterations[len(tree.SearchIterations)-1]
}

func createNode(move Move, alpha int, beta int) SearchDebugNode {
	return SearchDebugNode{
		Move:          move.String(),
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

func (node *SearchDebugNode) Sprint(indent int, prefix string) string {
	result := ""
	for i := 0; i < indent; i++ {
		result += " "
	}
	result += fmt.Sprint(prefix, " ", node.Move, " <", node.StartingAlpha, ",", node.StartingBeta, "> => <", node.EndingAlpha, ",", node.EndingBeta, ">")
	result += "\n"

	numChildren := len(node.Children)

	for i, child := range node.Children {
		childPrefix := fmt.Sprintf("(%v/%v)", i, numChildren)
		result += child.Sprint(indent+2, childPrefix)
	}
	return result
}

func (node *SearchDebugTree) Sprint() string {
	result := ""
	result += node.FenString
	result += "\n"

	numIterations := len(node.SearchIterations)

	f := func(i int) {
		searchRoots := node.SearchIterations[i]
		numSearchRoots := len(searchRoots)
		result += fmt.Sprintf("  search (%v/%v, %v)\n", i, numIterations, numSearchRoots)
		for j, root := range searchRoots {
			result += root.Sprint(4, fmt.Sprintf("(%v/%v)", j, numSearchRoots))
		}
	}

	f(0)
	f(len(node.SearchIterations) - 2)
	f(len(node.SearchIterations) - 1)

	return result
}

type searcher struct {
	Logger Logger

	OutOfTime bool

	Game      *GameState
	Bitboards *Bitboards

	Alpha            int
	Beta             int
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
		Alpha:            -Inf,
		Beta:             Inf,
		MaximizingPlayer: game.Player,
		DebugTree: SearchDebugTree{
			FenString: FenStringForGame(game),
		},
	}
}

func (s *searcher) scoreDirectionForPlayer(player Player) int {
	if player == s.MaximizingPlayer {
		return 1
	} else {
		return -1
	}
}

func (s *searcher) EvaluateMove(move Move) (int, SearchDebugNode, []error) {
	var score int
	var errors []error

	debugChild := createNode(move, s.Alpha, s.Beta)
	defer debugChild.finalize(s.Alpha, s.Beta, score)

	var update BoardUpdate
	err := s.Game.PerformMove(move, &update, s.Bitboards)
	if err != nil {
		errors = append(errors, err)
		return 0, debugChild, errors
	}

	defer func() {
		err = s.Game.UndoUpdate(&update, s.Bitboards)
		errors = append(errors, err)
	}()

	s.DebugTotalEvaluations++
	score = Evaluate(s.Bitboards, s.MaximizingPlayer)

	return score, debugChild, errors
}

func (s *searcher) Search() (Optional[Move], []error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for depth := 1; ; depth++ {
		debugSearches := s.DebugTree.addIteration()

		for i := range *moves {
			score, debugNode, errs := s.EvaluateMove((*moves)[i])
			*debugSearches = append(*debugSearches, debugNode)

			if len(errs) > 0 {
				return Empty[Move](), nil
			}

			(*moves)[i].Evaluation = Some(score)

			if s.OutOfTime {
				break
			}
		}

		sort.SliceStable(*moves, func(i, j int) bool {
			return (*moves)[j].Evaluation.Value() < (*moves)[i].Evaluation.Value()
		})

		s.Logger.Println("evaluated ",
			"- total evals", s.DebugTotalEvaluations,
			"- alpha", s.Alpha,
			"- beta", s.Beta,
			"- best move", (*moves)[0].String())

		if s.OutOfTime {
			break
		}
	}

	if len(*moves) == 0 {
		return Empty[Move](), nil
	}

	return Some((*moves)[0]), nil
}

func evaluateCapturesInner(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, error) {
	if KingIsInCheck(b, g.Enemy(), g.Player) {
		return SearchResult{Inf, 1, 1}, nil
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoCaptures(b, g, moves)

	if len(*moves) == 0 {
		score := Evaluate(b, g.Player)
		return SearchResult{score, 1, 1}, nil
	}

	totalSearched := 0

	for _, move := range *moves {
		if move.Evaluation.Value() < 100 {
			continue
		}

		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if err != nil {
			return SearchResult{}, fmt.Errorf("perform evaluateCapturesInner %v: %w", move.String(), err)
		}

		result, err := evaluateCapturesInner(g, b,
			-enemyCanForceScore,
			-playerCanForceScore)
		if err != nil {
			return SearchResult{}, fmt.Errorf("recurse evaluateCapturesInner %v: %w", move.String(), err)
		}
		enemyScore := result.Score
		totalSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if err != nil {
			return SearchResult{}, fmt.Errorf("undo evaluateCapturesInner %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, totalSearched}, nil
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, totalSearched}, nil
}

func evaluateCaptures(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, error) {
	standPat := Evaluate(b, g.Player)
	if standPat > enemyCanForceScore {
		return SearchResult{enemyCanForceScore, 1, 1}, nil
	} else if standPat > playerCanForceScore {
		playerCanForceScore = standPat
	}

	return evaluateCapturesInner(g, b, playerCanForceScore, enemyCanForceScore)
}

// func evaluateCaptures2(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (int, error) {
// 	result, err := evaluateCaptures(g, b, playerCanForceScore, enemyCanForceScore)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return result.Score, nil
// }

type SearchResult struct {
	Score              int
	TotalSearched      int
	QuiescenceSearched int
}

func evaluateSearch(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int, depth int) (SearchResult, error) {
	if KingIsInCheck(b, g.Enemy(), g.Player) {
		return SearchResult{Inf, 1, 0}, nil
	}

	if depth == 0 {
		score := Evaluate(b, g.Player)
		return SearchResult{score, 1, 0}, nil
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(b, g, moves)

	totalSearched := 0
	quiescenceSearched := 0

	for _, move := range *moves {
		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if err != nil {
			return SearchResult{}, fmt.Errorf("perform evaluateSearch %v: %w", move.String(), err)
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
		if err != nil {
			return SearchResult{}, fmt.Errorf("%v %v: %w", move.String(), depth, err)
		}

		enemyScore := result.Score
		totalSearched += result.TotalSearched
		quiescenceSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if err != nil {
			return SearchResult{}, fmt.Errorf("undo evaluateSearch %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, quiescenceSearched}, nil
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, quiescenceSearched}, nil
}

func Search(g *GameState, b *Bitboards, depth int, logger Logger) (Optional[Move], error) {
	defer profile.Start(profile.ProfilePath("../data/Search")).Stop()

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
		if err != nil {
			return Empty[Move](), fmt.Errorf("perform Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		result, err := evaluateSearch(g, b,
			-Inf, Inf, depth)
		if err != nil {
			return Empty[Move](), fmt.Errorf("evaluate Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		enemyScore := result.Score
		totalSearched += result.TotalSearched
		quiescenceSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if err != nil {
			return Empty[Move](), fmt.Errorf("undo Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		currentScore := -enemyScore
		logger.Println(i, "/", len(*moves), "searched", result.TotalSearched, "with initial search", result.TotalSearched-result.QuiescenceSearched, "and ending captures", result.QuiescenceSearched, "under", move.String(), "and found score", currentScore)

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

	return bestMoveSoFar, nil
}
