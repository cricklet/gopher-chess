package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"

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
		fmt.Println(" > accuracy clean")
		return
	}

	cachePath := RootDir() + "/data/epd_cache.json"
	cache := &[]EpdResult{}
	found, err := unmarshalEpdCache(cachePath, cache)
	if err.HasError() {
		panic(err)
	}

	fmt.Println("found cache:", found)

	if args[0] == "clean" {
		err = Wrap(os.Remove(RootDir() + "/data/epd_cache.json"))
		if err.HasError() {
			panic(err)
		}
	} else if args[0] == "cache" {
		priorSuccess := map[string]bool{}
		for _, result := range *cache {
			priorSuccess[result.Epd] = result.StockfishSuccess
		}

		for i, epd := range EigenmannRapidEpds {
			prefix := fmt.Sprintf("%d/%d", i, len(EigenmannRapidEpds))

			if prior, ok := priorSuccess[epd]; ok {
				if prior {
					fmt.Println(prefix, "skipping", epd)
					continue
				} else {
					prefix += " (retry)"
				}
			}

			result := CalculateEpdResult(epd)
			*cache = append(*cache, result)

			if result.StockfishSuccess {
				fmt.Println(prefix, "success", epd)
			} else {
				fmt.Println(prefix, "failure", epd)
			}

			err = marshalEpdCache(cachePath, cache)
			if err.HasError() {
				panic(err)
			}
		}

	} else {
		// duration := time.Millisecond * 100
		// if len(args) > 1 {
		// 	secondsString := args[1]
		// 	seconds, err := WrapReturn(strconv.Atoi(secondsString))
		// 	if err.HasError() {
		// 		panic(err)
		// 	}
		// 	duration = time.Second * time.Duration(seconds)
		// 	fmt.Println("using duration:", duration)
		// }

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
