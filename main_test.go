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

func TestNotation(t *testing.T) {
	s := "8/5k2/3p4/1p1Pp2p/pP2Pp1P/P4P1K/8/8 b - - 99 50"
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

	assert.Nil(t, g.enPassantTarget)

	assert.Equal(t, g.whiteCanCastleKingside, false)

	assert.Equal(t, g.whiteCanCastleKingside, false)
	assert.Equal(t, g.whiteCanCastleQueenside, false)
	assert.Equal(t, g.blackCanCastleKingside, false)
	assert.Equal(t, g.blackCanCastleQueenside, false)

	assert.Equal(t, g.halfMoveClock, 99)
	assert.Equal(t, g.fullMoveClock, 50)
}
