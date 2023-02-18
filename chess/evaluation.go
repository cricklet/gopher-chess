package chess

import (
	. "github.com/cricklet/chessgo/internal/helpers"
)

type EvaluationBitboard struct {
	multiplier int
	b          Bitboard
}

var DEVELOPMENT_SCALE = 10

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
				b |= SingleBitboard(index)
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
		evaluationsFromArray(FlipArray(whiteOrientedEvalArray), scale),
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
	development += evaluateDevelopment(b.Players[player].Pieces[Rook], ROOK_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.Players[player].Pieces[Knight], KNIGHT_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.Players[player].Pieces[Bishop], BISHOP_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.Players[player].Pieces[Queen], QUEEN_DEVELOPMENT_BITBOARDS[player])
	development += evaluateDevelopment(b.Players[player].Pieces[Pawn], PAWN_DEVELOPMENT_BITBOARDS[player])
	return development
}

func (b *Bitboards) evaluate(player Player) int {
	enemy := player.Other()

	pieceValues :=
		500*OnesCount(b.Players[player].Pieces[Rook]) +
			300*OnesCount(b.Players[player].Pieces[Knight]) +
			350*OnesCount(b.Players[player].Pieces[Bishop]) +
			900*OnesCount(b.Players[player].Pieces[Queen]) +
			100*OnesCount(b.Players[player].Pieces[Pawn])

	enemyValues :=
		500*OnesCount(b.Players[enemy].Pieces[Rook]) +
			300*OnesCount(b.Players[enemy].Pieces[Knight]) +
			350*OnesCount(b.Players[enemy].Pieces[Bishop]) +
			900*OnesCount(b.Players[enemy].Pieces[Queen]) +
			100*OnesCount(b.Players[enemy].Pieces[Pawn])

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
	return PIECE_SCORES[g.Board[index].PieceType()]
}

func EvaluateMove(m *Move, g *GameState) int {
	score := 0
	if m.MoveType == CaptureMove {
		score += g.pieceScore(m.EndIndex) - g.pieceScore(m.StartIndex)
	}
	if m.MoveType == EnPassantMove {
		score += 100
	}
	if m.MoveType == CastlingMove {
		score += 500
	}

	startDevelopment := evaluateDevelopment(
		SingleBitboard(m.EndIndex),
		DEVELOPMENT_BITBOARDS[g.Board[m.StartIndex].PieceType()][g.Player])
	endDevelopment :=
		evaluateDevelopment(
			SingleBitboard(m.StartIndex),
			DEVELOPMENT_BITBOARDS[g.Board[m.StartIndex].PieceType()][g.Player])

	score += startDevelopment - endDevelopment
	return score
}
