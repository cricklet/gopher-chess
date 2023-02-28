package runner

import (
	"fmt"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestIndexBug2(t *testing.T) {
	r := ChessGoRunner{}
	err := r.SetupPosition(Position{
		Fen:   "2kr3r/p1p2ppp/2n1b3/2bqp3/Pp1p4/1P1P1N1P/2PBBPP1/R2Q1RK1 w - - 24 13",
		Moves: []string{},
	})
	assert.Nil(t, err)

	err = r.PerformMoveFromString("g2g4")
	assert.Nil(t, err)
	move, err := r.Search()
	assert.Nil(t, err)
	assert.True(t, move.HasValue())
}

func TestIndexBug3(t *testing.T) {
	r := ChessGoRunner{}
	err := r.SetupPosition(Position{
		Fen:   "2k1r3/8/2np2p1/p1bq4/Pp2P1P1/1P1p4/2PBQ3/R4RK1 w - - 48 25",
		Moves: []string{},
	})
	assert.Nil(t, err)

	err = r.PerformMoveFromString("d2e3")
	assert.Nil(t, err)
	move, err := r.Search()
	assert.Nil(t, err)
	assert.True(t, move.HasValue())
}

func TestCastlingBug1(t *testing.T) {
	fen := "rn1qk2r/ppp3pp/3b1n2/3ppb2/8/2NPBNP1/PPP2PBP/R2QK2R b KQkq - 15 8"
	moves := []string{
		"e8g8",
		"d3d4",
	}

	{
		r := ChessGoRunner{}
		err := r.SetupPosition(Position{
			Fen:   fen,
			Moves: []string{},
		})
		assert.Nil(t, err)

		for _, m := range moves {
			err := r.PerformMoveFromString(m)
			assert.Nil(t, err)
		}

		kingMoves, err := r.MovesForSelection("g8")
		assert.Nil(t, err)

		for _, m := range kingMoves {
			assert.NotEqual(t, "g8f8", m)
		}
	}
	{
		r := ChessGoRunner{}
		err := r.SetupPosition(Position{
			Fen:   fen,
			Moves: moves,
		})
		assert.Nil(t, err)

		kingMoves, err := r.MovesForSelection("g8")
		assert.Nil(t, err)

		for _, m := range kingMoves {
			assert.NotEqual(t, "g8f8", m)
		}
	}
}

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
		r := StockfishRunner{}
		err := r.SetupPosition(Position{
			Fen:   fen,
			Moves: []string{},
		})
		assert.Nil(t, err)

		for _, m := range moves {
			err := r.PerformMoveFromString(m)
			assert.Nil(t, err)
		}

		move, err := r.Search()
		assert.Nil(t, err)
		assert.True(t, move.HasValue())
	}
}

func TestBattle(t *testing.T) {
	chessgo := ChessGoRunner{}
	stockfish := StockfishRunner{}

	// Setup both runners
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	startPosition := Position{
		Fen:   fen,
		Moves: []string{},
	}

	err := chessgo.SetupPosition(startPosition)
	assert.Nil(t, err)
	err = stockfish.SetupPosition(startPosition)
	assert.Nil(t, err)

	for i := 0; i < 2; i++ {
		var err error
		var move Optional[string]

		move, err = chessgo.Search()
		assert.Nil(t, err)
		assert.True(t, move.HasValue())

		fmt.Println("> chessgo: ", move)

		err = chessgo.PerformMoveFromString(move.Value())
		assert.Nil(t, err)
		err = stockfish.PerformMoveFromString(move.Value())
		assert.Nil(t, err)

		move, err = stockfish.Search()
		assert.Nil(t, err)
		assert.True(t, move.HasValue())

		fmt.Println("> stockfish: ", move)

		err = chessgo.PerformMoveFromString(move.Value())
		assert.Nil(t, err)
		err = stockfish.PerformMoveFromString(move.Value())
		assert.Nil(t, err)
	}

	assert.Equal(t, 4, len(chessgo.history))
}
