package chessgo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleBoards(t *testing.T) {
	b := BoardArray{
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
	}

	assert.Equal(t, b.string(), strings.Join([]string{
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        ",
		"        \n",
	}, "\n"))
}
