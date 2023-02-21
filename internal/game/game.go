package game

import (
	"fmt"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type GameState struct {
	Board                         BoardArray
	Player                        Player
	PlayerAndCastlingSideAllowed  [2][2]bool
	EnPassantTarget               Optional[FileRank]
	HalfMoveClock                 int
	FullMoveClock                 int
	FenAndMoveHistoryForDebugging [][2]string
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

func EnPassantTarget(move Move) int {
	if move.EndIndex > move.StartIndex {
		return move.StartIndex + OffsetN
	} else {
		return move.StartIndex + OffsetS
	}
}

func setupBoardUpdate(g *GameState, move Move, output *BoardUpdate) error {
	startPiece := g.Board[move.StartIndex]

	switch move.MoveType {
	case QuietMove:
		{
			if startPiece.PieceType() == Pawn && SingleBitboard(move.EndIndex)&PawnPromotionBitboard != 0 {
				output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
				output.Add(g.Board[move.EndIndex], move.EndIndex, PieceForPlayer[g.Player][Queen])
			} else {
				output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
				output.Add(g.Board[move.EndIndex], move.EndIndex, startPiece)
			}
		}
	case CaptureMove:
		{
			output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
			output.Add(g.Board[move.EndIndex], move.EndIndex, startPiece)
		}
	case EnPassantMove:
		{
			startPlayer := startPiece.Player()
			backwardsDir := S
			if startPlayer == Black {
				backwardsDir = N
			}

			captureIndex := move.EndIndex + Offsets[backwardsDir]
			output.Add(g.Board[captureIndex], captureIndex, XX)
			output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
			output.Add(g.Board[move.EndIndex], move.EndIndex, startPiece)
		}
	case CastlingMove:
		{
			rookStartIndex, rookEndIndex, err := RookMoveForCastle(move.StartIndex, move.EndIndex)
			if err != nil {
				return err
			}
			rookPiece := g.Board[rookStartIndex]

			output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
			output.Add(g.Board[rookStartIndex], rookStartIndex, XX)
			output.Add(g.Board[move.EndIndex], move.EndIndex, startPiece)
			output.Add(g.Board[rookEndIndex], rookEndIndex, rookPiece)
		}
	}

	output.PrevPlayer = g.Player
	output.PrevPlayerAndCastlingSideAllowed = g.PlayerAndCastlingSideAllowed
	output.PrevEnPassantTarget = g.EnPassantTarget
	output.PrevFullMoveClock = g.FullMoveClock
	output.PrevHalfMoveClock = g.HalfMoveClock

	return nil
}

func (g *GameState) updateCastlingRequirementsFor(moveBitboard Bitboard, player Player, side CastlingSide) {
	if moveBitboard&AllCastlingRequirements[player][side].Pieces != 0 {
		g.PlayerAndCastlingSideAllowed[player][side] = false
	}
}

func (g *GameState) PerformMove(move Move, update *BoardUpdate, b *Bitboards) error {
	setupBoardUpdate(g, move, update)

	g.FenAndMoveHistoryForDebugging = append(g.FenAndMoveHistoryForDebugging, [2]string{FenStringForGame(g), move.DebugString()})

	err := g.applyMoveToBitboards(b, move)
	if err != nil {
		return err
	}

	startPiece := g.Board[move.StartIndex]

	g.EnPassantTarget = Empty[FileRank]()
	if move.MoveType == QuietMove && isPawnSkip(startPiece, move) {
		g.EnPassantTarget = Some(FileRankFromIndex(EnPassantTarget(move)))
	}

	for i := 0; i < update.Num; i++ {
		g.Board[update.Indices[i]] = update.Pieces[i]
	}

	g.HalfMoveClock++
	if g.Player == Black {
		g.FullMoveClock++
	}
	g.Player = g.Player.Other()

	startBitboard := SingleBitboard(move.StartIndex)
	endBitboard := SingleBitboard(move.EndIndex)
	moveBitboard := startBitboard | endBitboard
	g.updateCastlingRequirementsFor(moveBitboard, White, Kingside)
	g.updateCastlingRequirementsFor(moveBitboard, White, Queenside)
	g.updateCastlingRequirementsFor(moveBitboard, Black, Kingside)
	g.updateCastlingRequirementsFor(moveBitboard, Black, Queenside)

	if move.MoveType == CastlingMove {
		g.PlayerAndCastlingSideAllowed[g.Player][Kingside] = false
		g.PlayerAndCastlingSideAllowed[g.Player][Queenside] = false
	}

	return nil
}

func (g *GameState) applyMoveToBitboards(b *Bitboards, move Move) error {
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
				return fmt.Errorf("%v clearing %v %v: %w (%v)", move.DebugString(), StringFromBoardIndex(endIndex), endPiece, err, FenStringForGame(g))
			}

			// Move the capturing piece
			err = b.ClearSquare(startIndex, startPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w (%v)", move.DebugString(), StringFromBoardIndex(startIndex), startPiece, err, FenStringForGame(g))
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
				return fmt.Errorf("%v clearing %v %v: %w (%v)", move.DebugString(), StringFromBoardIndex(captureIndex), capturePiece, err, FenStringForGame(g))
			}
			err = b.ClearSquare(startIndex, startPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w (%v)", move.DebugString(), StringFromBoardIndex(startIndex), startPiece, err, FenStringForGame(g))
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
				return fmt.Errorf("%v clearing %v %v: %w (%v)", move.DebugString(), StringFromBoardIndex(startIndex), startPiece, err, FenStringForGame(g))
			}
			b.SetSquare(endIndex, startPiece)

			err = b.ClearSquare(rookStartIndex, rookPiece)
			if err != nil {
				return fmt.Errorf("%v clearing %v %v: %w (%v)", move.DebugString(), StringFromBoardIndex(rookStartIndex), rookPiece, err, FenStringForGame(g))
			}
			b.SetSquare(rookEndIndex, rookPiece)
		}
	}

	return nil
}

func (g *GameState) UndoUpdate(update *BoardUpdate, b *Bitboards) error {
	err := g.applyUndoToBitboards(update, b)
	if err != nil {
		return err
	}

	g.Player = update.PrevPlayer
	g.PlayerAndCastlingSideAllowed = update.PrevPlayerAndCastlingSideAllowed
	g.EnPassantTarget = update.PrevEnPassantTarget
	g.FullMoveClock = update.PrevFullMoveClock
	g.HalfMoveClock = update.PrevHalfMoveClock

	for i := update.Num - 1; i >= 0; i-- {
		index := update.Indices[i]
		piece := update.PrevPieces[i]

		g.Board[index] = piece
	}

	g.FenAndMoveHistoryForDebugging = g.FenAndMoveHistoryForDebugging[:len(g.FenAndMoveHistoryForDebugging)-1]
	return nil
}

func (g *GameState) applyUndoToBitboards(update *BoardUpdate, b *Bitboards) error {
	for i := update.Num - 1; i >= 0; i-- {
		index := update.Indices[i]
		current := update.Pieces[i]
		previous := update.PrevPieces[i]

		if current == XX {
			if previous == XX {
			} else {
				b.SetSquare(index, previous)
			}
		} else {
			var err error
			if previous == XX {
				err = b.ClearSquare(index, current)
			} else {
				err = b.ClearSquare(index, current)
				b.SetSquare(index, previous)
			}
			if err != nil {
				return fmt.Errorf("undo %v %v %v: %w", StringFromBoardIndex(index), current, previous, err)
			}
		}
	}
	return nil
}

func (g *GameState) Enemy() Player {
	return g.Player.Other()
}

func (g *GameState) WhiteCanCastleKingside() bool {
	return g.PlayerAndCastlingSideAllowed[White][Kingside]
}
func (g *GameState) WhiteCanCastleQueenside() bool {
	return g.PlayerAndCastlingSideAllowed[Black][Queenside]
}
func (g *GameState) BlackCanCastleKingside() bool {
	return g.PlayerAndCastlingSideAllowed[White][Kingside]
}
func (g *GameState) BlackCanCastleQueenside() bool {
	return g.PlayerAndCastlingSideAllowed[Black][Queenside]
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
