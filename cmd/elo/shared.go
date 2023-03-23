package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cricklet/chessgo/internal/binary"
	"github.com/cricklet/chessgo/internal/chessgo"
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

const _footerEval = 1
const _footerCurrent = 2
const _footerBoard = 4
const _footerPgn = 6
const _footerHistory = 8

var logger = NewLiveLogger()

func MakeDirIfMissing(dir string) Error {
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

func RmIfExists(path string) Error {
	_, err := os.Stat(path)
	if IsNil(err) {
		return Wrap(os.Remove(path))
	}
	return NilError
}

func Exists(binaryPath string) (bool, Error) {
	_, err := os.Stat(binaryPath)
	if IsNil(err) {
		return true, NilError
	}
	if os.IsNotExist(err) {
		return false, NilError
	}
	return false, Wrap(err)
}

func BuildChessGoIfMissing(binaryPath string) Error {
	exists, err := Exists(binaryPath)
	if !IsNil(err) {
		return err
	}
	if exists {
		return NilError
	}

	logger.Println("go", "build", "-o", binaryPath, "cmd/uci/main.go")
	err = Wrap(exec.Command("go", "build", "-o", binaryPath, "cmd/uci/main.go").Run())
	if !IsNil(err) {
		return err
	}
	return NilError
}

func findMoveInOutput(output []string) (string, Error) {
	if len(output) == 0 {
		return "", Errorf("output was empty")
	}
	bestMoveString := FindInSlice(output, func(v string) bool {
		return strings.HasPrefix(v, "bestmove ")
	})
	if bestMoveString.HasValue() {
		return strings.Split(bestMoveString.Value(), " ")[1], NilError
	}
	return "", Errorf("couldn't find bestmove in output %v", output)
}

func Search(player Player, binary *binary.BinaryRunner, fen string, moveHistory []string, expectedFen string) []string {
	fenInput := fmt.Sprintf("position fen %v moves %v", fen, strings.Join(moveHistory, " "))
	RunAsync(binary, fenInput)

	if binary.CmdName() == "chessgo" {
		binaryFenOpt := FindInSlice(Run(binary, "fen", Some("position fen ")), func(v string) bool {
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

	results, err := RunThenStop(binary, "go", time.Millisecond*1000, "stop", Some("bestmove"))
	if !IsNil(err) {
		panic(err)
	}
	move, err := findMoveInOutput(results)
	if !IsNil(err) {
		panic(err)
	}
	moveHistory = append(moveHistory, move)

	logger.Printf("%v (%v) > %v\n", binary.CmdName(), player.String(), move)

	return moveHistory
}

type Evaluator struct {
	stockfish *binary.BinaryRunner
}

func NewEvaluator() (*Evaluator, Error) {
	evaluator, err := binary.SetupBinaryRunner("stockfish", "stockfish", []string{}, time.Millisecond*1000, binary.WithLogger(&SilentLogger))
	if !IsNil(err) {
		return nil, err
	}
	return &Evaluator{evaluator}, NilError
}

func (e *Evaluator) Close() {
	defer e.stockfish.Close()
}

func (e *Evaluator) Evaluate(fen string) (int, Error) {
	fenInput := fmt.Sprintf("position fen %v", fen)
	RunAsync(e.stockfish, fenInput)
	results, err := RunThenStop(e.stockfish, "go", time.Millisecond*10, "stop", Some("bestmove"))
	if !IsNil(err) {
		return 0, err
	}

	scoreStrs := FilterSlice(results, func(v string) bool {
		return strings.Contains(v, "score cp ") || strings.Contains(v, "score mate")
	})
	if len(scoreStrs) == 0 {
		return 0, Errorf("failed to find score in %v", Indent(strings.Join(results, "\n"), " > "))
	}

	scoreStr := Last(scoreStrs)
	if strings.Contains(scoreStr, "mate") {
		scoreStr = strings.Split(
			strings.Split(scoreStr, "score mate ")[1], " ")[0]
		score, err := ParseInt(scoreStr)
		if !IsNil(err) {
			return 0, err
		}
		if score > 0 {
			return 99999, NilError
		} else {
			return -99999, NilError
		}
	}

	scoreStr = strings.Split(
		strings.Split(scoreStr, "score cp ")[1], " ")[0]
	score, err := ParseInt(scoreStr)
	if !IsNil(err) {
		return 0, err
	}
	return score, NilError
}

func PlayBinaries(player0 *binary.BinaryRunner, player1 *binary.BinaryRunner,
	runner *chessgo.ChessGoRunner,
	callback func(),
) (float32, Error) {
	var err Error

	moveHistory := []string{}

	if !IsNil(err) {
		return 0.5, err
	}

	binaryToPlayer := map[*binary.BinaryRunner]Player{
		player0: White,
		player1: Black,
	}

	nextBinary := player0

	history := map[string]int{}

	for i := 0; i < 400; i++ {
		currentBinary := nextBinary
		if nextBinary == player0 {
			nextBinary = player1
		} else {
			nextBinary = player0
		}

		moveHistory = Search(binaryToPlayer[currentBinary], currentBinary, runner.StartFen, moveHistory, runner.FenString())
		if Last(moveHistory) == "forfeit" {
			if currentBinary == player0 {
				return 1, NilError
			} else {
				return 0, NilError
			}
		}
		err = runner.PerformMoveFromString(Last(moveHistory))
		if !IsNil(err) {
			return 0.5, err
		}

		boardString := game.FenStringForBoard(runner.Board())
		if _, contains := history[boardString]; !contains {
			history[boardString] = 0
		}
		history[boardString]++

		if history[boardString] >= 3 {
			return 0.5, NilError
		}

		callback()

		var noValidMoves bool
		noValidMoves, err = runner.NoValidMoves()
		if !IsNil(err) {
			return 0.5, err
		}

		if noValidMoves {
			if runner.PlayerIsInCheck() {
				if currentBinary == player0 {
					return 0, NilError
				} else {
					return 1, NilError
				}
			} else {
				return 0.5, NilError
			}
		}

		if runner.DrawClock() >= 50 {
			return 0.5, NilError
		}
	}

	return 0.5, NilError
}

func RunAsync(binary *binary.BinaryRunner, cmd string) {
	binary.Logger.Print("in=>", cmd)
	err := binary.RunAsync(cmd)
	if !IsNil(err) {
		panic(err)
	}
}

func Run(binary *binary.BinaryRunner, cmd string, waitFor Optional[string]) []string {
	var result []string
	binary.Logger.Print("in=>", cmd)
	result, err := binary.Run(cmd, waitFor)
	if !IsNil(err) {
		panic(err)
	}
	return result
}

func RunThenStop(binary *binary.BinaryRunner, cmd string, wait time.Duration, stopCmd string, waitFor Optional[string]) ([]string, Error) {
	returnErr := NilError

	var result []string
	binary.Logger.Print("in=>", cmd)

	done := false

	go func() {
		time.Sleep(wait)
		if done {
			return
		}
		binary.Logger.Print("in=>", stopCmd)
		err := binary.RunAsync(stopCmd)
		if !IsNil(err) {
			returnErr = Join(returnErr, err)
		}
	}()

	result, err := binary.Run(cmd, waitFor)
	if !IsNil(err) {
		panic(err)
	}

	done = true
	return result, returnErr
}
