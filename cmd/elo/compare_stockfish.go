package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cricklet/chessgo/internal/binary"
	. "github.com/cricklet/chessgo/internal/chessgo"
	. "github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/search"
	elo "github.com/kortemy/elo-go"
	combinations "github.com/mxschmitt/golang-combinations"
)

const _footerEval = 1
const _footerCurrent = 2
const _footerBoard = 4
const _footerPgn = 6
const _footerHistory = 8

var logger = NewLiveLogger()

type stockfishMatchResult struct {
	Won          bool
	Draw         bool
	Unknown      bool
	StartFen     string
	PositionFen  string
	EndingFen    string
	PgnMoves     string
	StockfishElo int
}
type stockfishEloResults struct {
	Cmd         string
	Matches     []stockfishMatchResult
	EloEstimate int
}

func (r stockfishEloResults) statsString() string {
	cmdName := Last(strings.Split(r.Cmd, "/"))
	wins := 0
	losses := 0
	draws := 0

	for _, match := range r.Matches {
		if match.Won {
			wins++
		} else if match.Draw {
			draws++
		} else if match.Unknown {
			draws++
		} else {
			losses++
		}
	}
	return fmt.Sprintf(" %4v %-60v (%2v %2v %2v) %2v ",
		r.computeElo(), cmdName[:MinInt(60, len(cmdName))], wins, draws, losses, len(r.Matches))
}

func (r stockfishEloResults) computeElo() int {
	rating := 800
	e := elo.NewElo()
	for _, match := range r.Matches {
		var result float64
		if match.Won {
			result = 1
		} else if match.Draw {
			result = 0.5
		} else if match.Unknown {
			result = 0.5
		} else {
			result = 0
		}
		outcome, _ := e.Outcome(rating, match.StockfishElo, result)
		rating = outcome.Rating
	}

	return rating
}

func (r stockfishEloResults) numMatches() int {
	return len(r.Matches)
}

func (r stockfishEloResults) matchHistory() string {
	if len(r.Matches) == 0 {
		return "<new>"
	}
	wins := []int{}
	losses := []int{}
	draws := []int{}
	for _, match := range r.Matches {
		if match.Won {
			wins = append(wins, match.StockfishElo)
		} else if match.Draw {
			draws = append(draws, match.StockfishElo)
		} else if match.Unknown {
			draws = append(draws, match.StockfishElo)
		} else {
			losses = append(losses, match.StockfishElo)
		}
	}

	sort.Ints(wins)
	sort.Ints(losses)
	sort.Ints(draws)
	return fmt.Sprintf("wins: %v\ndraws: %v\nlosses: %v", wins, draws, losses)
}

func unmarshalEloResults(path string, results *stockfishEloResults) Error {
	_, err := os.Stat(path)
	if !IsNil(err) {
		// It's fine if it doesn't exist
		return NilError
	}
	input, err := os.ReadFile(path)
	if !IsNil(err) {
		return Wrap(err)
	}
	err = json.Unmarshal(input, results)
	if !IsNil(err) {
		return Wrap(err)
	}

	return NilError
}

func marshalEloResults(path string, results *stockfishEloResults) Error {
	output, err := json.MarshalIndent(results, "", "  ")
	if !IsNil(err) {
		return Wrap(err)
	}
	err = os.WriteFile(path, output, 0600)
	return Wrap(err)
}

func findMoveInOutput(output []string) string {
	if len(output) == 0 {
		panic(Errorf("output was empty"))
	}
	bestMoveString := FindInSlice(output, func(v string) bool {
		return strings.HasPrefix(v, "bestmove ")
	})
	if bestMoveString.HasValue() {
		return strings.Split(bestMoveString.Value(), " ")[1]
	}
	panic(Errorf("couldn't find bestmove in output %v", output))
}

type stockfishResult int

const (
	StockfishWin stockfishResult = iota
	OpponentWin
	Draw
	Unknown
)

func playGame(
	stockfish *binary.BinaryRunner,
	stockfishElo int,
	opponent *binary.BinaryRunner,
	runner *ChessGoRunner,
) (stockfishResult, Error) {
	Run(stockfish, "isready", Some("readyok"))
	Run(stockfish, "uci", Some("uciok"))
	RunAsync(stockfish, "ucinewgame")
	RunAsync(stockfish, "setoption name UCI_LimitStrength value true")
	RunAsync(stockfish, fmt.Sprintf("setoption name UCI_Elo value %v", stockfishElo))
	RunAsync(stockfish, "position startpos")

	Run(opponent, "isready", Some("readyok"))
	Run(opponent, "uci", Some("uciok"))
	RunAsync(opponent, "ucinewgame")
	RunAsync(opponent, "position startpos")

	result, err := PlayBinaries(stockfish, opponent, runner)
	if !IsNil(err) {
		return Unknown, Wrap(err)
	}

	if result == 0 {
		return StockfishWin, NilError
	} else if result == 1 {
		return OpponentWin, NilError
	} else if result == 0.5 {
		return Draw, NilError
	} else {
		return Unknown, Errorf("unexpected result %v", result)
	}
}

func playGameBinaries(
	binaryPath string,
	binaryArgs []string,
	stockfishElo int,
) stockfishMatchResult {
	var err Error

	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	runner := NewChessGoRunner()
	err = runner.SetupPosition(Position{
		Fen:   fen,
		Moves: []string{},
	})
	if !IsNil(err) {
		panic(err)
	}

	opponentPlays := Black

	var stockfish *binary.BinaryRunner
	stockfishLogger := FuncLogger(func(s string) {
		if strings.Contains(s, "score cp ") {
			evalStr := strings.Split(
				strings.Split(s, "score cp ")[1], " ")[0]
			centipawnScore, err := WrapReturn(strconv.Atoi(evalStr))
			if !IsNil(err) {
				panic(err)
			}

			logger.SetFooter(HintText(fmt.Sprintf("eval: %v, piece: %v",
				-centipawnScore,
				runner.EvaluateSimple(opponentPlays))),
				_footerEval)
		}

		if strings.Contains(s, "info") {
			return
		}
		logger.Println("stockfish > " + Indent(s, "$ "))
	})
	stockfish, err = binary.SetupBinaryRunner("stockfish", "stockfish", []string{}, time.Millisecond*1000, binary.WithLogger(stockfishLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer stockfish.Close()

	var opponent *binary.BinaryRunner
	chessgoLogger := FuncLogger(func(s string) { logger.Println("chessgo > " + Indent(s, "$ ")) })
	opponent, err = binary.SetupBinaryRunner(binaryPath, "chessgo", binaryArgs, time.Millisecond*10000, binary.WithLogger(chessgoLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()

	var result stockfishResult
	result, err = playGame(stockfish, stockfishElo, opponent, &runner)
	if !IsNil(err) {
		panic(err)
	}

	newResult := stockfishMatchResult{
		StartFen:     runner.StartFen,
		PositionFen:  runner.StartFen + " moves " + strings.Join(runner.MoveHistory(), " "),
		EndingFen:    runner.FenString(),
		PgnMoves:     runner.PgnFromMoveHistory(),
		StockfishElo: stockfishElo,
	}

	switch result {
	case StockfishWin:
		newResult.Won = false
		logger.Println("stockfish won")
	case OpponentWin:
		newResult.Won = true
		logger.Println("opponent won")
	case Draw:
		newResult.Draw = true
		logger.Println("draw")
	case Unknown:
		newResult.Unknown = true
		logger.Println("wat")
	}

	RunAsync(stockfish, "quit")
	RunAsync(opponent, "quit")
	opponent.Wait()
	stockfish.Wait()

	return newResult
}

func allJsonFilesInDir(dir string) ([]string, Error) {
	filePaths := []string{}

	err := Wrap(filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			filePaths = append(filePaths, path)
		}
		return nil
	}))
	if !IsNil(err) {
		return filePaths, err
	}

	return filePaths, NilError
}

func allEloResultsInDir(dir string) ([]stockfishEloResults, Error) {
	filePaths, err := allJsonFilesInDir(dir)
	if !IsNil(err) {
		return nil, err
	}

	results := []stockfishEloResults{}
	for _, filePath := range filePaths {
		result := stockfishEloResults{}
		unmarshalEloResults(filePath, &result)
		results = append(results, result)
	}
	return results, NilError
}

func mainInner(shouldClean bool, binaryArgs []string, binaryPath string, jsonPath string) {
	var err Error
	if shouldClean {
		logger.Printf("cleaning %v\n     and %v\n", binaryPath, jsonPath)
		err = RmIfExists(binaryPath)
		if !IsNil(err) {
			panic(err)
		}
		err = RmIfExists(jsonPath)
		if !IsNil(err) {
			panic(err)
		}
		return
	}

	err = BuildChessGoIfMissing(binaryPath)
	if !IsNil(err) {
		panic(err)
	}

	results := stockfishEloResults{
		Cmd:     binaryPath,
		Matches: []stockfishMatchResult{},
	}

	err = unmarshalEloResults(jsonPath, &results)
	if !IsNil(err) {
		panic(err)
	}

	randomOffset := []int{-100, -50, 0, 50, 100}[rand.Intn(5)]
	if len(results.Matches) < 5 {
		randomOffset = []int{-50, 0, 50}[rand.Intn(3)]
	}
	stockfishElo := results.computeElo() + randomOffset

	currentSuffix := HintText(fmt.Sprintf(
		"stockfish elo: %v, chessgo elo: %v (%v)",
		stockfishElo,
		results.computeElo(),
		Last(strings.Split(binaryPath, "/"))))
	historySuffix := HintText(results.matchHistory())
	logger.SetFooter(currentSuffix, _footerCurrent)
	logger.SetFooter(historySuffix, _footerHistory)

	result := playGameBinaries(binaryPath, binaryArgs, stockfishElo)

	err = unmarshalEloResults(jsonPath, &results)
	if !IsNil(err) {
		panic(err)
	}

	results.Matches = append(results.Matches, result)
	results.EloEstimate = results.computeElo()

	logger.Printf("elo so far: %v\n", results.computeElo())
	logger.FlushFooter()

	err = marshalEloResults(jsonPath, &results)
	if !IsNil(err) {
		panic(err)
	}
}

func CompareStockfishMain(args []string) {
	var err Error

	dateString := time.Now().Format("2006_01_02")

	shouldClean := false
	printStats := false
	userSpecifiedBinaryArgs := []string{}

	performArgPermutations := false
	eachArg := false
	shouldProfile := false

	tags := []string{}

	for _, arg := range args {
		if arg == "clean" {
			shouldClean = true
		} else if strings.HasPrefix(arg, "v=") {
			version, err := WrapReturn(strconv.Atoi(arg[2:]))
			if !IsNil(err) {
				panic(err)
			}
			tags = append([]string{fmt.Sprintf("v%v", version)}, tags...)

		} else if arg == "stats" {
			printStats = true
		} else if arg == "permutations" {
			performArgPermutations = true
		} else if arg == "each" {
			eachArg = true
		} else if arg == "profile" {
			shouldProfile = true
		} else {
			userSpecifiedBinaryArgs = append(userSpecifiedBinaryArgs, arg)
		}
	}

	if len(tags) == 0 {
		tags = append(tags, "default")
	}

	resultsDir := RootDir() + "/data/elo_results"

	if printStats {
		allEloResults, err := allEloResultsInDir(resultsDir)
		if !IsNil(err) {
			panic(err)
		}
		reset := "\x1b[0m"
		foreground := "\033[38;5;255m"
		background1 := "\033[48;5;232m"
		background2 := "\033[48;5;235m"
		currentPrefix := ""
		for i, result := range allEloResults {
			line := result.statsString()

			cmdParts := strings.Split(Last(strings.Split(result.Cmd, "/")), "_")
			prefix := strings.Join(cmdParts[:MinInt(4, len(cmdParts))], "_")
			if prefix != currentPrefix {
				fmt.Println()
				currentPrefix = prefix
			}
			if i%2 == 0 {
				fmt.Printf("%v%v%v%v\n", background1, foreground, line, reset)
			} else {
				fmt.Printf("%v%v%v%v\n", background2, foreground, line, reset)
			}
		}

	}

	if printStats {
		return
	}

	err = MakeDirIfMissing(resultsDir)
	if !IsNil(err) {
		panic(err)
	}

	allBinaryArgsToTry := [][]string{
		userSpecifiedBinaryArgs,
	}

	if performArgPermutations {
		if len(userSpecifiedBinaryArgs) > 0 {
			allBinaryArgsToTry = append(combinations.All(userSpecifiedBinaryArgs), []string{})
		} else {
			allBinaryArgsToTry = append(combinations.All(AllSearchOptions), []string{})
		}
	} else if eachArg {
		if len(userSpecifiedBinaryArgs) > 0 {
			allBinaryArgsToTry = append(MapSlice(userSpecifiedBinaryArgs, func(arg string) []string {
				return []string{arg}
			}), []string{})
		} else {
			allBinaryArgsToTry = append(MapSlice(AllSearchOptions, func(arg string) []string {
				return []string{arg}
			}), []string{})
		}
	}
	allBinaryArgsToTry = FilterDisallowedSearchOptions(allBinaryArgsToTry)

	logger.Println("trying", len(allBinaryArgsToTry), "arg permutations")
	for _, binaryArgs := range allBinaryArgsToTry {
		logger.Println("  ", binaryArgs)
	}
	time.Sleep(time.Second * 1)

	numRuns := 1000
	if shouldClean {
		numRuns = len(allBinaryArgsToTry)
	}

	for i := 0; i < numRuns; i++ {
		func() {
			nextBinaryArgs := allBinaryArgsToTry[i%len(allBinaryArgsToTry)]
			nextTags := append(tags, nextBinaryArgs...)

			if shouldProfile {
				nextBinaryArgs = append(nextBinaryArgs, "profile")
				shouldProfile = false
			}
			fileNameBase := strings.Join(append([]string{dateString}, nextTags...), "_")
			binaryPath := fmt.Sprintf("%s/%v", resultsDir, fileNameBase)
			jsonPath := fmt.Sprintf("%s/%v.json", resultsDir, fileNameBase)
			mainInner(shouldClean, nextBinaryArgs, binaryPath, jsonPath)
		}()

		if !shouldClean {
			time.Sleep(time.Second * 10)
		}
	}
}