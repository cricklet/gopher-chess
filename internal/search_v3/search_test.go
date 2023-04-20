package searchv3

import (
	"fmt"
	"testing"

	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/search"
	"github.com/stretchr/testify/assert"
)

func TestOpening(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	fmt.Println(result)

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true, "g1f3": true, "b1c3": true}
	assert.True(t, expectedOpenings[result.Value().String()])
}

func TestPointlessSacrifice(t *testing.T) {
	fen := "rnbqkbnr/ppp2ppp/8/3pp3/4P3/3P1N2/PPP2PPP/RNBQKB1R b KQkq - 5 3"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	fmt.Println(result.Value().String())

	assert.NotEqual(t, "c8f5", result.Value().String())
}

func TestNoLegalMoves(t *testing.T) {
	fen := "rn1qkb1r/ppp3pp/5n2/3ppb2/8/2NP1NP1/PPP2PBP/R1BQK2R b KQkq - 13 7"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	fmt.Println(result.Value().String())

	assert.True(t, result.HasValue())
}

func TestCheckMateSearch(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8 b KQkq - 13 7"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	assert.False(t, result.HasValue(), result)
}

func TestCheckMateDetection(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8/8 b KQkq - 13 7"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	assert.False(t, result.HasValue(), result)

	game, err := game.GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	isCheckMate := search.PlayerIsInCheck(&game, &bitboards)
	assert.True(t, isCheckMate)
}

func TestCheckMateInOne(t *testing.T) {
	fen := "1K6/8/1b6/5k2/1p2p3/8/2q5/n7 b - - 2 2"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	assert.True(t, result.HasValue())

	checkMateMoves := map[string]bool{"c2c7": true, "c2c8": true}
	assert.True(t, checkMateMoves[result.Value().String()], result.Value().String())
}

func TestCheckMateInOne2(t *testing.T) {
	fen := "5b2/3kp2p/4r3/1p6/4n3/p3P1p1/3p1r2/6K1 b - - 1 46"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	assert.True(t, result.HasValue())

	checkMateMoves := map[string]bool{"d2d1q": true}
	assert.True(t, checkMateMoves[result.Value().String()], result.Value().String())
}

func TestQuiescence(t *testing.T) {
	fen := "r1bqk2r/p1p2ppp/1pnp1n2/4p3/1bPPP3/2N3P1/PP2NPBP/R1BQK2R b KQkq d3 0 7"

	_, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)
}

func TestCrash1(t *testing.T) {
	fen := "r1b1q3/2p2k2/pp3Q2/3pBn2/8/2N5/PP4PP/5R1K b - - 2 24"

	result, err := Search(fen, WithMaxDepth{3})
	assert.True(t, IsNil(err), err)

	assert.True(t, result.HasValue())
	assert.True(t, IsNil(err), err)
}
func TestCrash2(t *testing.T) {
	fen := "4qk1r/3R3p/5p1p/2Q1p3/p6K/6PP/8/8 b - - 9 38"

	result, err := Search(fen, WithMaxDepth{4})
	assert.True(t, IsNil(err), err)

	assert.True(t, result.HasValue())
	assert.True(t, IsNil(err), err)
}
func TestCrash3(t *testing.T) {
	fen := "rk1R1r2/pp4Q1/7p/4pN2/P1Pp4/3P4/2P3PP/R1n4K b - - 0 24"

	result, err := Search(fen, WithMaxDepth{4})
	assert.True(t, IsNil(err), err)

	assert.True(t, result.HasValue())
	assert.True(t, IsNil(err), err)
}
