package binary

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"

	"github.com/stretchr/testify/assert"
)

func TestTee(t *testing.T) {
	runner, err := SetupBinaryRunner("tee", "tee", []string{})
	assert.True(t, err.IsNil())

	for i := 0; i < 10; i++ {
		v := fmt.Sprintf("hello world %d", i)
		err = runner.RunSync(v, func(line string) (LoopResult, Error) {
			assert.Equal(t, v, line)
			return LoopBreak, NilError
		}, Some(time.Second))

		assert.True(t, err.IsNil())
	}

	err = runner.RunSync("hello world", func(line string) (LoopResult, Error) {
		assert.Equal(t, "hello world", line)
		return LoopBreak, NilError
	}, Empty[time.Duration]())

	assert.True(t, err.IsNil())

	runner.Close()
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, 0, runner.openGoRoutines)
}
func TestFixWithMultiWriterCopy(t *testing.T) {
	cmd := exec.Command("tee")

	stdinWriter, err := cmd.StdinPipe()
	assert.True(t, err == nil, err)

	stdoutReader, err := cmd.StdoutPipe()
	assert.True(t, err == nil, err)

	stdoutLoggingReader, stdoutLoggingWriter := io.Pipe()
	stdoutSyncReader, stdoutSyncWriter := io.Pipe()

	stdoutCombinedWriter := io.MultiWriter(stdoutLoggingWriter, stdoutSyncWriter)

	stdoutDoneCopying := make(chan bool)
	go func() {
		// This is necessary: https://stackoverflow.com/questions/47486128/why-does-io-pipe-continue-to-block-even-when-eof-is-reached
		defer stdoutLoggingWriter.Close()
		defer stdoutSyncWriter.Close()

		_, err = io.Copy(stdoutCombinedWriter, stdoutReader)
		assert.True(t, err == nil, err)
		fmt.Println("done copying")
		stdoutDoneCopying <- true
	}()

	stdoutLog := []string{}
	stdoutLoggingFinished := make(chan bool)

	go func() {
		reader := bufio.NewReader(stdoutLoggingReader)
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			stdoutLog = append(stdoutLog, scanner.Text())
		}
		fmt.Println("done logging")
		stdoutLoggingFinished <- true
	}()

	err = cmd.Start()
	assert.True(t, err == nil, err)

	{
		reader := bufio.NewReader(stdoutSyncReader)
		scanner := bufio.NewScanner(reader)

		for i := 0; i < 100; i++ {
			v := fmt.Sprintf("hello %d", i)
			_, err = stdinWriter.Write([]byte(v + "\n"))
			assert.True(t, err == nil, err)

			time.Sleep(time.Millisecond * 10)

			assert.Equal(t, Last(stdoutLog), v)

			for scanner.Scan() {
				output := scanner.Text()
				assert.Equal(t, v, output)

				if v == output {
					break
				}
			}
		}

		fmt.Println("successfully wrote to tee")
	}

	err = cmd.Process.Signal(syscall.SIGTERM)
	assert.True(t, err == nil, err)

	err = cmd.Wait()
	assert.Equal(t, err.Error(), "signal: terminated")

	assert.True(t, <-stdoutDoneCopying)
	assert.True(t, <-stdoutLoggingFinished)
}

func TestFixWithCopy(t *testing.T) {
	cmd := exec.Command("tee")

	stdinWriter, err := cmd.StdinPipe()
	assert.True(t, err == nil, err)

	stdoutReader, err := cmd.StdoutPipe()
	assert.True(t, err == nil, err)

	stdoutLoggingReader, stdoutLoggingWriter := io.Pipe()

	stdoutDoneCopying := make(chan bool)
	go func() {
		// This is necessary: https://stackoverflow.com/questions/47486128/why-does-io-pipe-continue-to-block-even-when-eof-is-reached
		defer stdoutLoggingWriter.Close()

		_, err = io.Copy(stdoutLoggingWriter, stdoutReader)
		assert.True(t, err == nil, err)
		fmt.Println("done copying")
		stdoutDoneCopying <- true
	}()

	stdoutLog := []string{}
	stdoutLoggingFinished := make(chan bool)

	go func() {
		reader := bufio.NewReader(stdoutLoggingReader)
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			stdoutLog = append(stdoutLog, scanner.Text())
		}
		fmt.Println("done logging")
		stdoutLoggingFinished <- true
	}()

	err = cmd.Start()
	assert.True(t, err == nil, err)

	{
		for i := 0; i < 100; i++ {
			v := fmt.Sprintf("hello %d", i)
			_, err = stdinWriter.Write([]byte(v + "\n"))
			assert.True(t, err == nil, err)

			time.Sleep(time.Millisecond * 10)

			assert.Equal(t, Last(stdoutLog), v)
		}

		fmt.Println("successfully wrote to tee")
	}

	err = cmd.Process.Kill()
	assert.True(t, err == nil, err)

	fmt.Println("killed")

	assert.True(t, <-stdoutDoneCopying)
	assert.True(t, <-stdoutLoggingFinished)
}

func TestFixWithoutCopy(t *testing.T) {
	cmd := exec.Command("tee")

	stdinWriter, err := cmd.StdinPipe()
	assert.True(t, err == nil, err)

	stdoutReader, err := cmd.StdoutPipe()
	assert.True(t, err == nil, err)

	stdoutLog := []string{}
	stdoutLoggingFinished := make(chan bool)

	go func() {
		reader := bufio.NewReader(stdoutReader)
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			stdoutLog = append(stdoutLog, scanner.Text())
		}
		fmt.Println("done logging")
		stdoutLoggingFinished <- true
	}()

	err = cmd.Start()
	assert.True(t, err == nil, err)

	{
		for i := 0; i < 100; i++ {
			v := fmt.Sprintf("hello %d", i)
			_, err = stdinWriter.Write([]byte(v + "\n"))
			assert.True(t, err == nil, err)

			time.Sleep(time.Millisecond * 10)

			assert.Equal(t, Last(stdoutLog), v)
		}

		fmt.Println("successfully wrote to tee")
	}

	err = cmd.Process.Kill()
	assert.True(t, err == nil, err)

	fmt.Println("killed")

	assert.True(t, <-stdoutLoggingFinished)
}
