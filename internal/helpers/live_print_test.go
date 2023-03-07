package helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrintLive(t *testing.T) {
	fmt.Print("\033[B")
	fmt.Print("\033[B")
	PrintLive(Some("1\n"), "", "a")
	time.Sleep(500 * time.Millisecond)
	PrintLive(Some("2\n"), "a", "ab")
	time.Sleep(500 * time.Millisecond)
	PrintLive(Some("3\n"), "ab", "abc")
}

func TestLiveLogger(t *testing.T) {
	l := NewLiveLogger()
	l.SetFooter("a")
	l.Println("1")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("ab")
	l.Println("12")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("abc")
	l.Println("123")
}

func TestLiveLogger2(t *testing.T) {
	l := NewLiveLogger()
	l.SetFooter("a")
	l.Println("1")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("a\nb")
	l.Println("12")
	time.Sleep(500 * time.Millisecond)
	l.SetFooter("a\nb\nc")
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
