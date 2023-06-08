package search

import (
	"strings"
	"testing"

	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestEvaluation(t *testing.T) {
	s := "4k3/2R5/8/7r/8/r7/3R4/4K3 b - - 10 5"
	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	bitboards := g.CreateBitboards()
	assert.Equal(t, strings.Join([]string{
		"    k   ",
		"  R     ",
		"        ",
		"       r",
		"        ",
		"r       ",
		"   R    ",
		"    K   ",
	}, "\n"), g.Board.String())

	assert.Equal(t, EvaluateDevelopment(bitboards, White), 2*_developmentScale)
	assert.Equal(t, EvaluateDevelopment(bitboards, Black), 0*_developmentScale)
}

func EvaluateFen(t *testing.T, s string, args ...EvaluationOption) int {
	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	bitboards := g.CreateBitboards()
	return Evaluate(bitboards, g.Player, args...)
}

func TestEvaluationEndgame(t *testing.T) {
	assert.Less(t,
		EvaluateFen(t, "8/8/4k3/8/8/8/7R/4K3 w - - 10 5"),
		EvaluateFen(t, "4k3/8/8/8/8/8/7R/4K3 w - - 10 5"))
}
