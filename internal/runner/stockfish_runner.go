package runner

import (
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/binary_runner"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type StockfishRunner struct {
	Logger Logger
	binary *BinaryRunner

	startFen string
	moves    []string
}

var _ Runner = (*StockfishRunner)(nil)

func (r *StockfishRunner) SetupPosition(position Position) Error {
	var err Error

	if r.binary == nil {
		r.binary, err = SetupBinaryRunner("stockfish", time.Millisecond*500)
		if !IsNil(err) {
			return err
		}
	}

	var output []string

	output, err = r.binary.Run("isready", Some("readyok"))
	if !IsNil(err) {
		return err
	}
	if !Contains(output, "readyok") {
		return Errorf("needs readyok")
	}

	output, err = r.binary.Run("uci", Some("uciok"))
	if !IsNil(err) {
		return err
	}
	if !Contains(output, "uciok") {
		return Errorf("needs uciok")
	}

	r.startFen = position.Fen
	r.moves = position.Moves
	err = r.binary.RunAsync("position fen " + position.Fen + " moves " + strings.Join(position.Moves, " "))
	if !IsNil(err) {
		return err
	}

	return NilError
}

func (r *StockfishRunner) Reset() {
	if r.binary != nil {
		r.binary.Close()

		r.binary = nil
		r.startFen = ""
		r.moves = []string{}
	}
}

func (r *StockfishRunner) IsNew() bool {
	return r.binary == nil
}

func (r *StockfishRunner) PerformMoves(fen string, moves []string) Error {
	if fen != r.startFen {
		return Errorf("fen %s does not match start fen %s", fen, r.startFen)
	}

	err := r.binary.RunAsync("position fen " + fen + " moves " + strings.Join(moves, " "))
	r.moves = moves

	if !IsNil(err) {
		return err
	}
	return NilError
}

func (r *StockfishRunner) PerformMoveFromString(s string) Error {
	r.moves = append(r.moves, s)
	err := r.binary.RunAsync("position " + r.startFen + " moves " + strings.Join(r.moves, " ") + " " + s)

	if !IsNil(err) {
		return err
	}
	return NilError
}

func (r *StockfishRunner) MovesForSelection(selection string) ([]string, Error) {
	return []string{}, Errorf("not implemented")
}

func (r *StockfishRunner) Rewind(num int) Error {
	return Errorf("not implemented")
}

func (r *StockfishRunner) Search() (Optional[string], Error) {
	var err Error
	var result []string
	err = r.binary.RunAsync("go")
	if !IsNil(err) {
		return Empty[string](), err
	}
	time.Sleep(100 * time.Millisecond)
	result, err = r.binary.Run("stop", Some("bestmove"))
	if !IsNil(err) {
		return Empty[string](), err
	}

	bestMoveString := FindInSlice(result, func(v string) bool {
		return strings.HasPrefix(v, "bestmove ")
	})

	if bestMoveString.HasValue() {
		return Some(strings.Split(bestMoveString.Value(), " ")[1]), NilError
	}

	return Empty[string](), NilError
}
