package chess

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
	{4, 4, 4, 4, 4, 4, 4, 4},
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

var NULL_DEVELOPMENT_BITBOARDS = [2][]EvaluationBitboard{
	{},
	{},
}

var DEVELOPMENT_BITBOARDS = [][2][]EvaluationBitboard{
	ROOK_DEVELOPMENT_BITBOARDS,
	KNIGHT_DEVELOPMENT_BITBOARDS,
	BISHOP_DEVELOPMENT_BITBOARDS,
	NULL_DEVELOPMENT_BITBOARDS,
	QUEEN_DEVELOPMENT_BITBOARDS,
	PAWN_DEVELOPMENT_BITBOARDS,
	NULL_DEVELOPMENT_BITBOARDS,
}

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

var PIECE_SCORES = []int{
	500,
	300,
	350,
	0,
	900,
	100,
	0,
}

func (g *GameState) pieceScore(index int) int {
	return PIECE_SCORES[g.Board[index].pieceType()]
}

func (m *Move) Evaluate(g *GameState) int {
	score := 0
	if m.moveType == CAPTURE_MOVE {
		score += g.pieceScore(m.endIndex) - g.pieceScore(m.startIndex)
	}
	if m.moveType == EN_PASSANT_MOVE {
		score += 100
	}

	startDevelopment := evaluateDevelopment(
		singleBitboard(m.endIndex),
		DEVELOPMENT_BITBOARDS[g.Board[m.startIndex].pieceType()][g.player])
	endDevelopment :=
		evaluateDevelopment(
			singleBitboard(m.startIndex),
			DEVELOPMENT_BITBOARDS[g.Board[m.startIndex].pieceType()][g.player])

	score += startDevelopment - endDevelopment
	return score
}
