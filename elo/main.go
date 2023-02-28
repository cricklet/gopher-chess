package main

import (
	"os"
	"time"

	. "github.com/cricklet/chessgo/internal/binary_runner"
	. "github.com/cricklet/chessgo/internal/helpers"
)

func main() {
	arg := os.Args[1]

	var stockfish *BinaryRunner
	var opponent *BinaryRunner
	var err Error

	stockfish, err = SetupBinaryRunner("stockfish", time.Millisecond*100)
	if !IsNil(err) {
		panic(err)
	}
	defer stockfish.Close()

	opponent, err = SetupBinaryRunner(arg, time.Millisecond*100)
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()
}
