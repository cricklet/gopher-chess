package chess

import "strings"

type GameState struct {
	Board                        BoardArray
	player                       Player
	playerAndCastlingSideAllowed [2][2]bool
	enPassantTarget              Optional[FileRank]
	halfMoveClock                int
	fullMoveClock                int
	moveHistoryForDebugging      []Move
}

type OldGameState struct {
	player                       Player
	playerAndCastlingSideAllowed [2][2]bool
	enPassantTarget              Optional[FileRank]
	halfMoveClock                int
	fullMoveClock                int
}

type BoardUpdate struct {
	indices [4]int
	pieces  [4]Piece
	num     int

	old [4]Piece
}

func (g *GameState) HistoryString() string {
	return strings.TrimSpace(strings.Join(
		MapSlice(g.moveHistoryForDebugging, func(m Move) string { return m.String() }), " "))
}

func isPawnCapture(startPieceType PieceType, startIndex int, endIndex int) bool {
	if startPieceType != PAWN {
		return false
	}

	start := FileRankFromIndex(startIndex)
	end := FileRankFromIndex(endIndex)

	return absDiff(int(start.file), int(end.file)) == 1 && absDiff(int(start.rank), int(end.rank)) == 1
}

func (g *GameState) moveFromString(s string) Move {
	start := boardIndexFromString(s[0:2])
	end := boardIndexFromString(s[2:4])

	var moveType MoveType
	if g.Board[end] == XX {
		startPieceType := g.Board[start].pieceType()
		// either a quiet, castle, or en passant
		if startPieceType == KING && absDiff(start, end) == 2 {
			moveType = CASTLING_MOVE
		} else if isPawnCapture(startPieceType, start, end) {
			moveType = EN_PASSANT_MOVE
		} else {
			moveType = QUIET_MOVE
		}
	} else {
		moveType = CAPTURE_MOVE
	}
	return Move{moveType, start, end, Empty[int]()}
}

func isPawnSkip(startPiece Piece, move Move) bool {
	if move.moveType != QUIET_MOVE || startPiece.pieceType() != PAWN {
		return false
	}

	return absDiff(move.startIndex, move.endIndex) == OFFSET_N+OFFSET_N
}

func enPassantTarget(move Move) int {
	if move.endIndex > move.startIndex {
		return move.startIndex + OFFSET_N
	} else {
		return move.startIndex + OFFSET_S
	}
}

func (u *BoardUpdate) Add(g *GameState, index int, piece Piece) {
	u.indices[u.num] = index
	u.pieces[u.num] = piece
	u.old[u.num] = g.Board[index]
	u.num++
}

func SetupBoardUpdate(g *GameState, move Move, output *BoardUpdate) error {
	startPiece := g.Board[move.startIndex]

	switch move.moveType {
	case QUIET_MOVE:
		{
			if startPiece.pieceType() == PAWN && singleBitboard(move.endIndex)&PAWN_PROMOTION_BITBOARD != 0 {
				output.Add(g, move.startIndex, XX)
				output.Add(g, move.endIndex, PIECE_FOR_PLAYER[g.player][QUEEN])
			} else {
				output.Add(g, move.startIndex, XX)
				output.Add(g, move.endIndex, startPiece)
			}
		}
	case CAPTURE_MOVE:
		{
			output.Add(g, move.startIndex, XX)
			output.Add(g, move.endIndex, startPiece)
		}
	case EN_PASSANT_MOVE:
		{
			startPlayer := startPiece.player()
			backwardsDir := S
			if startPlayer == BLACK {
				backwardsDir = N
			}

			captureIndex := move.endIndex + OFFSETS[backwardsDir]
			output.Add(g, captureIndex, XX)
			output.Add(g, move.startIndex, XX)
			output.Add(g, move.endIndex, startPiece)
		}
	case CASTLING_MOVE:
		{
			rookStartIndex, rookEndIndex, err := rookMoveForCastle(move.startIndex, move.endIndex)
			if err != nil {
				return err
			}
			rookPiece := g.Board[rookStartIndex]

			output.Add(g, move.startIndex, XX)
			output.Add(g, rookStartIndex, XX)
			output.Add(g, move.endIndex, startPiece)
			output.Add(g, rookEndIndex, rookPiece)
		}
	}

	return nil
}

func RecordCurrentState(g *GameState, output *OldGameState) {
	output.player = g.player
	output.playerAndCastlingSideAllowed = g.playerAndCastlingSideAllowed
	output.enPassantTarget = g.enPassantTarget
	output.fullMoveClock = g.fullMoveClock
	output.halfMoveClock = g.halfMoveClock
}

func (g *GameState) updateCastlingRequirementsFor(moveBitboard Bitboard, player Player, side CastlingSide) {
	if moveBitboard&CASTLING_REQUIREMENTS[player][side].pieces != 0 {
		g.playerAndCastlingSideAllowed[player][side] = false
	}
}

func (g *GameState) performMove(move Move, update BoardUpdate) {
	startPiece := g.Board[move.startIndex]

	g.enPassantTarget = Empty[FileRank]()
	if move.moveType == QUIET_MOVE && isPawnSkip(startPiece, move) {
		g.enPassantTarget = Some(FileRankFromIndex(enPassantTarget(move)))
	}

	for i := 0; i < update.num; i++ {
		g.Board[update.indices[i]] = update.pieces[i]
	}

	g.halfMoveClock++
	if g.player == BLACK {
		g.fullMoveClock++
	}
	g.player = g.player.Other()
	g.moveHistoryForDebugging = append(g.moveHistoryForDebugging, move)

	startBitboard := singleBitboard(move.startIndex)
	endBitboard := singleBitboard(move.endIndex)
	moveBitboard := startBitboard | endBitboard
	g.updateCastlingRequirementsFor(moveBitboard, WHITE, KINGSIDE)
	g.updateCastlingRequirementsFor(moveBitboard, WHITE, QUEENSIDE)
	g.updateCastlingRequirementsFor(moveBitboard, BLACK, KINGSIDE)
	g.updateCastlingRequirementsFor(moveBitboard, BLACK, QUEENSIDE)

	if move.moveType == CASTLING_MOVE {
		g.playerAndCastlingSideAllowed[g.player][KINGSIDE] = false
		g.playerAndCastlingSideAllowed[g.player][QUEENSIDE] = false
	}
}

func (g *GameState) undoUpdate(undo OldGameState, update BoardUpdate) {
	g.player = undo.player
	g.playerAndCastlingSideAllowed = undo.playerAndCastlingSideAllowed
	g.enPassantTarget = undo.enPassantTarget
	g.fullMoveClock = undo.fullMoveClock
	g.halfMoveClock = undo.halfMoveClock

	for i := update.num - 1; i >= 0; i-- {
		index := update.indices[i]
		piece := update.old[i]

		g.Board[index] = piece
	}

	g.moveHistoryForDebugging = g.moveHistoryForDebugging[:len(g.moveHistoryForDebugging)-1]
}
func (g *GameState) enemy() Player {
	return g.player.Other()
}

func (g *GameState) whiteCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[WHITE][KINGSIDE]
}
func (g *GameState) whiteCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[BLACK][QUEENSIDE]
}
func (g *GameState) blackCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[WHITE][KINGSIDE]
}
func (g *GameState) blackCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[BLACK][QUEENSIDE]
}
