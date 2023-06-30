package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	. "github.com/cricklet/chessgo/internal/accuracy"
	"github.com/cricklet/chessgo/internal/chessgo"
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
		fmt.Println(" > accuracy chessgo <depth>")
		fmt.Println(" > accuracy stockfish <depth>")
		fmt.Println(" > accuracy cache <epds>")
		fmt.Println(" > accuracy try <epd>")
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

	if err.HasError() {
		panic(err)
	}

	if args[0] == "clean" {
		err = Wrap(os.Remove(RootDir() + "/data/epd_cache.json"))
		if err.HasError() {
			fmt.Println("no cache to clean")
		}
	} else if args[0] == "try" {
		if len(args) < 2 {
			fmt.Println("usage: accuracy cache-specific <epd>")
			return
		}

		stock, err := stockfish.NewStockfishRunner(
			// stockfish.WithLogger(&SilentLogger),
			// stockfish.WithLogger(logger),
			stockfish.WithLogger(NewFooterLogger(logger, 0)),
		)
		if err.HasError() {
			panic(err)
		}

		epd := args[1]
		logger.Println("epd:", epd)

		result := CalculateEpdResult(stock, logger, epd)
		*cache = append(*cache, result)

		stock.Close()

		logger.Println(result)

		// err = marshalEpdCache(cachePath, cache)
		// if err.HasError() {
		// 	panic(err)
		// }
	} else if args[0] == "cache" {
		if len(args) < 2 {
			fmt.Println("usage: accuracy cache <epds>")
			return
		}

		stock, err := stockfish.NewStockfishRunner(
			// stockfish.WithLogger(&SilentLogger),
			// stockfish.WithLogger(logger),
			stockfish.WithLogger(NewFooterLogger(logger, 0)),
		)
		defer stock.Close()

		if err.HasError() {
			panic(err)
		}

		epdResultMap := map[string]EpdResult{}
		for _, result := range *cache {
			epdResultMap[result.Epd] = result
		}

		epdsNames := args[1:]

		for _, epdName := range epdsNames {
			epdsPath := RootDir() + "/internal/accuracy/" + epdName + ".epd"

			epds, err := LoadEpd(epdsPath)
			if err.HasError() {
				panic(err)
			}

			for i, epd := range epds {
				prefix := fmt.Sprintf("%d/%d %v", i+1, len(epds), epdName)

				epdStr := fmt.Sprintf("\"%s\"", strings.ReplaceAll(epd, "\"", "\\\""))

				if r, ok := epdResultMap[epd]; ok {
					if r.StockfishSuccess {
						if r.StockfishScoreUncertainty {
							logger.Println(prefix, "cached ambiguous w/ depth", r.StockfishDepth, epdStr)
							continue
						} else {
							logger.Println(prefix, "cached success w/ depth", r.StockfishDepth, epdStr)
							continue
						}
					} else {
						logger.Println(prefix, "cached failure w/ depth", r.StockfishDepth, epdStr)
						continue
					}
				}

				logger.Println(prefix, "calculating", epdStr)

				result := CalculateEpdResult(stock, logger, epd)
				*cache = append(*cache, result)

				if result.StockfishSuccess {
					if result.StockfishScoreUncertainty {
						logger.Println(prefix, "ambiguous w/ depth", result.StockfishDepth, epdStr)
					} else {
						logger.Println(prefix, "success w/ depth", result.StockfishDepth, epdStr)
					}
				} else {
					logger.Println(prefix, "failure w/ depth", result.StockfishDepth, epdStr)
				}

				err = marshalEpdCache(cachePath, cache)
				if err.HasError() {
					panic(err)
				}
			}
		}
	} else if args[0] == "chessgo" || args[0] == "stockfish" {
		if len(args) < 2 {
			fmt.Println("usage: accuracy chessgo <depth>")
			return
		}

		depthStr := args[1]
		depth, err := WrapReturn(strconv.Atoi(depthStr))
		if err.HasError() {
			panic(err)
		}

		testEpds := []EpdResult{}

		for _, epdResult := range *cache {
			if epdResult.StockfishDepth > depth {
				continue
			}
			if !epdResult.StockfishSuccess || !epdResult.StockfishScoreUncertainty {
				continue
			}
			testEpds = append(testEpds, epdResult)
		}

		var runner Runner
		if args[0] == "chessgo" {
			r := chessgo.NewChessGoRunner(
				chessgo.WithLogger(&SilentLogger),
			)
			runner = &r
		} else if args[0] == "stockfish" {
			r, err := stockfish.NewStockfishRunner(
				stockfish.WithLogger(&SilentLogger),
			)
			defer r.Close()
			if err.HasError() {
				panic(err)
			}
			runner = r
		}

		successes := 0

		for i, epdResult := range testEpds {
			prefix := fmt.Sprintf("%d/%d (depth %v)", i+1, len(testEpds), epdResult.StockfishDepth)

			move, success, err := SearchEpd(runner, epdResult.Epd)
			if err.HasError() {
				panic(err)
			}

			runnerScore := epdResult.StockfishScores[move]
			bestScore := epdResult.StockfishScores[epdResult.StockfishMove]

			if success {
				successes++
				prefix += " success "
			} else {
				prefix += " failure "
			}
			prefix += fmt.Sprint(move, " ", runnerScore, " vs ideal ", bestScore)

			suffix := fmt.Sprintf("%d successes", successes)

			logger.Println(prefix, epdResult.Epd, suffix)
		}
	}
}
