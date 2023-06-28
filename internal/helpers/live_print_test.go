package helpers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLiveLogger(t *testing.T) {
	l := NewLiveLogger()
	l.SetFooter("a", 0)
	l.Println("1")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("ab", 0)
	l.Println("12")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("abc", 0)
	l.Println("123")
}

func TestLiveLogger2(t *testing.T) {
	l := NewLiveLogger()
	l.SetFooter("a", 0)
	l.Println("1")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("a\nb", 0)
	l.Println("12")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("a\nb\nc", 0)
	l.Println("123")
}

func TestWrapText(t *testing.T) {
	s := "asdf asdf asdf"
	assert.Equal(t,
		"asdf\nasdf\nasdf",
		wrapLine(s, 3),
	)

	board := BoardArray{
		WR, WN, WB, WQ, WK, WB, WN, WR,
		WP, WP, WP, WP, WP, WP, WP, WP,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		XX, XX, XX, XX, XX, XX, XX, XX,
		BP, BP, BP, BP, BP, BP, BP, BP,
		BR, BN, BB, BQ, BK, BB, BN, BR,
	}
	assert.Equal(t, board.Unicode(), wrapText(board.Unicode(), 80, ""))
}
