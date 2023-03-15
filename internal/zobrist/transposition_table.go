package zobrist

import (
	"fmt"
	"math"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type CachedEvaluation struct {
	Depth       int
	Score       int
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

var DefaultTranspositionTableSize = int(math.Pow(2, 24))

func NewTranspositionTable(size int) *TranspositionTable {
	return &TranspositionTable{
		Size:  size,
		Cache: make([]CachedEvaluation, size),
	}
}
func (t *TranspositionTable) Stats() string {
	return fmt.Sprintf("hits: %v, collisions: %v, depth too low: %v, misses: %v", t.Hits, t.Collisions, t.DepthTooLow, t.Misses)
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
	} else if v.ZobristHash != 0 {
		t.Collisions++
	} else {
		t.Misses++
	}
	return Empty[CachedEvaluation]()
}

func (t *TranspositionTable) Put(hash uint64, depth int, score int) {
	i := hash % uint64(t.Size)
	t.Cache[i] = CachedEvaluation{
		Depth:       depth,
		Score:       score,
		ZobristHash: hash,
	}
}
