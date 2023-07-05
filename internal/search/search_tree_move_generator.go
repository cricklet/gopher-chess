package search

import (
	"fmt"
	"strings"

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

	current *SearchTree
	history []*SearchTree
}

func CreateSearchTreeMoveGenerator(tree SearchTree) MoveGenConstructor {
	return func(g *game.GameState) (func(), MoveGen) {
		unregister, gen := NewSearchTreeMoveGenerator(tree, g)
		return unregister, gen
	}
}

func NewSearchTreeMoveGenerator(
	tree SearchTree, g *game.GameState,
) (func(), *SearchTreeMoveGenerator) {
	gen := &SearchTreeMoveGenerator{
		SearchTree: tree,
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

func (gen *SearchTreeMoveGenerator) generateMoves(g *game.GameState, mode MoveGenerationMode) (func(), MoveGenerationResult, *[]Move, Error) {
	moves := GetMovesBuffer()
	cleanup := func() { ReleaseMovesBuffer(moves) }

	result := AllLegalMoves

	if mode == OnlyCaptures {
		result = SomeLegalMoves
		GeneratePseudoCaptures(func(m Move) {
			*moves = append(*moves, m)
		}, g)
	} else {
		GeneratePseudoMoves(func(m Move) {
			*moves = append(*moves, m)
		}, g)
	}

	if gen.current != nil && gen.current.continueSearching {
		// Perform all moves
		return cleanup, result, moves, NilError
	}

	*moves = FilterSlice(*moves, func(m Move) bool {
		_, contains := gen.current.moves[m.String()]
		if !contains {
			result = SomeLegalMoves
			return false
		}

		return true
	})

	return cleanup, result, moves, NilError
}
