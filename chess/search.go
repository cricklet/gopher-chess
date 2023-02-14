package chess

import (
	"time"

	"github.com/pkg/profile"
)

var INF int = 999999

func evaluateCapturesInner(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) SearchResult {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return SearchResult{INF, 1, 1}
	}

	moves := GetMovesBuffer()
	defer func() {
		ReleaseMovesBuffer(moves)
	}()
	b.GenerateSortedPseudoCaptures(g, moves)

	if len(*moves) == 0 {
		score := b.evaluate(g.player)
		return SearchResult{score, 1, 1}
	}

	totalSearched := 0

	for _, move := range *moves {
		if move.evaluation.Value() < 200 {
			continue
		}

		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		result := evaluateCaptures(g, b,
			-enemyCanForceScore,
			-playerCanForceScore)
		enemyScore := result.score
		totalSearched += result.quiescenceSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, totalSearched}
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, totalSearched}
}

func evaluateCaptures(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) SearchResult {
	standPat := b.evaluate(g.player)
	if standPat > enemyCanForceScore {
		return SearchResult{enemyCanForceScore, 1, 1}
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

func evaluateSearch(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int, depth int) SearchResult {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return SearchResult{INF, 1, 0}
	}

	if depth == 0 {
		score := b.evaluate(g.player)
		return SearchResult{score, 1, 0}
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
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		str := g.Board.String()
		Ignore(str)

		b.performMove(g, move)
		g.performMove(move, update)

		var result SearchResult
		if depth == 1 && move.moveType == CAPTURE_MOVE {
			result = evaluateCaptures(g, b,
				-enemyCanForceScore,
				-playerCanForceScore)
		} else {
			result = evaluateSearch(g, b,
				-enemyCanForceScore,
				-playerCanForceScore,
				depth-1)
		}

		enemyScore := result.score
		totalSearched += result.totalSearched
		quiescenceSearched += result.quiescenceSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, quiescenceSearched}
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, quiescenceSearched}
}

func Search(g *GameState, b *Bitboards, depth int, logger Logger) Optional[Move] {
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
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		result := evaluateSearch(g, b,
			-INF, INF, depth)
		enemyScore := result.score
		totalSearched += result.totalSearched
		quiescenceSearched += result.quiescenceSearched

		b.undoUpdate(update)
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

	return bestMoveSoFar
}
