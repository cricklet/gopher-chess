package uci

import (
	"errors"
	"fmt"
	"strings"

	. "github.com/cricklet/chessgo/internal/runner"
)

type UciRunner struct {
	Runner Runner
}

func parseFen(input string) string {
	s := strings.TrimPrefix(input, "position ")

	if strings.HasPrefix(s, "fen ") {
		s = strings.TrimPrefix(s, "fen ")
		return strings.Split(s, " moves ")[0]
	} else if strings.HasPrefix(s, "startpos") {
		return "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}

	panic(fmt.Errorf("couldn't parse '%v'", s))
}

func parseMoves(input string) []string {
	result := []string{}
	if strings.Contains(input, " moves ") {
		fields := strings.Fields(strings.SplitN(input, " moves ", 2)[1])
		result = append(result, fields...)
	}
	return result
}

func parsePosition(input string) Position {
	return Position{Fen: parseFen(input), Moves: parseMoves(input)}
}

func (u *UciRunner) HandleInput(input string) ([]string, error) {
	result := []string{}
	if input == "uci" {
		result = append(result, "id name chessgo 1")
		result = append(result, "id author Kenrick Rilee")
		result = append(result, "uciok")
	} else if input == "ucinewgame" {
		u.Runner.Reset()
	} else if input == "isready" {
		result = append(result, "readyok")
	} else if strings.HasPrefix(input, "position ") {
		position := parsePosition(input)
		if u.Runner.IsNew() {
			err := u.Runner.SetupPosition(position)
			if err != nil {
				return result, err
			}
		} else {
			err := u.Runner.PerformMoves(position.Fen, position.Moves)
			if err != nil {
				return result, err
			}
		}
	} else if strings.HasPrefix(input, "go") {
		move, err := u.Runner.Search()
		if err != nil {
			return result, err
		}

		if move.IsEmpty() {
			return result, errors.New("no legal moves")
		}

		result = append(result, fmt.Sprintf("bestmove %v", move.Value()))
	}
	return result, nil
}
