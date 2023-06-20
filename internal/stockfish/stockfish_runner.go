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

	multiPVEnabled bool
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
		r.binary, err = binary.SetupBinaryRunner(
			"stockfish", "stockfish", []string{},
			1000*time.Millisecond,
			binary.WithLogger(r.logger))
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

func (r *StockfishRunner) SetMultiPV() (func() Error, Error) {
	err := r.binary.RunAsync("setoption name MultiPV value 80")
	r.multiPVEnabled = true
	if !IsNil(err) {
		return func() Error { return NilError }, err
	}
	return func() Error {
		r.multiPVEnabled = false
		return r.binary.RunAsync("setoption name MultiPV value 1")
	}, NilError
}

type SearchParams struct {
	Depth    Optional[int]
	Duration Optional[time.Duration]
}

func (r *StockfishRunner) SearchVerbose(params SearchParams) (
	map[string]int, []Pair[string, int], Error,
) {
	if !r.multiPVEnabled {
		return nil, nil, Errorf("use multi-pv for verbose search")
	}

	moveToScore := map[string]int{}

	err := r.SearchRaw(params, func(line string) Error {
		if strings.Contains(line, "score cp ") &&
			strings.Contains(line, "pv ") {
			scoreStr := strings.Split(
				strings.Split(line, "score cp ")[1],
				" ")[0]
			moveStr := strings.Split(
				strings.Split(line, " pv ")[1],
				" ")[0]

			score, err := WrapReturn(strconv.Atoi(scoreStr))
			if !IsNil(err) {
				return err
			}

			moveToScore[moveStr] = score
		}

		return NilError
	})

	if !IsNil(err) {
		return nil, nil, err
	}

	if !IsNil(err) {
		return nil, nil, err
	}

	moveAndScore := []Pair[string, int]{}
	for move, score := range moveToScore {
		moveAndScore = append(moveAndScore,
			Pair[string, int]{First: move, Second: score})
	}
	SortMaxFirst(&moveAndScore, func(p Pair[string, int]) int {
		return p.Second
	})

	return moveToScore, moveAndScore, NilError
}

func (r *StockfishRunner) SearchRaw(params SearchParams, callback func(line string) Error) Error {
	var err Error

	processLine := func(line string) LoopResult {
		err = callback(line)
		if !IsNil(err) {
			return LoopBreak
		}

		if strings.Contains(line, "bestmove") {
			return LoopBreak
		}
		return LoopContinue
	}

	if err.HasError() {
		return err
	}

	if params.Depth.HasValue() {
		err = r.binary.RunSync(fmt.Sprint("go depth ", params.Depth.Value()), processLine, Empty[time.Duration]())

		if !IsNil(err) {
			return err
		}

	} else if params.Duration.HasValue() {
		err = r.binary.RunAsync("go")
		if !IsNil(err) {
			return err
		}
		time.Sleep(params.Duration.Value())
		err = r.binary.RunSync(fmt.Sprint("stop", params.Depth.Value()), processLine, Empty[time.Duration]())

		if !IsNil(err) {
			return err
		}
	} else {
		return Errorf("no search params provided")
	}

	return NilError
}

func (r *StockfishRunner) Search() (Optional[string], Error) {
	bestMoveString := Empty[string]()

	err := r.SearchRaw(SearchParams{Duration: Some(1 * time.Second)}, func(v string) Error {
		if strings.HasPrefix(v, "bestmove ") {
			bestMoveString = Some(v)
		}
		return NilError
	})

	if !IsNil(err) {
		return Empty[string](), err
	}

	if bestMoveString.HasValue() {
		return Some(strings.Split(bestMoveString.Value(), " ")[1]), NilError
	}

	return Empty[string](), NilError
}
