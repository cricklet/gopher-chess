package game

import (
	"testing"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestCastlingRights(t *testing.T) {
	s := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	b := g.CreateBitboards()
	update := BoardUpdate{}
	err = g.PerformMove(g.MoveFromString("e1c1"), &update, &b)
	assert.Nil(t, err)

	assert.False(t, g.WhiteCanCastleKingside())
	assert.False(t, g.WhiteCanCastleQueenside())
	assert.True(t, g.BlackCanCastleKingside())
	assert.True(t, g.BlackCanCastleQueenside())
}

func TestPromotion(t *testing.T) {
	s := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	b := g.CreateBitboards()
	update := BoardUpdate{}

	err = g.PerformMove(g.MoveFromString("d7c8q"), &update, &b)
	assert.Nil(t, err)

	assert.True(t,
		b.Players[White].Pieces[Queen]&SingleBitboard(BoardIndexFromString("c8")) != 0)
}
