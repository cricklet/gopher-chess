package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexBug1(t *testing.T) {
	r := Runner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen rnbqkb1r/ppp2ppp/5n2/3pp3/4P3/2NP1N2/PPP1BPPP/R1BQK2R w KQkq - 10 6",
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}
	err := r.PerformMoveFromString("e1g1")
	assert.Nil(t, err)
	_, err = r.HandleInput("go")
	assert.Nil(t, err)
}

func TestIndexBug2(t *testing.T) {
	r := Runner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen 2kr3r/p1p2ppp/2n1b3/2bqp3/Pp1p4/1P1P1N1P/2PBBPP1/R2Q1RK1 w - - 24 13",
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}

	err := r.PerformMoveFromString("g2g4")
	assert.Nil(t, err)
	_, err = r.HandleInput("go")
	assert.Nil(t, err)
}

func TestIndexBug3(t *testing.T) {
	r := Runner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen 2k1r3/8/2np2p1/p1bq4/Pp2P1P1/1P1p4/2PBQ3/R4RK1 w - - 48 25",
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}

	err := r.PerformMoveFromString("d2e3")
	assert.Nil(t, err)
	_, err = r.HandleInput("go")
	assert.Nil(t, err)
}

func TestCastlingBug1(t *testing.T) {
	fen := "rn1qk2r/ppp3pp/3b1n2/3ppb2/8/2NPBNP1/PPP2PBP/R2QK2R b KQkq - 15 8"
	moves := []string{
		"e8g8",
		"d3d4",
	}
	r := Runner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen " + fen,
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}

	for _, m := range moves {
		err := r.PerformMoveFromString(m)
		assert.Nil(t, err)
	}

	kingMoves, err := r.MovesForSelection("g8")
	assert.Nil(t, err)

	for _, m := range kingMoves {
		assert.NotEqual(t, "f8", m.String())
	}
}
