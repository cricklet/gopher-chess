package binary_runner

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type UciBinary struct {
	cmdPath string
	cmd     *exec.Cmd
	stdin   io.Writer

	stdoutChan chan string
	stderrChan chan string

	timeout time.Duration
	logger  Logger
}

func SetupUciBinaryWithDefaultLogger(cmdPath string, timeout time.Duration) (*UciBinary, Error) {
	return SetupUciBinary(cmdPath, timeout, &DefaultLogger)
}

func SetupUciBinary(cmdPath string, delay time.Duration, logger Logger) (*UciBinary, Error) {
	var err Error

	u := &UciBinary{
		cmdPath: cmdPath,
		timeout: delay,
		logger:  logger,
	}

	if u.cmd == nil {
		u.cmd = exec.Command(cmdPath)
		u.stdin, err = WrapReturn(u.cmd.StdinPipe())
		if !IsNil(err) {
			return u, err
		}
		var stdout io.Reader
		var stderr io.Reader
		stdout, err = WrapReturn(u.cmd.StdoutPipe())
		if !IsNil(err) {
			return u, err
		}
		stderr, err = WrapReturn(u.cmd.StderrPipe())
		if !IsNil(err) {
			return u, err
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
			return u, err
		}
	}

	return u, NilError
}

func (u *UciBinary) RunAsync(input string) Error {
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

func (u *UciBinary) Run(input string, waitFor Optional[string]) ([]string, Error) {
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
	for !done {
		select {
		case <-timeoutChan:
			u.logger.Printf("%v > %v", u.cmdPath, "timeout")
			done = true
		case output := <-u.stdoutChan:
			result = append(result, output)
			if waitFor.HasValue() && strings.Contains(output, waitFor.Value()) {
				done = true
			}
		}
	}

	return result, NilError
}

func (u *UciBinary) Close() {
	if u.cmd != nil {
		_ = u.cmd.Process.Kill()
		u.cmd = nil
	}
}
