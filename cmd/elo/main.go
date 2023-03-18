package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cricklet/chessgo/internal/binary"
	. "github.com/cricklet/chessgo/internal/chessgo"
	"github.com/cricklet/chessgo/internal/game"
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

type MatchResult struct {
	Won          bool
	Draw         bool
	Unknown      bool
	StartFen     string
	PositionFen  string
	EndingFen    string
	PgnMoves     string
	StockfishElo int
}
type EloResults struct {
	Cmd         string
	Matches     []MatchResult
	EloEstimate int
}

func (r EloResults) statsString() string {
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
	return fmt.Sprintf("%v: %v (%v) wins %v, draws %v, losses %v", cmdName, r.computeElo(), len(r.Matches), wins, draws, losses)
}

func (r EloResults) computeElo() int {
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

func (r EloResults) numMatches() int {
	return len(r.Matches)
}

func (r EloResults) matchHistory() string {
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

func unmarshalEloResults(path string, results *EloResults) Error {
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

func marshalEloResults(path string, results *EloResults) Error {
	output, err := json.MarshalIndent(results, "", "  ")
	if !IsNil(err) {
		return Wrap(err)
	}
	err = os.WriteFile(path, output, 0600)
	return Wrap(err)
}

func makeDirIfMissing(dir string) Error {
	_, err := os.Stat(dir)
	if IsNil(err) {
		return NilError
	}
	err = os.Mkdir(dir, 0755)
	if !IsNil(err) {
		return Wrap(err)
	}
	return NilError
}

func rmIfExists(path string) Error {
	_, err := os.Stat(path)
	if IsNil(err) {
		return Wrap(os.Remove(path))
	}
	return NilError
}

func buildChessGoIfMissing(binaryPath string) Error {
	_, err := os.Stat(binaryPath)
	if IsNil(err) {
		return NilError
	}
	if os.IsNotExist(err) {
		logger.Println("go", "build", "-o", binaryPath, "cmd/uci/main.go")
		err = exec.Command("go", "build", "-o", binaryPath, "cmd/uci/main.go").Run()
		if !IsNil(err) {
			return Wrap(err)
		}
		return NilError
	}
	return Wrap(err)
}

func runAsync(binary *binary.BinaryRunner, cmd string) {
	binary.Logger.Print("in=>", cmd)
	err := binary.RunAsync(cmd)
	if !IsNil(err) {
		panic(err)
	}
}

func run(binary *binary.BinaryRunner, cmd string, waitFor Optional[string]) []string {
	var result []string
	binary.Logger.Print("in=>", cmd)
	result, err := binary.Run(cmd, waitFor)
	if !IsNil(err) {
		panic(err)
	}
	return result
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

func search(player Player, binary *binary.BinaryRunner, fen string, moveHistory []string, expectedFen string) []string {
	fenInput := fmt.Sprintf("position fen %v moves %v", fen, strings.Join(moveHistory, " "))
	runAsync(binary, fenInput)

	if strings.Contains(binary.CmdName(), "chessgo") {
		binaryFenOpt := FindInSlice(run(binary, "fen", Some("position fen ")), func(v string) bool {
			return strings.HasPrefix(v, "position fen ")
		})
		if binaryFenOpt.HasValue() {
			binaryFen, _ := strings.CutPrefix(binaryFenOpt.Value(), "position fen ")
			if binaryFen != expectedFen {
				logger.Println(binary.Flush())
				panic(Errorf("wat\nprocessing %v\n%v (%v) != \n%v (expected)", fenInput, binaryFen, binary.CmdName(), expectedFen))
			}
		} else {
			panic(Errorf("failed to get fen from %v", binary.CmdPath()))
		}
	}

	runAsync(binary, "go")
	time.Sleep(time.Millisecond * 100)
	move := findMoveInOutput(run(binary, "stop", Some("bestmove")))
	moveHistory = append(moveHistory, move)

	logger.Printf("%v (%v) > %v\n", binary.CmdName(), player.String(), move)

	return moveHistory
}

type Result int

const (
	StockfishWin Result = iota
	OpponentWin
	Draw
	Unknown
)

func playGame(
	stockfish *binary.BinaryRunner,
	stockfishElo int,
	opponent *binary.BinaryRunner,
	opponentPlays Player,
	runner *ChessGoRunner,
) (Result, Error) {
	var err Error

	run(stockfish, "isready", Some("readyok"))
	run(stockfish, "uci", Some("uciok"))
	runAsync(stockfish, "ucinewgame")
	runAsync(stockfish, "setoption name UCI_LimitStrength value true")
	runAsync(stockfish, fmt.Sprintf("setoption name UCI_Elo value %v", stockfishElo))
	runAsync(stockfish, "position startpos")

	run(opponent, "isready", Some("readyok"))
	run(opponent, "uci", Some("uciok"))
	runAsync(opponent, "ucinewgame")
	runAsync(opponent, "position startpos")

	moveHistory := []string{}

	if !IsNil(err) {
		return Unknown, err
	}

	binaryToPlayer := map[*binary.BinaryRunner]Player{
		stockfish: opponentPlays.Other(),
		opponent:  opponentPlays,
	}

	nextBinary := stockfish

	history := map[string]int{}

	for i := 0; i < 400; i++ {
		currentBinary := nextBinary
		if nextBinary == stockfish {
			nextBinary = opponent
		} else {
			nextBinary = stockfish
		}

		moveHistory = search(binaryToPlayer[currentBinary], currentBinary, runner.StartFen, moveHistory, runner.FenString())
		if Last(moveHistory) == "forfeit" {
			if currentBinary == stockfish {
				return OpponentWin, NilError
			} else {
				return StockfishWin, NilError
			}
		}
		err = runner.PerformMoveFromString(Last(moveHistory))
		if !IsNil(err) {
			return Unknown, err
		}

		boardString := game.FenStringForBoard(runner.Board())
		if _, contains := history[boardString]; !contains {
			history[boardString] = 0
		}
		history[boardString]++

		if history[boardString] >= 3 {
			return Draw, NilError
		}

		pgnString := fmt.Sprintf("%v\n%v", runner.PgnFromMoveHistory(), runner.FenString())
		logger.SetFooter(HintText(pgnString), _footerPgn)
		logger.SetFooter(runner.Board().Unicode(), _footerBoard)

		var noValidMoves bool
		noValidMoves, err = runner.NoValidMoves()
		if !IsNil(err) {
			return Unknown, err
		}

		if noValidMoves {
			if runner.PlayerIsInCheck() {
				if currentBinary == stockfish {
					return StockfishWin, NilError
				} else {
					return OpponentWin, NilError
				}
			} else {
				return Draw, NilError
			}
		}

		if runner.DrawClock() >= 50 {
			return Draw, NilError
		}
	}

	return Unknown, NilError
}

func playGameBinaries(
	binaryPath string,
	binaryArgs []string,
	stockfishElo int,
) MatchResult {
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
	stockfish, err = binary.SetupBinaryRunner("stockfish", []string{}, time.Millisecond*1000, binary.WithLogger(stockfishLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer stockfish.Close()

	var opponent *binary.BinaryRunner
	chessgoLogger := FuncLogger(func(s string) { logger.Println("chessgo > " + Indent(s, "$ ")) })
	opponent, err = binary.SetupBinaryRunner(binaryPath, binaryArgs, time.Millisecond*10000, binary.WithLogger(chessgoLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()

	var result Result
	result, err = playGame(stockfish, stockfishElo, opponent, opponentPlays, &runner)
	if !IsNil(err) {
		panic(err)
	}

	newResult := MatchResult{
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

	runAsync(stockfish, "quit")
	runAsync(opponent, "quit")
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

func allEloResultsInDir(dir string) ([]EloResults, Error) {
	filePaths, err := allJsonFilesInDir(dir)
	if !IsNil(err) {
		return nil, err
	}

	results := []EloResults{}
	for _, filePath := range filePaths {
		result := EloResults{}
		unmarshalEloResults(filePath, &result)
		results = append(results, result)
	}
	return results, NilError
}

func mainInner(shouldClean bool, binaryArgs []string, binaryPath string, jsonPath string) {
	var err Error
	if shouldClean {
		logger.Printf("cleaning %v\n     and %v\n", binaryPath, jsonPath)
		err = rmIfExists(binaryPath)
		if !IsNil(err) {
			panic(err)
		}
		err = rmIfExists(jsonPath)
		if !IsNil(err) {
			panic(err)
		}
		return
	}

	err = buildChessGoIfMissing(binaryPath)
	if !IsNil(err) {
		panic(err)
	}

	results := EloResults{
		Cmd:     binaryPath,
		Matches: []MatchResult{},
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

func main() {
	var err Error

	args := os.Args[1:]

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
		statsStrings := []string{}
		for _, results := range allEloResults {
			statsStrings = append(statsStrings, results.statsString())
		}

		logger.Println(strings.Join(statsStrings, "\n"))
	}

	if printStats {
		return
	}

	err = makeDirIfMissing(resultsDir)
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
