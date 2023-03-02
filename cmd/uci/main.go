package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/runner"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "recover()", r)
		}
	}()

	r := runner.UciRunner{
		Runner: &runner.ChessGoRunner{},
	}

	scanner := bufio.NewScanner(os.Stdin)

	done := false
	for !done && scanner.Scan() {
		input := scanner.Text()
		if input == "quit" {
			break
		}
		result, err := r.HandleInput(input)
		if !IsNil(err) {
			fmt.Fprintln(os.Stderr, "error:", err)
			time.Sleep(200 * time.Millisecond)
			break
		}
		for _, v := range result {
			fmt.Println(v)
		}
	}
}
