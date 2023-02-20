package search

import (
	"fmt"
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/fen"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/pkg/profile"
)

var Inf int = 999999

type Searcher struct {
	Logger    Logger
	Game      *GameState
	Bitboards *Bitboards

	evaluateWhenNumMovesApplied int

	currentLine [200]Move

	BestLine       [200]Move
	BestLineLength int

	DebugTotalEvaluations int
}

func (s *Searcher) Evaluate(numMovesApplied int, playerCanForceScore int, enemyCanForceScore int) (int, error) {
	if KingIsInCheck(s.Bitboards, s.Game.Enemy(), s.Game.Player) {
		return Inf, nil
	}

	if numMovesApplied == s.evaluateWhenNumMovesApplied {
		s.DebugTotalEvaluations++
		return Evaluate(s.Bitboards, s.Game.Player), nil
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for _, move := range *moves {
		s.currentLine[numMovesApplied] = move

		update := BoardUpdate{}
		err := s.Game.PerformMove(move, &update, s.Bitboards)
		if err != nil {
			return 0, err
		}

		var enemyScore int
		if numMovesApplied == s.evaluateWhenNumMovesApplied-1 && move.MoveType == CaptureMove {
			enemyScore, err = evaluateCaptures2(
				s.Game, s.Bitboards,
				-enemyCanForceScore,
				-playerCanForceScore)
		} else {
			enemyScore, err = s.Evaluate(
				numMovesApplied+1,
				-enemyCanForceScore,
				-playerCanForceScore)
		}
		if err != nil {
			return 0, err
		}

		err = s.Game.UndoUpdate(&update, s.Bitboards)
		if err != nil {
			return 0, fmt.Errorf("undo evaluateCapturesInner %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			// move is a suitable refutation -- it's good enough that the enemy will avoid this path
			// s.Logger.Println("refutation", move.String(), currentScore, enemyCanForceScore)
			return enemyCanForceScore, nil
		}

		if currentScore > playerCanForceScore {
			// move is a potential principle line -- it's our best option and the enemy can't avoid it
			// s.Logger.Println("principle line", move.String(), currentScore, playerCanForceScore)
			playerCanForceScore = currentScore

			s.BestLineLength = numMovesApplied
			for i := 0; i < numMovesApplied+1; i++ {
				s.BestLine[i] = s.currentLine[i]
			}
		}
	}

	return playerCanForceScore, nil
}

func (s *Searcher) Search(outOfTime *bool) (Optional[Move], error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(s.Bitboards, s.Game, moves)

	for s.evaluateWhenNumMovesApplied = 2; s.evaluateWhenNumMovesApplied <= 5; s.evaluateWhenNumMovesApplied++ {
		for _, m := range *moves {
			s.currentLine[0] = m

			update := BoardUpdate{}
			err := s.Game.PerformMove(m, &update, s.Bitboards)
			if err != nil {
				return Empty[Move](), err
			}

			enemyScore, err := s.Evaluate(1, -Inf, Inf)
			s.Logger.Println(m, -enemyScore)
			if err != nil {
				return Empty[Move](), err
			}

			m.Evaluation = Some(-enemyScore)

			err = s.Game.UndoUpdate(&update, s.Bitboards)
			if err != nil {
				return Empty[Move](), err
			}
		}

		if s.BestLineLength == 0 {
			return Empty[Move](), fmt.Errorf("failed to find best line for %v", FenStringForGame(s.Game))
		}

		if *outOfTime {
			break
		}
	}

	if len(*moves) == 0 {
		// Checkmate
		return Empty[Move](), nil
	}

	return Some(s.BestLine[0]), nil
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

func evaluateCaptures2(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (int, error) {
	result, err := evaluateCaptures(g, b, playerCanForceScore, enemyCanForceScore)
	if err != nil {
		return 0, err
	}

	return result.Score, nil
}

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

/*
type BestMove struct {
	move Move
	score int
	depth int
	scoreType
		Best // either exact score (eg part of a best-sequence) or the best we've found (eg improved alpha)
		Refutation // a move that was good enough that the enemy will try to avoid it (eg improved beta)
}

type SearchCache struct {
	map[ZobristHash]BestMove
}

type Searcher struct {
	maxDepth int
	principleVariation [64]Move
	principleVariationLength int

	cache *SearchCache
}

func (s *Searcher) Evaluate(g, b, depth, playerCanForceScore, enemyCanForceScore) {
	GenerateSortedPseudoMoves(g, b, &moves)

	bestMove := Empty[Move]()
	bestScore := -Inf

	previousBestMove := InPrincipleVariation(...) ?
		s.principleVariation[depth] :
		s.cache.BestMove(g)

	for _, m := range Concat(previousBestMove, moves) {
		g.PerformMove(previousBestMove)
		enemyScore = Evaluate(g, b, depth + 1)
		g.UndoMove()

		if score >= enemyCanForceScore {
			// Refutation move, enemy will avoid.
			s.cache.Add(g, m, score, s.maxDepth - depth, Refutation)
			return enemyCanForceScore
		} else if score > playerCanForceScore {
			playerCanForceScore = score
		}

		if score > bestScore {
			bestMove = m
			bestScore = score
		}
	}

	s.cache.Add(g, bestMove, bestScore, s.maxDepth - depth, Best)
	return playerCanForceScore
}

func (s *Searcher) Search(g, b, outOfTime *bool) {
	GenerateSortedPseudoMoves(g, b, &moves)

	for ; s.maxDepth < 8; s.maxDepth ++ {
		for m := range moves {
			g.PerformMove(m)
			m.evaluation := Some(s.Evaluate(g, b, 0))
			g.UndoMove()
		}

		if outOfTime {
			break
		}
	}

	Sort(&moves)
	return Last(moves)
}
*/
