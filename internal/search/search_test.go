package search

import (
	"fmt"
	"testing"
	"time"

	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestOpeningWithoutQuiescence(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	result, score, err := Search(fen, WithMaxDepth{5}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	fmt.Println("searching depth 5", score, result)

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true, "g1f3": true, "b1c3": true}
	assert.True(t, expectedOpenings[result[0].String()])

	// Note that searching an even depth will cause us to play overly cautiously
	// because we don't have quiescence turned on so we can't see that we are
	// able to trade when an emeny captures a piece after us
	result, _, err = Search(fen, WithMaxDepth{4}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	fmt.Println("searching depth 4", score, result)

	assert.False(t, expectedOpenings[result[0].String()])
}
func TestOpeningWithoutQuiescenceE2E4(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	searchMoves, err := SearchTreeFromLines(
		fen,
		[][]string{
			{"e2e4"},
			{"e2e3"},
		},
		true, // continue searching past e2e4 and e2e3
	)
	assert.True(t, IsNil(err), err)

	// with search depth 4, we will be conservative
	result, _, err := Search(fen,
		WithMaxDepth{4}, WithSearch{searchMoves}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	assert.Equal(t, "e2e3", result[0].String())

	// with search depth 5, we will be aggressive
	result, _, err = Search(fen,
		WithMaxDepth{5}, WithSearch{searchMoves}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	assert.Equal(t, "e2e4", result[0].String())
}

func TestOpeningWithQuiescenceE2E4(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	searchMoves, err := SearchTreeFromLines(
		fen,
		[][]string{
			{"e2e4"},
			{"e2e3"},
		},
		true, // continue searching past e2e4 and e2e3
	)
	assert.True(t, IsNil(err), err)

	// with search depth 4 and quiescence enabled, we should be aggressive
	result, _, err := Search(fen,
		WithMaxDepth{4}, WithSearch{searchMoves})
	assert.True(t, IsNil(err), err)

	assert.Equal(t, "e2e4", result[0].String())
}

func TestOpeningCaptureWithoutQuiescence(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	searchMoves, err := SearchTreeFromLines(
		fen,
		[][]string{
			{
				"e2e4", "f7f5", "b1c3", "f5e4", "c3e4",
			},
		},
		false, // only search the specified line
	)
	assert.True(t, IsNil(err))

	// without quiescence, if we don't search far enough, we don't see trades
	result, score, err := Search(fen, WithSearch{searchMoves}, WithMaxDepth{4}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
	fmt.Println(result, score)
	assert.Less(t, score, 0)
	assert.Equal(t,
		"e2e4, f7f5, b1c3, f5e4",
		ConcatStringify(result))

	result, score, err = Search(fen, WithSearch{searchMoves}, WithMaxDepth{5}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
	fmt.Println(result, score)
	assert.Greater(t, score, 0)
	assert.Equal(t,
		"e2e4, f7f5, b1c3, f5e4, c3e4",
		ConcatStringify(result))
}

func TestOpeningCaptureWithQuiescence(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	searchMoves, err := SearchTreeFromLines(
		fen,
		[][]string{
			{
				"e2e4", "f7f5", "b1c3", "f5e4", "c3e4",
			},
		},
		false, // only search the specified line
	)
	assert.True(t, IsNil(err))

	// we should see the trades because of quiescence
	result, score, err := Search(fen, WithSearch{searchMoves}, WithMaxDepth{4}, WithDebugLogging{})
	assert.True(t, IsNil(err), err)
	fmt.Println(score, ConcatStringify(result))

	assert.Greater(t, score, 0)
	assert.Equal(t,
		"e2e4, f7f5, b1c3, f5e4, c3e4",
		ConcatStringify(result))

	result, score, err = Search(fen, WithSearch{searchMoves}, WithMaxDepth{5})
	assert.True(t, IsNil(err), err)
	fmt.Println(score, ConcatStringify(result))

	assert.Greater(t, score, 0)
	assert.Equal(t,
		"e2e4, f7f5, b1c3, f5e4, c3e4",
		ConcatStringify(result))
}

func TestOpeningResponse(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR b KQkq - 0 1"

	result, score, err := Search(fen, WithMaxDepth{2}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	fmt.Println(score, result)

	expectedOpenings := map[string]bool{"d7d5": true, "e7e5": true, "g8f6": true, "b8c6": true}
	assert.True(t, expectedOpenings[result[0].String()])
}

func TestPointlessSacrifice(t *testing.T) {
	fen := "rnbqkbnr/ppp2ppp/8/3pp3/4P3/3P1N2/PPP2PPP/RNBQKB1R b KQkq - 5 3"

	result, score, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	fmt.Println(score, result[0].String())

	assert.NotEqual(t, "c8f5", result[0].String())
}

func TestNoLegalMoves(t *testing.T) {
	fen := "rn1qkb1r/ppp3pp/5n2/3ppb2/8/2NP1NP1/PPP2PBP/R1BQK2R b KQkq - 13 7"

	result, score, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	fmt.Println(score, result[0].String())

	assert.True(t, result != nil)
}

func TestCheckMateSearch(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8 b KQkq - 13 7"

	result, _, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	assert.Equal(t, len(result), 0)
}

func TestCheckMateDetection(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8/8 b KQkq - 13 7"

	result, _, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	assert.Equal(t, len(result), 0)

	game, err := game.GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	isCheckMate := PlayerIsInCheck(&game, &bitboards)
	assert.True(t, isCheckMate)
}

func TestCheckMateInTwo(t *testing.T) {
	fen := "1K6/8/1b6/5k2/1p2p3/8/2q5/n7 b - - 2 2"

	// searching 3 ahead doesn't see the checkmate because we aren't able
	// to see that the enemy has no moves allowed
	result, score, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
	assert.True(t, result != nil)

	checkMateMoves := map[string]bool{"c2c7": true}
	assert.Less(t, score, 9999)
	assert.False(t, checkMateMoves[result[0].String()], result[0].String())

	// instead search 4 ahead
	result, score, err = Search(fen, WithMaxDepth{4}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
	assert.True(t, result != nil)

	assert.Greater(t, score, 9999)
	assert.True(t, checkMateMoves[result[0].String()], result[0].String())
}
func TestCheckMateInTwoSpecific(t *testing.T) {
	fen := "1K6/8/1b6/5k2/1p2p3/8/2q5/n7 b - - 2 2"

	searchMoves, err := SearchTreeFromLines(
		fen,
		[][]string{
			{
				"c2c7", "b8a8", "e4e3",
			},
			{
				"c2c7", "b8a8", "c7a7",
			},
		},
		true, // search everything past these two lines
	)
	assert.True(t, IsNil(err))

	result, score, err := Search(fen, WithSearch{searchMoves}, WithMaxDepth{4}, WithoutQuiescence{})
	assert.True(t, IsNil(err))
	assert.Greater(t, score, 9999)
	assert.Equal(t,
		"c2c7, b8a8, c7a7",
		ConcatStringify(result))
}

func TestCheckMateInOne2(t *testing.T) {
	fen := "5b2/3kp2p/4r3/1p6/4n3/p3P1p1/3p1r2/6K1 b - - 1 46"

	result, score, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)

	assert.True(t, result != nil)

	fmt.Println(score, result)

	checkMateMoves := map[string]bool{"d2d1q": true}
	assert.True(t, checkMateMoves[result[0].String()], result[0].String())
}

func TestQuiescence(t *testing.T) {
	fen := "r1bqk2r/p1p2ppp/1pnp1n2/4p3/1bPPP3/2N3P1/PP2NPBP/R1BQK2R b KQkq d3 0 7"

	_, _, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
}

func TestCrash1(t *testing.T) {
	fen := "r1b1q3/2p2k2/pp3Q2/3pBn2/8/2N5/PP4PP/5R1K b - - 2 24"

	result, score, err := Search(fen, WithMaxDepth{3}, WithoutQuiescence{})

	assert.True(t, result != nil)
	assert.True(t, IsNil(err), err)

	fmt.Println(score, result)
}
func TestCrash2(t *testing.T) {
	fen := "4qk1r/3R3p/5p1p/2Q1p3/p6K/6PP/8/8 b - - 9 38"

	result, score, err := Search(fen, WithMaxDepth{4}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
	assert.True(t, result != nil)
	fmt.Println(score, result)
}
func TestCrash3(t *testing.T) {
	fen := "rk1R1r2/pp4Q1/7p/4pN2/P1Pp4/3P4/2P3PP/R1n4K b - - 0 24"

	result, score, err := Search(fen, WithMaxDepth{4}, WithoutQuiescence{})
	assert.True(t, IsNil(err), err)
	assert.True(t, result != nil)
	fmt.Println(score, result)
}

func TestSearchDepthTime(t *testing.T) {
	fen := "r3k2r/1bq1bppp/pp2p3/2p1n3/P3PP2/2PBN3/1P1BQ1PP/R4RK1 b kq - 0 16"

	{
		start := time.Now()
		_, _, err := Search(fen, WithMaxDepth{2}, WithoutQuiescence{}, WithoutIterativeDeepening{})
		elapsed := time.Now().Sub(start)
		assert.True(t, IsNil(err), err)

		fmt.Println("NO quiescence, NO iterative", elapsed.Milliseconds(), "ms")
	}

	{
		start := time.Now()
		_, _, err := Search(fen, WithMaxDepth{2}, WithoutQuiescence{})
		elapsed := time.Now().Sub(start)
		assert.True(t, IsNil(err), err)

		fmt.Println("NO quiescence, WITH iterative", elapsed.Milliseconds(), "ms")
	}

	{
		start := time.Now()
		_, _, err := Search(fen, WithMaxDepth{2}, WithoutIterativeDeepening{})
		elapsed := time.Now().Sub(start)
		assert.True(t, IsNil(err), err)

		fmt.Println("WITH quiescence, NO iterative", elapsed.Milliseconds(), "ms")
	}

	{
		start := time.Now()
		_, _, err := Search(fen, WithMaxDepth{2})
		elapsed := time.Now().Sub(start)
		assert.True(t, IsNil(err), err)

		fmt.Println("WITH quiescence, WITH iterative", elapsed.Milliseconds(), "ms")
	}
}
