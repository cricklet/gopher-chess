package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cricklet/chessgo/internal/binary"
	. "github.com/cricklet/chessgo/internal/chessgo"
	. "github.com/cricklet/chessgo/internal/helpers"
)

var logger = NewLiveLogger()

type MatchResult struct {
	Won   bool
	Moves int
}
type EloResults struct {
	Elo     int
	Cmd     string
	Matches []MatchResult
}

func makeDirIfMissing(dir string) Error {
	_, err := os.Stat(dir)
	if IsNil(err) {
		return NilError
	}
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if !IsNil(err) {
			return Wrap(err)
		}
		return NilError
	}
	return Wrap(err)
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
		err = exec.Command("go", "build", "-o", binaryPath, "cmd/uci/main.go").Run()
		if !IsNil(err) {
			return Wrap(err)
		}
		return NilError
	}
	return Wrap(err)
}

func runAsync(binary *binary.BinaryRunner, cmd string) {
	err := binary.RunAsync(cmd)
	if !IsNil(err) {
		panic(err)
	}
}

func run(binary *binary.BinaryRunner, cmd string, waitFor Optional[string]) []string {
	var result []string
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

	logger.Printf("%v (%v) > %v\n\n", binary.CmdName(), player.String(), move)

	return moveHistory
}

type Result int

const (
	StockfishWin Result = iota
	OpponentWin
	Draw
	Unknown
)

const hintColor = "\033[38;5;255m"
const resetColors = "\033[0m"

func playGame(
	stockfish *binary.BinaryRunner,
	opponent *binary.BinaryRunner,
	runner *ChessGoRunner,
) (Result, Error) {
	var err Error

	run(stockfish, "isready", Some("readyok"))
	run(stockfish, "uci", Some("uciok"))
	runAsync(stockfish, "ucinewgame")
	runAsync(stockfish, "setoption name UCI_LimitStrength value true")
	runAsync(stockfish, "setoption name UCI_Elo value 800")
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
		stockfish: White,
		opponent:  Black,
	}

	nextBinary := stockfish

	for i := 0; i < 1000; i++ {
		currentBinary := nextBinary
		if nextBinary == stockfish {
			nextBinary = opponent
		} else {
			nextBinary = stockfish
		}

		moveHistory = search(binaryToPlayer[currentBinary], currentBinary, runner.StartFen, moveHistory, runner.FenString())
		if Last(moveHistory) == "forfeit" {
			logger.Println("finished, winner is:", nextBinary.CmdName())
			break
		}
		err = runner.PerformMoveFromString(Last(moveHistory))
		if !IsNil(err) {
			return Unknown, err
		}
		// logger.Println(fen + " moves " + strings.Join(moveHistory, " "))

		logger.SetFooter(
			fmt.Sprintf("\n%v%v\n\nfen: %v\npiece score: %v%v",
				runner.Board().Unicode(),
				hintColor,
				runner.StartFen+" moves "+strings.Join(moveHistory, " "),
				runner.Evaluate(binaryToPlayer[opponent]),
				resetColors),
		)

		var noValidMoves bool
		noValidMoves, err = runner.NoValidMoves()
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
	}

	return Unknown, NilError
}

func main() {
	var err Error

	args := os.Args[1:]

	resultsDir := RootDir() + "/data/elo_results"
	binaryPath := fmt.Sprintf("%s/%v_chessgo", resultsDir, time.Now().Format("2006_01_02"))
	jsonPath := fmt.Sprintf("%s/%v_results.json", resultsDir, time.Now().Format("2006_01_02"))

	if len(args) == 1 {
		if args[0] == "clean" {
			err = rmIfExists(binaryPath)
			if !IsNil(err) {
				panic(err)
			}
			err = rmIfExists(jsonPath)
			if !IsNil(err) {
				panic(err)
			}
		}
	}

	err = makeDirIfMissing(resultsDir)
	if !IsNil(err) {
		panic(err)
	}

	err = buildChessGoIfMissing(binaryPath)
	if !IsNil(err) {
		panic(err)
	}

	var stockfish *binary.BinaryRunner
	stockfish, err = binary.SetupBinaryRunner("stockfish", time.Millisecond*1000, binary.WithLogger(&SilentLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer stockfish.Close()

	printlnLogger := FuncLogger(func(s string) { logger.Println("chessgo > " + Indent(s, "$ ")) })
	var opponent *binary.BinaryRunner
	opponent, err = binary.SetupBinaryRunner(binaryPath, time.Millisecond*10000, binary.WithLogger(printlnLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()

	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	runner := NewChessGoRunner(nil)
	err = runner.SetupPosition(Position{
		Fen:   fen,
		Moves: []string{},
	})
	if !IsNil(err) {
		panic(err)
	}

	var result Result
	result, err = playGame(stockfish, opponent, &runner)
	if !IsNil(err) {
		panic(err)
	}

	switch result {
	case StockfishWin:
		logger.Println("stockfish won")
	case OpponentWin:
		logger.Println("opponent won")
	case Draw:
		logger.Println("draw")
	}
}
