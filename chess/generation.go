package chess

import (
	"fmt"
	"sort"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

func generateWalkMovesWithMagic(
	pieces Bitboard,
	allOccupied Bitboard,
	selfOccupied Bitboard,
	magicTable MagicMoveTable,
	onlyCaptures bool,
	output []Move,
) []Move {
	startIndex, tempPieces := 0, Bitboard(pieces)
	for tempPieces != 0 {
		startIndex, tempPieces = tempPieces.NextIndexOfOne()

		blockerBoard := magicTable.BlockerMasks[startIndex] & allOccupied
		magicValues := magicTable.Magics[startIndex]
		magicIndex := MagicIndex(magicValues.Magic, blockerBoard, magicValues.BitsInMagicIndex)

		potential := magicTable.Moves[startIndex][magicIndex]
		potential = potential & ^selfOccupied

		quiet := potential & ^allOccupied
		capture := potential & ^quiet

		if !onlyCaptures {
			endIndex, tempQuiet := 0, Bitboard(quiet)
			for tempQuiet != 0 {
				endIndex, tempQuiet = tempQuiet.NextIndexOfOne()
				output = append(output, Move{MoveType: QuietMove, StartIndex: startIndex, EndIndex: endIndex, Evaluation: Empty[int]()})
			}
		}
		{
			captureIndex, tempCapture := 0, Bitboard(capture)
			for tempCapture != 0 {
				captureIndex, tempCapture = tempCapture.NextIndexOfOne()

				output = append(output, Move{MoveType: CaptureMove, StartIndex: startIndex, EndIndex: captureIndex, Evaluation: Empty[int]()})
			}
		}
	}

	return output
}

func generateJumpMovesByLookup(
	pieces Bitboard,
	allOccupied Bitboard,
	selfOccupied Bitboard,
	attackMasks [64]Bitboard,
	onlyCaptures bool,
	output []Move,
) []Move {
	startIndex, tempPieces := 0, Bitboard(pieces)
	for tempPieces != 0 {
		startIndex, tempPieces = tempPieces.NextIndexOfOne()

		attackMask := attackMasks[startIndex]
		potential := attackMask & ^selfOccupied

		quiet := potential & ^allOccupied
		capture := potential & ^quiet

		if !onlyCaptures {
			endIndex, tempQuiet := 0, Bitboard(quiet)
			for tempQuiet != 0 {
				endIndex, tempQuiet = tempQuiet.NextIndexOfOne()
				output = append(output, Move{MoveType: QuietMove, StartIndex: startIndex, EndIndex: endIndex, Evaluation: Empty[int]()})
			}
		}
		{
			captureIndex, tempCapture := 0, Bitboard(capture)
			for tempCapture != 0 {
				captureIndex, tempCapture = tempCapture.NextIndexOfOne()

				output = append(output, Move{MoveType: CaptureMove, StartIndex: startIndex, EndIndex: captureIndex, Evaluation: Empty[int]()})
			}
		}
	}

	return output
}

var GetMovesBuffer, ReleaseMovesBuffer, StatsMoveBuffer = CreatePool(func() []Move { return make([]Move, 0, 256) }, func(t *[]Move) { *t = (*t)[:0] })

func GeneratePseudoMoves(b *Bitboards, g *GameState, moves *[]Move) {
	GeneratePseudoMovesInternal(b, g, moves, false /* onlyCaptures */)
}
func GenerateSortedPseudoMoves(b *Bitboards, g *GameState, moves *[]Move) {
	GeneratePseudoMoves(b, g, moves)

	for i := range *moves {
		(*moves)[i].Evaluation = Some(EvaluateMove(&(*moves)[i], g))
	}

	sort.SliceStable(*moves, func(i, j int) bool {
		return (*moves)[i].Evaluation.Value() > (*moves)[j].Evaluation.Value()
	})
}
func GenerateSortedPseudoCaptures(b *Bitboards, g *GameState, moves *[]Move) {
	GeneratePseudoCaptures(b, g, moves)

	for i := range *moves {
		(*moves)[i].Evaluation = Some(EvaluateMove(&(*moves)[i], g))
	}

	sort.SliceStable(*moves, func(i, j int) bool {
		return (*moves)[i].Evaluation.Value() > (*moves)[j].Evaluation.Value()
	})
}
func GeneratePseudoCaptures(b *Bitboards, g *GameState, moves *[]Move) {
	GeneratePseudoMovesInternal(b, g, moves, true /* onlyCaptures */)
}

func GeneratePseudoMovesInternal(b *Bitboards, g *GameState, moves *[]Move, onlyCaptures bool) {
	player := g.Player
	playerBoards := b.Players[player]
	enemyBoards := &b.Players[player.Other()]

	{
		pushOffset := PawnPushOffsets[player]

		// generate one step
		if !onlyCaptures {
			potential := RotateTowardsIndex64(playerBoards.Pieces[Pawn]&PremoveMaskFromOffset(pushOffset), pushOffset)
			potential = potential & ^b.Occupied

			index, tempPotential := 0, Bitboard(potential)
			for tempPotential != 0 {
				index, tempPotential = tempPotential.NextIndexOfOne()

				*moves = append(*moves, Move{MoveType: QuietMove, StartIndex: index - pushOffset, EndIndex: index, Evaluation: Empty[int]()})
			}
		}

		// generate skip step
		if !onlyCaptures {
			potential := playerBoards.Pieces[Pawn]
			potential = potential & MaskStartingPawnsForPlayer(player)
			potential = RotateTowardsIndex64(potential, pushOffset)
			potential = potential & ^b.Occupied
			potential = RotateTowardsIndex64(potential, pushOffset)
			potential = potential & ^b.Occupied

			index, tempPotential := 0, Bitboard(potential)
			for tempPotential != 0 {
				index, tempPotential = tempPotential.NextIndexOfOne()

				*moves = append(*moves, Move{MoveType: QuietMove, StartIndex: index - 2*pushOffset, EndIndex: index, Evaluation: Empty[int]()})
			}
		}

		// generate captures
		{
			for _, captureOffset := range PawnCaptureOffsets[player] {
				potential := playerBoards.Pieces[Pawn] & PremoveMaskFromOffset(captureOffset)
				potential = RotateTowardsIndex64(potential, captureOffset)
				potential = potential & enemyBoards.Occupied

				index, tempPotential := 0, Bitboard(potential)
				for tempPotential != 0 {
					index, tempPotential = tempPotential.NextIndexOfOne()

					*moves = append(*moves, Move{MoveType: CaptureMove, StartIndex: index - captureOffset, EndIndex: index, Evaluation: Empty[int]()})
				}
			}
		}

		// generate en-passant
		{
			if g.EnPassantTarget.HasValue() {
				enPassantBoard := SingleBitboard(IndexFromFileRank(g.EnPassantTarget.Value()))
				for _, captureOffset := range []int{pushOffset + OffsetE, pushOffset + OffsetW} {
					potential := playerBoards.Pieces[Pawn] & PremoveMaskFromOffset(captureOffset)
					potential = RotateTowardsIndex64(potential, captureOffset)
					potential = potential & enPassantBoard

					index, tempPotential := 0, Bitboard(potential)
					for tempPotential != 0 {
						index, tempPotential = tempPotential.NextIndexOfOne()

						*moves = append(*moves, Move{MoveType: EnPassantMove, StartIndex: index - captureOffset, EndIndex: index, Evaluation: Empty[int]()})
					}
				}
			}
		}
	}

	{
		// generate rook / bishop / queen moves
		// *moves = generateWalkMoves(playerBoards.pieces[ROOK], b.occupied, enemyBoards.occupied, N, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[ROOK], b.occupied, enemyBoards.occupied, S, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[ROOK], b.occupied, enemyBoards.occupied, E, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[ROOK], b.occupied, enemyBoards.occupied, W, *moves)

		// *moves = generateWalkMoves(playerBoards.pieces[BISHOP], b.occupied, enemyBoards.occupied, NE, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[BISHOP], b.occupied, enemyBoards.occupied, SE, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[BISHOP], b.occupied, enemyBoards.occupied, NW, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[BISHOP], b.occupied, enemyBoards.occupied, SW, *moves)

		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, N, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, S, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, E, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, W, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, NE, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, SE, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, NW, *moves)
		// *moves = generateWalkMoves(playerBoards.pieces[QUEEN], b.occupied, enemyBoards.occupied, SW, *moves)

		*moves = generateWalkMovesWithMagic(playerBoards.Pieces[Rook], b.Occupied, playerBoards.Occupied, RookMagicTable, onlyCaptures, *moves)
		*moves = generateWalkMovesWithMagic(playerBoards.Pieces[Bishop], b.Occupied, playerBoards.Occupied, BishopMagicTable, onlyCaptures, *moves)
		*moves = generateWalkMovesWithMagic(playerBoards.Pieces[Queen], b.Occupied, playerBoards.Occupied, RookMagicTable, onlyCaptures, *moves)
		*moves = generateWalkMovesWithMagic(playerBoards.Pieces[Queen], b.Occupied, playerBoards.Occupied, BishopMagicTable, onlyCaptures, *moves)
	}

	{
		// generate knight moves
		*moves = generateJumpMovesByLookup(playerBoards.Pieces[Knight], b.Occupied, playerBoards.Occupied, KnightAttackMasks, onlyCaptures, *moves)

		// generate king moves
		*moves = generateJumpMovesByLookup(playerBoards.Pieces[King], b.Occupied, playerBoards.Occupied, KingAttackMasks, onlyCaptures, *moves)
	}

	if !onlyCaptures {
		// generate king castle
		for _, castlingSide := range AllCastlingSides {
			canCastle := true
			if g.PlayerAndCastlingSideAllowed[player][castlingSide] {
				requirements := AllCastlingRequirements[player][castlingSide]
				if b.Occupied&requirements.Empty != 0 {
					canCastle = false
				}
				for _, index := range requirements.Safe {
					if playerIndexIsAttacked(player, index, b.Occupied, enemyBoards) {
						canCastle = false
						break
					}
				}

				if canCastle {
					*moves = append(*moves, requirements.Move)
				}
			}
		}
	}
}

func playerIndexIsAttacked(player Player, startIndex int, occupied Bitboard, enemyBitboards *PlayerBitboards) bool {
	startBoard := SingleBitboard(startIndex)

	// Bishop attacks
	{
		blockerBoard := BishopMagicTable.BlockerMasks[startIndex] & occupied
		magicValues := BishopMagicTable.Magics[startIndex]
		magicIndex := MagicIndex(magicValues.Magic, blockerBoard, magicValues.BitsInMagicIndex)

		potential := BishopMagicTable.Moves[startIndex][magicIndex]
		potential = potential & (enemyBitboards.Pieces[Bishop] | enemyBitboards.Pieces[Queen])

		if potential != 0 {
			return true
		}
	}
	// Rook attacks
	{
		blockerBoard := RookMagicTable.BlockerMasks[startIndex] & occupied
		magicValues := RookMagicTable.Magics[startIndex]
		magicIndex := MagicIndex(magicValues.Magic, blockerBoard, magicValues.BitsInMagicIndex)

		potential := RookMagicTable.Moves[startIndex][magicIndex]
		potential = potential & (enemyBitboards.Pieces[Rook] | enemyBitboards.Pieces[Queen])

		if potential != 0 {
			return true
		}
	}

	attackers := Bitboard(0)

	// Pawn attacks
	{
		enemyPlayer := player.Other()
		enemyPawns := enemyBitboards.Pieces[Pawn]
		captureOffset0 := PawnCaptureOffsets[enemyPlayer][0]
		captureOffset1 := PawnCaptureOffsets[enemyPlayer][1]
		captureMask0 := enemyPawns & PremoveMaskFromOffset(captureOffset0)
		captureMask1 := enemyPawns & PremoveMaskFromOffset(captureOffset1)

		potential := RotateTowardsIndex64(captureMask0, captureOffset0)
		potential |= RotateTowardsIndex64(captureMask1, captureOffset1)
		potential &= startBoard
		attackers |= potential
	}
	// Knight, king attacks
	{
		{
			knightMask := KnightAttackMasks[startIndex]
			potential := enemyBitboards.Pieces[Knight] & knightMask
			attackers |= potential
		}
		{
			kingMask := KingAttackMasks[startIndex]
			potential := enemyBitboards.Pieces[King] & kingMask
			attackers |= potential
		}
	}

	return attackers != 0
}

func KingIsInCheck(b *Bitboards, player Player, enemy Player) bool {
	kingBoard := b.Players[player].Pieces[King]
	kingIndex := kingBoard.FirstIndexOfOne()
	return playerIndexIsAttacked(player, kingIndex, b.Occupied, &b.Players[enemy])
}

func DangerBoard(b *Bitboards, player Player) Bitboard {
	enemyPlayer := player.Other()
	enemyBoards := &b.Players[enemyPlayer]
	result := Bitboard(0)
	for i := 0; i < 64; i++ {
		if playerIndexIsAttacked(player, i, b.Occupied, enemyBoards) {
			result |= SingleBitboard(i)
		}
	}
	return result
}

type BoardCorrupted struct {
	Message error
}

func (e *BoardCorrupted) Error() string {
	return fmt.Sprintf("corruption during update: %q", e.Message)
}

func GenerateLegalMoves(b *Bitboards, g *GameState, legalMovesOutput *[]Move) error {
	player := g.Player
	enemy := g.Enemy()
	potentialMoves := GetMovesBuffer()
	defer ReleaseMovesBuffer(potentialMoves)
	GeneratePseudoMoves(b, g, potentialMoves)

	for _, move := range *potentialMoves {
		update := BoardUpdate{}
		err := SetupBoardUpdate(g, move, &update)
		if err != nil {
			return fmt.Errorf("GenerateLegalMoves: %w", err)
		}

		err = g.PerformMove(move, &update, b)
		if err != nil {
			return &BoardCorrupted{err}
		}
		if !KingIsInCheck(b, player, enemy) {
			*legalMovesOutput = append(*legalMovesOutput, move)
		}

		err = g.UndoUpdate(&update, b)
		if err != nil {
			return fmt.Errorf("GenerateLegalMoves: %w", err)
		}
	}

	return nil
}
