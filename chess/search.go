package chess

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/profile"
)

var INF int = 999999

func evaluateCaptures(g *GameState, b *Bitboards, alpha int, beta int) (Optional[int], int) {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return Empty[int](), 1
	}

	moves := GetMovesBuffer()
	defer func() {
		ReleaseMovesBuffer(moves)
	}()
	b.GeneratePseudoCaptures(g, moves)

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
			-beta,
			-alpha)
		totalSearched += numSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= beta {
				return Some(beta), totalSearched
			}

			if currentScore > alpha {
				alpha = currentScore
			}
		}
	}

	return Some(alpha), totalSearched
}

func evaluateSearch(g *GameState, b *Bitboards, alpha int, beta int, depth int) (Optional[int], int) {
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

	b.GeneratePseudoMoves(g, moves)

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
				-beta,
				-alpha)
		} else {
			enemyScore, numSearched = evaluateSearch(g, b,
				-beta,
				-alpha,
				depth-1)
		}
		totalSearched += numSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= beta {
				return Some(beta), totalSearched
			}

			if currentScore > alpha {
				alpha = currentScore
			}
		}
	}

	return Some(alpha), totalSearched
}

func Search(g *GameState, b *Bitboards, depth int) Optional[Move] {
	defer profile.Start(profile.ProfilePath("../data/Search")).Stop()

	moves := GetMovesBuffer()
	b.GeneratePseudoMoves(g, moves)

	bestMove := Empty[Move]()
	bestScore := -INF

	totalSearched := 0

	startTime := time.Now()

	for i, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		enemyScore, numSearched := evaluateSearch(g, b, -INF, INF, depth)
		totalSearched += numSearched

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			log.Println(i, "/", len(*moves), "searched", numSearched, "under", move.String(), "and found score", currentScore)

			if currentScore > bestScore {
				bestScore = currentScore
				bestMove = Some(move)
			}
		} else {
			log.Println("searched", move.String(), "and and failed to find score")
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
			fmt.Println("searched", totalSearched, "nodes in", time.Now().Sub(startTime), ", which is less than a perft of ply", i, "(", PLY_COUNTS[i], ")")
			break
		}
	}

	return bestMove
}
