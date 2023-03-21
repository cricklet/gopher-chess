package main

import (
	"os"
	"os/exec"

	. "github.com/cricklet/chessgo/internal/helpers"
)

func MakeDirIfMissing(dir string) Error {
	_, err := os.Stat(dir)
	if IsNil(err) {
		return NilError
	}
	err = os.Mkdir(dir, 0755)
	if !IsNil(err) {
		return Wrap(err)
	}
	return NilError
}

func RmIfExists(path string) Error {
	_, err := os.Stat(path)
	if IsNil(err) {
		return Wrap(os.Remove(path))
	}
	return NilError
}

func Exists(binaryPath string) (bool, Error) {
	_, err := os.Stat(binaryPath)
	if IsNil(err) {
		return true, NilError
	}
	if os.IsNotExist(err) {
		return false, NilError
	}
	return false, Wrap(err)
}

func BuildChessGoIfMissing(binaryPath string) Error {
	exists, err := Exists(binaryPath)
	if !IsNil(err) {
		return err
	}
	if exists {
		return NilError
	}

	logger.Println("go", "build", "-o", binaryPath, "cmd/uci/main.go")
	err = Wrap(exec.Command("go", "build", "-o", binaryPath, "cmd/uci/main.go").Run())
	if !IsNil(err) {
		return err
	}
	return NilError
}
