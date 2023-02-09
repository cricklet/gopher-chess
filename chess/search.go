package chess

import (
	"time"

	"github.com/pkg/profile"
)

var INF int = 999999

func evaluateCaptures(g *GameState, b *Bitboards, bestScoreSoFar int, enemyWillAvoidIfBetterThan int) (Optional[int], int) {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return Empty[int](), 1
	}

	moves := GetMovesBuffer()
	defer func() {
		ReleaseMovesBuffer(moves)
	}()
	b.GenerateSortedPseudoCaptures(g, moves)

	if len(*moves) == 0 {
		score := b.evaluate(g.player)
		return Some(score), 1
	}

	totalSearched := 0

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		enemyScore, numSearched := evaluateCaptures(g, b,
			-enemyWillAvoidIfBetterThan,
			-bestScoreSoFar)
		totalSearched += numSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= enemyWillAvoidIfBetterThan {
				return Some(enemyWillAvoidIfBetterThan), totalSearched
			}

			if currentScore > bestScoreSoFar {
				bestScoreSoFar = currentScore
			}
		}
	}

	return Some(bestScoreSoFar), totalSearched
}

func evaluateSearch(g *GameState, b *Bitboards, bestScoreSoFar int, enemyWillAvoidIfBetterThan int, depth int) (Optional[int], int) {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return Empty[int](), 1
	}

	if depth == 0 {
		score := b.evaluate(g.player)
		return Some(score), 1
	}

	moves := GetMovesBuffer()
	defer func() {
		ReleaseMovesBuffer(moves)
	}()

	b.GenerateSortedPseudoMoves(g, moves)

	totalSearched := 0

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		var enemyScore Optional[int]
		var numSearched int
		if depth == 1 && move.moveType == CAPTURE_MOVE {
			enemyScore, numSearched = evaluateCaptures(g, b,
				-enemyWillAvoidIfBetterThan,
				-bestScoreSoFar)
		} else {
			enemyScore, numSearched = evaluateSearch(g, b,
				-enemyWillAvoidIfBetterThan,
				-bestScoreSoFar,
				depth-1)
		}
		totalSearched += numSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= enemyWillAvoidIfBetterThan {
				return Some(enemyWillAvoidIfBetterThan), totalSearched
			}

			if currentScore > bestScoreSoFar {
				bestScoreSoFar = currentScore
			}
		}
	}

	return Some(bestScoreSoFar), totalSearched
}

func Search(g *GameState, b *Bitboards, depth int, logger Logger) Optional[Move] {
	defer profile.Start(profile.ProfilePath("../data/Search")).Stop()

	moves := GetMovesBuffer()
	b.GenerateSortedPseudoMoves(g, moves)

	bestMoveSoFar := Empty[Move]()
	bestScoreSoFar := -INF

	totalSearched := 0

	startTime := time.Now()

	for i, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		enemyScore, numSearched := evaluateSearch(g, b,
			-INF, INF, depth)
		totalSearched += numSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			logger.Println(i, "/", len(*moves), "searched", numSearched, "under", move.String(), "and found score", currentScore)

			// if currentScore >= enemyWillAvoidIfBetterThan {
			// 	enemyWillAvoidIfBetterThan = currentScore
			// } else
			if currentScore > bestScoreSoFar {
				bestScoreSoFar = currentScore
				bestMoveSoFar = Some(move)
			}
		} else {
			logger.Println("searched", move.String(), "and and failed to find score")
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
			logger.Println("searched", totalSearched, "nodes in", time.Since(startTime), ", ~ perft of ply", i, "(", PLY_COUNTS[i], ")")
			break
		}
	}

	return bestMoveSoFar
}
