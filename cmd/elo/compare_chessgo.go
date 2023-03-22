package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

func runCommand(cmdName string, args []string) (string, Error) {
	result, err := WrapReturn(exec.Command(cmdName, args...).Output())
	if !IsNil(err) {
		return "", err
	}

	fmt.Println(string(result))
	return string(result), err
}

func allSubDirectories(dirPath string) ([]string, Error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, Wrap(err)
	}

	result := []string{}
	for _, file := range files {
		if file.IsDir() {
			result = append(result, file.Name())
		}
	}
	return result, NilError
}

func getBinaryOptions(binaryPath string) ([]string, Error) {
	output, err := runCommand(binaryPath, []string{"options"})
	if !IsNil(err) {
		return nil, err
	}
	return FilterSlice(
		strings.Split(output, "\n"),
		func(s string) bool { return s != "" }), NilError
}

type BinaryInfo struct {
	Date    string
	Options []string
}

func marshalBinaryInfo(jsonPath string, info BinaryInfo) Error {
	output, err := json.MarshalIndent(info, "", "  ")
	if !IsNil(err) {
		return Wrap(err)
	}
	err = os.WriteFile(jsonPath, output, 0644)
	return Wrap(err)
}

func unmarshalBinaryInfo(jsonPath string, info *BinaryInfo) (bool, Error) {
	_, err := os.Stat(jsonPath)
	if !IsNil(err) {
		return false, NilError
	}
	input, err := os.ReadFile(jsonPath)
	if !IsNil(err) {
		return false, Wrap(err)
	}
	err = json.Unmarshal(input, info)
	if !IsNil(err) {
		return false, Wrap(err)
	}

	return true, NilError
}

func runTournament(binaryPath string) {
	options, err := getBinaryOptions(binaryPath)

	for i := 0; i < 10; i++ {
	}
}

var _dateFormat = "2006-01-02"

func CompareChessGo(args []string) {
	if len(args) == 0 {
		panic("missing arg")
	}

	buildsDir := RootDir() + "/data/builds"
	fmt.Println("buildsDir", buildsDir)

	err := MakeDirIfMissing(buildsDir)
	if !IsNil(err) {
		panic(err)
	}

	if args[0] == "runLatest" {
		subdirs, err := allSubDirectories(buildsDir)
		if !IsNil(err) {
			panic(err)
		}

		i := IndexOfMax(subdirs, func(subdir string) int {
			infoPath := fmt.Sprintf("%s/%s/info.json", buildsDir, subdir)
			info := BinaryInfo{}
			exists, err := unmarshalBinaryInfo(infoPath, &info)
			if !IsNil(err) {
				panic(err)
			}
			if !exists {
				panic(fmt.Errorf("info.json doesn't exist for %s", subdir))
			}
			date, err := WrapReturn(time.Parse(info.Date, _dateFormat))
			if !IsNil(err) {
				panic(err)
			}
			return int(date.Unix())
		})

		binaryDir := buildsDir + "/" + subdirs[i]
		binaryPath := fmt.Sprintf("%s/main", binaryDir)
		jsonPath := fmt.Sprintf("%s/info.json", binaryDir)
		fmt.Println("binaryPath", binaryPath)
		fmt.Println("jsonPath", jsonPath)

	}

	if args[0] == "build" || args[0] == "clean" {
		gitHash, err := runCommand("git", []string{"rev-parse", "--short", "HEAD"})
		if !IsNil(err) {
			panic(err)
		}
		gitHash = strings.TrimSpace(gitHash)

		binaryDir := buildsDir + "/" + gitHash
		fmt.Println("binaryDir", binaryDir)
		err = MakeDirIfMissing(binaryDir)
		if !IsNil(err) {
			panic(err)
		}
		binaryPath := fmt.Sprintf("%s/main", binaryDir)
		jsonPath := fmt.Sprintf("%s/info.json", binaryDir)
		fmt.Println("binaryPath", binaryPath)
		fmt.Println("jsonPath", jsonPath)

		if args[0] == "clean" {
			err = RmIfExists(jsonPath)
			if !IsNil(err) {
				panic(err)
			}
			err = RmIfExists(binaryPath)
			if !IsNil(err) {
				panic(err)
			}
			err = RmIfExists(binaryDir)
			if !IsNil(err) {
				panic(err)
			}
			return
		}

		info := BinaryInfo{}
		foundInfo, err := unmarshalBinaryInfo(jsonPath, &info)
		if !IsNil(err) {
			panic(err)
		}

		if foundInfo {
			exists, err := Exists(binaryPath)
			if !IsNil(err) {
				panic(err)
			} else if !exists {
				panic("info.json exists but binary doesn't")
			} else {
				fmt.Println("already built")
				fmt.Println("date:", info.Date)
				fmt.Println("options:", info.Options)
			}
			return
		} else {
			BuildChessGoIfMissing(binaryPath)
			fmt.Println("built")

			info.Date = time.Now().Format(_dateFormat)
			info.Options, err = getBinaryOptions(binaryPath)
			if !IsNil(err) {
				panic(err)
			}
			fmt.Println("date:", info.Date)
			fmt.Println("options:", info.Options)

			err := marshalBinaryInfo(jsonPath, info)
			if !IsNil(err) {
				panic(err)
			}
		}
	}
}
