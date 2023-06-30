package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/cricklet/chessgo/internal/accuracy"
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
	output, err := WrapReturn(json.MarshalIndent(results, "", "  "))
	if !IsNil(err) {
		return err
	}
	err = Wrap(os.WriteFile(jsonPath, output, 0644))
	return err
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

	} else if args[0] == "cache" {
		if len(args) < 2 {
			fmt.Println("usage: accuracy cache <epds>")
			return
		}

		epdResultMap := map[string]EpdResult{}
		for _, result := range *cache {
			epdResultMap[result.Epd] = result
		}

		epdsNames := args[1:]

		updateCacheHelper(epdsNames, epdResultMap, logger, func(result EpdResult) {
			*cache = append(*cache, result)

			err = marshalEpdCache(cachePath, cache)
			if err.HasError() {
				panic(err)
			}
		})

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
			if epdResult.StockfishResult != EpdCacheResultSuccess {
				continue
			}
			testEpds = append(testEpds, epdResult)
		}

		logger.Println("test epds:", len(testEpds))

		var runner Runner
		if args[0] == "chessgo" {
			// NEXT: make it easier to pass in options here
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

		checkRunner(runner, testEpds, logger)
	}
}

func updateCacheHelper(
	epdsNames []string,
	epdResultMap map[string]EpdResult,
	logger *LiveLogger,
	callback func(EpdResult),
) {
	stock, err := stockfish.NewStockfishRunner(
		// stockfish.WithLogger(&SilentLogger),
		// stockfish.WithLogger(logger),
		stockfish.WithLogger(NewFooterLogger(logger, 0)),
	)
	defer stock.Close()

	if err.HasError() {
		panic(err)
	}

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
				switch r.StockfishResult {
				case EpdCacheResultSuccess:
					logger.Println(prefix, "cached success w/ depth", r.StockfishDepth, epdStr)
					continue
				case EpdCacheResultFailure:
					logger.Println(prefix, "cached failure w/ depth", r.StockfishDepth, epdStr)
					continue
				case EpdCacheResultAmbiguous:
					logger.Println(prefix, "cached ambiguous w/ depth", r.StockfishDepth, epdStr)
					continue
				}
			}

			logger.Println(prefix, "calculating", epdStr)

			result := CalculateEpdResult(stock, logger, epd)
			callback(result)

			switch result.StockfishResult {
			case EpdCacheResultSuccess:
				logger.Println(prefix, "success w/ depth", result.StockfishDepth, epdStr)
			case EpdCacheResultFailure:
				logger.Println(prefix, "failure w/ depth", result.StockfishDepth, epdStr)
			case EpdCacheResultAmbiguous:
				logger.Println(prefix, "ambiguous w/ depth", result.StockfishDepth, epdStr)
			}
		}
	}
}

func checkRunner(runner Runner, testEpds []accuracy.EpdResult, logger Logger) {
	for i, epdResult := range testEpds {

		move, _, depth, err := accuracy.SearchEpd(runner, epdResult.Epd)
		if err.HasError() {
			panic(err)
		}

		runnerScore := epdResult.StockfishScores[move]
		bestScore := epdResult.StockfishScores[epdResult.StockfishMove]

		acc := accuracy.AccuracyForScores(runnerScore, bestScore)

		prefix := fmt.Sprintf("%2d/%d depth cached %2v, ", i+1, len(testEpds), epdResult.StockfishDepth)
		prefix += fmt.Sprintf("searched %v: %5.1f (%4v) %6v vs %6v", depth, acc, move, ScoreString(runnerScore), ScoreString(bestScore))

		logger.Println(prefix, epdResult.Epd)
	}
}
