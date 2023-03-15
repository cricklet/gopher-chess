package search

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bluele/psort"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestOpening(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards, SearcherOptions{})

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 2000)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.True(t, IsNil(err), err)
	assert.Empty(t, errs)

	expectedOpenings := map[string]bool{"e2e4": true, "d2d4": true, "g1f3": true, "b1c3": true}
	assert.True(t, expectedOpenings[result.Value().String()])
}

func TestPointlessSacrifice(t *testing.T) {
	fen := "rnbqkbnr/ppp2ppp/8/3pp3/4P3/3P1N2/PPP2PPP/RNBQKB1R b KQkq - 5 3"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards, SearcherOptions{})

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 2000)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.True(t, IsNil(err), err)

	fmt.Println(result.Value().String())
	fmt.Println(game.Board.String())

	assert.NotEqual(t, "c8f5", result.Value().String())
}

func TestNoLegalMoves(t *testing.T) {
	fen := "rn1qkb1r/ppp3pp/5n2/3ppb2/8/2NP1NP1/PPP2PBP/R1BQK2R b KQkq - 13 7"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards, SearcherOptions{})

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 10000)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.True(t, IsNil(err), err)

	fmt.Println(result.Value().String())
	fmt.Println(game.Board.String())

	assert.True(t, result.HasValue())
}

func TestCheckMateSearch(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8 b KQkq - 13 7"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&DefaultLogger, &game, &bitboards, SearcherOptions{
		debugSearchTree: &debugSearchTree{},
	})

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 100)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.False(t, result.HasValue(), result)

	debugString := searcher.options.debugSearchTree.DebugString(2)
	fmt.Println(debugString)
	err = Wrap(os.WriteFile(RootDir()+"/data/TestCheckMateSearch.tree", []byte(debugString), 0600))
	assert.True(t, IsNil(err), err)
}

func TestCheckMateDetection(t *testing.T) {
	fen := "kQK5/8/8/8/8/8/8/8/8 b KQkq - 13 7"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	var noValidMoves bool
	noValidMoves, err = NoValidMoves(&game, &bitboards)
	assert.True(t, IsNil(err), err)
	assert.True(t, noValidMoves)

	isCheckMate := noValidMoves && PlayerIsInCheck(&game, &bitboards)
	assert.True(t, isCheckMate)
}

func TestCheckMateInOne(t *testing.T) {
	fen := "1K6/8/1b6/5k2/1p2p3/8/2q5/n7 b - - 2 2"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&SilentLogger, &game, &bitboards,
		SearcherOptions{
			debugSearchTree:   &debugSearchTree{},
			handleLegality:    true,
			evaluationOptions: []EvaluationOption{EndgamePushEnemyKing},
		})

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 50)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.True(t, result.HasValue())

	debugString := searcher.options.debugSearchTree.DebugString(3)
	fmt.Println(debugString)
	checkMateMoves := map[string]bool{"c2c7": true, "c2c8": true}
	assert.True(t, checkMateMoves[result.Value().String()], result.Value().String())

	err = Wrap(os.WriteFile(RootDir()+"/data/TestCheckMateInOne.tree", []byte(debugString), 0600))
	assert.True(t, IsNil(err), err)
}

func TestCheckMateInOne2(t *testing.T) {
	fen := "5b2/3kp2p/4r3/1p6/4n3/p3P1p1/3p1r2/6K1 b - - 1 46"
	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&SilentLogger, &game, &bitboards,
		SearcherOptions{
			debugSearchTree: &debugSearchTree{},
			handleLegality:  true,
		})

	var result Optional[Move]
	var errs []Error

	go func() {
		time.Sleep(time.Millisecond * 100)
		searcher.OutOfTime = true
	}()

	result, errs = searcher.Search()

	assert.Empty(t, errs)
	assert.True(t, result.HasValue())

	debugString := searcher.options.debugSearchTree.DebugString(2)
	fmt.Println(debugString)
	checkMateMoves := map[string]bool{"d2d1q": true}
	assert.True(t, checkMateMoves[result.Value().String()], result.Value().String())

	err = Wrap(os.WriteFile(RootDir()+"/data/TestCheckMateInOne2.tree", []byte(debugString), 0600))
	assert.True(t, IsNil(err), err)
}

func TestPartialSort(t *testing.T) {
	xs := []int{3, 1, 8, 10, 2, 7, 5, 6, 4, 9}
	psort.Slice(xs, func(i, j int) bool {
		return xs[i] > xs[j]
	}, 3)
	assert.Equal(t, 10, xs[0])
	assert.Equal(t, 9, xs[1])
	assert.Equal(t, 8, xs[2])
	assert.Equal(t, 10, len(xs))
	fmt.Println(xs)
}

func TestDisallowedOptions(t *testing.T) {
	options := [][]string{
		{"sortPartial", "incDepthForCheck"},
		{"sortPartial", "sortPartial=0"},
		{"sortPartial=10", "sortPartial=0"},
	}

	options = FilterDisallowedSearchOptions(options)
	assert.Equal(t, 1, len(options))
	assert.Equal(t, []string{"sortPartial", "incDepthForCheck"}, options[0])
}

func TestShouldMateInsteadOfDraw(t *testing.T) {
	fen := "2K5/6k1/1q6/3p4/8/5p2/4r3/8 b"

	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&SilentLogger, &game, &bitboards,
		SearcherOptions{
			// debugSearchTree: &debugSearchTree{},
			handleLegality: true,
		})

	{
		drawScore, legality, errs := searcher.evaluateMove(MoveFromString("e2e7", QuietMove), -Inf, Inf, 5)
		assert.Empty(t, errs)
		assert.True(t, legality)

		var winScore int
		winScore, legality, errs = searcher.evaluateMove(MoveFromString("e2a2", QuietMove), -Inf, Inf, 5)
		assert.Empty(t, errs)
		assert.True(t, legality)

		assert.Less(t, drawScore, winScore)
		assert.Equal(t, drawScore, 0)
		assert.Equal(t, winScore, Inf)
	}

	{
		drawScore, legality, errs := searcher.evaluateMove(MoveFromString("e2e7", QuietMove), -Inf, Inf, 2)
		assert.Empty(t, errs)
		assert.True(t, legality)

		var winScore int
		winScore, legality, errs = searcher.evaluateMove(MoveFromString("e2a2", QuietMove), -Inf, Inf, 2)
		assert.Empty(t, errs)
		assert.True(t, legality)

		assert.Less(t, drawScore, winScore)
		assert.Equal(t, drawScore, 0)
		assert.Less(t, winScore, Inf) // We aren't able to see the check-mate yet
	}
}

func TestDeeperSearchesAvoidPins(t *testing.T) {
	fen := "r1bqk2r/p1p2ppp/1pnp1n2/4p3/1bPPP3/2N3P1/PP2NPBP/R1BQK2R b KQkq d3 0 7"

	game, err := GamestateFromFenString(fen)
	assert.True(t, IsNil(err), err)
	bitboards := game.CreateBitboards()

	searcher := NewSearcherV2(&SilentLogger, &game, &bitboards,
		SearcherOptions{
			debugSearchTree:  &debugSearchTree{},
			debugSearchStack: &[]string{},
			handleLegality:   true,
		})

	// {
	// 	go func() {
	// 		time.Sleep(time.Millisecond * 200)
	// 		searcher.OutOfTime = true
	// 	}()

	// 	_, errs := searcher.Search()
	// 	assert.Empty(t, errs)

	// 	debugString := searcher.options.debugSearchTree.DebugString(10)
	// 	err = Wrap(os.WriteFile(RootDir()+"/data/TestPreventPin.tree", []byte(debugString), 0600))
	// 	assert.True(t, IsNil(err), err)
	// }

	{
		// perform the forking moves
		score0, errs := searcher.evaluateSubtree(-Inf, Inf, 2)
		assert.Empty(t, errs)

		searcher.PerformMoveAndReturnLegality(MoveFromString("c8e6", QuietMove), &BoardUpdate{})
		score1, errs := searcher.evaluateSubtree(-Inf, Inf, 2)
		assert.Empty(t, errs)

		searcher.PerformMoveAndReturnLegality(MoveFromString("d4d5", QuietMove), &BoardUpdate{})
		score2, errs := searcher.evaluateSubtree(-Inf, Inf, 2)
		assert.Empty(t, errs)

		searcher.PerformMoveAndReturnLegality(MoveFromString("e8g8", QuietMove), &BoardUpdate{})
		score3, errs := searcher.evaluateSubtree(-Inf, Inf, 2)
		assert.Empty(t, errs)

		searcher.PerformMoveAndReturnLegality(MoveFromString("d5c6", QuietMove), &BoardUpdate{})
		score4, errs := searcher.evaluateSubtree(-Inf, Inf, 2)
		assert.Empty(t, errs)

		fmt.Println(score0, score1, score2, score3, score4)

		assert.Greater(t, score0, score1)
		assert.Greater(t, score1, score2)
		assert.Greater(t, score2, score3)
		assert.Greater(t, score2, score4)
	}
}
