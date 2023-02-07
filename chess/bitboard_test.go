package chess

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/profile"
	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/assert"
)

func TestSingleBoards(t *testing.T) {
	assert.Equal(t, singleBitboard(63).String(), strings.Join([]string{
		"00000001",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))
	assert.Equal(t, singleBitboard(0).String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"10000000",
	}, "\n"))
	assert.Equal(t, singleBitboard(7).String(), strings.Join([]string{
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
	assert.Equal(t, ALL_ONES.String(), strings.Join([]string{
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
	assert.Equal(t, PRE_MOVE_MASKS[N].String(), strings.Join([]string{
		"00000000",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
	}, "\n"))
	assert.Equal(t, PRE_MOVE_MASKS[NE].String(), strings.Join([]string{
		"00000000",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
	}, "\n"))
	assert.Equal(t, PRE_MOVE_MASKS[SSW].String(), strings.Join([]string{
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

	bitboards := SetupBitboards(&g)
	assert.Equal(t, bitboards.occupied.String(), strings.Join([]string{
		"11111111",
		"11111111",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"11111111",
	}, "\n"))
	assert.Equal(t, bitboards.players[WHITE].occupied.String(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"11111111",
	}, "\n"))
	assert.Equal(t, bitboards.players[WHITE].pieces[PAWN].String(), strings.Join([]string{
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
	board := singleBitboard(63)
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

	board = rotateTowardsIndex0(board, 3)
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
	board = rotateTowardsIndex0(board, 60)
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
	board = rotateTowardsIndex0(board, 3)
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
	board = rotateTowardsIndex64(board, 3)
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
	board = rotateTowardsIndex64(board, -3)
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

func TestEvaluation(t *testing.T) {
	s := "4k3/2R5/8/7r/8/r7/3R4/4K3 b - - 10 5"
	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	bitboards := SetupBitboards(&g)
	assert.Equal(t, strings.Join([]string{
		"    k   ",
		"  R     ",
		"        ",
		"       r",
		"        ",
		"r       ",
		"   R    ",
		"    K   ",
	}, "\n"), g.Board.String())

	assert.Equal(t, bitboards.evaluateDevelopment(WHITE), 2*DEVELOPMENT_SCALE)
	assert.Equal(t, bitboards.evaluateDevelopment(BLACK), -2*DEVELOPMENT_SCALE)

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

	bitboards := SetupBitboards(&g)

	result := []string{}
	moves := GetMovesBuffer()
	bitboards.generatePseudoMoves(&g, moves)
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

	assert.Equal(t, g.enPassantTarget.Value().String(), "g6")

	bitboards := SetupBitboards(&g)

	result := []string{}

	moves := GetMovesBuffer()
	bitboards.generatePseudoMoves(&g, moves)
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
	board := singleBitboard(63) | singleBitboard(3) | singleBitboard(5) | singleBitboard(30)
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
	for _, index := range *board.eachIndexOfOne(buffer) {
		result = append(result, stringFromBoardIndex(index))
	}
	ReleaseIndicesBuffer(buffer)

	sort.Strings(result)
	sort.Strings(expected)

	assert.Equal(t, result, expected)
}

func TestStringFromBoardIndex(t *testing.T) {
	for _, str := range []string{"a4", "c2", "h7"} {
		fileRank, err := fileRankFromString(str)
		if err != nil {
			panic(err)
		}

		assert.Equal(t, fileRank.String(), str)

		i := boardIndexFromString(str)
		j := boardIndexFromFileRank(fileRank)
		assert.Equal(t, str, stringFromBoardIndex(i))
		assert.Equal(t, str, stringFromBoardIndex(j))
	}
}

func TestBitboardFromStrings(t *testing.T) {
	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00100000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
		}).String(),
		singleBitboard(boardIndexFromString("c7")).String())
}

func TestBlockerMasks(t *testing.T) {
	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00001000",
			"00001000",
			"00001000",
			"00001000",
			"00001000",
			"01110110",
			"00000000",
		}).String(),
		ROOK_MAGIC_TABLE.blockerMasks[boardIndexFromString("e2")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00010000",
			"00010000",
			"01101110",
			"00010000",
			"00010000",
			"00010000",
			"00000000",
		}).String(),
		ROOK_MAGIC_TABLE.blockerMasks[boardIndexFromString("d5")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"10000000",
			"10000000",
			"10000000",
			"10000000",
			"10000000",
			"10000000",
			"01111110",
		}).String(),
		ROOK_MAGIC_TABLE.blockerMasks[boardIndexFromString("a1")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"01000100",
			"00101000",
			"00000000",
			"00101000",
			"01000100",
			"00000010",
			"00000000",
		}).String(),
		BISHOP_MAGIC_TABLE.blockerMasks[boardIndexFromString("d5")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00000010",
			"00000100",
			"00001000",
			"00010000",
			"00100000",
			"01000000",
			"00000000",
		}).String(),
		BISHOP_MAGIC_TABLE.blockerMasks[boardIndexFromString("a1")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"01000000",
			"00100000",
			"00010000",
			"00001000",
			"00000100",
			"00000010",
			"00000000",
		}).String(),
		BISHOP_MAGIC_TABLE.blockerMasks[boardIndexFromString("h1")].String())
}

func TestWhiteCastling(t *testing.T) {
	s := "r3k2r/pp1bb2p/2npPn2/q1p2Pp1/2B5/2N1BN2/PPP1QPPP/R3K2R w KQkq - 1 11"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	assert.Equal(t, true, g.enPassantTarget.IsEmpty())
	assert.Equal(t, WHITE, g.player)
	assert.Equal(t, [2][2]bool{{true, true}, {true, true}}, g.playerAndCastlingSideAllowed)

	bitboards := SetupBitboards(&g)

	result := []string{}

	moves := GetMovesBuffer()
	bitboards.generatePseudoMoves(&g, moves)
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

	assert.Equal(t, true, g.enPassantTarget.IsEmpty())
	assert.Equal(t, BLACK, g.player)
	assert.Equal(t, [2][2]bool{{true, true}, {true, true}}, g.playerAndCastlingSideAllowed)

	bitboards := SetupBitboards(&g)

	result := []string{}

	moves := GetMovesBuffer()
	bitboards.generatePseudoMoves(&g, moves)
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

	bitboards := SetupBitboards(&g)

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
		bitboardFromStrings([8]string{
			"01111110",
			"10111101",
			"11111110",
			"11111001",
			"11011111",
			"10100000",
			"10000000",
			"00000000",
		}).String(),
		bitboards.dangerBoard(WHITE).String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00010100",
			"10011010",
			"01111110",
			"10111101",
			"11111111",
			"10111101",
			"01111110",
		}).String(),
		bitboards.dangerBoard(BLACK).String())
}

func TestKnightMasks(t *testing.T) {
	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"01000000",
			"00100000",
			"00000000",
		}).String(),
		KNIGHT_ATTACK_MASKS[boardIndexFromString("a1")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00000000",
			"00010100",
			"00100010",
			"00000000",
			"00100010",
			"00010100",
			"00000000",
		}).String(),
		KNIGHT_ATTACK_MASKS[boardIndexFromString("e4")].String())

	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000010",
			"00000100",
			"00000000",
			"00000100",
		}).String(),
		KNIGHT_ATTACK_MASKS[boardIndexFromString("h2")].String())
	assert.Equal(t,
		bitboardFromStrings([8]string{
			"00000000",
			"00000100",
			"00000010",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
			"00000000",
		}).String(),
		KNIGHT_ATTACK_MASKS[boardIndexFromString("h8")].String())
}

func TestCheck(t *testing.T) {
	s := "r3k2r/pp1bb3/3pPPQp/qBp1n1p1/6n1/2N1BN2/PPP2PPP/R3K2R b KQkq - 1 14"

	g, err := GamestateFromFenString(s)
	assert.Nil(t, err)

	bitboards := SetupBitboards(&g)

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
	bitboards.generateLegalMoves(&g, &moves)
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

	bitboards := SetupBitboards(&g)

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
	bitboards.generateLegalMoves(&g, &moves)
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

type X struct {
	a []int
	b [2]int
	c int
}

func (x X) updateValue(v int) {
	x.a[0] = v
	x.a[1] = v
	x.b[0] = v
	x.b[1] = v
	x.c = v
}

func (x *X) updatePointer(v int) {
	x.a[0] = v
	x.a[1] = v
	x.b[0] = v
	x.b[1] = v
	x.c = v
}

func updateValueX(x X, v int) {
	x.a[0] = v
	x.a[1] = v
	x.b[0] = v
	x.b[1] = v
	x.c = v
}

func updatePointerX(x *X, v int) {
	x.a[0] = v
	x.a[1] = v
	x.b[0] = v
	x.b[1] = v
	x.c = v
}

func TestArraysArePassedByReference(t *testing.T) {
	x := X{[]int{1, 1}, [2]int{1, 1}, 1}

	x.updateValue(9)
	assert.Equal(t, X{[]int{9, 9}, [2]int{1, 1}, 1}, x)

	updateValueX(x, 99)
	assert.Equal(t, X{[]int{99, 99}, [2]int{1, 1}, 1}, x)

	x.updatePointer(999)
	assert.Equal(t, X{[]int{999, 999}, [2]int{999, 999}, 999}, x)

	updatePointerX(&x, 9999)
	assert.Equal(t, X{[]int{9999, 9999}, [2]int{9999, 9999}, 9999}, x)
}

func TestBitboardsCopyingIsDeep(t *testing.T) {
	b := Bitboards{}
	b.occupied = 7
	b.players[WHITE].occupied = 7
	b.players[WHITE].pieces[ROOK] = 7

	c := b
	c.occupied = 11
	c.players[WHITE].occupied = 11
	c.players[WHITE].pieces[ROOK] = 11

	assert.Equal(t, b.occupied, Bitboard(7))
	assert.Equal(t, b.players[WHITE].occupied, Bitboard(7))
	assert.Equal(t, b.players[WHITE].pieces[ROOK], Bitboard(7))

	assert.Equal(t, c.occupied, Bitboard(11))
	assert.Equal(t, c.players[WHITE].occupied, Bitboard(11))
	assert.Equal(t, c.players[WHITE].pieces[ROOK], Bitboard(11))
}

func TestGameStateCopyingIsDeep(t *testing.T) {
	b := GameState{}
	b.Board[0] = WQ
	b.halfMoveClock = 9
	b.playerAndCastlingSideAllowed[0][0] = true
	b.playerAndCastlingSideAllowed[0][1] = false

	c := b
	c.Board[0] = BQ
	c.halfMoveClock = 11
	c.playerAndCastlingSideAllowed[0][0] = false
	c.playerAndCastlingSideAllowed[0][1] = true

	assert.Equal(t, b.Board[0], WQ)
	assert.Equal(t, b.halfMoveClock, 9)
	assert.Equal(t, b.playerAndCastlingSideAllowed[0][0], true)
	assert.Equal(t, b.playerAndCastlingSideAllowed[0][1], false)

	assert.Equal(t, c.Board[0], BQ)
	assert.Equal(t, c.halfMoveClock, 11)
	assert.Equal(t, c.playerAndCastlingSideAllowed[0][0], false)
	assert.Equal(t, c.playerAndCastlingSideAllowed[0][1], true)
}

type PerftMap map[string]int

func countAndPerftForDepth(t *testing.T, g *GameState, b *Bitboards, n int, progress *chan int, perft *PerftMap) int {
	if b.kingIsInCheck(g.enemy(), g.player) {
		return 0
	}

	if n == 0 {
		return 1
	}

	num := 0

	moves := GetMovesBuffer()
	b.generatePseudoMoves(g, moves)

	for _, move := range *moves {
		update := BoardUpdate{}
		previous := OldGameState{}
		SetupBoardUpdate(g, move, &update)
		RecordCurrentState(g, &previous)

		b.performMove(g, move)
		g.performMove(move, update)

		countUnderMove := countAndPerftForDepth(t, g, b, n-1, nil, nil)

		b.undoUpdate(update)
		g.undoUpdate(previous, update)

		num += countUnderMove

		if perft != nil {
			(*perft)[move.String()] = countUnderMove
		}
		if progress != nil {
			*progress <- num
		}
	}

	ReleaseMovesBuffer(moves)

	return num
}

func CountAndPerftForDepthWithProgress(t *testing.T, g *GameState, b *Bitboards, n int, expectedCount int) (int, PerftMap) {
	perft := make(PerftMap)

	var progressBar *progressbar.ProgressBar
	var startTime time.Time
	if expectedCount > 9999999 {
		progressBar = progressbar.Default(int64(expectedCount), fmt.Sprint("depth ", n))
		startTime = time.Now()
	}

	progressChan := make(chan int)

	var result int
	go func() {
		result = countAndPerftForDepth(t, g, b, n, &progressChan, &perft)
		close(progressChan)
	}()

	for p := range progressChan {
		if progressBar != nil {
			progressBar.Set(p)
		}
	}

	if progressBar != nil {
		progressBar.Close()
		fmt.Println("             |", time.Now().Sub(startTime))
		fmt.Println()
	}

	return result, perft
}

type PerftComparison int

const (
	MOVE_IS_INVALID PerftComparison = iota
	MOVE_IS_MISSING
	COUNT_TOO_HIGH
	COUNT_TOO_LOW
)

func parsePerft(s string) (map[string]int, int) {
	expectedPerft := make(map[string]int)

	ok := false
	for _, line := range strings.Split(s, "\n") {
		if ok {
			if len(line) == 0 {
				continue
			} else if strings.HasPrefix(line, "Nodes searched: ") {
				expectedCountStr := strings.TrimPrefix(line, "Nodes searched: ")
				expectedCount, err := strconv.Atoi(expectedCountStr)
				if err != nil {
					panic(fmt.Sprint("couldn't parse searched nodes", line, err))
				}

				return expectedPerft, expectedCount
			} else {
				lineParts := strings.Split(line, ": ")
				moveStr := lineParts[0]
				moveCount, err := strconv.Atoi(lineParts[1])
				if err != nil {
					panic(fmt.Sprint("couldn't parse count from move", line, err))
				}

				expectedPerft[moveStr] = moveCount
			}
		} else if line == "uciok" {
			ok = true
		}
	}

	panic(fmt.Sprint("could not parse", s))
}

func computeIncorrectPerftMoves(t *testing.T, g *GameState, b *Bitboards, depth int) map[string]PerftComparison {
	if depth == 0 {
		panic("0 depth not valid for stockfish")
	}
	input := fmt.Sprintf("echo \"isready\nuci\nposition fen %v\ngo perft %v\" | stockfish", g.fenString(), depth)
	cmd := exec.Command("bash", "-c", input)
	output, _ := cmd.CombinedOutput()

	expectedPerft, expectedTotal := parsePerft(string(output))

	total, perft := CountAndPerftForDepthWithProgress(t, g, b, depth, expectedTotal)

	result := make(map[string]PerftComparison)

	for move, count := range perft {
		expectedCount, ok := expectedPerft[move]
		if ok == false {
			result[move] = MOVE_IS_INVALID
		} else if count > expectedCount {
			result[move] = COUNT_TOO_HIGH
		} else if count < expectedCount {
			result[move] = COUNT_TOO_LOW
		}
	}
	for expectedMove, _ := range expectedPerft {
		_, ok := perft[expectedMove]
		if ok == false {
			result[expectedMove] = MOVE_IS_MISSING
		}
	}

	if total != expectedTotal && len(result) == 0 {
		panic("should have found a discrepancy between perft")
	}

	return result
}

type MoveToSearch struct {
	move    string
	issue   PerftComparison
	initial string
}

func (p PerftComparison) String() string {
	switch p {
	case MOVE_IS_INVALID:
		return "invalid"
	case MOVE_IS_MISSING:
		return "missing"
	case COUNT_TOO_HIGH:
		return "high"
	case COUNT_TOO_LOW:
		return "low"
	}
	panic(fmt.Sprint("unknown issue"))
}

func (m MoveToSearch) String() string {
	return fmt.Sprintf("%v %v at \"%v\"",
		m.issue.String(),
		m.move,
		m.initial,
	)
}

var totalInvalidMoves int = 0

const MAX_TOTAL_INVALID_MOVES int = 20

func findInvalidMoves(t *testing.T, initialString string, maxDepth int) []string {
	result := []string{}
	movesToSearch := []MoveToSearch{}

	g, err := GamestateFromFenString(initialString)
	assert.Nil(t, err)
	b := SetupBitboards(&g)

	for i := 1; i <= maxDepth; i++ {
		incorrectMoves := computeIncorrectPerftMoves(t, &g, &b, i)
		if len(incorrectMoves) > 0 {
			for move, issue := range incorrectMoves {
				movesToSearch = append(movesToSearch, MoveToSearch{move, issue, initialString})
			}
			break
		}
	}

	for _, search := range movesToSearch {
		if totalInvalidMoves > MAX_TOTAL_INVALID_MOVES {
			break
		}
		if search.issue == MOVE_IS_INVALID || search.issue == MOVE_IS_MISSING {
			result = append(result, search.String())
			totalInvalidMoves++
		} else {
			move := g.moveFromString(search.move)

			update := BoardUpdate{}
			previous := OldGameState{}
			SetupBoardUpdate(&g, move, &update)
			RecordCurrentState(&g, &previous)

			b.performMove(&g, move)
			g.performMove(move, update)

			nextString := g.fenString()

			b.undoUpdate(update)
			g.undoUpdate(previous, update)

			result = append(result, findInvalidMoves(t, nextString, maxDepth-1)...)
		}
	}

	if len(result) == 0 && len(movesToSearch) > 0 && totalInvalidMoves < MAX_TOTAL_INVALID_MOVES {
		panic("we weren't able to find the invalid move")
	}
	return result
}

func TestIncorrectEnPassantOutOfBounds(t *testing.T) {
	s := "rnbqkb1r/1ppppppp/5n2/p7/6PP/8/PPPPPP2/RNBQKBNR/ w KQkq a6 2 2"
	invalidMoves := findInvalidMoves(t, s, 2)

	for _, move := range invalidMoves {
		assert.Equal(t, nil, move)
	}
}

func TestIncorrectUndoBoard(t *testing.T) {
	s := "rnbqkbnr/pp1p1ppp/2p5/4pP2/8/2P5/PP1PP1PP/RNBQKBNR/ b KQkq - 5 3"
	invalidMoves := findInvalidMoves(t, s, 3)

	for _, move := range invalidMoves {
		assert.Equal(t, nil, move)
	}
}

func TestFindIncorrectMoves(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	invalidMoves := findInvalidMoves(t, s, 3)

	for _, move := range invalidMoves {
		assert.Equal(t, nil, move)
	}
}

func TestMovesAtDepthForPawnOutOfBoundsCapture(t *testing.T) {
	s := "rnbqkbnr/1ppppppp/8/p7/8/7P/PPPPPPP1/RNBQKBNR w KQkq - 0 2"

	EXPECTED_COUNT := []int{
		1,
		19,
		399,
	}

	for depth, expectedCount := range EXPECTED_COUNT {
		g, err := GamestateFromFenString(s)
		assert.Nil(t, err)
		b := SetupBitboards(&g)
		actualCount, _ := CountAndPerftForDepthWithProgress(t, &g, &b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualCount)
	}
}

type TestBuffer []int

var GetTestBuffer, ReleaseTestBuffer, StatsTestBuffer = createPool(func() TestBuffer { return make(TestBuffer, 0, 64) }, func(x *TestBuffer) { *x = (*x)[:0] })

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

func TestMovesAtDepth(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	EXPECTED_COUNT := []int{
		1,
		20,
		400,
		8902,
		197281,
		4865609,
		119060324,
		// 3195901860,
	}

	defer profile.Start(profile.ProfilePath("../data/TestMovesAtDepth")).Stop()
	for depth, expectedCount := range EXPECTED_COUNT {
		g, err := GamestateFromFenString(s)
		assert.Nil(t, err)
		b := SetupBitboards(&g)
		actualCount, _ := CountAndPerftForDepthWithProgress(t, &g, &b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualCount)
	}
	for depth, expectedCount := range EXPECTED_COUNT {
		g, err := GamestateFromFenString(s)
		assert.Nil(t, err)
		b := SetupBitboards(&g)
		actualCount, _ := CountAndPerftForDepthWithProgress(t, &g, &b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualCount)
	}

	fmt.Println("indices pool ", StatsIndicesBuffer().String())
	fmt.Println("move pool ", StatsMoveBuffer().String())
}

type TestSlice []int

var GetTestSlice, ReleaseTestSlice, StatsTestSlice = createPool(
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
func (xs *TestArray) get(i int) int {
	return xs._values[i]
}

var GetTestArray, ReleaseTestArray, StatsTestArray = createPool(
	func() TestArray { return TestArray{} },
	func(x *TestArray) { x.size = 0 },
)

func TestSliceVsArray(t *testing.T) {
	defer profile.Start(profile.ProfilePath("../data/TestSliceVsArray")).Stop()
	var wg sync.WaitGroup

	competingThreads := 25
	allocationsPerThread := 99999
	sliceProgress := progressbar.Default(int64(competingThreads*allocationsPerThread), "slice")
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

	arrayProgress := progressbar.Default(int64(competingThreads*allocationsPerThread), "array")
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

	fmt.Println("slices ", StatsTestSlice().String())
	fmt.Println("array ", StatsTestArray().String())
}

func TestEachIndexOfOneCallbackVsRange(t *testing.T) {
	defer profile.Start(profile.ProfilePath("../data/TestEachIndexOfOneCallbackVsRange")).Stop()

	testNum := uint64(9999999)

	buffer := GetIndicesBuffer()

	bufferProgress := progressbar.Default(int64(testNum), "array")
	for i := uint64(0); i < testNum; i++ {
		for range *Bitboard(i).eachIndexOfOne(buffer) {
		}
		if i%1000 == 0 {
			bufferProgress.Add(1000)
		}
	}
	bufferProgress.Close()

	var f = func(index int) {
	}
	callbackProgress := progressbar.Default(int64(testNum), "callback")
	for i := uint64(0); i < testNum; i++ {
		Bitboard(i).eachIndexOfOne2(f)
		if i%1000 == 0 {
			callbackProgress.Add(1000)
		}
	}
	callbackProgress.Close()

	manualProgress := progressbar.Default(int64(testNum), "manual")
	for i := uint64(0); i < testNum; i++ {
		temp := Bitboard(i)
		for temp != 0 {
			_, temp = temp.nextIndexOfOne()
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

	singleProgress := progressbar.Default(int64(testNum), "single")
	for i := 0; i < testNum; i++ {
		for j := range single {
			_ = single[j]
		}
		if i%1000 == 0 {
			singleProgress.Add(1000)
		}
	}
	singleProgress.Close()

	doubleProgress := progressbar.Default(int64(testNum), "double")
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

	ifProgress := progressbar.Default(int64(testNum), "if")
	for i := 0; i < testNum; i++ {
		for j := 0; j <= int(BP); j++ {
			piece := Piece(j)
			_ = piece.player()
		}
		if i%1000 == 0 {
			ifProgress.Add(1000)
		}
	}
	ifProgress.Close()

	lookupProgress := progressbar.Default(int64(testNum), "lookup")
	for i := 0; i < testNum; i++ {
		for j := 0; j <= int(BP); j++ {
			piece := Piece(j)
			_ = piece.player2()
		}
		if i%1000 == 0 {
			lookupProgress.Add(1000)
		}
	}
	lookupProgress.Close()

}
