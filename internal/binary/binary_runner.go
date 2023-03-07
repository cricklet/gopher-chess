package binary

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type BinaryRunner struct {
	cmdPath string
	cmd     *exec.Cmd

	stdin      ReadableWriter
	stdoutChan chan string

	stdRecord []string

	timeout time.Duration
	Logger  Logger
}

type BinaryRunnerOption func(*BinaryRunner)

func (b *BinaryRunner) CmdPath() string {
	return b.cmdPath
}

func (b *BinaryRunner) CmdName() string {
	return Last(strings.Split(Last(strings.Split(b.cmdPath, "/")), "_"))
}

func WithLogger(logger Logger) BinaryRunnerOption {
	return func(u *BinaryRunner) {
		u.Logger = logger
	}
}

func (u *BinaryRunner) flush(indent string) string {
	return Indent(strings.Join(u.stdRecord, "\n"), indent)
}

func (u *BinaryRunner) Flush() string {
	return "> " + u.flush("> ")
}

func wrapError(u *BinaryRunner, err error) Error {
	if !IsNil(err) {
		return Wrap(fmt.Errorf("%w\n.  %v\n", err, u.flush(".  ")))
	}
	return NilError
}

func SetupBinaryRunner(cmdPath string, delay time.Duration, options ...BinaryRunnerOption) (*BinaryRunner, Error) {
	u := &BinaryRunner{
		cmdPath: cmdPath,
		timeout: delay,
	}

	for _, option := range options {
		option(u)
	}

	if u.Logger == nil {
		u.Logger = &DefaultLogger
	}

	if u.cmd == nil {
		u.cmd = exec.Command(cmdPath)

		var err error
		u.stdin.Writer, err = u.cmd.StdinPipe()
		u.stdin.ReadChan = make(chan string)
		if !IsNil(err) {
			return u, wrapError(u, err)
		}

		var stdout io.Reader
		var stderr io.Reader
		stdout, err = u.cmd.StdoutPipe()
		if !IsNil(err) {
			return u, wrapError(u, err)
		}
		stderr, err = u.cmd.StderrPipe()
		if !IsNil(err) {
			return u, wrapError(u, err)
		}

		recordLock := sync.Mutex{}

		go func() {
			for {
				line := <-u.stdin.ReadChan
				u.stdRecord = AppendSafe(&recordLock, u.stdRecord, "in:  "+strings.TrimSpace(line))
			}
		}()

		u.stdoutChan = make(chan string)
		go func() {
			stdoutScanner := bufio.NewScanner(bufio.NewReader(stdout))
			for stdoutScanner.Scan() {
				line := stdoutScanner.Text()
				u.stdRecord = AppendSafe(&recordLock, u.stdRecord, "out: "+line)
				u.stdoutChan <- line
			}
		}()

		go func() {
			stderrScanner := bufio.NewScanner(bufio.NewReader(stderr))
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				u.stdRecord = AppendSafe(&recordLock, u.stdRecord, "err: "+line)
			}
		}()

		err = u.cmd.Start()
		if !IsNil(err) {
			return u, wrapError(u, err)
		}
	}

	return u, NilError
}

func (u *BinaryRunner) RunAsync(input string) Error {
	if u.cmd == nil {
		return wrapError(u, Errorf("cmd not setup: %v\n", u.cmdPath))
	}

	if u.cmd.ProcessState != nil && u.cmd.ProcessState.Exited() {
		return wrapError(u, Errorf("cmd exited: %v\n", u.cmdPath))
	}

	_, err := u.stdin.Write([]byte(input + "\n"))
	if !IsNil(err) {
		return wrapError(u, err)
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
			u.Logger.Printf("%v", "timeout")
			done = true
		case output := <-u.stdoutChan:
			u.Logger.Printf("%v", output)
			result = append(result, output)
			if waitFor.HasValue() && strings.Contains(output, waitFor.Value()) {
				foundOutput = true
				done = true
			}
		}
	}

	if waitFor.HasValue() && !foundOutput {
		return result, wrapError(u, fmt.Errorf("timeout waiting for %v", waitFor.Value()))
	}

	return result, NilError
}

func (u *BinaryRunner) Close() {
	if u.cmd != nil {
		_ = u.cmd.Process.Kill()
		u.cmd = nil
	}
}
