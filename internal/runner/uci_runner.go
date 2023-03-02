package runner

import (
	"fmt"
	"strings"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type UciRunner struct {
	Runner ChessGoRunner
}

func parseFen(input string) (string, Error) {
	s := strings.TrimPrefix(input, "position ")

	if strings.HasPrefix(s, "fen ") {
		s = strings.TrimPrefix(s, "fen ")
		return strings.Split(s, " moves ")[0], NilError
	} else if strings.HasPrefix(s, "startpos") {
		return "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", NilError
	}

	return "", Errorf("couldn't parse fen '%v'", input)
}

func parseMoves(input string) []string {
	result := []string{}
	if strings.Contains(input, " moves ") {
		fields := strings.Fields(strings.SplitN(input, " moves ", 2)[1])
		result = append(result, fields...)
	}
	return result
}

func parsePosition(input string) (Position, Error) {
	fen, err := parseFen(input)
	return Position{Fen: fen, Moves: parseMoves(input)}, err
}

func (u *UciRunner) HandleInput(input string) ([]string, Error) {
	result := []string{}
	if input == "uci" {
		result = append(result, "id name chessgo 1")
		result = append(result, "id author Kenrick Rilee")
		result = append(result, "uciok")
	} else if input == "ucinewgame" {
		u.Runner.Reset()
	} else if input == "isready" {
		result = append(result, "readyok")
	} else if input == "fen" {
		result = append(result, "position fen "+u.Runner.FenString())
	} else if input == "fullfen" {
		result = append(result, "position fen "+u.Runner.StartFen+" moves "+strings.Join(u.Runner.MoveHistory(), " "))
	} else if strings.HasPrefix(input, "position ") {
		position, err := parsePosition(input)
		if !IsNil(err) {
			return result, err
		}

		if u.Runner.IsNew() {
			err = u.Runner.SetupPosition(position)
			if !IsNil(err) {
				return result, err
			}
		} else {
			err = u.Runner.PerformMoves(position.Fen, position.Moves)
			if !IsNil(err) {
				return result, err
			}
		}
	} else if strings.HasPrefix(input, "go") {
		move, err := u.Runner.Search()
		if !IsNil(err) {
			return result, err
		}

		if move.IsEmpty() {
			result = append(result, "bestmove forfeit")
		} else {
			result = append(result, fmt.Sprintf("bestmove %v", move.Value()))
		}
	}
	return result, NilError
}
