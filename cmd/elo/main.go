package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/cricklet/chessgo/internal/binary_runner"
	. "github.com/cricklet/chessgo/internal/helpers"
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
	return Wrap(err)
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
	opponent, err = SetupBinaryRunner(binaryPath, time.Millisecond*100)
	if !IsNil(err) {
		panic(err)
	}
	defer opponent.Close()

	var runAsyncCheckingErrors = func(binary *BinaryRunner, cmd string) {
		if !IsNil(err) {
			panic(err)
		}
		err = binary.RunAsync(cmd)
	}

	var runCheckingErrors = func(binary *BinaryRunner, cmd string, waitFor Optional[string]) []string {
		if !IsNil(err) {
			panic(err)
		}
		var result []string
		result, err = binary.Run(cmd, waitFor)
		return result
	}

	runCheckingErrors(stockfish, "isready", Some("readyok"))
	runCheckingErrors(stockfish, "uci", Some("uciok"))
	runAsyncCheckingErrors(stockfish, "ucinewgame")
	runAsyncCheckingErrors(stockfish, "setoption name UCI_LimitStrength value true")
	runAsyncCheckingErrors(stockfish, "setoption name UCI_Elo value 800")
	runAsyncCheckingErrors(stockfish, "position startpos")
	runAsyncCheckingErrors(stockfish, "go")
	time.Sleep(time.Second)
	runCheckingErrors(stockfish, "stop", Some("bestmove"))
}
