package runner

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestIndexBug2(t *testing.T) {
	r := ChessGoRunner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen 2kr3r/p1p2ppp/2n1b3/2bqp3/Pp1p4/1P1P1N1P/2PBBPP1/R2Q1RK1 w - - 24 13",
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}

	err := r.PerformMoveFromString("g2g4")
	assert.Nil(t, err)
	_, err = r.HandleInput("go")
	assert.Nil(t, err)
}

func TestIndexBug3(t *testing.T) {
	r := ChessGoRunner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen 2k1r3/8/2np2p1/p1bq4/Pp2P1P1/1P1p4/2PBQ3/R4RK1 w - - 48 25",
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}

	err := r.PerformMoveFromString("d2e3")
	assert.Nil(t, err)
	_, err = r.HandleInput("go")
	assert.Nil(t, err)
}

func TestCastlingBug1(t *testing.T) {
	fen := "rn1qk2r/ppp3pp/3b1n2/3ppb2/8/2NPBNP1/PPP2PBP/R2QK2R b KQkq - 15 8"
	moves := []string{
		"e8g8",
		"d3d4",
	}
	r := ChessGoRunner{}
	for _, line := range []string{
		"isready",
		"uci",
		"position fen " + fen,
	} {
		_, err := r.HandleInput(line)
		assert.Nil(t, err)
	}

	for _, m := range moves {
		err := r.PerformMoveFromString(m)
		assert.Nil(t, err)
	}

	kingMoves, err := r.MovesForSelection("g8")
	assert.Nil(t, err)

	for _, m := range kingMoves {
		assert.NotEqual(t, "g8f8", m)
	}
}

type UciIteration struct {
	Input string
	Wait  time.Duration

	ExpectedOutput       Optional[string]
	ExpectedOutputPrefix Optional[string]
}

func TestStockfishManually(t *testing.T) {
	cmd := exec.Command("stockfish")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdOut, err := cmd.StdoutPipe()
	assert.Nil(t, err)

	stdOutScanner := bufio.NewScanner(bufio.NewReader(stdOut))

	defer func() {
		_ = cmd.Process.Kill()
	}()

	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	stdOutChan := make(chan string)
	go func() {
		for stdOutScanner.Scan() {
			stdOutChan <- stdOutScanner.Text()
		}
	}()

	for _, it := range []UciIteration{
		{"isready\n", time.Millisecond * 100, Some("readyok"), Empty[string]()},
		{"uci\n", time.Millisecond * 100, Some("uciok"), Empty[string]()},
		{"position startpos\n", time.Millisecond * 100, Empty[string](), Empty[string]()},
		{"go\n", time.Millisecond * 100, Empty[string](), Empty[string]()},
		{"stop\n", time.Millisecond * 100, Empty[string](), Some("bestmove")},
	} {
		timeoutChan := make(chan bool)
		go func() {
			time.Sleep(it.Wait)
			timeoutChan <- true
		}()

		_, err := stdin.Write([]byte(it.Input))
		if err != nil {
			t.Fatal(err)
		}

		expectsOutput := it.ExpectedOutput.HasValue() || it.ExpectedOutputPrefix.HasValue()

		done := false
		for !done {
			select {
			case <-timeoutChan:
				if expectsOutput {
					t.Fatal(fmt.Errorf("timeout for %v without correct output", it))
				}
				done = true
			case line := <-stdOutChan:
				fmt.Println("$", strings.TrimSpace(it.Input), ">", line)

				if it.ExpectedOutput.HasValue() && line == it.ExpectedOutput.Value() {
					done = true
				} else if it.ExpectedOutputPrefix.HasValue() && strings.HasPrefix(line, it.ExpectedOutputPrefix.Value()) {
					done = true
				}
			}
		}
	}

}

func TestStockfish(t *testing.T) {
	r := StockfishRunner{delay: time.Millisecond * 100}

	for _, it := range []UciIteration{
		{"isready\n", time.Millisecond * 100, Some("readyok"), Empty[string]()},
		{"uci\n", time.Millisecond * 100, Some("uciok"), Empty[string]()},
		{"position startpos\n", time.Millisecond * 100, Empty[string](), Empty[string]()},
		{"go\n", time.Millisecond * 100, Empty[string](), Empty[string]()},
		{"stop\n", time.Millisecond * 100, Empty[string](), Some("bestmove")},
	} {
		result, err := r.HandleInput(it.Input)
		if err != nil {
			t.Fatal(err)
		}

		expectsOutput := it.ExpectedOutput.HasValue() || it.ExpectedOutputPrefix.HasValue()

		foundExpectedOutput := false
		for _, line := range result {
			fmt.Println("$", strings.TrimSpace(it.Input), ">", line)

			if it.ExpectedOutput.HasValue() && line == it.ExpectedOutput.Value() {
				foundExpectedOutput = true
			} else if it.ExpectedOutputPrefix.HasValue() && strings.HasPrefix(line, it.ExpectedOutputPrefix.Value()) {
				foundExpectedOutput = true
			}
		}

		if expectsOutput && !foundExpectedOutput {
			t.Fatal(fmt.Errorf("expected output not found: %s", it.Input))
		}
	}

}

func TestBattle(t *testing.T) {
	chessgo := ChessGoRunner{}
	stockfish := StockfishRunner{delay: time.Millisecond * 100}

	// Setup both runners
	for _, line := range []string{
		"isready",
		"uci",
		"position startpos",
	} {
		_, err := chessgo.HandleInput(line)
		assert.Nil(t, err)
		_, err = stockfish.HandleInput(line)
		assert.Nil(t, err)
	}

	// Play a game
	var getMoveFromRunner = func(r Runner) (string, error) {
		var err error
		var stopResult, goResult []string
		goResult, err = r.HandleInput("go")
		assert.Nil(t, err)
		stopResult, err = r.HandleInput("stop")
		assert.Nil(t, err)

		for _, line := range append(goResult, stopResult...) {
			if strings.HasPrefix(line, "bestmove") {
				return strings.Split(line, " ")[1], nil
			}
		}
		return "", errors.New("no bestmove found")
	}

	moveHistory := []string{}

	for i := 0; i < 2; i++ {
		var err error
		var move string
		move, err = getMoveFromRunner(&chessgo)
		assert.Nil(t, err)
		moveHistory = append(moveHistory, move)

		fmt.Println("> chessgo: ", move)

		_, err = chessgo.HandleInput("position startpos moves " + strings.Join(moveHistory, " "))
		assert.Nil(t, err)

		_, err = stockfish.HandleInput("position startpos moves " + strings.Join(moveHistory, " "))
		assert.Nil(t, err)

		move, err = getMoveFromRunner(&stockfish)
		assert.Nil(t, err)
		moveHistory = append(moveHistory, move)

		_, err = chessgo.HandleInput("position startpos moves " + strings.Join(moveHistory, " "))
		assert.Nil(t, err)

		_, err = stockfish.HandleInput("position startpos moves " + strings.Join(moveHistory, " "))
		assert.Nil(t, err)

		fmt.Println("> stockfish: ", move)
	}

	assert.Equal(t, 4, len(chessgo.history))
}
