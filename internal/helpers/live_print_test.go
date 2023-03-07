package helpers

import (
	"fmt"
	"testing"
	"time"
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
