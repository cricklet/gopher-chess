package runner

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"time"
)

type StockfishRunner struct {
	cmd   *exec.Cmd
	stdin io.Writer

	stdoutChan chan string
	stderrChan chan string

	delay time.Duration
}

func (r *StockfishRunner) HandleInput(input string) ([]string, error) {
	result := []string{}
	var err error

	if r.cmd == nil {
		r.cmd = exec.Command("stockfish")
		r.stdin, err = r.cmd.StdinPipe()
		if err != nil {
			return result, err
		}
		var stdout io.Reader
		var stderr io.Reader
		stdout, err = r.cmd.StdoutPipe()
		if err != nil {
			return result, err
		}
		stderr, err = r.cmd.StderrPipe()
		if err != nil {
			return result, err
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
			return result, err
		}
	}

	_, err = r.stdin.Write([]byte(input + "\n"))
	if err != nil {
		return result, err
	}

	timeoutChan := make(chan bool)
	go func() {
		time.Sleep(r.delay)
		timeoutChan <- true
	}()

	done := false
	for !done {
		select {
		case <-timeoutChan:
			done = true
		case output := <-r.stdoutChan:
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
