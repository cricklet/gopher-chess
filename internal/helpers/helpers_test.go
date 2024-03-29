package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlice(t *testing.T) {
	a := make([]int, 0, 5)
	b := append(a[:0], 1, 2, 3, 4)
	c := append(a[:0], 4, 5, 6)

	assert.Equal(t, []int{}, a)
	assert.Equal(t, []int{4, 5, 6, 4}, b)
	assert.Equal(t, []int{4, 5, 6}, c)
}

func TestPrintColumns(t *testing.T) {
	result := PrintColumns(
		[]string{"012", "01234", "0"},
		[]int{4, 10, 4},
		"",
	)
	assert.Equal(t, "012 01234     0   ", result)
}
