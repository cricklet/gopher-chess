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
	stockfish, err = SetupBinaryRunner("stockfish", time.Millisecond*100)
	if !IsNil(err) {
		panic(err)
	}
	defer stockfish.Close()

	var opponent *BinaryRunner
	opponent, err = SetupBinaryRunner(binaryPath, time.Second*10)
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()

	var runAsync = func(binary *BinaryRunner, cmd string) {
		err = binary.RunAsync(cmd)
		if !IsNil(err) {
			panic(err)
		}
	}

	var run = func(binary *BinaryRunner, cmd string, waitFor Optional[string]) []string {
		var result []string
		result, err = binary.Run(cmd, waitFor)
		if !IsNil(err) {
			panic(err)
		}
		return result
	}

	var findMoveInOutput = func(output []string) string {
		if !IsNil(err) {
			panic(err)
		}
		if len(output) == 0 {
			err = Errorf("output was empty")
			return ""
		}
		bestMoveString := FindInSlice(output, func(v string) bool {
			return strings.HasPrefix(v, "bestmove ")
		})
		if bestMoveString.HasValue() {
			return strings.Split(bestMoveString.Value(), " ")[1]
		}
		err = Errorf("couldn't find bestmove in output %v", output)
		return ""
	}

	run(stockfish, "isready", Some("readyok"))
	run(stockfish, "uci", Some("uciok"))
	runAsync(stockfish, "ucinewgame")
	runAsync(stockfish, "setoption name UCI_LimitStrength value true")
	runAsync(stockfish, "setoption name UCI_Elo value 800")
	runAsync(stockfish, "position startpos")

	run(opponent, "isready", Some("readyok"))
	run(opponent, "uci", Some("uciok"))
	runAsync(opponent, "ucinewgame")
	runAsync(opponent, "setoption name UCI_LimitStrength value true")
	runAsync(opponent, "setoption name UCI_Elo value 800")
	runAsync(opponent, "position startpos")

	moveHistory := []string{}
	var move string

	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	runner := ChessGoRunner{}
	err = runner.SetupPosition(Position{
		Fen:   fen,
		Moves: []string{},
	})
	if !IsNil(err) {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		runAsync(stockfish, "go")
		time.Sleep(time.Second)
		move = findMoveInOutput(run(stockfish, "stop", Some("bestmove")))
		moveHistory = append(moveHistory, move)

		runAsync(opponent, fmt.Sprintf("position fen %v moves %v", fen, strings.Join(moveHistory, " ")))
		err = runner.PerformMoveFromString(move)
		if !IsNil(err) {
			panic(err)
		}
		fmt.Println("stockfish", move)
		fmt.Println(runner.Board().String())

		runAsync(opponent, "go")
		time.Sleep(time.Second)
		move = findMoveInOutput(run(opponent, "stop", Some("bestmove")))
		moveHistory = append(moveHistory, move)

		runAsync(stockfish, fmt.Sprintf("position fen %v moves %v", fen, strings.Join(moveHistory, " ")))
		err = runner.PerformMoveFromString(move)
		if !IsNil(err) {
			panic(err)
		}

		fmt.Println("opponent", move)
		fmt.Println(runner.Board().String())
	}
}
