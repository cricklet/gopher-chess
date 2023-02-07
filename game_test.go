package chessgo

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

	assert.Equal(t, b.String(), strings.Join([]string{
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
	}, "\n"))
}

func TestLocationDecoding(t *testing.T) {
	location, err := fileRankFromString("a1")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{0, 0})

	game, err := GamestateFromFenString("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	assert.Nil(t, err)

	assert.Equal(t, pieceAtFileRank(game.Board, location).String(), WR.String())

	location, err = fileRankFromString("e4")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{4, 3})

	assert.Equal(t, pieceAtFileRank(game.Board, location).String(), WP.String())

	location, err = fileRankFromString("d8")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{3, 7})

	assert.Equal(t, pieceAtFileRank(game.Board, location).String(), BQ.String())

	location, err = fileRankFromString("a1")
	assert.Nil(t, err)
	assert.Equal(t, boardIndexFromFileRank(location), 0)

	location, err = fileRankFromString("h1")
	assert.Nil(t, err)
	assert.Equal(t, boardIndexFromFileRank(location), 7)

	location, err = fileRankFromString("a8")
	assert.Nil(t, err)
	assert.Equal(t, boardIndexFromFileRank(location), 56)

	location, err = fileRankFromString("h8")
	assert.Nil(t, err)
	assert.Equal(t, boardIndexFromFileRank(location), 63)
}

func TestNotationDecoding(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.Board.String(), NaturalBoardArray{
		BR, BN, BB, BQ, BK, BB, BN, BR,
		BP, BP, BP, BP, BP, BP, BP, BP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, WP, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		WP, WP, WP, WP, XX, WP, WP, WP,
		WR, WN, WB, WQ, WK, WB, WN, WR,
	}.AsBoardArray().String())

	assert.Equal(t, g.player, BLACK)

	expectedLocation, err := fileRankFromString("e3")
	assert.Nil(t, err)
	assert.Equal(t, g.enPassantTarget.Value(), expectedLocation)

	assert.Equal(t, g.whiteCanCastleKingside(), true)
	assert.Equal(t, g.whiteCanCastleQueenside(), true)
	assert.Equal(t, g.blackCanCastleKingside(), true)
	assert.Equal(t, g.blackCanCastleQueenside(), true)

	assert.Equal(t, g.halfMoveClock, 0)
	assert.Equal(t, g.fullMoveClock, 1)
}

func TestNotationDecoding2(t *testing.T) {
	s := "8/5k2/3p4/1p1Pp2p/pP2Pp1P/P4P1K/8/8 w - - 99 50"
	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.Board, NaturalBoardArray{
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, BK, XX, XX,
		XX, XX, XX, BP, XX, XX, XX, XX,
		XX, BP, XX, WP, BP, XX, XX, BP,
		BP, WP, XX, XX, WP, BP, XX, WP,
		WP, XX, XX, XX, XX, WP, XX, WK,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
	}.AsBoardArray())
	assert.Equal(t, g.player, WHITE)

	assert.Equal(t, true, g.enPassantTarget.IsEmpty())

	assert.Equal(t, g.whiteCanCastleKingside(), false)
	assert.Equal(t, g.whiteCanCastleQueenside(), false)
	assert.Equal(t, g.blackCanCastleKingside(), false)
	assert.Equal(t, g.blackCanCastleQueenside(), false)

	assert.Equal(t, g.halfMoveClock, 99)
	assert.Equal(t, g.fullMoveClock, 50)
}

func TestUCI(t *testing.T) {
	inputs := []string{
		"isready",
		"uci",
		"position fen rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"go",
	}
	r := Runner{}
	for _, line := range inputs {
		r.HandleInputAndReturnDone(line)
	}
}
