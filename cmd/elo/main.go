package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprint(r))
			fmt.Fprintln(os.Stderr, string(debug.Stack()))
			// time.Sleep(60 * time.Second)
			// main()
		}
	}()

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "compareStockfish" {
		CompareStockfishMain(args[1:])
	} else {
		CompareChessGo(args)
	}
}
