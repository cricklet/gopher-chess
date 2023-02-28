package runner

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func allRunners() []Runner {
	return []Runner{
		&StockfishRunner{},
		&ChessGoRunner{},
	}
}

func TestUci(t *testing.T) {
	for _, runner := range allRunners() {
		inputs := []string{
			"isready",
			"uci",
			"position fen rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			"go",
		}
		r := UciRunner{runner}
		for _, line := range inputs {
			log.Println(r.HandleInput(line))
		}
	}
}

func TestUciIndexBug2(t *testing.T) {
	for _, runner := range allRunners() {
		r := UciRunner{runner}
		for _, line := range []string{
			"isready",
			"uci",
			"position fen 2kr3r/p1p2ppp/2n1b3/2bqp3/Pp1p4/1P1P1N1P/2PBBPP1/R2Q1RK1 w - - 24 13",
			"position fen 2kr3r/p1p2ppp/2n1b3/2bqp3/Pp1p4/1P1P1N1P/2PBBPP1/R2Q1RK1 w - - 24 13 moves g2g4",
		} {
			_, err := r.HandleInput(line)
			assert.Nil(t, err)
		}

		_, err := r.HandleInput("go")
		assert.Nil(t, err)
	}
}

func TestUciIndexBug3(t *testing.T) {
	for _, runner := range allRunners() {
		r := UciRunner{runner}
		for _, line := range []string{
			"isready",
			"uci",
			"position fen 2k1r3/8/2np2p1/p1bq4/Pp2P1P1/1P1p4/2PBQ3/R4RK1 w - - 48 25",
			"position fen 2k1r3/8/2np2p1/p1bq4/Pp2P1P1/1P1p4/2PBQ3/R4RK1 w - - 48 25 moves d2e3",
		} {
			_, err := r.HandleInput(line)
			assert.Nil(t, err)
		}

		_, err := r.HandleInput("go")
		assert.Nil(t, err)
	}
}

func TestUciCastlingBug1(t *testing.T) {
	for _, runner := range allRunners() {
		r := UciRunner{runner}
		fen := "rn1qk2r/ppp3pp/3b1n2/3ppb2/8/2NPBNP1/PPP2PBP/R2QK2R b KQkq - 15 8"
		moves := []string{
			"e8g8",
			"d3d4",
		}
		for _, line := range []string{
			"isready",
			"uci",
			"position fen " + fen,
			"position fen " + fen + " moves " + moves[0],
			"position fen " + fen + " moves " + moves[0] + " " + moves[1],
		} {
			_, err := r.HandleInput(line)
			assert.Nil(t, err)
		}
	}
}

func TestUciStockfishManually(t *testing.T) {
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
