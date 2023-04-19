package searchv3

import (
	"fmt"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestOpening(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	outOfTime := false
	Search(fen, &outOfTime)

	go func() {
		time.Sleep(time.Millisecond * 200)
		outOfTime = true
	}()

	result, err := Search(fen, &outOfTime)
	assert.True(t, IsNil(err), err)

	fmt.Println(result)

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true, "g1f3": true, "b1c3": true}
	assert.True(t, expectedOpenings[result.Value().String()])
}
