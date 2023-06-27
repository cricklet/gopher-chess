package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"

	. "github.com/cricklet/chessgo/internal/accuracy"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/stockfish"
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
		fmt.Println(" > accuracy chessgo <seconds> <epd>")
		fmt.Println(" > accuracy stockfish <seconds> <epd>")
		fmt.Println(" > accuracy cache <epd>")
		fmt.Println(" > accuracy clean")
		return
	}

	cachePath := RootDir() + "/data/epd_cache.json"
	cache := &[]EpdResult{}
	found, err := unmarshalEpdCache(cachePath, cache)
	if err.HasError() {
		panic(err)
	}

	logger := NewLiveLogger()
	logger.Println("found cache:", found)

	stock, err := stockfish.NewStockfishRunner(
		// stockfish.WithLogger(&SilentLogger),
		// stockfish.WithLogger(logger),
		stockfish.WithLogger(NewFooterLogger(logger, 0)),
	)
	if err.HasError() {
		panic(err)
	}

	if args[0] == "clean" {
		err = Wrap(os.Remove(RootDir() + "/data/epd_cache.json"))
		if err.HasError() {
			fmt.Println("no cache to clean")
		}
	} else if args[0] == "cache" {
		if len(args) < 2 {
			fmt.Println("usage: accuracy cache <epd>")
			return
		}

		priorSuccess := map[string]bool{}
		for _, result := range *cache {
			priorSuccess[result.Epd] = result.StockfishSuccess
		}

		epdName := args[1]
		epdPath := RootDir() + "/internal/accuracy/" + epdName + ".epd"

		epds, err := LoadEpd(epdPath)

		for i, epd := range epds {
			prefix := fmt.Sprintf("%d/%d", i+1, len(epds))

			if prior, ok := priorSuccess[epd]; ok {
				if prior {
					logger.Println(prefix, "skipping", epd)
					continue
				} else {
					prefix += " (retry)"
				}
			}

			result := CalculateEpdResult(stock, logger, epd)
			*cache = append(*cache, result)

			if result.StockfishSuccess {
				logger.Println(prefix, "success", epd)
			} else {
				logger.Println(prefix, "failure", epd)
			}

			err = marshalEpdCache(cachePath, cache)
			if err.HasError() {
				panic(err)
			}
		}

	}
	// else {
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
	// }
}
