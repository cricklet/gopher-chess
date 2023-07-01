package chessgo

import (
	"fmt"
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/search"
)

type ChessGoRunner struct {
	Logger Logger

	g         *GameState
	s         *search.SearchHelper
	outOfTime *bool

	StartFen string
	history  []HistoryValue
}

var _ Runner = (*ChessGoRunner)(nil)

type ChessGoOptions struct {
	SearchOptions search.SearchOptions
	Logger        Optional[Logger]
}

func NewChessGoRunner(opts ChessGoOptions) ChessGoRunner {
	r := ChessGoRunner{
		outOfTime: new(bool),
	}
	if opts.Logger.HasValue() {
		r.Logger = opts.Logger.Value()
	} else {
		r.Logger = &SilentLogger
	}
	return r
}

type HistoryValue struct {
	move   Move
	update BoardUpdate
}

func (r *ChessGoRunner) Reset() {
	r.g = nil
	r.s = nil
	r.StartFen = ""
	r.history = []HistoryValue{}
}

func (r *ChessGoRunner) IsNew() bool {
	return r.g == nil
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
		err := r.g.UndoUpdate(&h.update)
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

	err := r.g.PerformMove(move, &h.update)
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
	if !r.IsNew() {
		r.Reset()
	}

	game, err := GamestateFromFenString(position.Fen)
	if !IsNil(err) {
		return Errorf("couldn't create game from %v, %w", position, err)
	}
	r.g = game

	// We don't need to be careful about unregistering searcher because it
	// has the same lifecycle as GameState above. eg, the garbage collector
	// will clean up both at the same time
	_, searcher := search.NewSearchHelper(r.g, search.SearchOptions{
		MaxDepth:  Some(10),
		Logger:    Some(r.Logger),
		OutOfTime: r.outOfTime,
	})
	r.s = searcher

	r.StartFen = position.Fen

	for _, m := range position.Moves {
		err := r.PerformMove(r.g.MoveFromString(m))
		if !IsNil(err) {
			return err
		}
	}

	r.Logger.Println(MemUsageString())

	return NilError
}

func (r *ChessGoRunner) MovesForSelection(selection string) ([]string, Error) {
	selectionFileRank, err := FileRankFromString(selection)
	if !IsNil(err) {
		return nil, Errorf("failed to parse selection %w", err)
	}
	selectionIndex := IndexFromFileRank(selectionFileRank)

	legalMoves := []Move{}
	err = search.GenerateLegalMoves(r.g, &legalMoves)
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

func (r *ChessGoRunner) Bitboards() *Bitboards {
	return r.g.Bitboards
}

func (r *ChessGoRunner) Board() BoardArray {
	return r.g.Board
}

func (r *ChessGoRunner) Search() (Optional[string], Optional[int], int, Error) {
	var err Error

	*r.outOfTime = false

	go func() {
		time.Sleep(1000 * time.Millisecond)
		*r.outOfTime = true
	}()

	if r.s == nil {
		return Empty[string](), Empty[int](), 0, Errorf("position not setup")
	}

	moves, score, depth, err := r.s.Search()
	if !IsNil(err) {
		return Empty[string](), Empty[int](), depth, err
	}

	if len(moves) > 0 {
		return Some(moves[0].String()), Some(score), depth, NilError
	}

	return Empty[string](), Empty[int](), depth, NilError
}

func (r *ChessGoRunner) PlayerIsInCheck() bool {
	return search.PlayerIsInCheck(r.g)
}

func (r *ChessGoRunner) NoValidMoves() (bool, Error) {
	return search.NoValidMoves(r.g)
}

func (r *ChessGoRunner) Evaluate(player Player) int {
	return search.Evaluate(r.g.Bitboards, player)
}

func (r *ChessGoRunner) EvaluateSimple(player Player) int {
	return search.EvaluatePieces(r.g.Bitboards, player) - search.EvaluatePieces(r.g.Bitboards, player.Other())
}

func (r *ChessGoRunner) DrawClock() int {
	return r.g.HalfMoveClock
}
