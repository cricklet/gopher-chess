package chess

import (
	"fmt"
	"strings"

	. "github.com/cricklet/chessgo/internal/helpers"
)

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
}

type BoardUpdate struct {
	Indices [4]int
	Pieces  [4]Piece
	Num     int

	PrevPieces                       [4]Piece
	PrevPlayer                       Player
	PrevPlayerAndCastlingSideAllowed [2][2]bool
	PrevEnPassantTarget              Optional[FileRank]
	PrevHalfMoveClock                int
	PrevFullMoveClock                int
}

func (g *GameState) HistoryString() string {
	return strings.TrimSpace(strings.Join(
		MapSlice(g.moveHistoryForDebugging, func(m Move) string { return m.String() }), " "))
}

func isPawnCapture(startPieceType PieceType, startIndex int, endIndex int) bool {
	if startPieceType != Pawn {
		return false
	}

	start := FileRankFromIndex(startIndex)
	end := FileRankFromIndex(endIndex)

	return AbsDiff(int(start.File), int(end.File)) == 1 && AbsDiff(int(start.Rank), int(end.Rank)) == 1
}

func (g *GameState) MoveFromString(s string) Move {
	start := BoardIndexFromString(s[0:2])
	end := BoardIndexFromString(s[2:4])

	var moveType MoveType
	if g.Board[end] == XX {
		startPieceType := g.Board[start].PieceType()
		// either a quiet, castle, or en passant
		if startPieceType == King && AbsDiff(start, end) == 2 {
			moveType = CastlingMove
		} else if isPawnCapture(startPieceType, start, end) {
			moveType = EnPassantMove
		} else {
			moveType = QuietMove
		}
	} else {
		moveType = CaptureMove
	}
	return Move{
		MoveType: moveType, StartIndex: start, EndIndex: end, Evaluation: Empty[int]()}
}

func isPawnSkip(startPiece Piece, move Move) bool {
	if move.MoveType != QuietMove || startPiece.PieceType() != Pawn {
		return false
	}

	return AbsDiff(move.StartIndex, move.EndIndex) == OffsetN+OffsetN
}

func enPassantTarget(move Move) int {
	if move.EndIndex > move.StartIndex {
		return move.StartIndex + OffsetN
	} else {
		return move.StartIndex + OffsetS
	}
}

func (u *BoardUpdate) Add(g *GameState, index int, piece Piece) {
	u.Indices[u.Num] = index
	u.Pieces[u.Num] = piece
	u.PrevPieces[u.Num] = g.Board[index]
	u.Num++
}

func SetupBoardUpdate(g *GameState, move Move, output *BoardUpdate) error {
	startPiece := g.Board[move.StartIndex]

	switch move.MoveType {
	case QuietMove:
		{
			if startPiece.PieceType() == Pawn && SingleBitboard(move.EndIndex)&PawnPromotionBitboard != 0 {
				output.Add(g, move.StartIndex, XX)
				output.Add(g, move.EndIndex, PieceForPlayer[g.player][Queen])
			} else {
				output.Add(g, move.StartIndex, XX)
				output.Add(g, move.EndIndex, startPiece)
			}
		}
	case CaptureMove:
		{
			output.Add(g, move.StartIndex, XX)
			output.Add(g, move.EndIndex, startPiece)
		}
	case EnPassantMove:
		{
			startPlayer := startPiece.Player()
			backwardsDir := S
			if startPlayer == Black {
				backwardsDir = N
			}

			captureIndex := move.EndIndex + Offsets[backwardsDir]
			output.Add(g, captureIndex, XX)
			output.Add(g, move.StartIndex, XX)
			output.Add(g, move.EndIndex, startPiece)
		}
	case CastlingMove:
		{
			rookStartIndex, rookEndIndex, err := RookMoveForCastle(move.StartIndex, move.EndIndex)
			if err != nil {
				return err
			}
			rookPiece := g.Board[rookStartIndex]

			output.Add(g, move.StartIndex, XX)
			output.Add(g, rookStartIndex, XX)
			output.Add(g, move.EndIndex, startPiece)
			output.Add(g, rookEndIndex, rookPiece)
		}
	}

	output.PrevPlayer = g.player
	output.PrevPlayerAndCastlingSideAllowed = g.playerAndCastlingSideAllowed
	output.PrevEnPassantTarget = g.enPassantTarget
	output.PrevFullMoveClock = g.fullMoveClock
	output.PrevHalfMoveClock = g.halfMoveClock

	return nil
}

func (g *GameState) updateCastlingRequirementsFor(moveBitboard Bitboard, player Player, side CastlingSide) {
	if moveBitboard&AllCastlingRequirements[player][side].pieces != 0 {
		g.playerAndCastlingSideAllowed[player][side] = false
	}
}

func (g *GameState) performMove(move Move, update BoardUpdate) {
	startPiece := g.Board[move.StartIndex]

	g.enPassantTarget = Empty[FileRank]()
	if move.MoveType == QuietMove && isPawnSkip(startPiece, move) {
		g.enPassantTarget = Some(FileRankFromIndex(enPassantTarget(move)))
	}

	for i := 0; i < update.Num; i++ {
		g.Board[update.Indices[i]] = update.Pieces[i]
	}

	g.halfMoveClock++
	if g.player == Black {
		g.fullMoveClock++
	}
	g.player = g.player.Other()
	g.moveHistoryForDebugging = append(g.moveHistoryForDebugging, move)

	startBitboard := SingleBitboard(move.StartIndex)
	endBitboard := SingleBitboard(move.EndIndex)
	moveBitboard := startBitboard | endBitboard
	g.updateCastlingRequirementsFor(moveBitboard, White, Kingside)
	g.updateCastlingRequirementsFor(moveBitboard, White, Queenside)
	g.updateCastlingRequirementsFor(moveBitboard, Black, Kingside)
	g.updateCastlingRequirementsFor(moveBitboard, Black, Queenside)

	if move.MoveType == CastlingMove {
		g.playerAndCastlingSideAllowed[g.player][Kingside] = false
		g.playerAndCastlingSideAllowed[g.player][Queenside] = false
	}
}

func (g *GameState) undoUpdate(update BoardUpdate) {
	g.player = update.PrevPlayer
	g.playerAndCastlingSideAllowed = update.PrevPlayerAndCastlingSideAllowed
	g.enPassantTarget = update.PrevEnPassantTarget
	g.fullMoveClock = update.PrevFullMoveClock
	g.halfMoveClock = update.PrevHalfMoveClock

	for i := update.Num - 1; i >= 0; i-- {
		index := update.Indices[i]
		piece := update.PrevPieces[i]

		g.Board[index] = piece
	}

	g.moveHistoryForDebugging = g.moveHistoryForDebugging[:len(g.moveHistoryForDebugging)-1]
}
func (g *GameState) enemy() Player {
	return g.player.Other()
}

func (g *GameState) whiteCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[White][Kingside]
}
func (g *GameState) whiteCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[Black][Queenside]
}
func (g *GameState) blackCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[White][Kingside]
}
func (g *GameState) blackCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[Black][Queenside]
}

func (g *GameState) CreateBitboards() Bitboards {
	result := Bitboards{}
	for i, piece := range g.Board {
		if piece == XX {
			continue
		}
		pieceType := piece.PieceType()
		player := piece.Player()
		result.Players[player].Pieces[pieceType] |= SingleBitboard(i)

		if piece.IsWhite() {
			result.Occupied |= SingleBitboard(i)
			result.Players[White].Occupied |= SingleBitboard(i)
		}
		if piece.IsBlack() {
			result.Occupied |= SingleBitboard(i)
			result.Players[Black].Occupied |= SingleBitboard(i)
		}
	}
	return result
}

func (g *GameState) ApplyMoveToBitboards(b *Bitboards, move Move) error {
	startIndex := move.StartIndex
	endIndex := move.EndIndex

	startPiece := g.Board[startIndex]

	switch move.MoveType {
	case QuietMove:
		{
			err := b.ClearSquare(startIndex, startPiece)
			if err != nil {
				return fmt.Errorf("%v: %w", move.DebugString(), err)
			}
			b.SetSquare(endIndex, startPiece)
		}
	case CaptureMove:
		{
			// Remove captured piece
			endPiece := g.Board[endIndex]
			err := b.ClearSquare(endIndex, endPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(endIndex), endPiece, err)
			}

			// Move the capturing piece
			err = b.ClearSquare(startIndex, startPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(startIndex), startPiece, err)
			}
			b.SetSquare(endIndex, startPiece)
		}
	case EnPassantMove:
		{
			capturedPlayer := startPiece.Player().Other()
			capturedBackwards := N
			if capturedPlayer == Black {
				capturedBackwards = S
			}

			captureIndex := endIndex + Offsets[capturedBackwards]
			capturePiece := g.Board[captureIndex]

			err := b.ClearSquare(captureIndex, capturePiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(captureIndex), capturePiece, err)
			}
			err = b.ClearSquare(startIndex, startPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(startIndex), startPiece, err)
			}
			b.SetSquare(endIndex, startPiece)
		}
	case CastlingMove:
		{
			rookStartIndex, rookEndIndex, err := RookMoveForCastle(startIndex, endIndex)
			if err != nil {
				return err
			}
			rookPiece := g.Board[rookStartIndex]

			err = b.ClearSquare(startIndex, startPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(startIndex), startPiece, err)
			}
			b.SetSquare(endIndex, startPiece)

			err = b.ClearSquare(rookStartIndex, rookPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(rookStartIndex), rookPiece, err)
			}
			err = b.ClearSquare(rookEndIndex, rookPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w", move.DebugString(), StringFromBoardIndex(rookEndIndex), rookPiece, err)
			}
		}
	}

	return nil
}
