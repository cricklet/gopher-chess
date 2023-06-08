package search

import (
	"fmt"
	"strings"

	"github.com/cricklet/chessgo/internal/bitboards"
	"github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

type SearchTree struct {
	moves             map[string]*SearchTree
	continueSearching bool
}

func (tree *SearchTree) String() string {
	contents := []string{}
	if tree.continueSearching {
		contents = append(contents, "continue...")
	}
	for move, nextTree := range tree.moves {
		contents = append(contents, move+": "+nextTree.String())
	}

	return fmt.Sprintf("SearchTree[%s]", strings.Join(contents, ", "))
}

func SearchTreeFromLines(
	lines [][]string,
	continueSearchingPastLines bool,
) (SearchTree, Error) {
	result := SearchTree{
		moves:             map[string]*SearchTree{},
		continueSearching: false,
	}

	for _, line := range lines {
		currentTree := &result
		for _, move := range line {
			if nextTree, contains := currentTree.moves[move]; contains {
				currentTree = nextTree
			} else {
				currentTree.moves[move] = &SearchTree{
					moves:             map[string]*SearchTree{},
					continueSearching: false,
				}
				currentTree = currentTree.moves[move]
			}
		}

		if continueSearchingPastLines {
			currentTree.continueSearching = true
		}
	}

	return result, Error{}
}

type SearchTreeMoveGenerator struct {
	SearchTree
	*game.GameState
	*bitboards.Bitboards

	current *SearchTree
	history []*SearchTree
}

func NewSearchTreeMoveGenerator(
	tree SearchTree, g *game.GameState, b *bitboards.Bitboards,
) (func(), *SearchTreeMoveGenerator) {
	gen := &SearchTreeMoveGenerator{
		SearchTree: tree,
		GameState:  g,
		Bitboards:  b,
	}
	gen.current = &gen.SearchTree

	unregister := g.RegisterListener(gen)
	return unregister, gen
}

var _ MoveGen = (*SearchTreeMoveGenerator)(nil)
var _ game.MoveListener = (*SearchTreeMoveGenerator)(nil)

func (gen *SearchTreeMoveGenerator) AfterMove(move Move) {
	previous := gen.current

	if gen.current != nil {
		nextSearchTree, contains := gen.current.moves[move.String()]
		if contains {
			gen.current = nextSearchTree
		}
	}

	gen.history = append(gen.history, previous)
}

func (gen *SearchTreeMoveGenerator) AfterUndo() {
	gen.current, gen.history = PopPtr(gen.history)
}

func (gen *SearchTreeMoveGenerator) updatePrincipleVariations(variations []Pair[int, []SearchMove]) {
}

func (gen *SearchTreeMoveGenerator) searchingAllLegalMoves() bool {
	if gen.current.continueSearching {
		return true
	} else {
		return false
	}
}

func (gen *SearchTreeMoveGenerator) performEachMoveAndCall(mode MoveGenerationMode, callback func(move Move) (LoopResult, Error)) Error {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	if mode == OnlyCaptures {
		GeneratePseudoCaptures(func(m Move) {
			*moves = append(*moves, m)
		}, gen.Bitboards, gen.GameState)
	} else {
		GeneratePseudoMoves(func(m Move) {
			*moves = append(*moves, m)
		}, gen.Bitboards, gen.GameState)
	}

	if gen.current != nil && gen.current.continueSearching {
		// Perform all moves
		for _, move := range *moves {
			result, err := performMoveAndCall(gen.GameState, gen.Bitboards, move, callback)

			if !err.IsNil() {
				return err
			}
			if result == LoopBreak {
				break
			}
		}
		return NilError
	}

	for nextMoveStr := range gen.current.moves {
		nextMove := gen.GameState.MoveFromString(nextMoveStr)
		if mode == OnlyCaptures && !nextMove.MoveType.Captures() {
			// If we're in quiescence, don't search non-capture moves.
			// Note, this isn't an error because we could be hitting
			// quiescence early due to iterative deepening (eg searching
			// depth 1 or 2)
			continue
		}

		if !Contains(*moves, nextMove) {
			return Errorf("move %s not found", nextMove.DebugString())
		}

		result, err := performMoveAndCall(gen.GameState, gen.Bitboards, nextMove, callback)

		if !err.IsNil() {
			return err
		}
		if result == LoopBreak {
			break
		}
	}

	return NilError
}
