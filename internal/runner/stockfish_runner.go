package runner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type StockfishRunner struct {
	Delay  time.Duration
	Logger Logger

	cmd   *exec.Cmd
	stdin io.Writer

	stdoutChan chan string
	stderrChan chan string

	startFen string
	moves    []string
}

var _ Runner = (*StockfishRunner)(nil)

func (r *StockfishRunner) SetupPosition(position Position) error {
	var err error

	if r.cmd == nil {
		if r.Delay == 0 {
			r.Delay = time.Millisecond * 100
		}
		if r.Logger == nil {
			r.Logger = &DefaultLogger
		}

		r.cmd = exec.Command("stockfish")
		r.stdin, err = r.cmd.StdinPipe()
		if err != nil {
			return err
		}
		var stdout io.Reader
		var stderr io.Reader
		stdout, err = r.cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err = r.cmd.StderrPipe()
		if err != nil {
			return err
		}

		r.stdoutChan = make(chan string)
		go func() {
			stdoutScanner := bufio.NewScanner(bufio.NewReader(stdout))
			for stdoutScanner.Scan() {
				r.stdoutChan <- stdoutScanner.Text()
			}
		}()

		r.stderrChan = make(chan string)
		go func() {
			stderrChan := bufio.NewScanner(bufio.NewReader(stderr))
			for stderrChan.Scan() {
				r.stderrChan <- stderrChan.Text()
			}
		}()

		err = r.cmd.Start()
		if err != nil {
			return err
		}
	}

	_, err = r.run("isready")
	if err != nil {
		return err
	}
	_, err = r.run("uci")
	if err != nil {
		return err
	}

	r.startFen = position.Fen
	r.moves = position.Moves
	_, err = r.run("position fen " + position.Fen + " moves " + strings.Join(position.Moves, " "))
	if err != nil {
		return err
	}

	return nil
}

func (r *StockfishRunner) Reset() {
	if r.cmd != nil {
		_ = r.cmd.Process.Kill()
		r.cmd = nil

		r.startFen = ""
		r.moves = []string{}
	}
}

func (r *StockfishRunner) run(input string) ([]string, error) {
	result := []string{}
	var err error

	if r.cmd == nil || r.stdin == nil {
		return result, errors.New("call Setup()")
	}

	_, err = r.stdin.Write([]byte(input + "\n"))
	if err != nil {
		return result, err
	}

	timeoutChan := make(chan bool)
	go func() {
		time.Sleep(r.Delay)
		timeoutChan <- true
	}()

	done := false
	for !done {
		select {
		case <-timeoutChan:
			done = true
		case output := <-r.stdoutChan:
			if !strings.HasPrefix(output, "option") && !strings.HasPrefix(output, "id") && !strings.HasPrefix(output, "info") {
				r.Logger.Println(output)
			}
			result = append(result, output)
			if input == "isready" && output == "readyok" {
				done = true
			} else if input == "uci" && output == "uciok" {
				done = true
			} else if (input == "stop" || input == "go") && strings.HasPrefix(output, "bestmove") {
				done = true
			}
		}
	}

	return result, nil
}

func (r *StockfishRunner) IsNew() bool {
	return r.cmd == nil
}

func (r *StockfishRunner) PerformMoves(fen string, moves []string) error {
	if fen != r.startFen {
		return fmt.Errorf("fen %s does not match start fen %s", fen, r.startFen)
	}

	_, err := r.run("position fen " + fen + " moves " + strings.Join(moves, " "))
	r.moves = moves

	if err != nil {
		return err
	}
	return nil
}

func (r *StockfishRunner) PerformMoveFromString(s string) error {
	r.moves = append(r.moves, s)
	_, err := r.run("position " + r.startFen + " moves " + strings.Join(r.moves, " ") + " " + s)

	if err != nil {
		return err
	}
	return nil
}

func (r *StockfishRunner) MovesForSelection(selection string) ([]string, error) {
	return []string{}, errors.New("not implemented")
}

func (r *StockfishRunner) Rewind(num int) error {
	return errors.New("not implemented")
}

func (r *StockfishRunner) Search() (Optional[string], error) {
	var err error
	var goResult []string
	var stopResult []string
	goResult, err = r.run("go")
	if err != nil {
		return Empty[string](), err
	}
	time.Sleep(100 * time.Millisecond)
	stopResult, err = r.run("stop")
	if err != nil {
		return Empty[string](), err
	}

	results := append(goResult, stopResult...)

	bestMoveString := FindInSlice(results, func(v string) bool {
		return strings.HasPrefix(v, "bestmove ")
	})

	if bestMoveString.HasValue() {
		return Some(strings.Split(bestMoveString.Value(), " ")[1]), nil
	}

	return Empty[string](), nil
}
