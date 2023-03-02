package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/binary_runner"
	. "github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/runner"
)

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

func runAsync(binary *BinaryRunner, cmd string) {
	err := binary.RunAsync(cmd)
	if !IsNil(err) {
		panic(err)
	}
}

func run(binary *BinaryRunner, cmd string, waitFor Optional[string]) []string {
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

func search(binary *BinaryRunner, fen string, moveHistory []string, expectedFen string) []string {
	fenInput := fmt.Sprintf("position fen %v moves %v", fen, strings.Join(moveHistory, " "))
	runAsync(binary, fenInput)

	if strings.Contains(binary.CmdName(), "chessgo") {
		binaryFenOpt := FindInSlice(run(binary, "fen", Some("position fen ")), func(v string) bool {
			return strings.HasPrefix(v, "position fen ")
		})
		if binaryFenOpt.HasValue() {
			binaryFen, _ := strings.CutPrefix(binaryFenOpt.Value(), "position fen ")
			if binaryFen != expectedFen {
				fmt.Println(binary.Flush())
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

	fmt.Println(binary.CmdName(), ">", move)

	return moveHistory
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

	var stockfish *BinaryRunner
	stockfish, err = SetupBinaryRunner("stockfish", time.Millisecond*1000, WithLogger(&SilentLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer stockfish.Close()

	printlnLogger := FuncLogger(func(s string) { fmt.Println("chessgo > " + Indent(s, "$ ")) })
	var opponent *BinaryRunner
	opponent, err = SetupBinaryRunner(binaryPath, time.Millisecond*10000, WithLogger(printlnLogger))
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()

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

	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	runner := ChessGoRunner{}
	err = runner.SetupPosition(Position{
		Fen:   fen,
		Moves: []string{},
	})
	if !IsNil(err) {
		panic(err)
	}

	nextPlayer := stockfish

	for i := 0; i < 1000; i++ {
		currentPlayer := nextPlayer
		if nextPlayer == stockfish {
			nextPlayer = opponent
		} else {
			nextPlayer = stockfish
		}

		moveHistory = search(currentPlayer, fen, moveHistory, runner.FenString())
		if Last(moveHistory) == "forfeit" {
			fmt.Println("finished, winner is:", nextPlayer.CmdName())
			break
		}
		err = runner.PerformMoveFromString(Last(moveHistory))
		if !IsNil(err) {
			panic(err)
		}
		// fmt.Println(fen + " moves " + strings.Join(moveHistory, " "))

		fmt.Println()
		fmt.Println(runner.Board().Unicode())
		fmt.Println()

		var noValidMoves bool
		noValidMoves, err = runner.NoValidMoves()
		if noValidMoves {
			if runner.PlayerIsInCheck() {
				fmt.Println("finished, winner is:", currentPlayer.CmdName())
			} else {
				fmt.Println("finished, draw")
			}
			return
		}
	}
}
