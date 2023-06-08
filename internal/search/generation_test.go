package search

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"

	"github.com/stretchr/testify/assert"
)

type PerftValues struct {
	leaves     int
	captures   int
	enPassants int
	castles    int
}

func (p *PerftValues) add(o PerftValues) {
	p.leaves += o.leaves
	p.captures += o.captures
	p.enPassants += o.enPassants
	p.castles += o.castles
}

func (p *PerftValues) empty() bool {
	return p.leaves == 0 && p.captures == 0 && p.enPassants == 0 && p.castles == 0
}

type PerftResult struct {
	result      PerftValues
	startingFen string
	move        string
	endingFen   string
}

type PerftResults map[string]PerftResult

type PerftDiscrepancy struct {
	comparison  PerftComparison
	startingFen string
	moves       []string
	fenPerMove  []string
}

type PerftDiscrepancies map[string]PerftDiscrepancy

func createPerftDiscrepancy(result PerftResult, comparison PerftComparison) PerftDiscrepancy {
	return PerftDiscrepancy{
		startingFen: result.startingFen,
		moves:       []string{result.move},
		fenPerMove:  []string{result.endingFen},
		comparison:  comparison,
	}
}

func countAndPerftForDepth(t *testing.T, g *GameState, b *Bitboards, n int, progress *chan int, outputPerftResults *PerftResults) PerftValues {
	if n == 0 {
		return PerftValues{leaves: 1, captures: 0, enPassants: 0, castles: 0}
	}
	result := PerftValues{}

	movingPlayer := g.Player

	startingFen := ""
	if outputPerftResults != nil {
		startingFen = FenStringForGame(g)
	}

	GeneratePseudoMovesWithAllPromotions(
		func(move Move) {
			update := BoardUpdate{}
			err := g.PerformMove(move, &update, b)
			if !IsNil(err) {
				t.Error(Errorf("perform %v, %v: %w", FenStringForGame(g), move, err))
			}

			defer func() {
				err = g.UndoUpdate(&update, b)
				if !IsNil(err) {
					t.Error(Errorf("undo %v, %v: %w", FenStringForGame(g), move, err))
				}
			}()

			resultsForMove := PerftValues{}

			if KingIsInCheck(b, movingPlayer) {
			} else if n <= 1 {
				resultsForMove.leaves++
				if move.MoveType == CaptureMove {
					resultsForMove.captures++
				} else if move.MoveType == EnPassantMove {
					resultsForMove.enPassants++
				} else if move.MoveType == CastlingMove {
					resultsForMove.castles++
				}
			} else {
				resultsForMove = countAndPerftForDepth(t, g, b, n-1, nil, nil)
			}

			if !resultsForMove.empty() {
				result.add(resultsForMove)
				if outputPerftResults != nil {
					(*outputPerftResults)[move.String()] = PerftResult{
						result:      resultsForMove,
						startingFen: startingFen,
						move:        move.String(),
						endingFen:   FenStringForGame(g),
					}
				}
			}
			if progress != nil {
				*progress <- result.leaves
			}
		},
		b, g)

	return result
}

func CountAndPerftForDepthWithProgress(t *testing.T, g *GameState, b *Bitboards, n int, expectedCount int) (PerftValues, PerftResults) {
	perft := make(PerftResults)

	var progressBar *ProgressBar
	if expectedCount > 0 {
		p := CreateProgressBar(expectedCount, fmt.Sprint("depth ", n))
		progressBar = &p
	}

	progressChan := make(chan int)

	var result PerftValues
	go func() {
		result = countAndPerftForDepth(t, g, b, n, &progressChan, &perft)
		close(progressChan)
	}()

	for p := range progressChan {
		if progressBar != nil {
			progressBar.Set(p)
		}
	}

	if progressBar != nil {
		progressBar.Close()
	}

	return result, perft
}

type PerftComparison int

const (
	MOVE_IS_INVALID PerftComparison = iota
	MOVE_IS_MISSING
	COUNT_TOO_HIGH
	COUNT_TOO_LOW
)

func (p PerftComparison) String() string {
	switch p {
	case MOVE_IS_INVALID:
		return "move-is-illegal"
	case MOVE_IS_MISSING:
		return "missing-specific-move"
	case COUNT_TOO_HIGH:
		return "stockfish-found-fewer"
	case COUNT_TOO_LOW:
		return "stockfish-found-more"
	}
	panic("unknown issue")
}

func parsePerft(s string) (map[string]int, int, Error) {
	expectedPerft := make(map[string]int)

	ok := false
	for _, line := range strings.Split(s, "\n") {
		if ok {
			if len(line) == 0 {
				continue
			} else if strings.HasPrefix(line, "Nodes searched: ") {
				expectedCountStr := strings.TrimPrefix(line, "Nodes searched: ")
				expectedCount, err := strconv.Atoi(expectedCountStr)
				if !IsNil(err) {
					return expectedPerft, expectedCount,
						Errorf("couldn't parse searched nodes: %v, %w", line, err)
				}

				return expectedPerft, expectedCount, NilError
			} else {
				lineParts := strings.Split(line, ": ")
				moveStr := lineParts[0]
				moveCount, err := strconv.Atoi(lineParts[1])
				if !IsNil(err) {
					return expectedPerft, 0,
						Errorf("couldn't parse count from move: %v, %w", line, err)
				}

				expectedPerft[moveStr] = moveCount
			}
		} else if line == "uciok" {
			ok = true
		}
	}

	return expectedPerft, 0, Errorf("could not parse: %v", s)
}

func findPerftDiscrepancies(t *testing.T, g *GameState, b *Bitboards, depth int) PerftDiscrepancies {
	if depth == 0 {
		t.Error("0 depth not valid for stockfish")
	}
	initialFen := FenStringForGame(g)
	input := fmt.Sprintf("echo \"isready\nuci\nposition fen %v\ngo perft %v\" | stockfish", initialFen, depth)
	cmd := exec.Command("bash", "-c", input)
	output, _ := cmd.CombinedOutput()

	expectedPerftMap, expectedPerftTotal, err := parsePerft(string(output))
	if !IsNil(err) {
		t.Error(err)
	}

	actualPerftTotal, actualPerftMap := CountAndPerftForDepthWithProgress(t, g, b, depth, expectedPerftTotal)

	result := make(PerftDiscrepancies)

	for move, actualCountForMove := range actualPerftMap {
		expectedCount, ok := expectedPerftMap[move]
		if ok == false {
			result[move] = createPerftDiscrepancy(actualCountForMove, MOVE_IS_INVALID)
		} else if actualCountForMove.result.leaves > expectedCount {
			result[move] = createPerftDiscrepancy(actualCountForMove, COUNT_TOO_HIGH)
		} else if actualCountForMove.result.leaves < expectedCount {
			result[move] = createPerftDiscrepancy(actualCountForMove, COUNT_TOO_LOW)
		}
	}
	for expectedMove := range expectedPerftMap {
		_, ok := actualPerftMap[expectedMove]
		if ok == false {
			result[expectedMove] = PerftDiscrepancy{
				startingFen: initialFen,
				moves:       []string{expectedMove},
				fenPerMove:  []string{"???"},
				comparison:  MOVE_IS_MISSING,
			}
		}
	}

	if actualPerftTotal.leaves != expectedPerftTotal && len(result) == 0 {
		panic("should have found a discrepancy between perft")
	}

	return result
}

type InitialState struct {
	startingFen string
	moves       []string
	fenPerMove  []string
}

type InvalidMovesToSearch struct {
	move    string
	issue   PerftComparison
	initial InitialState
}

var totalInvalidMoves int = 0

const MAX_TOTAL_INVALID_MOVES int = 20

func findSpecificInvalidMoves(t *testing.T, initialState InitialState, maxDepth int) {
	result := []string{}
	invalidMovesToSearch := []InvalidMovesToSearch{}

	g, err := GamestateFromFenString(initialState.startingFen)
	assert.True(t, IsNil(err))
	b := g.CreateBitboards()

	for _, move := range initialState.moves {
		update := BoardUpdate{}
		err := g.PerformMove(g.MoveFromString(move), &update, b)
		if !IsNil(err) {
			t.Error(Errorf("perform %v => %v: %w", FenStringForGame(g), move, err))
		}
	}

	if len(initialState.fenPerMove) == 0 {
		assert.True(t, strings.HasPrefix(FenStringForGame(g), initialState.startingFen))
	} else {
		assert.True(t, strings.HasPrefix(FenStringForGame(g), Last(initialState.fenPerMove)))
	}

	for i := 1; i <= maxDepth; i++ {
		perftIssueMap := findPerftDiscrepancies(t, g, b, i)
		for k, m := range perftIssueMap {
			perftIssueMap[k] = PerftDiscrepancy{
				startingFen: initialState.startingFen,
				moves:       append(initialState.moves, m.moves...),
				fenPerMove:  append(initialState.fenPerMove, m.fenPerMove...),
				comparison:  m.comparison,
			}
		}

		if len(perftIssueMap) > 0 {
			t.Error(Errorf(PrettyPrint(perftIssueMap)))
			for move, issue := range perftIssueMap {
				invalidMovesToSearch = append(invalidMovesToSearch, InvalidMovesToSearch{move, issue.comparison, initialState})
			}
			break
		}
	}

	for _, search := range invalidMovesToSearch {
		if totalInvalidMoves > MAX_TOTAL_INVALID_MOVES {
			break
		}
		if search.issue == MOVE_IS_INVALID || search.issue == MOVE_IS_MISSING {
			result = append(result, PrettyPrint(search))
			totalInvalidMoves++
		} else {
			move := g.MoveFromString(search.move)

			update := BoardUpdate{}
			err := g.PerformMove(move, &update, b)
			if !IsNil(err) {
				t.Error(Errorf("perform %v => %v: %w", FenStringForGame(g), move, err))
			}

			findSpecificInvalidMoves(t,
				InitialState{
					initialState.startingFen,
					append(initialState.moves, move.String()),
					append(initialState.fenPerMove, FenStringForGame(g)),
				}, maxDepth-1)

			err = g.UndoUpdate(&update, b)
			if !IsNil(err) {
				t.Error(Errorf("undo %v => %v: %w", FenStringForGame(g), move, err))
			}
		}
	}

	if len(result) == 0 && len(invalidMovesToSearch) > 0 && totalInvalidMoves < MAX_TOTAL_INVALID_MOVES {
		t.Error(Errorf("couldn't find '%v' => %v", FenStringForGame(g), PrettyPrint(invalidMovesToSearch)))
	}
}

func TestIncorrectEnPassantOutOfBounds(t *testing.T) {
	s := "rnbqkb1r/1ppppppp/5n2/p7/6PP/8/PPPPPP2/RNBQKBNR w KQkq a6 2 2"
	findSpecificInvalidMoves(t, InitialState{s, []string{}, []string{}}, 2)
}

func TestIncorrectUndoBoard(t *testing.T) {
	s := "rnbqkbnr/pp1p1ppp/2p5/4pP2/8/2P5/PP1PP1PP/RNBQKBNR b KQkq - 5 3"
	findSpecificInvalidMoves(t, InitialState{s, []string{}, []string{}}, 3)
}

func TestFindIncorrectMoves(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	findSpecificInvalidMoves(t, InitialState{s, []string{}, []string{}}, 3)
}

func assertPerftCountsMatch(t *testing.T, s string, expectedCount []int) {
	for depth, expectedCount := range expectedCount {
		g, err := GamestateFromFenString(s)
		assert.True(t, IsNil(err))
		b := g.CreateBitboards()
		actualPerft, _ := CountAndPerftForDepthWithProgress(t, g, b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualPerft.leaves)
	}

	if t.Failed() {
		findSpecificInvalidMoves(t, InitialState{s, []string{}, []string{}}, 3)
	}
}

func TestMovesAtDepthForPawnOutOfBoundsCapture(t *testing.T) {
	s := "rnbqkbnr/1ppppppp/8/p7/8/7P/PPPPPPP1/RNBQKBNR w KQkq - 0 2"

	EXPECTED_COUNT := []int{
		1,
		19,
		399,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestMovesAtDepth(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	EXPECTED_COUNT := []int{
		1,
		20,
		400,
		8902,
		197281,
		// 4865609,
		// 119060324,
		// 3195901860,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestPosition2(t *testing.T) {
	s := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq -"

	EXPECTED_COUNT := []int{
		1,
		48,
		2039,
		97862,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestPosition2KingCheck(t *testing.T) {
	s := "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"

	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()
	update := BoardUpdate{}
	err = g.PerformMove(g.MoveFromString("a1b1"), &update, b)
	assert.True(t, IsNil(err))

	assert.True(t, KingIsInCheck(b, g.Enemy()))
}

func TestPosition3(t *testing.T) {
	s := "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - -"

	EXPECTED_COUNT := []int{
		1,
		14,
		191,
		2812,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestPositionPromotion(t *testing.T) {
	s := "k7/8/8/8/8/8/7p/K7 b - - 0 1"
	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()

	{
		moves := []Move{}
		GeneratePseudoMovesWithAllPromotions(func(m Move) {
			moves = append(moves, m)
		}, b, g)

		expectedMoves := []string{
			"h2h1q",
			"h2h1r",
			"h2h1b",
			"h2h1n",
			"a8a7",
			"a8b8",
			"a8b7",
		}

		assert.Equal(t, len(expectedMoves), len(moves))
		for _, m := range moves {
			assert.Contains(t, expectedMoves, m.String())
		}
	}

	{
		moves := []Move{}
		GeneratePseudoMoves(func(m Move) {
			moves = append(moves, m)
		}, b, g)

		expectedMoves := []string{
			"h2h1q",
			"a8a7",
			"a8b8",
			"a8b7",
		}

		assert.Equal(t, len(expectedMoves), len(moves))
		for _, m := range moves {
			assert.Contains(t, expectedMoves, m.String())
		}
	}
}
func TestPosition4F1F2(t *testing.T) {
	s := "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"
	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()

	err = g.PerformMove(g.MoveFromString("f1f2"), &BoardUpdate{}, b)
	assert.True(t, IsNil(err))
	err = g.PerformMove(g.MoveFromString("b2a1r"), &BoardUpdate{}, b)
	assert.True(t, IsNil(err))

	expectedFen := "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/P2P1RPP/r2Q2K1 w kq - 0 2"

	assert.Equal(t, expectedFen, FenStringForGame(g))

}

func TestPosition4(t *testing.T) {
	s := "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"

	EXPECTED_COUNT := []int{
		1,
		6,
		264,
		9467,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestPosition5(t *testing.T) {
	s := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"

	EXPECTED_COUNT := []int{
		1,
		44,
		1486,
		62379,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestPosition5QueenPin(t *testing.T) {
	s := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"

	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()
	update := BoardUpdate{}

	err = g.PerformMove(g.MoveFromString("d7c8q"), &update, b)
	assert.True(t, IsNil(err))

	err = g.PerformMove(g.MoveFromString("d8d6"), &update, b)
	assert.True(t, IsNil(err))

	assert.True(t, KingIsInCheck(b, g.Enemy()))
}

func TestPosition6(t *testing.T) {
	s := "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10 "

	EXPECTED_COUNT := []int{
		1,
		46,
		2079,
		89890,
	}
	assertPerftCountsMatch(t, s, EXPECTED_COUNT)
}

func TestPosition5QueenRetreat(t *testing.T) {
	s := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"

	g, err := GamestateFromFenString(s)
	assert.True(t, IsNil(err))

	b := g.CreateBitboards()

	err = g.PerformMove(g.MoveFromString("d1d4"), &BoardUpdate{}, b)
	assert.True(t, IsNil(err))

	err = g.PerformMove(g.MoveFromString("f2e4"), &BoardUpdate{}, b)
	assert.True(t, IsNil(err))

	expectedBoardString := []string{
		"rnbq k r",
		"pp Pbppp",
		"  p     ",
		"        ",
		"  BQn   ",
		"        ",
		"PPP N PP",
		"RNB K  R",
	}
	expectedBitboardString := MapSlice(expectedBoardString, func(s string) string {
		result := ""
		for _, c := range s {
			if c == ' ' {
				result += "0"
			} else {
				result += "1"
			}
		}
		return result
	})

	assert.Equal(t, strings.Join(expectedBoardString, "\n"), g.Board.String())
	assert.Equal(t, strings.Join(expectedBitboardString, "\n"), b.Occupied.String())

	moves := []Move{}
	GeneratePseudoMovesWithAllPromotions(func(m Move) {
		moves = append(moves, m)
	}, b, g)
	moveStrings := MapSlice(moves, func(m Move) string { return m.String() })

	numExpectedMoves := 55
	assert.Equal(t, numExpectedMoves, len(moves))
	assert.True(t, Contains(moveStrings, "d4g7"))
	assert.True(t, Contains(moveStrings, "d4g1"))
}
