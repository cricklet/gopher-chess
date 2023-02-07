package chess

import "fmt"

type EvaluationBitboard struct {
	multiplier int
	b          Bitboard
}

var DEVELOPMENT_SCALE = 50

var ROOK_DEVELOPMENT_BITBOARDS = evaluationsPerPlayer([8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},
	{1, 2, 2, 2, 2, 2, 2, 1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{0, 0, 0, 2, 2, 0, 0, 0},
}, DEVELOPMENT_SCALE)

var PAWN_DEVELOPMENT_BITBOARDS = evaluationsPerPlayer([8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},
	{3, 3, 3, 4, 4, 3, 3, 3},
	{2, 2, 2, 3, 3, 2, 2, 2},
	{2, 2, 2, 3, 3, 2, 2, 2},
	{1, 1, 1, 3, 3, 1, 1, 1},
	{0, 0, 0, 2, 2, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
}, DEVELOPMENT_SCALE)

var BISHOP_DEVELOPMENT_BITBOARDS = evaluationsPerPlayer([8][8]int{
	{-1, -1, -1, -1, -1, -1, -1, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 1, 2, 2, 1, 0, -1},
	{-1, 1, 1, 2, 2, 1, 1, -1},
	{-1, 0, 2, 2, 2, 2, 0, -1},
	{-1, 2, 2, 2, 2, 2, 2, -1},
	{-1, 1, 0, 0, 0, 0, 1, -1},
	{-1, -1, -1, -1, -1, -1, -1, -1},
}, DEVELOPMENT_SCALE)
var KNIGHT_DEVELOPMENT_BITBOARDS = evaluationsPerPlayer([8][8]int{
	{-2, -2, -2, -2, -2, -2, -2, -2},
	{-2, -1, 0, 0, 0, 0, -1, -2},
	{-2, 0, 1, 2, 2, 1, 0, -2},
	{-2, 1, 2, 2, 2, 2, 1, -2},
	{-2, 0, 2, 2, 2, 2, 0, -2},
	{-2, 1, 1, 2, 2, 1, 1, -2},
	{-2, -1, 0, 0, 0, 0, -1, -2},
	{-2, -2, -2, -2, -2, -2, -2, -2},
}, DEVELOPMENT_SCALE)
var QUEEN_DEVELOPMENT_BITBOARDS = evaluationsPerPlayer([8][8]int{
	{-2, -2, -2, -1, -1, -2, -2, -2},
	{-2, 0, 0, 0, 0, 0, 0, -2},
	{-2, 0, 1, 1, 1, 1, 0, -2},
	{-1, 0, 1, 1, 1, 1, 0, -1},
	{0, 0, 1, 1, 1, 1, 0, 0},
	{-2, 0, 1, 1, 1, 1, 0, -2},
	{-2, 0, 1, 0, 0, 1, 0, -2},
	{-2, -2, -2, -1, -1, -2, -2, -20},
}, DEVELOPMENT_SCALE)

func bitboardFromArray(lookup int, array [8][8]int) Bitboard {
	b := Bitboard(0)
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if array[i][j] == lookup {
				index := (7-i)*8 + j
				b |= singleBitboard(index)
			}
		}
	}
	return b
}

func evaluationsFromArray(array [8][8]int, scale int) []EvaluationBitboard {
	result := []EvaluationBitboard{}
	scores := map[int]bool{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			scores[array[i][j]] = true
		}
	}
	for k := range scores {
		eval := EvaluationBitboard{}
		eval.multiplier = k * scale
		eval.b = bitboardFromArray(k, array)
		result = append(result, eval)
	}
	return result
}

func evaluationsPerPlayer(whiteOrientedEvalArray [8][8]int, scale int) [2][]EvaluationBitboard {
	return [2][]EvaluationBitboard{
		evaluationsFromArray(whiteOrientedEvalArray, scale),
		evaluationsFromArray(flipArray(whiteOrientedEvalArray), scale),
	}
}

func evaluateDevelopment(b Bitboard, e []EvaluationBitboard) int {
	result := 0
	for _, eval := range e {
		result += eval.multiplier * OnesCount(eval.b&b)
	}
	return result
}

func (b *Bitboards) evaluateDevelopment(player Player) int {
	development := 0
	development += evaluateDevelopment(b.players[player].pieces[ROOK], ROOK_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.players[player].pieces[KNIGHT], KNIGHT_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.players[player].pieces[BISHOP], BISHOP_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.players[player].pieces[QUEEN], QUEEN_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.players[player].pieces[PAWN], PAWN_DEVELOPMENT_BITBOARDS[player])
	return development
}

func (b *Bitboards) evaluate(player Player) int {
	enemy := player.other()

	pieceValues :=
		500*OnesCount(b.players[player].pieces[ROOK]) +
			300*OnesCount(b.players[player].pieces[KNIGHT]) +
			350*OnesCount(b.players[player].pieces[BISHOP]) +
			900*OnesCount(b.players[player].pieces[QUEEN]) +
			100*OnesCount(b.players[player].pieces[PAWN])

	enemyValues :=
		500*OnesCount(b.players[enemy].pieces[ROOK]) +
			300*OnesCount(b.players[enemy].pieces[KNIGHT]) +
			350*OnesCount(b.players[enemy].pieces[BISHOP]) +
			900*OnesCount(b.players[enemy].pieces[QUEEN]) +
			100*OnesCount(b.players[enemy].pieces[PAWN])

	developmentValues := b.evaluateDevelopment(player)
	enemyDevelopmentValues := b.evaluateDevelopment(enemy)

	return pieceValues + developmentValues - enemyValues - enemyDevelopmentValues
}

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
