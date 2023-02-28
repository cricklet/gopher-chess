package runner

import (
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type StockfishRunner struct {
	Logger Logger
	binary *uciBinary

	startFen string
	moves    []string
}

var _ Runner = (*StockfishRunner)(nil)

func (r *StockfishRunner) SetupPosition(position Position) error {
	var err error

	if r.binary == nil {
		r.binary, err = SetupWithDefaultLogger("stockfish", time.Millisecond*500)
		if err != nil {
			return err
		}
	}

	var output []string

	output, err = r.binary.Run("isready", Some("readyok"))
	if err != nil {
		return err
	}
	if !Contains(output, "readyok") {
		return errors.New("needs readyok")
	}

	output, err = r.binary.Run("uci", Some("uciok"))
	if err != nil {
		return err
	}
	if !Contains(output, "uciok") {
		return errors.New("needs uciok")
	}

	r.startFen = position.Fen
	r.moves = position.Moves
	err = r.binary.RunAsync("position fen " + position.Fen + " moves " + strings.Join(position.Moves, " "))
	if err != nil {
		return err
	}

	return nil
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

func (r *StockfishRunner) PerformMoves(fen string, moves []string) error {
	if fen != r.startFen {
		return fmt.Errorf("fen %s does not match start fen %s", fen, r.startFen)
	}

	err := r.binary.RunAsync("position fen " + fen + " moves " + strings.Join(moves, " "))
	r.moves = moves

	if err != nil {
		return err
	}
	return nil
}

func (r *StockfishRunner) PerformMoveFromString(s string) error {
	r.moves = append(r.moves, s)
	err := r.binary.RunAsync("position " + r.startFen + " moves " + strings.Join(r.moves, " ") + " " + s)

	if err != nil {
		return err
	}
	return nil
}

func (r *StockfishRunner) MovesForSelection(selection string) ([]string, error) {
	return []string{}, errors.New("not implemented")
}

func (r *StockfishRunner) Rewind(num int) error {
	return errors.New("not implemented")
}

func (r *StockfishRunner) Search() (Optional[string], error) {
	var err error
	var result []string
	err = r.binary.RunAsync("go")
	if err != nil {
		return Empty[string](), err
	}
	time.Sleep(100 * time.Millisecond)
	result, err = r.binary.Run("stop", Some("bestmove"))
	if err != nil {
		return Empty[string](), err
	}

	bestMoveString := FindInSlice(result, func(v string) bool {
		return strings.HasPrefix(v, "bestmove ")
	})

	if bestMoveString.HasValue() {
		return Some(strings.Split(bestMoveString.Value(), " ")[1]), nil
	}

	return Empty[string](), nil
}
