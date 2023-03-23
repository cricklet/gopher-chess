package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cricklet/chessgo/internal/binary"
	"github.com/cricklet/chessgo/internal/chessgo"
	. "github.com/cricklet/chessgo/internal/helpers"
	elo "github.com/kortemy/elo-go"
)

func runCommand(cmdName string, args []string) (string, Error) {
	result, err := WrapReturn(exec.Command(cmdName, args...).Output())
	if !IsNil(err) {
		return "", err
	}

	logger.Println(string(result))
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
	return append([]string{""}, FilterSlice(
		strings.Split(output, "\n"),
		func(s string) bool { return s != "" })...), NilError
}

type BinaryInfo struct {
	Date    string   `json:"date"`
	Options []string `json:"options"`
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

func setupChessGoRunner(binaryPath string, options string, fen string) (*binary.BinaryRunner, Error) {
	var err Error

	var player *binary.BinaryRunner
	name := fmt.Sprintf("chessgo (%v)", options)
	logger := FuncLogger(func(s string) { logger.Println(name, ">", Indent(s, "$ ")) })
	player, err = binary.SetupBinaryRunner(binaryPath, "chessgo", strings.Split(options, " "),
		time.Millisecond*10000, binary.WithLogger(logger))
	if !IsNil(err) {
		return nil, err
	}

	Run(player, "isready", Some("readyok"))
	Run(player, "uci", Some("uciok"))
	RunAsync(player, "ucinewgame")
	RunAsync(player, "position fen "+fen)

	return player, err
}

func runGame(binaryPath string, opt1 string, opt2 string) (float32, Error) {
	evaluator, err := NewEvaluator()
	if !IsNil(err) {
		return 0.5, err
	}

	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	var player1 *binary.BinaryRunner
	player1, err = setupChessGoRunner(binaryPath, opt1, fen)
	if !IsNil(err) {
		return 0.5, err
	}
	defer player1.Close()

	var player2 *binary.BinaryRunner
	player2, err = setupChessGoRunner(binaryPath, opt2, fen)
	if !IsNil(err) {
		return 0.5, err
	}
	defer player2.Close()

	runner := chessgo.NewChessGoRunner()
	err = runner.SetupPosition(Position{
		Fen:   fen,
		Moves: []string{},
	})
	if !IsNil(err) {
		panic(err)
	}

	result, err := PlayBinaries(player1, player2, &runner, func() {
		player := runner.Player()

		pgnString := fmt.Sprintf("%v\n%v", runner.PgnFromMoveHistory(), runner.FenString())
		logger.SetFooter(HintText(pgnString), _footerPgn)
		logger.SetFooter(runner.Board().Unicode(), _footerBoard)

		score, err := evaluator.Evaluate(runner.FenString())
		if !IsNil(err) {
			panic(err)
		}

		if player == Black {
			score = -score
		}

		logger.SetFooter(HintText(fmt.Sprintf("eval: %v, piece: %v",
			score,
			runner.EvaluateSimple(White))),
			_footerEval)
	})
	if !IsNil(err) {
		return 0.5, err
	}

	return result, err
}

type matchResult struct {
	WhiteBinary string  `json:"whiteBinary"`
	WhiteOpts   string  `json:"whiteOpts"`
	BlackBinary string  `json:"blackBinary"`
	BlackOpts   string  `json:"blackOpts"`
	Result      float32 `json:"result"`
}

type binaryDefinition struct {
	BinaryPath string `json:"binaryPath"`
	Options    string `json:"options"`
}

type estimatedElo struct {
	CmdPath string `json:"cmdPath"`
	Options string `json:"options"`
	Elo     int    `json:"elo"`
}

type tournamentResults struct {
	Matches      []matchResult  `json:"matches"`
	Participants []estimatedElo `json:"participants"`
}

type updateTournamentResults interface {
	Update(result matchResult) Error
}

type JsonTournamentResults struct {
	jsonPath string
}

func unmarshalTournamentResults(jsonPath string, results *tournamentResults) (bool, Error) {
	_, err := os.Stat(jsonPath)
	if !IsNil(err) {
		return false, NilError
	}
	input, err := os.ReadFile(jsonPath)
	if !IsNil(err) {
		return false, Wrap(err)
	}
	err = json.Unmarshal(input, results)
	if !IsNil(err) {
		return false, Wrap(err)
	}

	return true, NilError
}

func marshalTournamentResults(jsonPath string, results *tournamentResults) Error {
	output, err := json.MarshalIndent(results, "", "  ")
	if !IsNil(err) {
		return Wrap(err)
	}
	err = os.WriteFile(jsonPath, output, 0644)
	return Wrap(err)
}

func (j *JsonTournamentResults) Update(result matchResult) Error {
	results := tournamentResults{}
	_, err := unmarshalTournamentResults(j.jsonPath, &results)
	if !IsNil(err) {
		return err
	}

	results.Matches = append(results.Matches, result)
	results.Participants = []estimatedElo{}

	elos := map[binaryDefinition]int{}
	e := elo.NewElo()
	for _, match := range results.Matches {
		white := binaryDefinition{match.WhiteBinary, match.WhiteOpts}
		black := binaryDefinition{match.BlackBinary, match.BlackOpts}
		elo1 := GetWithDefault(elos, white, 800)
		elo2 := GetWithDefault(elos, black, 800)
		outcome1, outcome2 := e.Outcome(elo1, elo2, float64(match.Result))

		elos[white] = outcome1.Rating
		elos[black] = outcome2.Rating
	}

	for binary, elo := range elos {
		results.Participants = append(results.Participants, estimatedElo{
			CmdPath: binary.BinaryPath,
			Options: binary.Options,
			Elo:     elo,
		})
	}

	return marshalTournamentResults(j.jsonPath, &results)
}

func runTournament(binaryPath string, updater updateTournamentResults) Error {
	options, err := getBinaryOptions(binaryPath)
	if !IsNil(err) {
		return err
	}

	for i := 0; i < 100; i++ {
		for _, opt1 := range options {
			for _, opt2 := range options {
				if opt1 == opt2 {
					continue
				}
				result, err := runGame(binaryPath, opt1, opt2)
				if !IsNil(err) {
					return err
				}

				err = updater.Update(matchResult{
					WhiteBinary: binaryPath,
					WhiteOpts:   opt1,
					BlackBinary: binaryPath,
					BlackOpts:   opt2,
					Result:      result,
				})
				if !IsNil(err) {
					return err
				}
			}
		}
	}

	return NilError
}

var _dateFormat = "2006-01-02 15:04:05"

func CompareChessGo(args []string) {
	if len(args) == 0 {
		panic("missing arg")
	}

	buildsDir := RootDir() + "/data/builds"
	logger.Println("buildsDir", buildsDir)

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
			date, err := WrapReturn(time.Parse(_dateFormat, info.Date))
			if !IsNil(err) {
				panic(err)
			}
			return int(date.Unix())
		})

		binaryDir := buildsDir + "/" + subdirs[i]
		binaryPath := fmt.Sprintf("%s/main", binaryDir)
		logger.Println("binaryPath", binaryPath)

		hostName, err := GetHostName()
		if !IsNil(err) {
			panic(err)
		}
		jsonPath := fmt.Sprintf("%s/tournament_%s.json", binaryDir, hostName)

		err = runTournament(binaryPath, &JsonTournamentResults{jsonPath: jsonPath})
		if !IsNil(err) {
			panic(err)
		}
	}

	if args[0] == "build" || args[0] == "clean" {
		gitHash, err := runCommand("git", []string{"rev-parse", "--short", "HEAD"})
		if !IsNil(err) {
			panic(err)
		}
		gitHash = strings.TrimSpace(gitHash)

		binaryDir := buildsDir + "/" + gitHash
		logger.Println("binaryDir", binaryDir)
		err = MakeDirIfMissing(binaryDir)
		if !IsNil(err) {
			panic(err)
		}
		binaryPath := fmt.Sprintf("%s/main", binaryDir)
		jsonPath := fmt.Sprintf("%s/info.json", binaryDir)
		logger.Println("binaryPath", binaryPath)
		logger.Println("jsonPath", jsonPath)

		if args[0] == "clean" {
			err = RmIfExists(jsonPath)
			if !IsNil(err) {
				panic(err)
			}
			err = RmIfExists(binaryPath)
			if !IsNil(err) {
				panic(err)
			}
			hostName, err := GetHostName()
			if !IsNil(err) {
				panic(err)
			}
			resultsPath := fmt.Sprintf("%s/tournament_%s.json", binaryDir, hostName)
			err = RmIfExists(resultsPath)
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
				logger.Println("already built")
				logger.Println("date:", info.Date)
				logger.Println("options:", info.Options)
			}
			return
		} else {
			err := BuildChessGoIfMissing(binaryPath)
			if !IsNil(err) {
				panic(err)
			}
			logger.Println("built")

			info.Date = time.Now().Format(_dateFormat)
			info.Options, err = getBinaryOptions(binaryPath)
			if !IsNil(err) {
				panic(err)
			}
			logger.Println("date:", info.Date)
			logger.Println("options:", info.Options)

			err = marshalBinaryInfo(jsonPath, info)
			if !IsNil(err) {
				panic(err)
			}
		}
	}
}
