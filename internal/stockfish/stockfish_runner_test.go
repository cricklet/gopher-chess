package stockfish

import (
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/search"
	"github.com/stretchr/testify/assert"
)

type UciIteration struct {
	Input string
	Wait  time.Duration

	ExpectedOutput       Optional[string]
	ExpectedOutputPrefix Optional[string]
}

func TestStockfish(t *testing.T) {
	fen := "rn1qk2r/ppp3pp/3b1n2/3ppb2/8/2NPBNP1/PPP2PBP/R2QK2R b KQkq - 15 8"
	moves := []string{
		"e8g8",
		"d3d4",
	}

	{
		r, err := NewStockfishRunner()
		assert.True(t, IsNil(err))

		err = r.SetupPosition(Position{
			Fen:   fen,
			Moves: []string{},
		})
		assert.True(t, IsNil(err))

		for _, m := range moves {
			err := r.PerformMoveFromString(m)
			assert.True(t, IsNil(err))
		}

		move, _, err := r.Search()
		assert.True(t, IsNil(err))
		assert.True(t, move.HasValue())
	}
}

func TestInfoMate(t *testing.T) {
	line := "info depth 31 seldepth 2 multipv 1 score mate 1 nodes 670 nps 670000 tbhits 0 time 1 pv a4e8	"
	move, score, err := MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "a4e8", move.Value())
	assert.Equal(t, search.ScoreString(score), "mate+1")

	line = "info depth 31 seldepth 2 multipv 1 score mate -1 nodes 670 nps 670000 tbhits 0 time 1 pv a4e8	"
	move, score, err = MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "a4e8", move.Value())
	assert.Equal(t, search.ScoreString(score), "mate-1")
}

func TestInfoScore(t *testing.T) {
	line := "info depth 1 seldepth 3 multipv 1 score cp 869 nodes 83 nps 83000 tbhits 0 time 1 pv a4e8 f7f6 e6f5 f6f5"
	move, score, err := MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "a4e8", move.Value())
	assert.Equal(t, score, 869)
}

func TestInfoMissingPv(t *testing.T) {
	line := "info depth 14 seldepth 16 multipv 1 score cp 133 nodes 46884 nps 390700 tbhits 0 time 120 pv b7e4 d3e4 c7c4 e2c4 c8c4 a4b6 d7b6 e4d3"
	move, score, err := MoveAndScoreFromInfoLine(line)
	assert.True(t, err.IsNil(), err)
	assert.Equal(t, "b7e4", move.Value())
	assert.Equal(t, score, 133)
}
