package internal

import (
	"log"
	"sort"
	"strings"
	"sync"
	"testing"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/runner"
	. "github.com/cricklet/chessgo/internal/search"

	"github.com/pkg/profile"
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
	location, err := FileRankFromString("a1")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{File: 0, Rank: 0})

	game, err := GamestateFromFenString("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	assert.Nil(t, err)

	assert.Equal(t, PieceAtFileRank(game.Board, location).String(), WR.String())

	location, err = FileRankFromString("e4")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{File: 4, Rank: 3})

	assert.Equal(t, PieceAtFileRank(game.Board, location).String(), WP.String())

	location, err = FileRankFromString("d8")
	assert.Nil(t, err)
	assert.Equal(t, location, FileRank{File: 3, Rank: 7})

	assert.Equal(t, PieceAtFileRank(game.Board, location).String(), BQ.String())

	location, err = FileRankFromString("a1")
	assert.Nil(t, err)
	assert.Equal(t, IndexFromFileRank(location), 0)

	location, err = FileRankFromString("h1")
	assert.Nil(t, err)
	assert.Equal(t, IndexFromFileRank(location), 7)

	location, err = FileRankFromString("a8")
	assert.Nil(t, err)
	assert.Equal(t, IndexFromFileRank(location), 56)

	location, err = FileRankFromString("h8")
	assert.Nil(t, err)
	assert.Equal(t, IndexFromFileRank(location), 63)
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

	assert.Equal(t, g.Player, Black)

	expectedLocation, err := FileRankFromString("e3")
	assert.Nil(t, err)
	assert.Equal(t, g.EnPassantTarget.Value(), expectedLocation)

	assert.Equal(t, g.WhiteCanCastleKingside(), true)
	assert.Equal(t, g.WhiteCanCastleQueenside(), true)
	assert.Equal(t, g.BlackCanCastleKingside(), true)
	assert.Equal(t, g.BlackCanCastleQueenside(), true)

	assert.Equal(t, g.HalfMoveClock, 0)
	assert.Equal(t, g.FullMoveClock, 1)
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
	assert.Equal(t, g.Player, White)

	assert.Equal(t, true, g.EnPassantTarget.IsEmpty())

	assert.Equal(t, g.WhiteCanCastleKingside(), false)
	assert.Equal(t, g.WhiteCanCastleQueenside(), false)
	assert.Equal(t, g.BlackCanCastleKingside(), false)
	assert.Equal(t, g.BlackCanCastleQueenside(), false)

	assert.Equal(t, g.HalfMoveClock, 99)
	assert.Equal(t, g.FullMoveClock, 50)
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
		log.Println(r.HandleInput(line))
	}
}

func TestSingleBoards(t *testing.T) {
	assert.Equal(t, SingleBitboard(63).String(), strings.Join([]string{
		"00000001",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))
	assert.Equal(t, SingleBitboard(0).String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"10000000",
	}, "\n"))
	assert.Equal(t, SingleBitboard(7).String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000001",
	}, "\n"))
}

func TestAllOnes(t *testing.T) {
	assert.Equal(t, AllOnes.String(), strings.Join([]string{
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
	}, "\n"))
}

func TestDirMasks(t *testing.T) {
	assert.Equal(t, PreMoveMasks[N].String(), strings.Join([]string{
		"00000000",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
	}, "\n"))
	assert.Equal(t, PreMoveMasks[NE].String(), strings.Join([]string{
		"00000000",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
	}, "\n"))
	assert.Equal(t, PreMoveMasks[SSW].String(), strings.Join([]string{
		"01111111",
		"01111111",
		"01111111",
		"01111111",
		"01111111",
		"01111111",
		"00000000",
		"00000000",
	}, "\n"))
}

func TestBitboardSetup(t *testing.T) {
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

	bitboards := g.CreateBitboards()
	assert.Equal(t, bitboards.Occupied.String(), strings.Join([]string{
		"11111111",
		"11111111",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"11111111",
	}, "\n"))
	assert.Equal(t, bitboards.Players[White].Occupied.String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"11111111",
	}, "\n"))
	assert.Equal(t, bitboards.Players[White].Pieces[Pawn].String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"00000000",
	}, "\n"))
}

func TestBitRotation(t *testing.T) {
	board := SingleBitboard(63)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00000001",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))

	board = RotateTowardsIndex0(board, 3)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00001000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))
	board = RotateTowardsIndex0(board, 60)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"10000000",
	}, "\n"))
	board = RotateTowardsIndex0(board, 3)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00000100",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))
	board = RotateTowardsIndex64(board, 3)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"10000000",
	}, "\n"))
	board = RotateTowardsIndex64(board, -3)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00000100",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))
}

func TestGeneratePseudoMovesEarly(t *testing.T) {
	s := "rnbqkbnr/pppp11pp/8/4pp2/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 1 2"
	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.Board.String(), NaturalBoardArray{
		BR, BN, BB, BQ, BK, BB, BN, BR,
		BP, BP, BP, BP, XX, XX, BP, BP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, BP, BP, XX, XX,
		XX, XX, XX, XX, WP, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		WP, WP, WP, WP, XX, WP, WP, WP,
		WR, WN, WB, WQ, WK, WB, WN, WR,
	}.AsBoardArray().String())

	bitboards := g.CreateBitboards()

	result := []string{}
	moves := GetMovesBuffer()
	GeneratePseudoMoves(&bitboards, &g, moves)
	for _, move := range *moves {
		result = append(result, move.String())
	}

	expected := []string{
		"a2a3",
		"b2b3",
		"c2c3",
		"d2d3",
		// "e4e5", <-- blocked
		"f2f3",
		"g2g3",
		"h2h3",

		// skip step
		"a2a4",
		"b2b4",
		"c2c4",
		"d2d4",
		// "e4e6", <-- not allowed
		"f2f4",
		"g2g4",
		"h2h4",

		// captures
		"e4f5",

		// bishop
		"f1e2",
		"f1d3",
		"f1c4",
		"f1b5",
		"f1a6",

		// queen
		"d1e2",
		"d1f3",
		"d1g4",
		"d1h5",

		// king
		"e1e2",

		// queenside knight
		"b1a3",
		"b1c3",

		// kingside knight
		"g1f3",
		"g1h3",
		"g1e2",
	}

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, expected, result)
}

func TestGeneratePseudoMovesEnPassant(t *testing.T) {
	s := "rnbqkbnr/pppp3p/8/4pPp1/8/5N2/PPPP1PPP/RNBQKB1R w KQkq g6 0 4"
	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, NaturalBoardArray{
		BR, BN, BB, BQ, BK, BB, BN, BR,
		BP, BP, BP, BP, XX, XX, XX, BP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, BP, WP, BP, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, WN, XX, XX,
		WP, WP, WP, WP, XX, WP, WP, WP,
		WR, WN, WB, WQ, WK, WB, XX, WR,
	}.AsBoardArray().String(), g.Board.String())

	assert.Equal(t, g.EnPassantTarget.Value().String(), "g6")

	bitboards := g.CreateBitboards()

	result := []string{}

	moves := GetMovesBuffer()
	GeneratePseudoMoves(&bitboards, &g, moves)
	for _, move := range *moves {
		result = append(result, move.String())
	}

	expected := []string{
		"a2a3",
		"b2b3",
		"c2c3",
		"d2d3",
		"f5f6", // e pawn
		// "f2f3", // f pawn blocked
		"g2g3",
		"h2h3",

		// skip step
		"a2a4",
		"b2b4",
		"c2c4",
		"d2d4",
		// "f2f4", // f pawn blocked
		"g2g4",
		"h2h4",

		// captures
		"f5g6",

		// bishop
		"f1e2",
		"f1d3",
		"f1c4",
		"f1b5",
		"f1a6",

		// queen
		"d1e2",

		// king
		"e1e2",

		// rook
		"h1g1",

		// queenside knight
		"b1a3",
		"b1c3",

		// kingside knight
		"f3g1",
		"f3d4",
		"f3e5",
		"f3g5",
		"f3h4",
	}

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, expected, result)
}

func TestEachIndexOfOne(t *testing.T) {
	board := SingleBitboard(63) | SingleBitboard(3) | SingleBitboard(5) | SingleBitboard(30)
	assert.Equal(t, board.String(), strings.Join([]string{
		"00000001",
		"00000000",
		"00000000",
		"00000000",
		"00000010",
		"00000000",
		"00000000",
		"00010100",
	}, "\n"))

	expected := []string{
		"d1", "f1", "g4", "h8",
	}
	result := []string{}
	buffer := GetIndicesBuffer()
	for _, index := range *board.EachIndexOfOne(buffer) {
		result = append(result, StringFromBoardIndex(index))
	}
	ReleaseIndicesBuffer(buffer)

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, result, expected)
}

func TestStringFromBoardIndex(t *testing.T) {
	for _, str := range []string{"a4", "c2", "h7"} {
		fileRank, err := FileRankFromString(str)
		if err != nil {
			panic(err)
		}

		assert.Equal(t, fileRank.String(), str)

		i := BoardIndexFromString(str)
		j := IndexFromFileRank(fileRank)
		assert.Equal(t, str, StringFromBoardIndex(i))
		assert.Equal(t, str, StringFromBoardIndex(j))
	}
}

func TestBitboardFromStrings(t *testing.T) {
	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00100000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
		}).String(),
		SingleBitboard(BoardIndexFromString("c7")).String())
}

func TestBlockerMasks(t *testing.T) {
	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00001000",
			"00001000",
			"00001000",
			"00001000",
			"00001000",
			"01110110",
			"00000000",
		}).String(),
		RookMagicTable.BlockerMasks[BoardIndexFromString("e2")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00010000",
			"00010000",
			"01101110",
			"00010000",
			"00010000",
			"00010000",
			"00000000",
		}).String(),
		RookMagicTable.BlockerMasks[BoardIndexFromString("d5")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"10000000",
			"10000000",
			"10000000",
			"10000000",
			"10000000",
			"10000000",
			"01111110",
		}).String(),
		RookMagicTable.BlockerMasks[BoardIndexFromString("a1")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"01000100",
			"00101000",
			"00000000",
			"00101000",
			"01000100",
			"00000010",
			"00000000",
		}).String(),
		BishopMagicTable.BlockerMasks[BoardIndexFromString("d5")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00000010",
			"00000100",
			"00001000",
			"00010000",
			"00100000",
			"01000000",
			"00000000",
		}).String(),
		BishopMagicTable.BlockerMasks[BoardIndexFromString("a1")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"01000000",
			"00100000",
			"00010000",
			"00001000",
			"00000100",
			"00000010",
			"00000000",
		}).String(),
		BishopMagicTable.BlockerMasks[BoardIndexFromString("h1")].String())
}

func TestWhiteCastling(t *testing.T) {
	s := "r3k2r/pp1bb2p/2npPn2/q1p2Pp1/2B5/2N1BN2/PPP1QPPP/R3K2R w KQkq - 1 11"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, true, g.EnPassantTarget.IsEmpty())
	assert.Equal(t, White, g.Player)
	assert.Equal(t, [2][2]bool{{true, true}, {true, true}}, g.PlayerAndCastlingSideAllowed)

	bitboards := g.CreateBitboards()

	result := []string{}

	moves := GetMovesBuffer()
	GeneratePseudoMoves(&bitboards, &g, moves)
	for _, move := range *moves {
		result = append(result, move.String())
	}

	expected := []string{
		// rook
		"a1b1",
		"a1c1",
		"a1d1",

		// pawns
		"a2a3",
		"a2a4",
		"b2b3",
		"b2b4",

		// knight
		"c3a4",
		"c3b1",
		"c3b5",
		"c3d1",
		"c3d5",
		"c3e4",

		// bishop
		"c4a6",
		"c4b3",
		"c4b5",
		"c4d3",
		"c4d5",

		// king
		"e1d1",
		"e1d2",
		"e1f1",

		// queen
		"e2d1",
		"e2d2",
		"e2d3",
		"e2f1",
		"e1g1", // <-- castling
		"e1c1", // <-- castling

		// bishop
		"e3c1",
		"e3c5",
		"e3d2",
		"e3d4",
		"e3f4",
		"e3g5",

		// pawn
		"e6d7",

		// knight
		"f3d2",
		"f3d4",
		"f3e5",
		"f3g1",
		"f3g5",
		"f3h4",

		// pawn
		"g2g3",
		"g2g4",
		"h2h3",
		"h2h4",

		// rook
		"h1f1",
		"h1g1",
	}

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, expected, result)
}

func TestBlackCastling(t *testing.T) {
	s := "r3k2r/pp1bb2p/2npPn2/q1p2Pp1/2B5/2NQBN2/PPP2PPP/R3K2R b KQkq - 2 11"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, true, g.EnPassantTarget.IsEmpty())
	assert.Equal(t, Black, g.Player)
	assert.Equal(t, [2][2]bool{{true, true}, {true, true}}, g.PlayerAndCastlingSideAllowed)

	bitboards := g.CreateBitboards()

	result := []string{}

	moves := GetMovesBuffer()
	GeneratePseudoMoves(&bitboards, &g, moves)
	for _, move := range *moves {
		result = append(result, move.String())
	}

	expected := []string{
		// queen
		"a5a2",
		"a5a3",
		"a5a4",
		"a5a6",
		"a5b4",
		"a5b5",
		"a5b6",
		"a5c3",
		"a5c7",
		"a5d8",

		// pawn
		"a7a6",

		// rook
		"a8b8",
		"a8c8",
		"a8d8",

		// pawn
		"b7b5",
		"b7b6",

		//knight
		"c6b4",
		"c6b8",
		"c6d4",
		"c6d8",
		"c6e5",

		// pawn
		"d6d5",

		// bishop
		"d7c8",
		"d7e6",

		// bishop
		"e7d8",
		"e7f8",

		// king
		"e8c8", // <- castling
		"e8d8",
		"e8f7",
		"e8f8",
		"e8g8", // <- castling

		// knight
		"f6d5",
		"f6e4",
		"f6g4",
		"f6g8",
		"f6h5",

		// pawn
		"g5g4",
		"h7h5",
		"h7h6",

		// rook
		"h8f8",
		"h8g8",
	}

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, expected, result)
}

func TestAttackMap(t *testing.T) {
	s := "r3k2r/pp1bb2p/2npPn2/q1p2Pp1/2B5/2NQBN2/PPP2PPP/R3K2R b KQkq - 2 11"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	bitboards := g.CreateBitboards()

	assert.Equal(t, strings.Join([]string{
		"r   k  r",
		"pp bb  p",
		"  npPn  ",
		"q p  Pp ",
		"  B     ",
		"  NQBN  ",
		"PPP  PPP",
		"R   K  R",
	}, "\n"), g.Board.String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"01111110",
			"10111101",
			"11111110",
			"11111001",
			"11011111",
			"10100000",
			"10000000",
			"00000000",
		}).String(),
		DangerBoard(&bitboards, White).String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00010100",
			"10011010",
			"01111110",
			"10111101",
			"11111111",
			"10111101",
			"01111110",
		}).String(),
		DangerBoard(&bitboards, Black).String())
}

func TestKnightMasks(t *testing.T) {
	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"01000000",
			"00100000",
			"00000000",
		}).String(),
		KnightAttackMasks[BoardIndexFromString("a1")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00000000",
			"00010100",
			"00100010",
			"00000000",
			"00100010",
			"00010100",
			"00000000",
		}).String(),
		KnightAttackMasks[BoardIndexFromString("e4")].String())

	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000010",
			"00000100",
			"00000000",
			"00000100",
		}).String(),
		KnightAttackMasks[BoardIndexFromString("h2")].String())
	assert.Equal(t,
		BitboardFromStrings([8]string{
			"00000000",
			"00000100",
			"00000010",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
		}).String(),
		KnightAttackMasks[BoardIndexFromString("h8")].String())
}

func TestCheck(t *testing.T) {
	s := "r3k2r/pp1bb3/3pPPQp/qBp1n1p1/6n1/2N1BN2/PPP2PPP/R3K2R b KQkq - 1 14"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	bitboards := g.CreateBitboards()

	assert.Equal(t, strings.Join([]string{
		"r   k  r",
		"pp bb   ",
		"   pPPQp",
		"qBp n p ",
		"      n ",
		"  N BN  ",
		"PPP  PPP",
		"R   K  R",
	}, "\n"), g.Board.String())

	result := []string{}
	moves := make([]Move, 0)
	err = GenerateLegalMoves(&bitboards, &g, &moves)
	if err != nil {
		t.Error(err)
	}
	for _, move := range moves {
		result = append(result, move.String())
	}

	expected := []string{
		"e8d8", // move king
		"e8f8",
		"e5g6", // capture queen
		"e5f7", // block queen
	}

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, expected, result)
}

func TestPin(t *testing.T) {
	s := "5k2/8/8/8/1q6/2N4p/2PK2pP/8 w - - 0 44"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	bitboards := g.CreateBitboards()

	assert.Equal(t, strings.Join([]string{
		"     k  ",
		"        ",
		"        ",
		"        ",
		" q      ",
		"  N    p",
		"  PK  pP",
		"        ",
	}, "\n"), g.Board.String())

	result := []string{}
	moves := make([]Move, 0)
	err = GenerateLegalMoves(&bitboards, &g, &moves)
	if err != nil {
		t.Error(err)
	}
	for _, move := range moves {
		result = append(result, move.String())
	}

	expected := []string{
		// we can move the king
		"d2c1", "d2d1", "d2d3", "d2e1", "d2e2", "d2e3",
		// but the knight is pinned

	}

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, expected, result)
}

// type X struct {
// 	a []int
// 	b [2]int
// 	c int
// }

// func (x X) updateValue(v int) {
// 	x.a[0] = v
// 	x.a[1] = v
// 	x.b[0] = v
// 	x.b[1] = v
// 	x.c = v // lint:ignore SA4005
// }

// func (x *X) updatePointer(v int) {
// 	x.a[0] = v
// 	x.a[1] = v
// 	x.b[0] = v
// 	x.b[1] = v
// 	x.c = v
// }

// func updateValueX(x X, v int) {
// 	x.a[0] = v
// 	x.a[1] = v
// 	x.b[0] = v
// 	x.b[1] = v
// 	x.c = v
// }

// func updatePointerX(x *X, v int) {
// 	x.a[0] = v
// 	x.a[1] = v
// 	x.b[0] = v
// 	x.b[1] = v
// 	x.c = v
// }

// func TestArraysArePassedByReference(t *testing.T) {
// 	x := X{[]int{1, 1}, [2]int{1, 1}, 1}

// 	x.updateValue(9)
// 	assert.Equal(t, X{[]int{9, 9}, [2]int{1, 1}, 1}, x)

// 	updateValueX(x, 99)
// 	assert.Equal(t, X{[]int{99, 99}, [2]int{1, 1}, 1}, x)

// 	x.updatePointer(999)
// 	assert.Equal(t, X{[]int{999, 999}, [2]int{999, 999}, 999}, x)

// 	updatePointerX(&x, 9999)
// 	assert.Equal(t, X{[]int{9999, 9999}, [2]int{9999, 9999}, 9999}, x)
// }

func TestBitboardsCopyingIsDeep(t *testing.T) {
	b := Bitboards{}
	b.Occupied = 7
	b.Players[White].Occupied = 7
	b.Players[White].Pieces[Rook] = 7

	c := b
	c.Occupied = 11
	c.Players[White].Occupied = 11
	c.Players[White].Pieces[Rook] = 11

	assert.Equal(t, b.Occupied, Bitboard(7))
	assert.Equal(t, b.Players[White].Occupied, Bitboard(7))
	assert.Equal(t, b.Players[White].Pieces[Rook], Bitboard(7))

	assert.Equal(t, c.Occupied, Bitboard(11))
	assert.Equal(t, c.Players[White].Occupied, Bitboard(11))
	assert.Equal(t, c.Players[White].Pieces[Rook], Bitboard(11))
}

func TestGameStateCopyingIsDeep(t *testing.T) {
	b := GameState{}
	b.Board[0] = WQ
	b.HalfMoveClock = 9
	b.PlayerAndCastlingSideAllowed[0][0] = true
	b.PlayerAndCastlingSideAllowed[0][1] = false

	c := b
	c.Board[0] = BQ
	c.HalfMoveClock = 11
	c.PlayerAndCastlingSideAllowed[0][0] = false
	c.PlayerAndCastlingSideAllowed[0][1] = true

	assert.Equal(t, b.Board[0], WQ)
	assert.Equal(t, b.HalfMoveClock, 9)
	assert.Equal(t, b.PlayerAndCastlingSideAllowed[0][0], true)
	assert.Equal(t, b.PlayerAndCastlingSideAllowed[0][1], false)

	assert.Equal(t, c.Board[0], BQ)
	assert.Equal(t, c.HalfMoveClock, 11)
	assert.Equal(t, c.PlayerAndCastlingSideAllowed[0][0], false)
	assert.Equal(t, c.PlayerAndCastlingSideAllowed[0][1], true)
}

type TestBuffer []int

var GetTestBuffer, ReleaseTestBuffer, StatsTestBuffer = CreatePool(func() TestBuffer { return make(TestBuffer, 0, 64) }, func(x *TestBuffer) { *x = (*x)[:0] })

func RecursivelySetBuffer(t *testing.T, limit int, x *TestBuffer) {
	if limit <= 0 {
		return
	}

	*x = (*x)[:0]
	for i := 0; i < 64; i++ {
		*x = append(*x, limit)
	}
	for i := 0; i < 64; i++ {
		assert.Equal(t, (*x)[i], limit)
	}

	RecursivelySetBuffer(t, limit-1, x)
}

func TestThreadSafetyForPool(t *testing.T) {
	for i := 0; i < 64; i++ {
		go func() {
			buffer := GetTestBuffer()
			RecursivelySetBuffer(t, 10, buffer)
			ReleaseTestBuffer(buffer)
		}()
	}
}

type TestSlice []int

var GetTestSlice, ReleaseTestSlice, StatsTestSlice = CreatePool(
	func() TestSlice { return make(TestSlice, 0, 64) },
	func(x *TestSlice) { *x = (*x)[:0] },
)

type TestArray struct {
	_values [64]int
	size    int
}

func (xs *TestArray) add(x int) {
	xs._values[xs.size] = x
	xs.size++
}

// func (xs *TestArray) get(i int) int {
// 	return xs._values[i]
// }

var GetTestArray, ReleaseTestArray, StatsTestArray = CreatePool(
	func() TestArray { return TestArray{} },
	func(x *TestArray) { x.size = 0 },
)

func TestSliceVsArray(t *testing.T) {
	defer profile.Start(profile.ProfilePath("../data/TestSliceVsArray")).Stop()
	var wg sync.WaitGroup

	competingThreads := 25
	allocationsPerThread := 99999
	sliceProgress := CreateProgressBar(competingThreads*allocationsPerThread, "slice")
	for t := 0; t < competingThreads; t++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < allocationsPerThread; i++ {
				debugValue := i
				slice := GetTestSlice()
				for j := 0; j < 64; j++ {
					*slice = append(*slice, debugValue)
				}
				ReleaseTestSlice(slice)
				if i%100 == 0 {
					sliceProgress.Add(100)
				}
			}
		}()
	}
	wg.Wait()
	sliceProgress.Close()

	arrayProgress := CreateProgressBar(competingThreads*allocationsPerThread, "array")
	for t := 0; t < competingThreads; t++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < allocationsPerThread; i++ {
				debugValue := i
				array := GetTestArray()
				for j := 0; j < 64; j++ {
					array.add(debugValue)
				}
				ReleaseTestArray(array)
				if i%100 == 0 {
					arrayProgress.Add(100)
				}
			}
		}()
	}

	wg.Wait()
	arrayProgress.Close()

	log.Println("slices ", StatsTestSlice().String())
	log.Println("array ", StatsTestArray().String())
}

func TestEachIndexOfOneCallbackVsRange(t *testing.T) {
	defer profile.Start(profile.ProfilePath("../data/TestEachIndexOfOneCallbackVsRange")).Stop()

	testNum := 9999999

	buffer := GetIndicesBuffer()

	bufferProgress := CreateProgressBar(testNum, "array")
	for i := 0; i < testNum; i++ {
		for range *Bitboard(i).EachIndexOfOne(buffer) {
		}
		if i%1000 == 0 {
			bufferProgress.Add(1000)
		}
	}
	bufferProgress.Close()

	var f = func(index int) {
	}
	callbackProgress := CreateProgressBar(testNum, "callback")
	for i := 0; i < testNum; i++ {
		Bitboard(i).EachIndexOfOneCallback(f)
		if i%1000 == 0 {
			callbackProgress.Add(1000)
		}
	}
	callbackProgress.Close()

	manualProgress := CreateProgressBar(testNum, "manual")
	for i := 0; i < testNum; i++ {
		temp := Bitboard(i)
		for temp != 0 {
			_, temp = temp.NextIndexOfOne()
		}
		if i%1000 == 0 {
			manualProgress.Add(1000)
		}
	}
	manualProgress.Close()
}

func TestIndexSingeVsDoubleArray(t *testing.T) {
	defer profile.Start(profile.ProfilePath("../data/TestEachIndexOfOneCallbackVsRange")).Stop()

	double := [64][64]int{}
	single := [64 * 64]int{}
	testNum := 100000

	singleProgress := CreateProgressBar(testNum, "single")
	for i := 0; i < testNum; i++ {
		for j := range single {
			_ = single[j]
		}
		if i%1000 == 0 {
			singleProgress.Add(1000)
		}
	}
	singleProgress.Close()

	doubleProgress := CreateProgressBar(testNum, "double")
	for i := 0; i < testNum; i++ {
		for j := range double {
			interior := &double[j]
			for k := range *interior {
				_ = double[j][k]
			}
		}
		if i%1000 == 0 {
			doubleProgress.Add(1000)
		}
	}
	doubleProgress.Close()

}

func TestPlayerFromPiece(t *testing.T) {
	defer profile.Start(profile.ProfilePath("../data/TestPlayerFromPiece")).Stop()

	testNum := 100000000

	ifProgress := CreateProgressBar(testNum, "if")
	for i := 0; i < testNum; i++ {
		for j := 0; j <= int(BP); j++ {
			piece := Piece(j)
			_ = piece.Player()
		}
		if i%1000 == 0 {
			ifProgress.Add(1000)
		}
	}
	ifProgress.Close()

	lookupProgress := CreateProgressBar(testNum, "lookup")
	for i := 0; i < testNum; i++ {
		for j := 0; j <= int(BP); j++ {
			piece := Piece(j)
			_ = piece.PlayerLookup()
		}
		if i%1000 == 0 {
			lookupProgress.Add(1000)
		}
	}
	lookupProgress.Close()

}
