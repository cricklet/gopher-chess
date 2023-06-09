package search

import (
	"testing"

	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestVariationMovePrioritizer(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w"
	g, err := game.GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)

	b := g.CreateBitboards()
	assert.True(t, IsNil(err), err)

	variations := [][]SearchMove{
		{
			{Move: MoveFromString("e2e4", QuietMove)},
			{Move: MoveFromString("e7e5", QuietMove)},
			{Move: MoveFromString("g1f3", QuietMove)},
		},
		{
			{Move: MoveFromString("d2d4", QuietMove)},
			{Move: MoveFromString("d7d5", QuietMove)},
		},
	}

	unregister, gen := NewVariationMovePrioritizer(g, variations)
	defer unregister()

	assert.Equal(t, "VariationMovePrioritizer[[e2e4, e7e5, g1f3], [d2d4, d7d5]]", gen.String())

	{
		undo1 := BoardUpdate{}
		g.PerformMove(MoveFromString("e2e4", QuietMove), &undo1, b)
		assert.Equal(t, "VariationMovePrioritizer[e7e5, g1f3]", gen.String())

		undo2 := BoardUpdate{}
		g.PerformMove(MoveFromString("e7e5", QuietMove), &undo2, b)
		assert.Equal(t, "VariationMovePrioritizer[g1f3]", gen.String())

		undo3 := BoardUpdate{}
		g.PerformMove(MoveFromString("g1f3", QuietMove), &undo3, b)
		assert.Equal(t, "VariationMovePrioritizer[empty]", gen.String())

		g.UndoUpdate(&undo3, b)
		assert.Equal(t, "VariationMovePrioritizer[g1f3]", gen.String())

		g.UndoUpdate(&undo2, b)
		assert.Equal(t, "VariationMovePrioritizer[e7e5, g1f3]", gen.String())

		g.UndoUpdate(&undo1, b)
	}

	assert.Equal(t, "VariationMovePrioritizer[[e2e4, e7e5, g1f3], [d2d4, d7d5]]", gen.String())

	{
		undo1 := BoardUpdate{}
		g.PerformMove(MoveFromString("d2d4", QuietMove), &undo1, b)
		assert.Equal(t, "VariationMovePrioritizer[d7d5]", gen.String())

		undo2 := BoardUpdate{}
		g.PerformMove(MoveFromString("e7e6", QuietMove), &undo2, b)
		assert.Equal(t, "VariationMovePrioritizer[empty]", gen.String())

		g.UndoUpdate(&undo2, b)
		assert.Equal(t, "VariationMovePrioritizer[d7d5]", gen.String())

		g.UndoUpdate(&undo1, b)
	}

	assert.Equal(t, "VariationMovePrioritizer[[e2e4, e7e5, g1f3], [d2d4, d7d5]]", gen.String())
}
