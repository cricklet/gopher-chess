package search

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestOpening(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err))
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards)

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 2000)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.True(t, IsNil(err))
	assert.Empty(t, errs)

	err = Wrap(os.WriteFile(RootDir()+"/data/debug-search-openings.tree", []byte(searcher.DebugTree.Sprint(10)), 0600))
	assert.True(t, IsNil(err))
	fmt.Println(searcher.DebugTree.Sprint(1))

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true}
	assert.True(t, expectedOpenings[result.Value().String()])
}

func TestPointlessSacrifice(t *testing.T) {
	fen := "rnbqkbnr/ppp2ppp/8/3pp3/4P3/3P1N2/PPP2PPP/RNBQKB1R b KQkq - 5 3"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err))
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards)

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 2000)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.True(t, IsNil(err))

	fmt.Println(result.Value().String())
	fmt.Println(game.Board.String())

	assert.NotEqual(t, "c8f5", result.Value().String())

	err = Wrap(os.WriteFile(RootDir()+"/data/debug-search-pointless-sacrifice.tree", []byte(searcher.DebugTree.Sprint(4)), 0600))
	assert.True(t, IsNil(err))
	fmt.Println(searcher.DebugTree.Sprint(1))
}

func TestNoLegalMoves(t *testing.T) {
	fen := "rn1qkb1r/ppp3pp/5n2/3ppb2/8/2NP1NP1/PPP2PBP/R1BQK2R b KQkq - 13 7"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err))
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards)

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 10000)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.True(t, IsNil(err))

	fmt.Println(result.Value().String())
	fmt.Println(game.Board.String())

	assert.True(t, result.HasValue())
	err = Wrap(os.WriteFile(RootDir()+"/data/debug-no-legal-move.tree", []byte(searcher.DebugTree.Sprint(4)), 0600))
	assert.True(t, IsNil(err))
	fmt.Println(searcher.DebugTree.Sprint(1))
}

func TestCheckMateSearch(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8/8 b KQkq - 13 7"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err))
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&SilentLogger, &game, &bitboards)

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 100)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.False(t, result.HasValue())
}

func TestCheckMateDetection(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8/8 b KQkq - 13 7"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err))
	bitboards := game.CreateBitboards()

	var noValidMoves bool
	noValidMoves, err = NoValidMoves(&game, &bitboards)
	assert.True(t, IsNil(err))
	assert.True(t, noValidMoves)

	isCheckMate := noValidMoves && PlayerIsInCheck(&game, &bitboards)
	assert.True(t, isCheckMate)
}
