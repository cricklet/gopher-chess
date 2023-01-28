package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoardPrint(t *testing.T) {
	b := BoardArray{
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
	}

	assert.Equal(t, b.string(), strings.Join([]string{
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        \n",
	}, "\n"))
}

func TestLocationDecoding(t *testing.T) {
	location, err := locationFromString("a1")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{0, 7})

	game, err := gamestateFromString("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	assert.Nil(t, err)

	assert.Equal(t, pieceAtFileRank(game.board, location), WR)

	location, err = locationFromString("e4")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{4, 4})

	assert.Equal(t, pieceAtFileRank(game.board, location), WP)
}

func TestNotationDecoding(t *testing.T) {
	s := "8/5k2/3p4/1p1Pp2p/pP2Pp1P/P4P1K/8/8 w - - 99 50"
	g, err := gamestateFromString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.board, BoardArray{
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, BK, XX, XX,
		XX, XX, XX, BP, XX, XX, XX, XX,
		XX, BP, XX, WP, BP, XX, XX, BP,
		BP, WP, XX, XX, WP, BP, XX, WP,
		WP, XX, XX, XX, XX, WP, XX, WK,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
	})
	assert.Equal(t, g.player, WHITE)

	assert.Nil(t, g.enPassantTarget)

	assert.Equal(t, g.whiteCanCastleKingside, false)
	assert.Equal(t, g.whiteCanCastleQueenside, false)
	assert.Equal(t, g.blackCanCastleKingside, false)
	assert.Equal(t, g.blackCanCastleQueenside, false)

	assert.Equal(t, g.halfMoveClock, 99)
	assert.Equal(t, g.fullMoveClock, 50)

	s = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	g, err = gamestateFromString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.board, BoardArray{
		BR, BN, BB, BQ, BK, BB, BN, BR,
		BP, BP, BP, BP, BP, BP, BP, BP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, WP, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		WP, WP, WP, WP, XX, WP, WP, WP,
		WR, WN, WB, WQ, WK, WB, WN, WR,
	})

	assert.Equal(t, g.player, BLACK)

	expectedLocation, err := locationFromString("e3")
	assert.Nil(t, err)
	assert.Equal(t, *g.enPassantTarget, expectedLocation)

	assert.Equal(t, g.whiteCanCastleKingside, true)
	assert.Equal(t, g.whiteCanCastleQueenside, true)
	assert.Equal(t, g.blackCanCastleKingside, true)
	assert.Equal(t, g.blackCanCastleQueenside, true)

	assert.Equal(t, g.halfMoveClock, 0)
	assert.Equal(t, g.fullMoveClock, 1)
}
