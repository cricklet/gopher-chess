package main

import (
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "compareStockfish" {
		CompareStockfishMain(args[1:])
	} else {
		CompareChessGo(args)
	}
}
