package stockfish

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cricklet/chessgo/internal/binary"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type StockfishRunner struct {
	logger Logger
	binary *binary.BinaryRunner

	elo      Optional[int]
	startFen string
	moves    []string
}

type StockfishRunnerOption func(*StockfishRunner)

func WithElo(elo int) StockfishRunnerOption {
	return func(r *StockfishRunner) {
		r.elo = Some(elo)
	}
}
func WithLogger(logger Logger) StockfishRunnerOption {
	return func(r *StockfishRunner) {
		r.logger = logger
	}
}

func NewStockfishRunner(options ...StockfishRunnerOption) *StockfishRunner {
	r := &StockfishRunner{}
	for _, o := range options {
		o(r)
	}
	if r.logger == nil {
		r.logger = &DefaultLogger
	}

	return r
}

var _ Runner = (*StockfishRunner)(nil)

func (r *StockfishRunner) SetupPosition(position Position) Error {
	var err Error

	if r.binary == nil {
		r.binary, err = binary.SetupBinaryRunner("stockfish", "stockfish", []string{}, 1000*time.Millisecond)
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

	err = r.binary.RunAsync("ucinewgame")
	if !IsNil(err) {
		return err
	}

	if r.elo.HasValue() && r.elo.Value() > 0 {
		err = r.binary.RunAsync("setoption name UCI_LimitStrength value true")
		if !IsNil(err) {
			return err
		}

		err = r.binary.RunAsync(fmt.Sprintf("setoption name UCI_Elo value %v", r.elo.Value()))
		if !IsNil(err) {
			return err
		}
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

func StockfishEvalFromGo(output []string) Optional[int] {
	centipawnScores := FilterSlice(output, func(v string) bool {
		return strings.Contains(v, "score cp ")
	})
	if len(centipawnScores) > 0 {
		centipawnScoreStr := Last(centipawnScores)
		centipawnScoreStr = strings.Split(
			strings.Split(centipawnScoreStr, "score cp ")[1],
			" ")[0]
		centipawnScore, err := WrapReturn(strconv.Atoi(centipawnScoreStr))
		if !IsNil(err) {
			return Empty[int]()
		}
		return Some(centipawnScore)
	}
	return Empty[int]()
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
