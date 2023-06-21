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
	cmdName string
	cmd     *exec.Cmd

	stdin ReadableWriter

	stdout StdOutBuffer

	record []string

	Logger Logger
}

type BinaryRunnerOption func(*BinaryRunner)

func (b *BinaryRunner) CmdPath() string {
	return b.cmdPath
}

func (b *BinaryRunner) CmdName() string {
	return b.cmdName
}

func WithLogger(logger Logger) BinaryRunnerOption {
	return func(u *BinaryRunner) {
		u.Logger = logger
	}
}

func (u *BinaryRunner) flush(indent string) string {
	return Indent(strings.Join(u.record, "\n"), indent)
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

func SetupBinaryRunner(cmdPath string, cmdName string, args []string, options ...BinaryRunnerOption) (*BinaryRunner, Error) {
	u := &BinaryRunner{
		cmdPath: cmdPath,
		cmdName: cmdName,
	}

	for _, option := range options {
		option(u)
	}

	if u.Logger == nil {
		u.Logger = &DefaultLogger
	}

	if u.cmd == nil {
		u.Logger.Println(cmdPath, args)
		u.cmd = exec.Command(cmdPath, args...)

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
				u.Logger.Println("stdin: ", line)
				u.record = AppendSafe(&recordLock, u.record, "in:  "+strings.TrimSpace(line))
			}
		}()

		u.stdout = StdOutBuffer{}

		avoidSpam := func(line string) bool {
			if strings.Contains(line, "multipv") && !strings.Contains(line, "multipv 1 ") {
				return true
			}
			if strings.Contains(line, "currmove") {
				return true
			}
			return false
		}

		go func() {
			stdoutScanner := bufio.NewScanner(bufio.NewReader(stdout))
			for stdoutScanner.Scan() {
				output := stdoutScanner.Text()
				for _, line := range strings.Split(output, "\n") {
					if !avoidSpam(line) {
						u.Logger.Println("stdout: ", Ellipses(line, 140))
					}

					u.record = AppendSafe(&recordLock, u.record, "out: "+line)
					u.stdout.Update(line)
				}
			}
		}()

		go func() {
			stderrScanner := bufio.NewScanner(bufio.NewReader(stderr))
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				u.record = AppendSafe(&recordLock, u.record, "err: "+line)
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

func (u *BinaryRunner) RunSync(input string, callback func(string) (LoopResult, Error), timeout Optional[time.Duration]) Error {
	err := u.RunAsync(input)
	if !IsNil(err) {
		return err
	}

	done := false

	timeoutChan := make(chan bool)
	go func() {
		if timeout.HasValue() {
			time.Sleep(timeout.Value())
			AsyncSend(&timeoutChan, true)
		} else {
			time.Sleep(time.Second * 10)
		}
		if !done {
			fmt.Println("possible timeout detected")
		}
	}()

	handleLine := func(line string) Error {
		result, err := callback(line)
		if result == LoopBreak {
			done = true
		}
		return err
	}

	for !done {
		select {
		case <-timeoutChan:
			err = u.stdout.Flush(func(line string) Error {
				_, err = callback(line)
				return err
			})
			u.Logger.Println("timeout")
			done = true
		case <-u.stdout.Wait():
			err = u.stdout.Flush(handleLine)
		}

		if !IsNil(err) {
			return err
		}
	}

	if !IsNil(err) {
		return err
	}

	return NilError
}

func (u *BinaryRunner) Run(input string, waitFor Optional[string]) ([]string, Error) {
	result := []string{}

	foundOutput := false

	err := u.RunSync(input, func(line string) (LoopResult, Error) {
		result = append(result, line)

		if waitFor.HasValue() && strings.Contains(line, waitFor.Value()) {
			foundOutput = true
			return LoopBreak, NilError
		}
		return LoopContinue, NilError
	}, Some(time.Second))

	if !IsNil(err) {
		return result, err
	}

	if waitFor.HasValue() && !foundOutput {
		return result, wrapError(u, fmt.Errorf("timeout waiting for %v", waitFor.Value()))
	}

	return result, NilError
}

func (u *BinaryRunner) Wait() {
	if u.cmd != nil {
		_ = u.cmd.Wait()
		u.cmd = nil
	}
}

func (u *BinaryRunner) Close() {
	if u.cmd != nil {
		_ = u.cmd.Process.Kill()
		u.cmd = nil
	}
}
