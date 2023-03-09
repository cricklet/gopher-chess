package chessgo

import (
	"fmt"
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	"github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/search"
)

type ChessGoRunner struct {
	Logger        Logger
	SearchOptions SearcherOptions

	g *GameState
	b *Bitboards

	StartFen string
	history  []HistoryValue
}

var _ Runner = (*ChessGoRunner)(nil)

type ChessGoOption func(*ChessGoRunner)

func WithLogger(l Logger) ChessGoOption {
	return func(r *ChessGoRunner) {
		r.Logger = l
	}
}

func WithSearchOptions(s SearcherOptions) ChessGoOption {
	return func(r *ChessGoRunner) {
		r.SearchOptions = s
	}
}

func NewChessGoRunner(opts ...ChessGoOption) ChessGoRunner {
	r := ChessGoRunner{}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

type HistoryValue struct {
	move   Move
	update BoardUpdate
}

func (r *ChessGoRunner) Reset() {
	r.g = nil
	r.b = nil
	r.StartFen = ""
	r.history = []HistoryValue{}
}

func (r *ChessGoRunner) IsNew() bool {
	return r.g == nil || r.b == nil
}

func (r *ChessGoRunner) LastMove() Optional[Move] {
	if len(r.history) > 0 {
		return Some(r.LastHistory().move)
	}
	return Empty[Move]()
}

func (r *ChessGoRunner) LastHistory() *HistoryValue {
	return &r.history[len(r.history)-1]
}
func (r *ChessGoRunner) Rewind(num int) Error {
	for i := 0; i < MinInt(num, len(r.history)); i++ {
		h := r.history[len(r.history)-1]
		err := r.g.UndoUpdate(&h.update, r.b)
		if !IsNil(err) {
			return Errorf("Rewind: %w", err)
		}
		r.history = r.history[:len(r.history)-1]
	}
	return NilError
}

func (r *ChessGoRunner) PerformMove(move Move) Error {
	r.history = append(r.history, HistoryValue{})

	h := r.LastHistory()
	h.move = move

	err := r.g.PerformMove(move, &h.update, r.b)
	if !IsNil(err) {
		return Errorf("PerformMove: %w", err)
	}

	return NilError
}

func (r *ChessGoRunner) PerformMoveFromString(s string) Error {
	m := r.g.MoveFromString(s)
	err := r.PerformMove(m)
	return err
}

func firstIndexMotMatching[A any, B any](a []A, b []B, matches func(A, B) bool) int {
	for i := 0; i < MinInt(len(a), len(b)); i++ {
		if !matches(a[i], b[i]) {
			return i
		}
	}
	return MinInt(len(a), len(b))
}

func (r *ChessGoRunner) PerformMoves(startPos string, moves []string) Error {
	if r.StartFen != startPos {
		return Errorf("positions don't match: %v != %v", r.StartFen, startPos)
	}

	startIndex := firstIndexMotMatching(r.history, moves, func(a HistoryValue, b string) bool {
		return a.move.String() == b
	})

	for i := startIndex; i < len(moves); i++ {
		err := r.PerformMove(r.g.MoveFromString(moves[i]))
		if !IsNil(err) {
			return err
		}
	}

	return NilError
}

func (r *ChessGoRunner) SetupPosition(position Position) Error {
	if r.Logger == nil {
		r.Logger = &DefaultLogger
	}
	if !r.IsNew() {
		return Errorf("please use ucinewgame")
	}

	game, err := GamestateFromFenString(position.Fen)
	if !IsNil(err) {
		return Errorf("couldn't create game from %v, %w", position, err)
	}
	r.g = &game

	bitboards := r.g.CreateBitboards()
	r.b = &bitboards

	r.StartFen = position.Fen

	for _, m := range position.Moves {
		err := r.PerformMove(r.g.MoveFromString(m))
		if !IsNil(err) {
			return err
		}
	}

	return NilError
}

func (r *ChessGoRunner) MovesForSelection(selection string) ([]string, Error) {
	selectionFileRank, err := FileRankFromString(selection)
	if !IsNil(err) {
		return nil, Errorf("failed to parse selection %w", err)
	}
	selectionIndex := IndexFromFileRank(selectionFileRank)

	legalMoves := []Move{}
	err = GenerateLegalMoves(r.b, r.g, &legalMoves)
	if !IsNil(err) {
		return nil, err
	}

	moves := FilterSlice(legalMoves, func(m Move) bool {
		return m.StartIndex == selectionIndex
	})
	return MapSlice(moves, func(m Move) string {
		return m.String()
	}), NilError
}

func (r *ChessGoRunner) FenString() string {
	return FenStringForGame(r.g)
}

func (r *ChessGoRunner) MoveHistory() []string {
	return MapSlice(r.history, func(h HistoryValue) string {
		return h.move.String()
	})
}

func (r *ChessGoRunner) PgnFromMoveHistory() string {
	result := ""
	fullMove := 1
	halfMove := 0
	for _, move := range r.history {
		if halfMove == 0 {
			result += fmt.Sprintf("%v. ", fullMove)
		}

		result += fmt.Sprintf("%v ", move.move.String())

		halfMove += 1
		if halfMove == 2 {
			halfMove = 0
			fullMove += 1
		}
	}
	return result
}

func (r *ChessGoRunner) Player() Player {
	return r.g.Player
}

func (r *ChessGoRunner) Board() BoardArray {
	return r.g.Board
}

func (r *ChessGoRunner) Search() (Optional[string], Error) {
	var move Optional[Move] = Empty[Move]()
	var err Error

	searcher := NewSearcherV2(r.Logger, r.g, r.b, r.SearchOptions)

	go func() {
		time.Sleep(2 * time.Second)
		searcher.OutOfTime = true
	}()

	move, err = JoinReturn(searcher.Search())
	if !IsNil(err) {
		return Empty[string](), err
	}

	if move.HasValue() {
		return Some(move.Value().String()), NilError
	}

	return MapOptional(move, func(m Move) string { return m.String() }), NilError
}

func (r *ChessGoRunner) PlayerIsInCheck() bool {
	return PlayerIsInCheck(r.g, r.b)
}

func (r *ChessGoRunner) NoValidMoves() (bool, Error) {
	return NoValidMoves(r.g, r.b)
}

func (r *ChessGoRunner) Evaluate(player Player) int {
	return evaluation.Evaluate(r.b, player)
}

func (r *ChessGoRunner) EvaluateSimple(player Player) int {
	return evaluation.EvaluatePieces(r.b, player)
}

func (r *ChessGoRunner) DrawClock() int {
	return r.g.HalfMoveClock
}
