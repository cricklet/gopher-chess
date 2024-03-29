package game

import (
	"strings"
	"unicode/utf8"

	"github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/zobrist"
)

type MoveListener interface {
	AfterMove(move Move)
	AfterUndo()
}

type GameState struct {
	Board                        BoardArray
	Player                       Player
	PlayerAndCastlingSideAllowed [2][2]bool
	EnPassantTarget              Optional[FileRank]
	HalfMoveClock                int
	FullMoveClock                int

	Bitboards *bitboards.Bitboards

	zobristHash Optional[uint64]

	moveListeners []MoveListener

	noDefaultConstruction bool
	noCopy                NoCopy
}

func NewGameState(
	board BoardArray,
	player Player,
	playerAndCastlingSideAllowed [2][2]bool,
	enPassantTarget Optional[FileRank],
	halfMoveClock int,
	fullMoveClock int,
) *GameState {
	game := GameState{
		Board:                        board,
		Player:                       player,
		PlayerAndCastlingSideAllowed: playerAndCastlingSideAllowed,
		EnPassantTarget:              enPassantTarget,
		HalfMoveClock:                halfMoveClock,
		FullMoveClock:                fullMoveClock,
		Bitboards:                    &Bitboards{},
		noDefaultConstruction:        true,
	}

	for i, piece := range game.Board {
		if piece == XX {
			continue
		}
		pieceType := piece.PieceType()
		player := piece.Player()
		game.Bitboards.Players[player].Pieces[pieceType] |= SingleBitboard(i)

		if piece.IsWhite() {
			game.Bitboards.Occupied |= SingleBitboard(i)
			game.Bitboards.Players[White].Occupied |= SingleBitboard(i)
		}
		if piece.IsBlack() {
			game.Bitboards.Occupied |= SingleBitboard(i)
			game.Bitboards.Players[Black].Occupied |= SingleBitboard(i)
		}
	}

	return &game
}

func (g *GameState) RegisterListener(listener MoveListener) func() {
	g.moveListeners = append(g.moveListeners, listener)

	return func() {
		g.moveListeners = FilterSlice(g.moveListeners, func(l MoveListener) bool {
			return l != listener
		})
	}
}

func (g *GameState) ZobristHash() uint64 {
	if g.zobristHash.HasValue() {
		return g.zobristHash.Value()
	}
	g.zobristHash = Some(zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget))
	return g.zobristHash.Value()
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
		p := PieceTypeFromString(s[4:5])
		if p != InvalidPiece {
			promotion = Some(p)
		}
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

func setupBoardUpdate(g *GameState, move Move, output *BoardUpdate) Error {
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
			if !IsNil(err) {
				return err
			}
			rookPiece := g.Board[rookStartIndex]

			output.Add(g.Board[rookStartIndex], rookStartIndex, XX)
			output.Add(g.Board[rookEndIndex], rookEndIndex, rookPiece)
		}
	}

	output.PrevPlayer = g.Player
	output.PreviousCastlingRights = g.PlayerAndCastlingSideAllowed
	output.PrevEnPassantTarget = g.EnPassantTarget
	output.PrevFullMoveClock = g.FullMoveClock
	output.PrevHalfMoveClock = g.HalfMoveClock

	return NilError
}

func (g *GameState) updateCastlingRequirementsFor(moveBitboard Bitboard, player Player, side CastlingSide) {
	if moveBitboard&AllCastlingRequirements[player][side].Pieces != 0 {
		g.PlayerAndCastlingSideAllowed[player][side] = false
	}
}

func (g *GameState) PerformMove(move Move, update *BoardUpdate) Error {
	if !g.noDefaultConstruction {
		return Errorf("GameState must be constructed with NewGameState")
	}

	prevZobristHash := g.ZobristHash()
	err := setupBoardUpdate(g, move, update)
	if !IsNil(err) {
		return err
	}

	err = g.applyMoveToBitboards(update)
	if !IsNil(err) {
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

	g.zobristHash = Some(zobrist.UpdateHash(prevZobristHash, update, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget))

	for _, listener := range g.moveListeners {
		listener.AfterMove(move)
	}

	return NilError
}

func (g *GameState) applyMoveToBitboards(update *BoardUpdate) Error {
	for i := 0; i < update.Num; i++ {
		index := update.Indices[i]
		prevPiece := update.PrevPieces[i]
		nextPiece := update.Pieces[i]
		if nextPiece == XX {
			if prevPiece == XX {
			} else {
				err := g.Bitboards.ClearSquare(index, prevPiece)
				if !IsNil(err) {
					return err
				}
			}
		} else {
			if prevPiece == XX {
				g.Bitboards.SetSquare(index, nextPiece)
			} else {
				err := g.Bitboards.ClearSquare(index, prevPiece)
				if !IsNil(err) {
					return err
				}
				g.Bitboards.SetSquare(index, nextPiece)
			}
		}
	}

	return NilError
}

func (g *GameState) UndoUpdate(update *BoardUpdate) Error {
	if g.zobristHash.IsEmpty() {
		return Errorf("zobrist hash should have been setup during original move")
	}
	g.zobristHash = Some(zobrist.UpdateHash(g.zobristHash.Value(), update, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget))

	err := g.applyUndoToBitboards(update)
	if !IsNil(err) {
		return err
	}

	g.Player = update.PrevPlayer
	g.PlayerAndCastlingSideAllowed = update.PreviousCastlingRights
	g.EnPassantTarget = update.PrevEnPassantTarget
	g.FullMoveClock = update.PrevFullMoveClock
	g.HalfMoveClock = update.PrevHalfMoveClock

	for i := update.Num - 1; i >= 0; i-- {
		index := update.Indices[i]
		piece := update.PrevPieces[i]

		g.Board[index] = piece
	}

	for _, listener := range g.moveListeners {
		listener.AfterUndo()
	}

	return NilError
}

func (g *GameState) applyUndoToBitboards(update *BoardUpdate) Error {
	for i := update.Num - 1; i >= 0; i-- {
		index := update.Indices[i]
		current := update.Pieces[i]
		previous := update.PrevPieces[i]

		if current == XX {
			if previous == XX {
			} else {
				g.Bitboards.SetSquare(index, previous)
			}
		} else {
			var err Error
			if previous == XX {
				err = g.Bitboards.ClearSquare(index, current)
			} else {
				err = g.Bitboards.ClearSquare(index, current)
				g.Bitboards.SetSquare(index, previous)
			}
			if !IsNil(err) {
				return Errorf("undo %v %v %v: %w", StringFromBoardIndex(index), current, previous, err)
			}
		}
	}
	return NilError
}

func (g *GameState) String() string {
	return g.Board.String()
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
