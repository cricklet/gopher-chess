package chess

import (
	"fmt"
	"sort"
)

type MoveType int

const (
	QUIET_MOVE MoveType = iota
	CAPTURE_MOVE
	CASTLING_MOVE
	EN_PASSANT_MOVE
)

type Move struct {
	moveType   MoveType
	startIndex int
	endIndex   int
	evaluation Optional[int]
}

type ReusableIndicesBuffers struct {
	startBuffer *IndicesBuffer
	endBuffer   *IndicesBuffer
}

func SetupBuffers() ReusableIndicesBuffers {
	return ReusableIndicesBuffers{GetIndicesBuffer(), GetIndicesBuffer()}
}

func (r ReusableIndicesBuffers) Release() {
	ReleaseIndicesBuffer(r.startBuffer)
	ReleaseIndicesBuffer(r.endBuffer)
}

func moveFromString(s string, m MoveType) Move {
	first := s[0:2]
	second := s[2:4]
	return Move{m, boardIndexFromString(first), boardIndexFromString(second), Empty[int]()}
}

func (m Move) String() string {
	return stringFromBoardIndex(m.startIndex) + stringFromBoardIndex(m.endIndex)
}

func (m Move) DebugString() string {
	return fmt.Sprintf("%v%v, %v", stringFromBoardIndex(m.startIndex), stringFromBoardIndex(m.endIndex), m.moveType)
}

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
		startIndex, tempPieces = tempPieces.nextIndexOfOne()

		blockerBoard := magicTable.blockerMasks[startIndex] & allOccupied
		magicValues := magicTable.magics[startIndex]
		magicIndex := magicIndex(magicValues.Magic, blockerBoard, magicValues.BitsInMagicIndex)

		potential := magicTable.moves[startIndex][magicIndex]
		potential = potential & ^selfOccupied

		quiet := potential & ^allOccupied
		capture := potential & ^quiet

		if !onlyCaptures {
			endIndex, tempQuiet := 0, Bitboard(quiet)
			for tempQuiet != 0 {
				endIndex, tempQuiet = tempQuiet.nextIndexOfOne()
				output = append(output, Move{QUIET_MOVE, startIndex, endIndex, Empty[int]()})
			}
		}
		{
			captureIndex, tempCapture := 0, Bitboard(capture)
			for tempCapture != 0 {
				captureIndex, tempCapture = tempCapture.nextIndexOfOne()

				output = append(output, Move{CAPTURE_MOVE, startIndex, captureIndex, Empty[int]()})
			}
		}
	}

	return output
}

func generateWalkBitboard(
	pieceBoard Bitboard,
	blockerBoard Bitboard,
	dir Dir,
	output Bitboard,
) Bitboard {
	mask := PRE_MOVE_MASKS[dir]
	offset := OFFSETS[dir]

	totalOffset := 0
	potential := pieceBoard

	for potential != 0 {
		potential = rotateTowardsIndex64(potential&mask, offset)
		totalOffset += offset

		quiet := potential & ^blockerBoard
		capture := potential & blockerBoard

		output |= quiet | capture

		potential = quiet
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
		startIndex, tempPieces = tempPieces.nextIndexOfOne()

		attackMask := attackMasks[startIndex]
		potential := attackMask & ^selfOccupied

		quiet := potential & ^allOccupied
		capture := potential & ^quiet

		if !onlyCaptures {
			endIndex, tempQuiet := 0, Bitboard(quiet)
			for tempQuiet != 0 {
				endIndex, tempQuiet = tempQuiet.nextIndexOfOne()
				output = append(output, Move{QUIET_MOVE, startIndex, endIndex, Empty[int]()})
			}
		}
		{
			captureIndex, tempCapture := 0, Bitboard(capture)
			for tempCapture != 0 {
				captureIndex, tempCapture = tempCapture.nextIndexOfOne()

				output = append(output, Move{CAPTURE_MOVE, startIndex, captureIndex, Empty[int]()})
			}
		}
	}

	return output
}

var GetMovesBuffer, ReleaseMovesBuffer, StatsMoveBuffer = createPool(func() []Move { return make([]Move, 0, 256) }, func(t *[]Move) { *t = (*t)[:0] })

func (b *Bitboards) GeneratePseudoMoves(g *GameState, moves *[]Move) {
	b.generatePseudoMovesInternal(g, moves, false /* onlyCaptures */)
}
func (b *Bitboards) GenerateSortedPseudoMoves(g *GameState, moves *[]Move) {
	b.GeneratePseudoMoves(g, moves)

	for i := range *moves {
		(*moves)[i].evaluation = Some((*moves)[i].Evaluate(g))
	}

	sort.SliceStable(*moves, func(i, j int) bool {
		return (*moves)[i].evaluation.Value() > (*moves)[j].evaluation.Value()
	})
}
func (b *Bitboards) GenerateSortedPseudoCaptures(g *GameState, moves *[]Move) {
	b.GeneratePseudoCaptures(g, moves)

	for i := range *moves {
		(*moves)[i].evaluation = Some((*moves)[i].Evaluate(g))
	}

	sort.SliceStable(*moves, func(i, j int) bool {
		return (*moves)[i].evaluation.Value() > (*moves)[j].evaluation.Value()
	})
}
func (b *Bitboards) GeneratePseudoCaptures(g *GameState, moves *[]Move) {
	b.generatePseudoMovesInternal(g, moves, true /* onlyCaptures */)
}

func (b *Bitboards) generatePseudoMovesInternal(g *GameState, moves *[]Move, onlyCaptures bool) {
	player := g.player
	playerBoards := b.players[player]
	enemyBoards := &b.players[player.Other()]

	{
		pushOffset := PAWN_PUSH_OFFSETS[player]

		// generate one step
		if !onlyCaptures {
			potential := rotateTowardsIndex64(playerBoards.pieces[PAWN]&PremoveMaskFromOffset(pushOffset), pushOffset)
			potential = potential & ^b.occupied

			index, tempPotential := 0, Bitboard(potential)
			for tempPotential != 0 {
				index, tempPotential = tempPotential.nextIndexOfOne()

				*moves = append(*moves, Move{QUIET_MOVE, index - pushOffset, index, Empty[int]()})
			}
		}

		// generate skip step
		if !onlyCaptures {
			potential := playerBoards.pieces[PAWN]
			potential = potential & maskStartingPawnsForPlayer(player)
			potential = rotateTowardsIndex64(potential, pushOffset)
			potential = potential & ^b.occupied
			potential = rotateTowardsIndex64(potential, pushOffset)
			potential = potential & ^b.occupied

			index, tempPotential := 0, Bitboard(potential)
			for tempPotential != 0 {
				index, tempPotential = tempPotential.nextIndexOfOne()

				*moves = append(*moves, Move{QUIET_MOVE, index - 2*pushOffset, index, Empty[int]()})
			}
		}

		// generate captures
		{
			for _, captureOffset := range PAWN_CAPTURE_OFFSETS[player] {
				potential := playerBoards.pieces[PAWN] & PremoveMaskFromOffset(captureOffset)
				potential = rotateTowardsIndex64(potential, captureOffset)
				potential = potential & enemyBoards.occupied

				index, tempPotential := 0, Bitboard(potential)
				for tempPotential != 0 {
					index, tempPotential = tempPotential.nextIndexOfOne()

					*moves = append(*moves, Move{CAPTURE_MOVE, index - captureOffset, index, Empty[int]()})
				}
			}
		}

		// generate en-passant
		{
			if g.enPassantTarget.HasValue() {
				enPassantBoard := singleBitboard(IndexFromFileRank(g.enPassantTarget.Value()))
				for _, captureOffset := range []int{pushOffset + OFFSET_E, pushOffset + OFFSET_W} {
					potential := playerBoards.pieces[PAWN] & PremoveMaskFromOffset(captureOffset)
					potential = rotateTowardsIndex64(potential, captureOffset)
					potential = potential & enPassantBoard

					index, tempPotential := 0, Bitboard(potential)
					for tempPotential != 0 {
						index, tempPotential = tempPotential.nextIndexOfOne()

						*moves = append(*moves, Move{EN_PASSANT_MOVE, index - captureOffset, index, Empty[int]()})
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

		*moves = generateWalkMovesWithMagic(playerBoards.pieces[ROOK], b.occupied, playerBoards.occupied, ROOK_MAGIC_TABLE, onlyCaptures, *moves)
		*moves = generateWalkMovesWithMagic(playerBoards.pieces[BISHOP], b.occupied, playerBoards.occupied, BISHOP_MAGIC_TABLE, onlyCaptures, *moves)
		*moves = generateWalkMovesWithMagic(playerBoards.pieces[QUEEN], b.occupied, playerBoards.occupied, ROOK_MAGIC_TABLE, onlyCaptures, *moves)
		*moves = generateWalkMovesWithMagic(playerBoards.pieces[QUEEN], b.occupied, playerBoards.occupied, BISHOP_MAGIC_TABLE, onlyCaptures, *moves)
	}

	{
		// generate knight moves
		*moves = generateJumpMovesByLookup(playerBoards.pieces[KNIGHT], b.occupied, playerBoards.occupied, KNIGHT_ATTACK_MASKS, onlyCaptures, *moves)

		// generate king moves
		*moves = generateJumpMovesByLookup(playerBoards.pieces[KING], b.occupied, playerBoards.occupied, KING_ATTACK_MASKS, onlyCaptures, *moves)
	}

	if !onlyCaptures {
		// generate king castle
		for _, castlingSide := range CASTLING_SIDES {
			canCastle := true
			if g.playerAndCastlingSideAllowed[player][castlingSide] {
				requirements := CASTLING_REQUIREMENTS[player][castlingSide]
				if b.occupied&requirements.empty != 0 {
					canCastle = false
				}
				for _, index := range requirements.safe {
					if playerIndexIsAttacked(player, index, b.occupied, enemyBoards) {
						canCastle = false
						break
					}
				}

				if canCastle {
					*moves = append(*moves, requirements.move)
				}
			}
		}
	}
}

func playerIndexIsAttacked(player Player, startIndex int, occupied Bitboard, enemyBitboards *PlayerBitboards) bool {
	startBoard := singleBitboard(startIndex)

	// Bishop attacks
	{
		blockerBoard := BISHOP_MAGIC_TABLE.blockerMasks[startIndex] & occupied
		magicValues := BISHOP_MAGIC_TABLE.magics[startIndex]
		magicIndex := magicIndex(magicValues.Magic, blockerBoard, magicValues.BitsInMagicIndex)

		potential := BISHOP_MAGIC_TABLE.moves[startIndex][magicIndex]
		potential = potential & (enemyBitboards.pieces[BISHOP] | enemyBitboards.pieces[QUEEN])

		if potential != 0 {
			return true
		}
	}
	// Rook attacks
	{
		blockerBoard := ROOK_MAGIC_TABLE.blockerMasks[startIndex] & occupied
		magicValues := ROOK_MAGIC_TABLE.magics[startIndex]
		magicIndex := magicIndex(magicValues.Magic, blockerBoard, magicValues.BitsInMagicIndex)

		potential := ROOK_MAGIC_TABLE.moves[startIndex][magicIndex]
		potential = potential & (enemyBitboards.pieces[ROOK] | enemyBitboards.pieces[QUEEN])

		if potential != 0 {
			return true
		}
	}

	attackers := Bitboard(0)

	// Pawn attacks
	{
		enemyPlayer := player.Other()
		enemyPawns := enemyBitboards.pieces[PAWN]
		captureOffset0 := PAWN_CAPTURE_OFFSETS[enemyPlayer][0]
		captureOffset1 := PAWN_CAPTURE_OFFSETS[enemyPlayer][1]
		captureMask0 := enemyPawns & PremoveMaskFromOffset(captureOffset0)
		captureMask1 := enemyPawns & PremoveMaskFromOffset(captureOffset1)

		potential := rotateTowardsIndex64(captureMask0, captureOffset0)
		potential |= rotateTowardsIndex64(captureMask1, captureOffset1)
		potential &= startBoard
		attackers |= potential
	}
	// Knight, king attacks
	{
		{
			knightMask := KNIGHT_ATTACK_MASKS[startIndex]
			potential := enemyBitboards.pieces[KNIGHT] & knightMask
			attackers |= potential
		}
		{
			kingMask := KING_ATTACK_MASKS[startIndex]
			potential := enemyBitboards.pieces[KING] & kingMask
			attackers |= potential
		}
	}

	return attackers != 0
}

func (b *Bitboards) kingIsInCheck(player Player, enemy Player) bool {
	kingBoard := b.players[player].pieces[KING]
	kingIndex := kingBoard.firstIndexOfOne()
	return playerIndexIsAttacked(player, kingIndex, b.occupied, &b.players[enemy])
}

func (b *Bitboards) dangerBoard(player Player) Bitboard {
	enemyPlayer := player.Other()
	enemyBoards := &b.players[enemyPlayer]
	result := Bitboard(0)
	for i := 0; i < 64; i++ {
		if playerIndexIsAttacked(player, i, b.occupied, enemyBoards) {
			result |= singleBitboard(i)
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

func (b *Bitboards) generateLegalMoves(g *GameState, legalMovesOutput *[]Move) error {
	player := g.player
	enemy := g.enemy()
	potentialMoves := GetMovesBuffer()
	defer ReleaseMovesBuffer(potentialMoves)
	b.GeneratePseudoMoves(g, potentialMoves)

	for _, move := range *potentialMoves {
		update := BoardUpdate{}
		err := SetupBoardUpdate(g, move, &update)
		if err != nil {
			return fmt.Errorf("generateLegalMoves: %w", err)
		}

		err = b.performMove(g, move)
		if err != nil {
			return &BoardCorrupted{err}
		}
		if !b.kingIsInCheck(player, enemy) {
			*legalMovesOutput = append(*legalMovesOutput, move)
		}

		err = b.undoUpdate(update)
		if err != nil {
			return fmt.Errorf("generateLegalMoves: %w", err)
		}
	}

	return nil
}
