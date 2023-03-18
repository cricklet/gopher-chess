package search

import (
	"fmt"
	"math"

	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/dustin/go-humanize"
)

type ScoreType int

const (
	NoneType ScoreType = iota
	AlphaFailUpperBound
	BetaFailLowerBound
	Exact
)

type CachedEvaluation struct {
	Depth       int
	Score       int
	ScoreType   ScoreType
	ZobristHash uint64
}

type TranspositionTable struct {
	Size        int
	Cache       []CachedEvaluation
	Hits        int
	Collisions  int
	DepthTooLow int
	Misses      int
}

var DefaultTranspositionTableSize = int(math.Pow(2, 28))

func NewTranspositionTable(size int) *TranspositionTable {
	return &TranspositionTable{
		Size:  size,
		Cache: make([]CachedEvaluation, size),
	}
}

var _defaultTranspositionTable *TranspositionTable

func DefaultTranspositionTable() *TranspositionTable {
	if _defaultTranspositionTable == nil {
		_defaultTranspositionTable = NewTranspositionTable(DefaultTranspositionTableSize)
	}
	return _defaultTranspositionTable
}

func (t *TranspositionTable) Clear() {
	t.Cache = make([]CachedEvaluation, t.Size)
	t.Hits = 0
	t.Collisions = 0
	t.DepthTooLow = 0
	t.Misses = 0
}

func (t *TranspositionTable) Stats() string {
	return fmt.Sprintf("hits: %v, collisions: %v, depth too low: %v, misses: %v",
		humanize.Comma(int64(t.Hits)), humanize.Comma(int64(t.Collisions)), humanize.Comma(int64(t.DepthTooLow)), humanize.Comma(int64(t.Misses)))
}

func (t *TranspositionTable) Get(hash uint64, depth int) Optional[CachedEvaluation] {
	i := hash % uint64(t.Size)
	v := t.Cache[i]
	if v.ZobristHash == hash {
		if v.Depth >= depth {
			t.Hits++
			return Some(v)
		} else {
			t.DepthTooLow++
		}
	} else if v.ScoreType != NoneType {
		t.Collisions++
	} else {
		t.Misses++
	}
	return Empty[CachedEvaluation]()
}

func (t *TranspositionTable) Put(hash uint64, depth int, score int, scoreType ScoreType) {
	i := hash % uint64(t.Size)
	t.Cache[i] = CachedEvaluation{
		Depth:       depth,
		Score:       score,
		ScoreType:   scoreType,
		ZobristHash: hash,
	}
}
