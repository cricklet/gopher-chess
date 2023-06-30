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

	MultiPVEnabled bool
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

func NewStockfishRunner(options ...StockfishRunnerOption) (*StockfishRunner, Error) {
	r := &StockfishRunner{}
	for _, o := range options {
		o(r)
	}
	if r.logger == nil {
		r.logger = &DefaultLogger
	}

	var err Error

	r.binary, err = binary.SetupBinaryRunner(
		"stockfish", "stockfish", []string{},
		binary.WithLogger(r.logger))
	if !IsNil(err) {
		return nil, err
	}

	var output []string

	output, err = r.binary.Run("isready", Some("readyok"))
	if !IsNil(err) {
		return nil, err
	}
	if !Contains(output, "readyok") {
		return nil, Errorf("needs readyok")
	}

	output, err = r.binary.Run("uci", Some("uciok"))
	if !IsNil(err) {
		return nil, err
	}
	if !Contains(output, "uciok") {
		return nil, Errorf("needs uciok")
	}

	if r.elo.HasValue() && r.elo.Value() > 0 {
		err = r.binary.RunAsync("setoption name UCI_LimitStrength value true")
		if !IsNil(err) {
			return nil, err
		}

		err = r.binary.RunAsync(fmt.Sprintf("setoption name UCI_Elo value %v", r.elo.Value()))
		if !IsNil(err) {
			return nil, err
		}
	}
	r.logger.Println("setup stockfish")

	return r, NilError
}

func (r *StockfishRunner) Close() {
	if r.binary != nil {
		r.binary.Close()
		r.binary = nil
	}
}

var _ Runner = (*StockfishRunner)(nil)

func (r *StockfishRunner) SetupPosition(position Position) Error {
	var err Error

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
	r.MultiPVEnabled = true
	if !IsNil(err) {
		return func() Error { return NilError }, err
	}
	return func() Error {
		r.MultiPVEnabled = false
		return r.binary.RunAsync("setoption name MultiPV value 1")
	}, NilError
}

func (r *StockfishRunner) SetHashSize(mb int) Error {
	err := r.binary.RunAsync(fmt.Sprint("setoption name Hash value ", mb))
	return err
}

func (r *StockfishRunner) ClearHash() Error {
	err := r.binary.RunAsync("setoption name Clear Hash")
	return err
}

func DepthFromInfoLine(line string) (int, Error) {
	// input: "info depth 14 seldepth 21 multipv 46 score cp -1846 nodes 24503461 nps 1286203 hashfull 1000 tbhits 0 time 19051 pv h5g5 e5d4 g1h1"
	// returns: 14
	if strings.Contains(line, "info depth ") {
		depthStr := strings.Split(
			strings.Split(line, "info depth ")[1],
			" ")[0]

		depth, err := WrapReturn(strconv.Atoi(depthStr))
		if !IsNil(err) {
			return 0, err
		}

		return depth, NilError
	}

	return 0, Errorf("depth not found")
}

func MoveAndScoreFromInfoLine(line string) (Optional[string], int, Error) {
	if strings.Contains(line, "pv ") {
		moveStr := strings.TrimSpace(strings.Split(
			strings.Split(line, " pv ")[1],
			" ")[0])

		if strings.Contains(line, "score cp ") {
			scoreStr := strings.Split(
				strings.Split(line, "score cp ")[1],
				" ")[0]

			score, err := WrapReturn(strconv.Atoi(scoreStr))
			if !IsNil(err) {
				return Some(moveStr), 0, err
			}

			return Some(moveStr), score, NilError
		}
		if strings.Contains(line, "score mate ") {
			mateStr := strings.Split(
				strings.Split(line, "score mate ")[1],
				" ")[0]

			mate, err := WrapReturn(strconv.Atoi(mateStr))
			if !IsNil(err) {
				return Some(moveStr), 0, err
			}

			score, err := MateInNScore(mate)
			if err.HasError() {
				return Some(moveStr), 0, err
			}

			return Some(moveStr), score, NilError
		}
	}

	return Empty[string](), 0, NilError
}

func MoveFromBestMoveLine(line string) Optional[string] {
	if strings.HasPrefix(line, "bestmove ") {
		v := strings.Split(line, " ")[1]
		return Some(strings.Split(v, " ponder ")[0])
	}

	return Empty[string]()
}

func (r *StockfishRunner) SearchUnlimitedRaw(callback func(line string) (LoopResult, Error)) Error {
	var err Error

	processLine := func(line string) (LoopResult, Error) {
		var result LoopResult
		result, err = callback(line)
		if !IsNil(err) {
			return LoopBreak, err
		}

		return result, NilError
	}

	err = r.binary.RunSync("go", processLine, Empty[time.Duration]())
	if !IsNil(err) {
		return err
	}

	_, err = r.binary.Run("stop", Some("bestmove"))

	return err
}

func (r *StockfishRunner) SearchDepthRaw(depth int, callback func(line string) (LoopResult, Error)) Error {
	var err Error

	if err.HasError() {
		return err
	}

	err = r.binary.RunSync(fmt.Sprint("go depth ", depth), func(line string) (LoopResult, Error) {
		result, err := callback(line)
		if !IsNil(err) {
			return LoopBreak, err
		}

		return result, NilError
	}, Empty[time.Duration]())

	if !IsNil(err) {
		return err
	}
	return NilError
}

func (r *StockfishRunner) SearchDurationRaw(duration time.Duration, callback func(line string) (LoopResult, Error)) Error {
	var err Error

	err = r.binary.RunAsync("go")
	if !IsNil(err) {
		return err
	}

	time.Sleep(time.Second)
	err = r.binary.RunSync(
		"stop",
		func(line string) (LoopResult, Error) {
			result, err := callback(line)
			if !IsNil(err) {
				return LoopBreak, err
			}

			return result, NilError
		},
		Some(time.Second), // timeout for reading stdout
	)

	if !IsNil(err) {
		return err
	}
	return NilError
}

type SearchReader struct {
	bestPVMove  Optional[string]
	bestPVScore Optional[int]
	bestMove    Optional[string]
	depth       int
	noCopy      NoCopy
}

func (r *SearchReader) ReadLine(line string) (LoopResult, Error) {
	{
		move, score, err := MoveAndScoreFromInfoLine(line)
		if !IsNil(err) {
			return LoopBreak, err
		}

		if move.HasValue() {
			r.bestPVMove = move
			r.bestPVScore = Some(score)

			depth, err := DepthFromInfoLine(line)
			if !IsNil(err) {
				return LoopBreak, err
			}

			r.depth = depth
		}
	}

	{
		move := MoveFromBestMoveLine(line)
		if move.HasValue() {
			r.bestMove = move

			if r.bestPVMove.HasValue() && r.bestPVMove != r.bestMove {
				return LoopBreak, Errorf("best move does not match best PV move (best move: %v, best PV move: %v, line %v)", r.bestMove, r.bestPVMove, line)
			} else {
				return LoopBreak, NilError
			}
		}
	}

	return LoopContinue, NilError
}

func (runner *StockfishRunner) SearchDepth(depth int) (Optional[string], Optional[int], Error) {
	if runner.MultiPVEnabled {
		return Empty[string](), Empty[int](), Errorf("cannot search with MultiPV enabled")
	}

	reader := &SearchReader{}

	err := runner.SearchDepthRaw(depth, reader.ReadLine)

	if !IsNil(err) {
		return Empty[string](), Empty[int](), err
	}

	return reader.bestMove, reader.bestPVScore, NilError
}

func (runner *StockfishRunner) Search() (Optional[string], Optional[int], int, Error) {
	if runner.MultiPVEnabled {
		return Empty[string](), Empty[int](), 0, Errorf("cannot search with MultiPV enabled")
	}

	reader := &SearchReader{}

	err := runner.SearchDurationRaw(time.Second, reader.ReadLine)

	if !IsNil(err) {
		return Empty[string](), Empty[int](), reader.depth, err
	}

	return reader.bestMove, reader.bestPVScore, reader.depth, NilError
}

func (runner *StockfishRunner) Eval() (int, Error) {
	eval := 0.0
	err := runner.binary.RunSync("eval", func(line string) (LoopResult, Error) {
		var err Error

		// parse "Final evaluation       +0.31 (white side) [with scaled NNUE, hybrid, ...]"
		if strings.Contains(line, "Final evaluation") {
			evalStr := strings.TrimSpace(strings.TrimPrefix(line, "Final evaluation"))
			evalStr = strings.Split(evalStr, " ")[0]
			eval, err = WrapReturn(strconv.ParseFloat(evalStr, 64))
			if !IsNil(err) {
				return LoopBreak, err
			}

			return LoopBreak, NilError
		}
		return LoopContinue, NilError
	}, Empty[time.Duration]())

	return int(eval*100 + 0.5), err
}
