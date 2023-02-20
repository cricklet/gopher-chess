package search

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/fen"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestOpening(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	game, err := GamestateFromFenString(fen)
	assert.Nil(t, err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcher(&DefaultLogger, &game, &bitboards)

	var wg sync.WaitGroup
	wg.Add(1)

	var result Optional[Move]
	var errs []error

	go func() {
		result, errs = searcher.Search()
		wg.Done()
	}()

	go func() {
		time.Sleep(time.Second)
		searcher.OutOfTime = true
	}()

	wg.Wait()
	assert.Nil(t, err)
	assert.Empty(t, errs)

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true}

	assert.True(t, expectedOpenings[result.Value().String()])
}

func TestPointlessSacrifice(t *testing.T) {
	fen := "rnbqkbnr/ppp2ppp/8/3pp3/4P3/3P1N2/PPP2PPP/RNBQKB1R b KQkq - 5 3"
	game, err := GamestateFromFenString(fen)
	assert.Nil(t, err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcher(&DefaultLogger, &game, &bitboards)

	var wg sync.WaitGroup
	wg.Add(1)

	var result Optional[Move]
	var errs []error

	go func() {
		result, errs = searcher.Search()
		wg.Done()
	}()

	go func() {
		time.Sleep(time.Millisecond * 100)
		searcher.OutOfTime = true
	}()

	wg.Wait()
	assert.Empty(t, errs)
	assert.Nil(t, err)

	fmt.Println(result.Value().String())
	fmt.Println(game.Board.String())

	assert.Nil(t, err)

	err = os.WriteFile(RootDir()+"/data/debug-search-pointless-sacrifice.tree", []byte(searcher.DebugTree.Sprint(4)), 0600)
	assert.Nil(t, err)
	fmt.Println(searcher.DebugTree.Sprint(2))
}
