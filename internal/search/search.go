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

type SearchDebugTree struct {
	Move           string
	FenString      string
	StartingAlpha  int
	StartingBeta   int
	EndingAlpha    int
	EndingBeta     int
	ReturnedScore  int
	MoveEvaluation MoveEvaluation
	Children       []SearchDebugTree
}

func (tree *SearchDebugTree) AddChild(move Move, fenString string, alpha, beta int) *SearchDebugTree {
	tree.Children = append(tree.Children, SearchDebugTree{
		Move:          move.String(),
		FenString:     fenString,
		StartingAlpha: alpha,
		StartingBeta:  beta,
		Children:      []SearchDebugTree{},
	})
	return &tree.Children[len(tree.Children)-1]
}

func (child *SearchDebugTree) Finalize(alpha, beta, score int, evaluation MoveEvaluation) {
	child.EndingAlpha = alpha
	child.EndingBeta = beta
	child.ReturnedScore = score
	child.MoveEvaluation = evaluation
}

type searcher struct {
	Logger    Logger
	Game      *GameState
	Bitboards *Bitboards

	alpha            int
	beta             int
	maximizingPlayer Player
	minimizingPlayer Player

	evaluateWhenNumMovesApplied int

	DebugTotalEvaluations         int
	DebugEvaluationsThisIteration int

	DebugTreeRoot SearchDebugTree
}

func NewSearcher(logger Logger, game *GameState, bitboards *Bitboards) searcher {
	return searcher{
		Logger:           logger,
		Game:             game,
		Bitboards:        bitboards,
		alpha:            -Inf,
		beta:             Inf,
		maximizingPlayer: game.Player,
		minimizingPlayer: game.Enemy(),
		DebugTreeRoot:    SearchDebugTree{},
	}
}

type MoveEvaluation int

const (
	IllegalMove MoveEvaluation = iota
	AllMove
	BestMove
	RefutationMove
)

func (s *searcher) EvaluateMove(numMovesApplied int, move Move, debugTree *SearchDebugTree) (int, MoveEvaluation, error) {
	var moveScore int
	moveEvaluation := AllMove

	childDebugTree := debugTree.AddChild(move, FenStringForGame(s.Game), s.alpha, s.beta)
	defer childDebugTree.Finalize(s.alpha, s.beta, moveScore, moveEvaluation)

	if numMovesApplied == s.evaluateWhenNumMovesApplied {
		s.DebugTotalEvaluations++
		s.DebugEvaluationsThisIteration++
		moveScore = Evaluate(s.Bitboards, s.maximizingPlayer)
		return moveScore, AllMove, nil
	}

	update := BoardUpdate{}
	err := s.Game.PerformMove(move, &update, s.Bitboards)
	if err != nil {
		return 0, moveEvaluation, err
	}

	if KingIsInCheck(s.Bitboards, s.Game.Enemy(), s.Game.Player) {
		moveEvaluation = IllegalMove
		if s.maximizingPlayer == s.Game.Enemy() {
			moveScore = Inf
		} else {
			moveScore = -Inf
		}
	} else {
		moveScore, moveEvaluation, err = s.Evaluate(numMovesApplied+1, childDebugTree)
		if err != nil {
			return 0, moveEvaluation, err
		}
	}

	err = s.Game.UndoUpdate(&update, s.Bitboards)
	if err != nil {
		return 0, moveEvaluation, fmt.Errorf("undo evaluateCapturesInner %v: %w", move.String(), err)
	}

	if s.maximizingPlayer == s.Game.Player {
		if moveScore >= s.beta {
			moveEvaluation = RefutationMove
			return s.beta, moveEvaluation, nil
		} else if moveScore > s.alpha {
			moveEvaluation = BestMove
			s.alpha = moveScore
		}
	} else {
		if moveScore <= s.alpha {
			moveEvaluation = RefutationMove
			return s.alpha, moveEvaluation, nil
		} else if moveScore < s.beta {
			moveEvaluation = BestMove
			s.beta = moveScore
		}
	}
	return moveScore, moveEvaluation, nil
}

func (s *searcher) Evaluate(
	numMovesApplied int,
	debugTree *SearchDebugTree,
	// TODO exhaust captures
) (int, MoveEvaluation, error) {
	evaluation := AllMove

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for _, move := range *moves {
		score, evaluation, err := s.EvaluateMove(numMovesApplied, move, debugTree)
		if err != nil {
			return 0, evaluation, err
		}

		if evaluation == RefutationMove {
			return score, evaluation, nil
		}
	}

	if s.maximizingPlayer == s.Game.Player {
		return s.alpha, evaluation, nil
	} else {
		return s.beta, evaluation, nil
	}
}

func (s *searcher) Search(outOfTime *bool) (Optional[Move], error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for s.evaluateWhenNumMovesApplied = 3; s.evaluateWhenNumMovesApplied <= 10; s.evaluateWhenNumMovesApplied++ {
		s.DebugTreeRoot = SearchDebugTree{}
		s.DebugEvaluationsThisIteration = 0

		for _, m := range *moves {
			err := func() error {
				var err error
				var score int
				childDebugTree := s.DebugTreeRoot.AddChild(m, FenStringForGame(s.Game), s.alpha, s.beta)
				defer childDebugTree.Finalize(s.alpha, s.beta, score, 0)

				update := BoardUpdate{}
				err = s.Game.PerformMove(m, &update, s.Bitboards)
				if err != nil {
					return err
				}

				score, _, err = s.Evaluate(1, &s.DebugTreeRoot)
				if err != nil {
					return err
				}

				m.Evaluation = Some(score)

				err = s.Game.UndoUpdate(&update, s.Bitboards)
				if err != nil {
					return err
				}

				return nil
			}()

			if err != nil {
				return Empty[Move](), err
			}

			if *outOfTime {
				break
			}
		}

		sort.SliceStable(*moves, func(i, j int) bool {
			return (*moves)[i].Evaluation.Value() > (*moves)[j].Evaluation.Value()
		})

		s.Logger.Println("evaluated up to", s.evaluateWhenNumMovesApplied,
			"- iteration evals", s.DebugEvaluationsThisIteration,
			"- total evals", s.DebugTotalEvaluations,
			"- alpha", s.alpha,
			"- beta", s.beta,
			"- best move", (*moves)[0].String())

		if *outOfTime {
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
