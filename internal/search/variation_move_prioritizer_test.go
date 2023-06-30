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

	unregister, gen := NewVariationMovePrioritizer(g)
	defer unregister()

	gen.resetSortedVariations(variations)

	assert.Equal(t, "VariationMovePrioritizer[[e2e4, e7e5, g1f3], [d2d4, d7d5]]", gen.String())

	{
		undo1 := BoardUpdate{}
		err = g.PerformMove(MoveFromString("e2e4", QuietMove), &undo1)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[e7e5, g1f3]", gen.String())

		undo2 := BoardUpdate{}
		err = g.PerformMove(MoveFromString("e7e5", QuietMove), &undo2)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[g1f3]", gen.String())

		undo3 := BoardUpdate{}
		err = g.PerformMove(MoveFromString("g1f3", QuietMove), &undo3)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[empty]", gen.String())

		err = g.UndoUpdate(&undo3)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[g1f3]", gen.String())

		err = g.UndoUpdate(&undo2)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[e7e5, g1f3]", gen.String())

		err = g.UndoUpdate(&undo1)
		assert.True(t, err.IsNil())
	}

	assert.Equal(t, "VariationMovePrioritizer[[e2e4, e7e5, g1f3], [d2d4, d7d5]]", gen.String())

	{
		undo1 := BoardUpdate{}
		err = g.PerformMove(MoveFromString("d2d4", QuietMove), &undo1)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[d7d5]", gen.String())

		undo2 := BoardUpdate{}
		err = g.PerformMove(MoveFromString("e7e6", QuietMove), &undo2)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[empty]", gen.String())

		err = g.UndoUpdate(&undo2)
		assert.True(t, err.IsNil())
		assert.Equal(t, "VariationMovePrioritizer[d7d5]", gen.String())

		err = g.UndoUpdate(&undo1)
		assert.True(t, err.IsNil())
	}

	assert.Equal(t, "VariationMovePrioritizer[[e2e4, e7e5, g1f3], [d2d4, d7d5]]", gen.String())
}
