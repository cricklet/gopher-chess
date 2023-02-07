package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime/debug"

	chessgo "github.com/cricklet/chessgo/chess"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(string(debug.Stack()))
		}
	}()

	r := chessgo.Runner{}

	scanner := bufio.NewScanner(os.Stdin)
	done := false
	for !done && scanner.Scan() {
		input := scanner.Text()
		done = r.HandleInputAndReturnDone(input)
	}
}
