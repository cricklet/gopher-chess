package binary_runner

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type BinaryRunner struct {
	cmdPath string
	cmd     *exec.Cmd
	stdin   io.Writer

	stdoutChan chan string
	stderrChan chan string

	timeout time.Duration
	logger  Logger
}

type BinaryRunnerOption func(*BinaryRunner)

func WithLogger(logger Logger) BinaryRunnerOption {
	return func(u *BinaryRunner) {
		u.logger = logger
	}
}

func SetupBinaryRunner(cmdPath string, delay time.Duration, options ...BinaryRunnerOption) (*BinaryRunner, Error) {
	var err Error

	u := &BinaryRunner{
		cmdPath: cmdPath,
		timeout: delay,
	}

	for _, option := range options {
		option(u)
	}

	if u.logger == nil {
		u.logger = &DefaultLogger
	}

	if u.cmd == nil {
		u.cmd = exec.Command(cmdPath)
		u.stdin, err = WrapReturn(u.cmd.StdinPipe())
		if !IsNil(err) {
			return u, Wrap(err)
		}
		var stdout io.Reader
		var stderr io.Reader
		stdout, err = WrapReturn(u.cmd.StdoutPipe())
		if !IsNil(err) {
			return u, Wrap(err)
		}
		stderr, err = WrapReturn(u.cmd.StderrPipe())
		if !IsNil(err) {
			return u, Wrap(err)
		}

		u.stdoutChan = make(chan string)
		go func() {
			stdoutScanner := bufio.NewScanner(bufio.NewReader(stdout))
			for stdoutScanner.Scan() {
				line := stdoutScanner.Text()
				u.logger.Printf("%v > %v", u.cmdPath, line)
				u.stdoutChan <- line
			}
		}()

		u.stderrChan = make(chan string)
		go func() {
			stderrScanner := bufio.NewScanner(bufio.NewReader(stderr))
			for stderrScanner.Scan() {
				u.stderrChan <- stderrScanner.Text()
			}
		}()

		err = Wrap(u.cmd.Start())
		if !IsNil(err) {
			return u, Wrap(err)
		}
	}

	return u, NilError
}

func (u *BinaryRunner) RunAsync(input string) Error {
	if u.cmd == nil || u.stdin == nil {
		return Errorf("cmd not setup: %v", u.cmdPath)
	}

	if u.cmd.ProcessState != nil && u.cmd.ProcessState.Exited() {
		return Errorf("cmd exited: %v", u.cmdPath)
	}

	_, err := u.stdin.Write([]byte(input + "\n"))
	if !IsNil(err) {
		return Wrap(err)
	}

	return NilError
}

func (u *BinaryRunner) Run(input string, waitFor Optional[string]) ([]string, Error) {
	result := []string{}

	err := u.RunAsync(input)
	if !IsNil(err) {
		return result, err
	}

	timeoutChan := make(chan bool)
	go func() {
		time.Sleep(u.timeout)
		timeoutChan <- true
	}()

	done := false
	foundOutput := false
	for !done {
		select {
		case <-timeoutChan:
			u.logger.Printf("%v > %v", u.cmdPath, "timeout")
			done = true
		case output := <-u.stdoutChan:
			result = append(result, output)
			if waitFor.HasValue() && strings.Contains(output, waitFor.Value()) {
				foundOutput = true
				done = true
			}
		}
	}

	if waitFor.HasValue() && !foundOutput {
		return result, Errorf("timeout waiting for %v", waitFor.Value())
	}

	return result, NilError
}

func (u *BinaryRunner) Close() {
	if u.cmd != nil {
		_ = u.cmd.Process.Kill()
		u.cmd = nil
	}
}
