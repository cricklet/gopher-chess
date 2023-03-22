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
	"github.com/pkg/profile"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "recover()", r)
		}
	}()

	args := os.Args[1:]

	if Contains(args, "profile") {
		profilePath := RootDir() + "/data/CmdUciMain"
		p := profile.Start(profile.ProfilePath(profilePath))
		defer p.Stop()
	}
	args = FilterSlice(args, func(arg string) bool {
		return arg != "profile"
	})

	if len(args) > 0 && args[0] == "options" {
		for _, option := range search.AllSearchOptions {
			fmt.Println(option)
		}
		return
	}

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
