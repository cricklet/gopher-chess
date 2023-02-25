package game

import (
	"fmt"
	"strings"
	"unicode/utf8"

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
	s = strings.TrimSpace(s)
	start := BoardIndexFromString(s[0:2])
	end := BoardIndexFromString(s[2:4])

	promotion := Empty[PieceType]()
	if utf8.RuneCountInString(s) >= 5 {
		promotion = Some(PieceTypeFromString(s[4:5]))
	}

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
		MoveType:       moveType,
		StartIndex:     start,
		EndIndex:       end,
		PromotionPiece: promotion,
		Evaluation:     Empty[int](),
	}
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
	startPlayer := startPiece.Player()

	if startPiece.PieceType() == Pawn && move.PromotionPiece.HasValue() {
		endPiece := PieceForPlayer[startPlayer][move.PromotionPiece.Value()]
		output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
		output.Add(g.Board[move.EndIndex], move.EndIndex, endPiece)
	} else {
		output.Add(g.Board[move.StartIndex], move.StartIndex, XX)
		output.Add(g.Board[move.EndIndex], move.EndIndex, startPiece)
	}

	switch move.MoveType {
	case QuietMove:
	case CaptureMove:
	case EnPassantMove:
		{
			backwardsDir := S
			if startPlayer == Black {
				backwardsDir = N
			}

			captureIndex := move.EndIndex + Offsets[backwardsDir]
			output.Add(g.Board[captureIndex], captureIndex, XX)
		}
	case CastlingMove:
		{
			rookStartIndex, rookEndIndex, err := RookMoveForCastle(move.StartIndex, move.EndIndex)
			if err != nil {
				return err
			}
			rookPiece := g.Board[rookStartIndex]

			output.Add(g.Board[rookStartIndex], rookStartIndex, XX)
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
	err := setupBoardUpdate(g, move, update)
	if err != nil {
		return err
	}

	g.FenAndMoveHistoryForDebugging = append(g.FenAndMoveHistoryForDebugging, [2]string{FenStringForGame(g), move.DebugString()})

	err = g.applyMoveToBitboards(b, update)
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

	if move.MoveType == CaptureMove || startPiece.PieceType() == Pawn {
		g.HalfMoveClock = 0
	} else {
		g.HalfMoveClock++
	}

	if g.Player == Black {
		g.FullMoveClock++
	}

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

	g.Player = g.Player.Other()

	return nil
}

func (g *GameState) applyMoveToBitboards(b *Bitboards, update *BoardUpdate) error {
	for i := 0; i < update.Num; i++ {
		index := update.Indices[i]
		prevPiece := update.PrevPieces[i]
		nextPiece := update.Pieces[i]
		if nextPiece == XX {
			if prevPiece == XX {
			} else {
				err := b.ClearSquare(index, prevPiece)
				if err != nil {
					return err
				}
			}
		} else {
			if prevPiece == XX {
				b.SetSquare(index, nextPiece)
			} else {
				err := b.ClearSquare(index, prevPiece)
				if err != nil {
					return err
				}
				b.SetSquare(index, nextPiece)
			}
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
	return g.PlayerAndCastlingSideAllowed[White][Queenside]
}
func (g *GameState) BlackCanCastleKingside() bool {
	return g.PlayerAndCastlingSideAllowed[Black][Kingside]
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
