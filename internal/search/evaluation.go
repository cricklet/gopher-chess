package search

import (
	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type EvaluationBitboard struct {
	multiplier int
	b          Bitboard
}

var _developmentScale = 10

var RookDevelopmentBitboards = evaluationsPerPlayer([8][8]int{
	{0, 0, 0, 1, 1, 0, 0, 0},
	{0, 2, 2, 2, 2, 2, 2, 0},
	{1, 0, 0, 0, 0, 0, 0, 1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{0, 0, 0, 2, 2, 0, 0, 0},
}, _developmentScale)

var PawnDevelopmentBitboards = evaluationsPerPlayer([8][8]int{
	{4, 4, 4, 4, 4, 4, 4, 4},
	{3, 3, 3, 4, 4, 3, 3, 3},
	{3, 3, 3, 3, 3, 3, 3, 3},
	{2, 2, 2, 1, 1, 2, 2, 2},
	{1, 1, 1, 3, 3, 1, 1, 1},
	{0, 1, 1, 2, 2, 1, 1, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
}, _developmentScale*2)

var BishopDevelopmentBitboards = evaluationsPerPlayer([8][8]int{
	{-1, -1, -1, -1, -1, -1, -1, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 1, 1, 1, 1, 0, -1},
	{-1, 1, 1, 2, 2, 1, 1, -1},
	{-1, 0, 1, 2, 2, 1, 0, -1},
	{-1, 2, 2, 2, 2, 2, 2, -1},
	{-1, 1, 0, 0, 0, 0, 1, -1},
	{-1, -1, -1, -1, -1, -1, -1, -1},
}, _developmentScale)
var KnightDevelopmentBitboards = evaluationsPerPlayer([8][8]int{
	{-2, -2, -2, -2, -2, -2, -2, -2},
	{-2, -1, 0, 0, 0, 0, -1, -2},
	{-2, 0, 1, 2, 2, 1, 0, -2},
	{-2, 1, 2, 2, 2, 2, 1, -2},
	{-2, 0, 2, 2, 2, 2, 0, -2},
	{-2, 1, 1, 2, 2, 1, 1, -2},
	{-2, -1, 0, 0, 0, 0, -1, -2},
	{-2, -2, -2, -2, -2, -2, -2, -2},
}, _developmentScale)
var QueenDevelopmentBitboards = evaluationsPerPlayer([8][8]int{
	{-1, -1, -1, -1, -1, -1, -1, -1},
	{-1, 1, 1, 1, 1, 1, 1, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{-1, 0, 0, 0, 0, 0, 0, -1},
	{-1, 0, 0, 1, 1, 0, 0, -1},
	{-1, -1, -1, 0, 0, -1, -1, -1},
}, _developmentScale/2)
var EnemyKingEndgameBitboards = evaluationsPerPlayer([8][8]int{
	{4, 4, 3, 3, 3, 3, 4, 4},
	{4, 3, 2, 2, 2, 2, 3, 4},
	{3, 2, 0, 0, 0, 0, 2, 3},
	{3, 2, 0, 0, 0, 0, 2, 3},
	{3, 2, 0, 0, 0, 0, 2, 3},
	{3, 2, 0, 0, 0, 0, 2, 3},
	{4, 3, 2, 2, 2, 2, 3, 4},
	{4, 4, 3, 3, 3, 3, 4, 4},
}, _developmentScale*3)

var NullDevelopmentBitboards = [2][]EvaluationBitboard{
	{},
	{},
}

var AllDevelopmentBitboards = [][2][]EvaluationBitboard{
	RookDevelopmentBitboards,
	KnightDevelopmentBitboards,
	BishopDevelopmentBitboards,
	NullDevelopmentBitboards,
	QueenDevelopmentBitboards,
	PawnDevelopmentBitboards,
	NullDevelopmentBitboards,
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

func evaluateDevelopmentForPiece(b Bitboard, e []EvaluationBitboard) int {
	result := 0
	for _, eval := range e {
		result += eval.multiplier * OnesCount(eval.b&b)
	}
	return result
}

var PawnCenterBitboards = [2]Bitboard{
	BitboardFromStrings([8]string{
		"00000000",
		"00000000",
		"00001000",
		"00001000",
		"00001000",
		"00001000",
		"00000000",
		"00000000",
	}),
	BitboardFromStrings([8]string{
		"00000000",
		"00000000",
		"00010000",
		"00010000",
		"00010000",
		"00010000",
		"00000000",
		"00000000",
	}),
}

func EvaluateDevelopment(b *Bitboards, player Player) int {
	development := 0
	development += evaluateDevelopmentForPiece(b.Players[player].Pieces[Rook], RookDevelopmentBitboards[player])
	development += evaluateDevelopmentForPiece(b.Players[player].Pieces[Knight], KnightDevelopmentBitboards[player])
	development += evaluateDevelopmentForPiece(b.Players[player].Pieces[Bishop], BishopDevelopmentBitboards[player])
	development += evaluateDevelopmentForPiece(b.Players[player].Pieces[Pawn], PawnDevelopmentBitboards[player])

	pawnsInCenter := 0
	for _, pawnCenter := range PawnCenterBitboards {
		if pawnCenter&b.Players[player].Pieces[Pawn] != 0 {
			pawnsInCenter += 1
		}
	}
	if pawnsInCenter == 2 {
		development += _developmentScale * 2
	} else if pawnsInCenter == 1 {
		development += _developmentScale * 1
	}

	return development
}

func EvaluatePieces(b *Bitboards, player Player) int {
	pieceValues :=
		500*OnesCount(b.Players[player].Pieces[Rook]) +
			300*OnesCount(b.Players[player].Pieces[Knight]) +
			350*OnesCount(b.Players[player].Pieces[Bishop]) +
			900*OnesCount(b.Players[player].Pieces[Queen]) +
			100*OnesCount(b.Players[player].Pieces[Pawn])

	return pieceValues
}

func Evaluate(b *Bitboards, player Player, args ...EvaluationOption) int {
	enemy := player.Other()

	developmentValues := EvaluateDevelopment(b, player)
	enemyDevelopmentValues := EvaluateDevelopment(b, enemy)

	pieceValues := EvaluatePieces(b, player)
	enemyPieceValues := EvaluatePieces(b, enemy)

	result := pieceValues - enemyPieceValues + developmentValues - enemyDevelopmentValues

	for _, arg := range args {
		if arg == EndgamePushEnemyKing {
			if enemyPieceValues <= 500 {
				result += evaluateDevelopmentForPiece(
					b.Players[enemy].Pieces[King],
					EnemyKingEndgameBitboards[enemy])
				if KingIsInCheck(b, enemy) {
					result += 10
				}
			}
		}
	}

	return result
}

var _pieceScores = []int{
	500,
	300,
	350,
	0,
	900,
	100,
	0,
}

func pieceScore(g *GameState, index int) int {
	return _pieceScores[g.Board[index].PieceType()]
}

type EvaluationOption int

const (
	Default EvaluationOption = iota
	EndgamePushEnemyKing
)

func EvaluateMove(m *Move, g *GameState, args ...EvaluationOption) int {
	score := 0
	if m.MoveType == CaptureMove {
		score += pieceScore(g, m.EndIndex) - pieceScore(g, m.StartIndex)
	}
	if m.MoveType == EnPassantMove {
		score += 100
	}
	if m.MoveType == CastlingMove {
		score += 500
	}

	startDevelopment := evaluateDevelopmentForPiece(
		SingleBitboard(m.EndIndex),
		AllDevelopmentBitboards[g.Board[m.StartIndex].PieceType()][g.Player])
	endDevelopment :=
		evaluateDevelopmentForPiece(
			SingleBitboard(m.StartIndex),
			AllDevelopmentBitboards[g.Board[m.StartIndex].PieceType()][g.Player])

	score += startDevelopment - endDevelopment
	return score
}
