package chess

import "fmt"

var INF int = 999999

func evaluateSearch(g *GameState, b *Bitboards, bestScore int, ignoreScoresOver int, depth int) Optional[int] {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return Empty[int]()
	}

	if depth == 0 {
		return Some(b.evaluate(g.player))
	}

	ignoreCaptures := false
	if depth == 1 {
		ignoreCaptures = true
	}

	moves := GetMovesBuffer()
	b.GeneratePseudoMoves(g, moves)

	for _, move := range *moves {
		if ignoreCaptures && move.moveType != CAPTURE_MOVE {
			continue
		}
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		enemyScore := evaluateSearch(g, b, -ignoreScoresOver, -bestScore, depth-1)

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= ignoreScoresOver {
				return Some(ignoreScoresOver)
			}

			if currentScore > bestScore {
				bestScore = currentScore
			}
		}
	}

	ReleaseMovesBuffer(moves)

	return Some(bestScore)
}

func Search(g *GameState, b *Bitboards, depth int) Optional[Move] {
	moves := GetMovesBuffer()
	b.GeneratePseudoMoves(g, moves)

	bestMove := Empty[Move]()
	bestScore := -INF

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		enemyScore := evaluateSearch(g, b, -INF, INF, depth)

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			fmt.Println("searched", move.String(), "and found score", currentScore)

			if currentScore > bestScore {
				bestScore = currentScore
				bestMove = Some(move)
			}
		} else {
			fmt.Println("searched", move.String(), "and failed to find score")
		}
	}

	return bestMove
}
