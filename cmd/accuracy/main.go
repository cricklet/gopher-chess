package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	. "github.com/cricklet/chessgo/internal/accuracy"
	. "github.com/cricklet/chessgo/internal/helpers"
)

func unmarshalEpdCache(jsonPath string, results *[]EpdResult) (bool, Error) {
	_, err := os.Stat(jsonPath)
	if !IsNil(err) {
		return false, NilError
	}
	input, err := os.ReadFile(jsonPath)
	if !IsNil(err) {
		return false, Wrap(err)
	}
	err = json.Unmarshal(input, results)
	if !IsNil(err) {
		return false, Wrap(err)
	}

	return true, NilError
}

func marshalEpdCache(jsonPath string, results *[]EpdResult) Error {
	output, err := json.MarshalIndent(results, "", "  ")
	if !IsNil(err) {
		return Wrap(err)
	}
	err = os.WriteFile(jsonPath, output, 0644)
	return Wrap(err)
}
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
	if len(args) == 0 {
		fmt.Println("usage:")
		fmt.Println(" > accuracy chessgo <seconds>")
		fmt.Println(" > accuracy stockfish <seconds>")
		fmt.Println(" > accuracy cache")
		return
	}

	cache := &[]EpdResult{}
	found, err := unmarshalEpdCache(RootDir()+"/data/epd_cache.json", cache)
	if err.HasError() {
		panic(err)
	}

	fmt.Println("found cache:", found)

	if args[0] == "cache" {
		duration := time.Millisecond * 100
		if len(args) > 1 {
			secondsString := args[1]
			seconds, err := WrapReturn(strconv.Atoi(secondsString))
			if err.HasError() {
				panic(err)
			}
			duration = time.Second * time.Duration(seconds)
			fmt.Println("using duration:", duration)
		}

		ComputeResults(cache, duration, func() {
			err = marshalEpdCache(RootDir()+"/data/epd_cache.json", cache)
			if err.HasError() {
				panic(err)
			}
		})
	} else {
		// var runner Runner
		// if args[0] == "chessgo" {
		// 	r := chessgo.NewChessGoRunner()
		// 	runner = &r
		// } else if args[0] == "stockfish" {
		// 	runner = stockfish.NewStockfishRunner()
		// }

		// secondsString := args[1]
		// seconds, err := strconv.Atoi(secondsString)
		// if err.HasError() {
		// 	panic(err)
		// }
	}
}
