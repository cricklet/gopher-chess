package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchTreeFromLines(t *testing.T) {
	lines := [][]string{
		{"e4", "e5"},
	}

	{
		tree, err := SearchTreeFromLines(
			lines,
			true,
		)

		assert.True(t, err.IsNil())
		assert.Equal(t, tree.String(), "SearchTree[e4: SearchTree[e5: SearchTree[continue...]]]")
	}

	{
		tree, err := SearchTreeFromLines(
			lines,
			false,
		)

		assert.True(t, err.IsNil())
		assert.Equal(t, tree.String(), "SearchTree[e4: SearchTree[e5: SearchTree[]]]")
	}
}
