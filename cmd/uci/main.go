package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/cricklet/chessgo/internal/chessgo"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/search"
	"github.com/cricklet/chessgo/internal/uci"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "recover()", r)
		}
	}()

	args := os.Args[1:]

	searchOptions, err := search.SearcherOptionsFromArgs(args...)
	if !IsNil(err) {
		panic(err)
	}

	r := uci.NewUciRunner(chessgo.NewChessGoRunner(
		chessgo.WithSearchOptions(searchOptions),
		chessgo.WithLogger(FuncLogger(
			func(s string) {
				fmt.Print(s)
			})),
	))

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
