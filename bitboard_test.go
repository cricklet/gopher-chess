package chessgo

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleBoards(t *testing.T) {
	assert.Equal(t, singleBitboard(63).string(), strings.Join([]string{
		"00000001",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
	}, "\n"))
	assert.Equal(t, singleBitboard(0).string(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"10000000",
	}, "\n"))
	assert.Equal(t, singleBitboard(7).string(), strings.Join([]string{
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
	assert.Equal(t, ALL_ONES.string(), strings.Join([]string{
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
	assert.Equal(t, PRE_MOVE_MASKS[N].string(), strings.Join([]string{
		"00000000",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
		"11111111",
	}, "\n"))
	assert.Equal(t, PRE_MOVE_MASKS[NE].string(), strings.Join([]string{
		"00000000",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
		"11111110",
	}, "\n"))
	assert.Equal(t, PRE_MOVE_MASKS[SSW].string(), strings.Join([]string{
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
	g, err := gamestateFromString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.board.string(), NaturalBoardArray{
		BR, BN, BB, BQ, BK, BB, BN, BR,
		BP, BP, BP, BP, BP, BP, BP, BP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, WP, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		WP, WP, WP, WP, XX, WP, WP, WP,
		WR, WN, WB, WQ, WK, WB, WN, WR,
	}.AsBoardArray().string())

	bitboards := setupBitboards(g)
	assert.Equal(t, bitboards.occupied.string(), strings.Join([]string{
		"11111111",
		"11111111",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"11111111",
	}, "\n"))
	assert.Equal(t, bitboards.players[WHITE].occupied.string(), strings.Join([]string{
		"00000000",
		"00000000",
		"00000000",
		"00000000",
		"00001000",
		"00000000",
		"11110111",
		"11111111",
	}, "\n"))
	assert.Equal(t, bitboards.players[WHITE].pawns.string(), strings.Join([]string{
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
	assert.Equal(t, board.string(), strings.Join([]string{
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
	assert.Equal(t, board.string(), strings.Join([]string{
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
	assert.Equal(t, board.string(), strings.Join([]string{
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
	assert.Equal(t, board.string(), strings.Join([]string{
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
	assert.Equal(t, board.string(), strings.Join([]string{
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
	assert.Equal(t, board.string(), strings.Join([]string{
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

func TestGeneratePseudoMoves(t *testing.T) {
	s := "rnbqkbnr/pppp11pp/8/4pp2/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 1 2"
	g, err := gamestateFromString(s)
	assert.Nil(t, err)

	assert.Equal(t, g.board.string(), NaturalBoardArray{
		BR, BN, BB, BQ, BK, BB, BN, BR,
		BP, BP, BP, BP, XX, XX, BP, BP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, BP, BP, XX, XX,
		XX, XX, XX, XX, WP, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		WP, WP, WP, WP, XX, WP, WP, WP,
		WR, WN, WB, WQ, WK, WB, WN, WR,
	}.AsBoardArray().string())

	bitboards := setupBitboards(g)

	result := []string{}
	for _, move := range bitboards.generatePseudoMoves(g.player) {
		result = append(result, move.string())
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

func TestEachIndexOfOne(t *testing.T) {
	board := singleBitboard(63) | singleBitboard(3) | singleBitboard(5) | singleBitboard(30)
	assert.Equal(t, board.string(), strings.Join([]string{
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
	for _, index := range board.eachIndexOfOne() {
		result = append(result, stringFromBoardIndex(index))
	}

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

		assert.Equal(t, fileRank.string(), str)

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
		}).string(),
		singleBitboard(boardIndexFromString("c7")).string())
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
		}).string(),
		ROOK_BLOCKER_MASKS[boardIndexFromString("e2")].string())

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
		}).string(),
		ROOK_BLOCKER_MASKS[boardIndexFromString("d5")].string())

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
		}).string(),
		ROOK_BLOCKER_MASKS[boardIndexFromString("a1")].string())

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
		}).string(),
		BISHOP_BLOCKER_MASKS[boardIndexFromString("d5")].string())

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
		}).string(),
		BISHOP_BLOCKER_MASKS[boardIndexFromString("a1")].string())

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
		}).string(),
		BISHOP_BLOCKER_MASKS[boardIndexFromString("h1")].string())
}
