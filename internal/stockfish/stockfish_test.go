package stockfish

import (
	"testing"

	"github.com/cricklet/chessgo/internal/search"
	"github.com/stretchr/testify/assert"
)

func TestInfoMate(t *testing.T) {
	line := "info depth 31 seldepth 2 multipv 1 score mate 1 nodes 670 nps 670000 tbhits 0 time 1 pv a4e8	"
	move, score, err := MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "a4e8", move.Value())
	assert.Greater(t, score, search.Inf)

	line = "info depth 31 seldepth 2 multipv 1 score mate -1 nodes 670 nps 670000 tbhits 0 time 1 pv a4e8	"
	move, score, err = MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "a4e8", move.Value())
	assert.Less(t, score, -search.Inf)
}

func TestInfoScore(t *testing.T) {
	line := "info depth 1 seldepth 3 multipv 1 score cp 869 nodes 83 nps 83000 tbhits 0 time 1 pv a4e8 f7f6 e6f5 f6f5"
	move, score, err := MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "a4e8", move.Value())
	assert.Equal(t, score, 869)
}
