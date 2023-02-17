package chess

import (
	"fmt"
	"time"

	"github.com/pkg/profile"
)

var INF int = 999999

func evaluateCapturesInner(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, error) {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return SearchResult{INF, 1, 1}, nil
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	b.GenerateSortedPseudoCaptures(g, moves)

	if len(*moves) == 0 {
		score := b.evaluate(g.player)
		return SearchResult{score, 1, 1}, nil
	}

	totalSearched := 0

	for _, move := range *moves {
		if move.evaluation.Value() < 200 {
			continue
		}

		update := BoardUpdate{}
		previous := OldGameState{}
		err := SetupBoardUpdate(g, move, &update)
		if err != nil {
			return SearchResult{}, fmt.Errorf("setup evaluateCapturesInner %v: %w", move.String(), err)
		}

		RecordCurrentState(g, &previous)

		err = b.performMove(g, move)
		if err != nil {
			return SearchResult{}, fmt.Errorf("perform evaluateCapturesInner %v: %w", move.String(), err)
		}
		g.performMove(move, update)

		result, err := evaluateCapturesInner(g, b,
			-enemyCanForceScore,
			-playerCanForceScore)
		if err != nil {
			return SearchResult{}, fmt.Errorf("recurse evaluateCapturesInner %v: %w", move.String(), err)
		}
		enemyScore := result.score
		totalSearched += result.quiescenceSearched

		err = b.undoUpdate(update)
		if err != nil {
			return SearchResult{}, fmt.Errorf("undo evaluateCapturesInner %v: %w", move.String(), err)
		}
		g.undoUpdate(previous, update)

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
	standPat := b.evaluate(g.player)
	if standPat > enemyCanForceScore {
		return SearchResult{enemyCanForceScore, 1, 1}, nil
	} else if standPat > playerCanForceScore {
		playerCanForceScore = standPat
	}

	return evaluateCapturesInner(g, b, playerCanForceScore, enemyCanForceScore)
}

type SearchResult struct {
	score              int
	totalSearched      int
	quiescenceSearched int
}

func evaluateSearch(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int, depth int) (SearchResult, error) {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return SearchResult{INF, 1, 0}, nil
	}

	if depth == 0 {
		score := b.evaluate(g.player)
		return SearchResult{score, 1, 0}, nil
	}

	moves := GetMovesBuffer()
	defer func() {
		ReleaseMovesBuffer(moves)
	}()

	b.GenerateSortedPseudoMoves(g, moves)

	totalSearched := 0
	quiescenceSearched := 0

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		err := SetupBoardUpdate(g, move, &update)
		if err != nil {
			return SearchResult{}, fmt.Errorf("setup evaluateSearch %v: %w", move.String(), err)
		}
		RecordCurrentState(g, &previous)

		str := g.Board.String()
		Ignore(str)

		err = b.performMove(g, move)
		if err != nil {
			return SearchResult{}, fmt.Errorf("perform evaluateSearch %v: %w", move.String(), err)
		}

		g.performMove(move, update)

		var result SearchResult
		if depth == 1 && move.moveType == CAPTURE_MOVE {
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

		enemyScore := result.score
		totalSearched += result.totalSearched
		quiescenceSearched += result.quiescenceSearched

		err = b.undoUpdate(update)
		if err != nil {
			return SearchResult{}, fmt.Errorf("undo evaluateSearch %v: %w", move.String(), err)
		}
		g.undoUpdate(previous, update)

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
	b.GenerateSortedPseudoMoves(g, moves)

	bestMoveSoFar := Empty[Move]()
	bestScoreSoFar := -INF

	quiescenceSearched := 0
	totalSearched := 0

	startTime := time.Now()

	for i, move := range *moves {
		update, previous := BoardUpdate{}, OldGameState{}
		err := SetupBoardUpdate(g, move, &update)
		if err != nil {
			return Empty[Move](), fmt.Errorf("setup Search %v => %v: %w", g.fenString(), move.String(), err)
		}
		RecordCurrentState(g, &previous)

		err = b.performMove(g, move)
		if err != nil {
			return Empty[Move](), fmt.Errorf("perform Search %v => %v: %w", g.fenString(), move.String(), err)
		}
		g.performMove(move, update)

		result, err := evaluateSearch(g, b,
			-INF, INF, depth)
		if err != nil {
			return Empty[Move](), fmt.Errorf("evaluate Search %v => %v: %w", g.fenString(), move.String(), err)
		}

		enemyScore := result.score
		totalSearched += result.totalSearched
		quiescenceSearched += result.quiescenceSearched

		err = b.undoUpdate(update)
		if err != nil {
			return Empty[Move](), fmt.Errorf("undo Search %v => %v: %w", g.fenString(), move.String(), err)
		}
		g.undoUpdate(previous, update)

		currentScore := -enemyScore
		logger.Println(i, "/", len(*moves), "searched", result.totalSearched, "with initial search", result.totalSearched-result.quiescenceSearched, "and ending captures", result.quiescenceSearched, "under", move.String(), "and found score", currentScore)

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
