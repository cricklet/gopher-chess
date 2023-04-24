package game

import (
	"testing"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/zobrist"
	"github.com/stretchr/testify/assert"
)

func TestCastlingRights(t *testing.T) {
	s := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()
	update := BoardUpdate{}
	err = g.PerformMove(g.MoveFromString("e1c1"), &update, &b)
	assert.True(t, IsNil(err))

	assert.False(t, g.WhiteCanCastleKingside())
	assert.False(t, g.WhiteCanCastleQueenside())
	assert.True(t, g.BlackCanCastleKingside())
	assert.True(t, g.BlackCanCastleQueenside())
}

func TestPromotion(t *testing.T) {
	s := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"

	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()
	update := BoardUpdate{}

	err = g.PerformMove(g.MoveFromString("d7c8q"), &update, &b)
	assert.True(t, IsNil(err))

	assert.True(t,
		b.Players[White].Pieces[Queen]&SingleBitboard(BoardIndexFromString("c8")) != 0)
}

func TestZobristHashIsKeptUpToDate(t *testing.T) {
	s := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()

	hash0 := zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget)
	assert.Equal(t, hash0, g.ZobristHash())

	update := BoardUpdate{}
	err = g.PerformMove(g.MoveFromString("e1c1"), &update, &b)
	assert.True(t, IsNil(err))

	hash1 := zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget)
	assert.Equal(t, hash1, g.ZobristHash())

	err = g.UndoUpdate(&update, &b)
	assert.True(t, IsNil(err))

	hash2 := zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget)
	assert.Equal(t, hash2, g.ZobristHash())

	assert.Equal(t, hash0, hash2)
}

func TestZobristHashSimple(t *testing.T) {
	s := "7K/8/8/8/8/8/8/7k w - - 0 1"

	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))
	b := g.CreateBitboards()

	hash0 := zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget)
	assert.Equal(t, hash0, g.ZobristHash())

	update := BoardUpdate{}
	err = g.PerformMove(g.MoveFromString("h1h2"), &update, &b)
	assert.True(t, IsNil(err))

	hash1 := zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget)
	assert.Equal(t, hash1, g.ZobristHash())

	err = g.UndoUpdate(&update, &b)
	assert.True(t, IsNil(err))

	hash2 := zobrist.HashForBoardPosition(&g.Board, g.Player, &g.PlayerAndCastlingSideAllowed, g.EnPassantTarget)
	assert.Equal(t, hash2, g.ZobristHash())

	assert.Equal(t, hash0, hash2)
}
