package runner

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type uciBinary struct {
	cmdPath string
	cmd     *exec.Cmd
	stdin   io.Writer

	stdoutChan chan string
	stderrChan chan string

	timeout time.Duration
	logger  Logger
}

func SetupWithDefaultLogger(cmdPath string, timeout time.Duration) (*uciBinary, error) {
	return Setup(cmdPath, timeout, &DefaultLogger)
}

func Setup(cmdPath string, delay time.Duration, logger Logger) (*uciBinary, error) {
	var err error

	u := &uciBinary{
		cmdPath: cmdPath,
		timeout: delay,
		logger:  logger,
	}

	if u.cmd == nil {
		u.cmd = exec.Command(cmdPath)
		u.stdin, err = u.cmd.StdinPipe()
		if err != nil {
			return u, err
		}
		var stdout io.Reader
		var stderr io.Reader
		stdout, err = u.cmd.StdoutPipe()
		if err != nil {
			return u, err
		}
		stderr, err = u.cmd.StderrPipe()
		if err != nil {
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

		err = u.cmd.Start()
		if err != nil {
			return u, err
		}
	}

	return u, nil
}

func (u *uciBinary) RunAsync(input string) error {
	if u.cmd == nil || u.stdin == nil {
		return fmt.Errorf("cmd not setup: %v", u.cmdPath)
	}

	if u.cmd.ProcessState != nil && u.cmd.ProcessState.Exited() {
		return fmt.Errorf("cmd exited: %v", u.cmdPath)
	}

	_, err := u.stdin.Write([]byte(input + "\n"))
	if err != nil {
		return err
	}

	return nil
}

func (u *uciBinary) Run(input string, waitFor Optional[string]) ([]string, error) {
	result := []string{}

	err := u.RunAsync(input)
	if err != nil {
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

	return result, nil
}

func (u *uciBinary) Close() {
	if u.cmd != nil {
		_ = u.cmd.Process.Kill()
		u.cmd = nil
	}
}
