package chess

import "fmt"

var INF int = 999999

func evaluateCaptures(g *GameState, b *Bitboards, alpha int, beta int) Optional[int] {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return Empty[int]()
	}

	moves := GetMovesBuffer()
	b.GeneratePseudoCaptures(g, moves)

	if len(*moves) == 0 {
		score := b.evaluate(g.player)
		ReleaseMovesBuffer(moves)
		return Some(score)
	}

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		enemyScore := evaluateCaptures(g, b,
			-beta,
			-alpha)

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= beta {
				return Some(beta)
			}

			if currentScore > alpha {
				alpha = currentScore
			}
		}
	}

	ReleaseMovesBuffer(moves)

	return Some(alpha)
}

func evaluateSearch(g *GameState, b *Bitboards, alpha int, beta int, depth int) Optional[int] {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return Empty[int]()
	}

	if depth == 0 {
		score := b.evaluate(g.player)
		return Some(score)
	}

	moves := GetMovesBuffer()
	b.GeneratePseudoMoves(g, moves)

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		var enemyScore Optional[int]
		if depth == 1 && move.moveType == CAPTURE_MOVE {
			enemyScore = evaluateCaptures(g, b,
				-beta,
				-alpha)
		} else {
			enemyScore = evaluateSearch(g, b,
				-beta,
				-alpha,
				depth-1)
		}

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		if enemyScore.HasValue() {
			currentScore := -enemyScore.Value()
			if currentScore >= beta {
				return Some(beta)
			}

			if currentScore > alpha {
				alpha = currentScore
			}
		}
	}

	ReleaseMovesBuffer(moves)

	return Some(alpha)
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
