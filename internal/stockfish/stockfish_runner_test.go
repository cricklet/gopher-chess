package stockfish

import (
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
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
		r := NewStockfishRunner()
		err := r.SetupPosition(Position{
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
