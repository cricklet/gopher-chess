package accuracy

import (
	"fmt"
	"testing"

	"github.com/cricklet/chessgo/internal/game"
	"github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestFindsTheRightCapture(t *testing.T) {
	epd := "r1bqk1r1/1p1p1n2/p1n2pN1/2p1b2Q/2P1Pp2/1PN5/PB4PP/R4RK1 w q - - bm Rxf4; id \"ERET 001 - Relief\";"

	fen := EpdToFen(epd)
	game, err := game.GamestateFromFenString(fen)
	assert.True(t, err.IsNil(), err)

	bitboards := game.CreateBitboards()

	bestMoves, err := MovesFromEpd("bm", epd, game, bitboards)

	assert.True(t, err.IsNil(), err)
	assert.Equal(t, 1, len(bestMoves))
	if len(bestMoves) == 1 {
		assert.Equal(t, "f1f4", bestMoves[0])
	}
}

func TestEpdPawn(t *testing.T) {
	epd := "r1b2r1k/ppp2ppp/8/4p3/2BPQ3/P3P1K1/1B3PPP/n3q1NR w - - bm dxe5; id \"ERET 011 - Attacking Castle\";"

	fen := EpdToFen(epd)
	game, err := game.GamestateFromFenString(fen)
	assert.True(t, err.IsNil(), err)

	bitboards := game.CreateBitboards()

	bestMoves, err := MovesFromEpd("bm", epd, game, bitboards)

	assert.True(t, err.IsNil(), err)
	assert.Equal(t, 1, len(bestMoves), bestMoves)
	if len(bestMoves) == 1 {
		assert.Equal(t, "d4e5", bestMoves[0])
	}
}

func TestDisambiguation(t *testing.T) {
	fen := "5k2/8/1p6/2P5/1b6/8/8/5K2 b - - 0 1"

	game, err := game.GamestateFromFenString(fen)
	assert.True(t, err.IsNil(), err)

	bitboards := game.CreateBitboards()

	move, err := MoveFromShorthand("Bxc5", game, bitboards)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "b4c5", move)

	move, err = MoveFromShorthand("bxc5", game, bitboards)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "b6c5", move)
}

func TestPawnPush(t *testing.T) {
	fen := "r1b1r1k1/1pqn1pbp/p2pp1p1/P7/1n1NPP1Q/2NBBR2/1PP3PP/R6K w"

	game, err := game.GamestateFromFenString(fen)
	assert.True(t, err.IsNil(), err)

	bitboards := game.CreateBitboards()

	move, err := MoveFromShorthand("f5", game, bitboards)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "f4f5", move)
}

func TestDisambiguateKnight(t *testing.T) {
	epd := "2rq1rk1/pb1n1ppN/4p3/1pb5/3P1Pn1/P1N5/1PQ1B1PP/R1B2RK1 b - - bm Nde5; id \"ERET 007 - Bishop Pair\""

	fen := EpdToFen(epd)
	game, err := game.GamestateFromFenString(fen)
	assert.True(t, err.IsNil(), err)

	bitboards := game.CreateBitboards()

	bestMoves, err := MovesFromEpd("bm", epd, game, bitboards)

	assert.True(t, err.IsNil(), err)
	assert.Equal(t, 1, len(bestMoves), bestMoves)
	if len(bestMoves) == 1 {
		assert.Equal(t, "d7e5", bestMoves[0])
	}
}

func CheckEpdParsing(t *testing.T, name string) {
	epds, err := LoadEpd(helpers.RootDir() + "/internal/accuracy/" + name)
	assert.True(t, err.IsNil(), err)

	for _, epd := range epds {
		fen := EpdToFen(epd)
		game, err := game.GamestateFromFenString(fen)
		assert.True(t, err.IsNil(), err)

		bitboards := game.CreateBitboards()

		_, err = MovesFromEpd("bm", epd, game, bitboards)
		assert.True(t, err.IsNil(),
			fmt.Sprintf("epd: %s, %v", epd, err))

		_, err = MovesFromEpd("am", epd, game, bitboards)
		assert.True(t, err.IsNil(), err)
		assert.True(t, err.IsNil(),
			fmt.Sprintf("epd: %s, %v", epd, err))
	}
}

func TestDecoding(t *testing.T) {
	CheckEpdParsing(t, "ccr.epd")
	CheckEpdParsing(t, "eigenmann.epd")
	CheckEpdParsing(t, "kaufman.epd")
	CheckEpdParsing(t, "louguet.epd")
	CheckEpdParsing(t, "wac.epd")
}
