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

	stdoutScanner *bufio.Scanner

	record []string

	Logger Logger
}

type BinaryRunnerOption func(*BinaryRunner)

func (b *BinaryRunner) Close() {
	if b.cmd != nil {
		_ = b.cmd.Process.Kill()
		b.cmd = nil
	}
	b.stdin.Close()
}

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

		var err Error
		u.stdin.Writer, err = WrapReturn(u.cmd.StdinPipe())
		u.stdin.ReadChan = make(chan string)
		if !IsNil(err) {
			return u, wrapError(u, err)
		}

		stdoutLoggingReader, stdoutLoggingWriter := io.Pipe()
		stdoutSyncReader, stdoutSyncWriter := io.Pipe()
		stdoutCombinedWriter := io.MultiWriter(stdoutLoggingWriter, stdoutSyncWriter)

		stdoutReader, err := WrapReturn(u.cmd.StdoutPipe())
		if err.HasError() {
			return u, err
		}

		go func() {
			// This is necessary: https://stackoverflow.com/questions/47486128/why-does-io-pipe-continue-to-block-even-when-eof-is-reached
			defer stdoutLoggingWriter.Close()
			defer stdoutSyncReader.Close()

			_, err = WrapReturn(io.Copy(stdoutCombinedWriter, stdoutReader))
			if err.HasError() {
				panic("failed to copy data from stdout")
			}
		}()

		stderrReader, err := WrapReturn(u.cmd.StderrPipe())
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

		avoidSpam := func(line string) bool {
			if strings.Contains(line, "multipv") && !strings.Contains(line, "multipv 1 ") {
				return true
			}
			if strings.Contains(line, "currmove") {
				return true
			}
			return false
		}

		stdoutLoggingScanner := bufio.NewScanner(bufio.NewReader(stdoutLoggingReader))
		go func() {
			for stdoutLoggingScanner.Scan() {
				output := stdoutLoggingScanner.Text()
				for _, line := range strings.Split(output, "\n") {
					if !avoidSpam(line) {
						u.Logger.Println("stdout: ", Ellipses(line, 140))
					}

					u.record = AppendSafe(&recordLock, u.record, "out: "+line)
				}
			}
		}()

		stderrScanner := bufio.NewScanner(bufio.NewReader(stderrReader))
		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				u.record = AppendSafe(&recordLock, u.record, "err: "+line)
			}
		}()

		u.stdoutScanner = bufio.NewScanner(bufio.NewReader(stdoutSyncReader))

		err = Wrap(u.cmd.Start())
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

	for u.stdoutScanner.Scan() {
		line := u.stdoutScanner.Text()
		result, err := callback(line)
		if !IsNil(err) {
			return err
		}

		if result == LoopBreak {
			break
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
