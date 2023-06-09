package search

import (
	"fmt"
	"strings"

	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type VariationMovePrioritizer struct {
	sortedVariations [][]SearchMove

	currentVariationIndex Optional[int]
	currentDepth          int

	historyVariationIndex []Optional[int]

	noCopy NoCopy
}

var _ game.MoveListener = (*VariationMovePrioritizer)(nil)
var _ MoveSorter = (*VariationMovePrioritizer)(nil)

/*
	SortMaxFirst(&variations, func(t Pair[int, []SearchMove]) int {
		return t.First
	})

	sortedVariations := [][]SearchMove{}
	for _, variation := range variations {
		sortedVariations = append(sortedVariations, variation.Second)
	}

	unregister, gen.prioritizer = NewVariationMovePrioritizer(g, sortedVariations)
*/

func NewVariationMovePrioritizer(
	g *game.GameState,
) (func(), *VariationMovePrioritizer) {
	gen := &VariationMovePrioritizer{}

	unregister := g.RegisterListener(gen)
	return unregister, gen
}

func (gen *VariationMovePrioritizer) reset(variations []Pair[int, []SearchMove]) {
	SortMaxFirst(&variations, func(t Pair[int, []SearchMove]) int {
		return t.First
	})

	sortedVariations := [][]SearchMove{}
	for _, variation := range variations {
		sortedVariations = append(sortedVariations, variation.Second)
	}

	gen.resetSortedVariations(sortedVariations)
}

func (gen *VariationMovePrioritizer) resetSortedVariations(sortedVariations [][]SearchMove) {
	gen.sortedVariations = sortedVariations
	gen.currentVariationIndex = Empty[int]()
	gen.currentDepth = 0
	gen.historyVariationIndex = []Optional[int]{}
}

func (gen *VariationMovePrioritizer) AfterMove(move Move) {
	previous := gen.currentVariationIndex

	if gen.currentDepth == 0 {
		i := IndexOf(gen.sortedVariations, func(v []SearchMove) bool {
			return v[0].Move == move
		})
		if i.HasValue() {
			gen.currentVariationIndex = i
		} else {
			gen.currentVariationIndex = Empty[int]()
		}

	} else if gen.currentVariationIndex.HasValue() {
		i := gen.currentVariationIndex.Value()
		j := gen.currentDepth

		variation := gen.sortedVariations[i]
		if j < len(variation) && variation[j].Move == move {
			// we're still in the variation
		} else {
			gen.currentVariationIndex = Empty[int]()
		}
	}

	gen.currentDepth++
	gen.historyVariationIndex = append(gen.historyVariationIndex, previous)
}

func (gen *VariationMovePrioritizer) AfterUndo() {
	gen.currentVariationIndex, gen.historyVariationIndex = PopValue(gen.historyVariationIndex, Empty[int]())
	gen.currentDepth--
}

func (gen *VariationMovePrioritizer) String() string {
	if gen.currentDepth == 0 {
		result := strings.Join(MapSlice(gen.sortedVariations, func(variation []SearchMove) string {
			return "[" + ConcatStringify(variation) + "]"
		}), ", ")
		return fmt.Sprintf("VariationMovePrioritizer[%v]", result)
	} else if gen.currentVariationIndex.HasValue() {
		i := gen.currentVariationIndex.Value()
		j := gen.currentDepth
		variation := gen.sortedVariations[i]
		if j < len(variation) {
			return fmt.Sprintf("VariationMovePrioritizer[%v]", ConcatStringify(variation[j:]))
		}
	}

	return fmt.Sprintf("VariationMovePrioritizer[empty]")
}

func (gen *VariationMovePrioritizer) sortMoves(moves *[]Move) Error {
	moveScores := map[Move]int{}

	if gen.currentDepth == 0 {
		for i, variation := range gen.sortedVariations {
			moveScores[variation[0].Move] = i
		}
	} else if gen.currentVariationIndex.HasValue() {
		i := gen.currentVariationIndex.Value()
		j := gen.currentDepth
		variation := gen.sortedVariations[i]
		if j < len(variation) {
			moveScores[variation[j].Move] = i
		}
	}

	SortMinFirst(moves, func(move Move) int {
		if score, contains := moveScores[move]; contains {
			return score
		} else {
			return Inf
		}
	})

	return NilError
}
