package search

import (
	"testing"

	"github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestBoardPrint(t *testing.T) {
	assert.Equal(t, helpers.UnwrapReturn(MateInNScore(1)), Inf-1)
	assert.Equal(t, helpers.UnwrapReturn(MateInNScore(2)), Inf-2)
	assert.Equal(t, helpers.UnwrapReturn(MateInNScore(-1)), -Inf+1)
	assert.Equal(t, helpers.UnwrapReturn(MateInNScore(-2)), -Inf+2)

	assert.True(t, IsMate(helpers.UnwrapReturn(MateInNScore(1))))
	assert.True(t, IsMate(helpers.UnwrapReturn(MateInNScore(2))))
	assert.True(t, IsMate(helpers.UnwrapReturn(MateInNScore(-1))))
	assert.True(t, IsMate(helpers.UnwrapReturn(MateInNScore(-2))))

	assert.Equal(t, ScoreString(helpers.UnwrapReturn(MateInNScore(1))), "mate+1")
	assert.Equal(t, ScoreString(helpers.UnwrapReturn(MateInNScore(2))), "mate+2")
	assert.Equal(t, ScoreString(helpers.UnwrapReturn(MateInNScore(-1))), "mate-1")
	assert.Equal(t, ScoreString(helpers.UnwrapReturn(MateInNScore(-2))), "mate-2")
}
