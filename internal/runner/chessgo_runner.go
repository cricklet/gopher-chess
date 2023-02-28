package runner

import (
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/search"
)

type Runner interface {
	PerformMoveFromString(s string) Error
	SetupPosition(position Position) Error
	PerformMoves(startPos string, moves []string) Error
	MovesForSelection(s string) ([]string, Error)
	Rewind(num int) Error
	Reset()
	Search() (Optional[string], Error)
	IsNew() bool
}

type ChessGoRunner struct {
	Logger Logger

	g *GameState
	b *Bitboards

	StartFen string
	history  []HistoryValue
}

var _ Runner = (*ChessGoRunner)(nil)

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

type Position struct {
	Fen   string
	Moves []string
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

func (r *ChessGoRunner) Player() Player {
	return r.g.Player
}

func (r *ChessGoRunner) Search() (Optional[string], Error) {
	searcher := NewSearcher(r.Logger, r.g, r.b)

	go func() {
		time.Sleep(2 * time.Second)
		searcher.OutOfTime = true
	}()

	move, errs := searcher.Search()
	if len(errs) != 0 {
		return Empty[string](), Join(errs...)
	}

	if move.HasValue() {
		return Some(move.Value().String()), NilError
	}

	return Empty[string](), NilError
}
