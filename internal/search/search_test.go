package search

import (
	"encoding/json"
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
	outOfTime := false

	var wg sync.WaitGroup
	wg.Add(1)

	var result Optional[Move]

	go func() {
		result, err = searcher.Search(&outOfTime)
		wg.Done()
	}()

	go func() {
		time.Sleep(time.Second)
		outOfTime = true
	}()

	wg.Wait()
	assert.Nil(t, err)

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true}

	assert.True(t, expectedOpenings[result.Value().String()])
}

func TestPointlessSacrifice(t *testing.T) {
	fen := "rnbqkbnr/ppp2ppp/8/3pp3/4P3/3P1N2/PPP2PPP/RNBQKB1R b KQkq - 5 3"
	game, err := GamestateFromFenString(fen)
	assert.Nil(t, err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcher(&DefaultLogger, &game, &bitboards)
	outOfTime := false

	var wg sync.WaitGroup
	wg.Add(1)

	var result Optional[Move]

	go func() {
		result, err = searcher.Search(&outOfTime)
		wg.Done()
	}()

	go func() {
		time.Sleep(time.Second)
		outOfTime = true
	}()

	wg.Wait()
	assert.Nil(t, err)

	fmt.Println(result.Value().String())

	output, err := json.MarshalIndent(searcher.DebugTreeRoot, "", " ")
	assert.Nil(t, err)
	err = os.WriteFile(RootDir()+"/data/debug-search-pointless-sacrifice.json", output, 0600)
	assert.Nil(t, err)
}
