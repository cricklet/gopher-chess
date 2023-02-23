package search

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"

	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/assert"
)

type PerftResult struct {
	leaves     int
	captures   int
	enPassants int
	castles    int
}

func (p *PerftResult) add(o PerftResult) {
	p.leaves += o.leaves
	p.captures += o.captures
	p.enPassants += o.enPassants
	p.castles += o.castles
}

type PerftMap map[string]PerftResult

func countAndPerftForDepth(t *testing.T, g *GameState, b *Bitboards, n int, progress *chan int, perftPerMove *PerftMap) PerftResult {
	if n == 0 {
		return PerftResult{leaves: 1, captures: 0, enPassants: 0, castles: 0}
	}
	result := PerftResult{}

	moves := GetMovesBuffer()
	GeneratePseudoMoves(b, g, moves)

	for _, move := range *moves {
		func() {
			update := BoardUpdate{}
			err := g.PerformMove(move, &update, b)
			if err != nil {
				t.Error(fmt.Errorf("perform %v, %v: %w", FenStringForGame(g), move, err))
			}

			defer func() {
				err = g.UndoUpdate(&update, b)
				if err != nil {
					t.Error(fmt.Errorf("undo %v, %v: %w", FenStringForGame(g), move, err))
				}
			}()

			resultsForMove := PerftResult{}

			if KingIsInCheck(b, g.Enemy(), g.Player) {
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

			result.add(resultsForMove)
			if perftPerMove != nil {
				(*perftPerMove)[move.String()] = resultsForMove
			}
		}()

		if progress != nil {
			*progress <- result.leaves
		}
	}

	ReleaseMovesBuffer(moves)

	return result
}

func CountAndPerftForDepthWithProgress(t *testing.T, g *GameState, b *Bitboards, n int, expectedCount int) (PerftResult, PerftMap) {
	perft := make(PerftMap)

	var progressBar *progressbar.ProgressBar
	var startTime time.Time
	if expectedCount > 0 {
		progressBar = progressbar.Default(int64(expectedCount), fmt.Sprint("depth ", n))
		startTime = time.Now()
	}

	progressChan := make(chan int)

	var result PerftResult
	go func() {
		result = countAndPerftForDepth(t, g, b, n, &progressChan, &perft)
		close(progressChan)
	}()

	for p := range progressChan {
		if progressBar != nil {
			_ = progressBar.Set(p)
		}
	}

	if progressBar != nil {
		progressBar.Close()
		log.Println("             |", time.Since(startTime))
		log.Println()
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

func parsePerft(s string) (map[string]int, int, error) {
	expectedPerft := make(map[string]int)

	ok := false
	for _, line := range strings.Split(s, "\n") {
		if ok {
			if len(line) == 0 {
				continue
			} else if strings.HasPrefix(line, "Nodes searched: ") {
				expectedCountStr := strings.TrimPrefix(line, "Nodes searched: ")
				expectedCount, err := strconv.Atoi(expectedCountStr)
				if err != nil {
					return expectedPerft, expectedCount,
						fmt.Errorf("couldn't parse searched nodes: %v, %w", line, err)
				}

				return expectedPerft, expectedCount, nil
			} else {
				lineParts := strings.Split(line, ": ")
				moveStr := lineParts[0]
				moveCount, err := strconv.Atoi(lineParts[1])
				if err != nil {
					return expectedPerft, 0,
						fmt.Errorf("couldn't parse count from move: %v, %w", line, err)
				}

				expectedPerft[moveStr] = moveCount
			}
		} else if line == "uciok" {
			ok = true
		}
	}

	return expectedPerft, 0, fmt.Errorf("could not parse: %v", s)
}

func computeIncorrectPerftMoves(t *testing.T, g *GameState, b *Bitboards, depth int) map[string]PerftComparison {
	if depth == 0 {
		t.Error("0 depth not valid for stockfish")
	}
	input := fmt.Sprintf("echo \"isready\nuci\nposition fen %v\ngo perft %v\" | stockfish", FenStringForGame(g), depth)
	cmd := exec.Command("bash", "-c", input)
	output, _ := cmd.CombinedOutput()

	expectedPerft, expectedTotal, err := parsePerft(string(output))
	if err != nil {
		t.Error(err)
	}

	overallPerft, perftPerMove := CountAndPerftForDepthWithProgress(t, g, b, depth, expectedTotal)

	result := make(map[string]PerftComparison)

	for move, perftForMove := range perftPerMove {
		expectedCount, ok := expectedPerft[move]
		if ok == false {
			result[move] = MOVE_IS_INVALID
		} else if perftForMove.leaves > expectedCount {
			fmt.Println(perftForMove.leaves, expectedCount)
			result[move] = COUNT_TOO_HIGH
		} else if perftForMove.leaves < expectedCount {
			result[move] = COUNT_TOO_LOW
		}
	}
	for expectedMove := range expectedPerft {
		_, ok := perftPerMove[expectedMove]
		if ok == false {
			result[expectedMove] = MOVE_IS_MISSING
		}
	}

	if overallPerft.leaves != expectedTotal && len(result) == 0 {
		panic("should have found a discrepancy between perft")
	}

	return result
}

type InitialState struct {
	startingFen                 string
	moves                       []Move
	exectedFenAfterMovesApplied string
}

func (i InitialState) String() string {
	if len(i.moves) == 0 {
		return i.startingFen
	}
	return fmt.Sprint(i.startingFen, "+", i.moves, "=>", i.exectedFenAfterMovesApplied)
}

type InvalidMovesToSearch struct {
	move    string
	issue   PerftComparison
	initial InitialState
}

func (p PerftComparison) String() string {
	switch p {
	case MOVE_IS_INVALID:
		return "invalid"
	case MOVE_IS_MISSING:
		return "missing"
	case COUNT_TOO_HIGH:
		return "high"
	case COUNT_TOO_LOW:
		return "low"
	}
	panic("unknown issue")
}

func (m InvalidMovesToSearch) String() string {
	return fmt.Sprintf("%v %v at \"%v\"",
		m.issue.String(),
		m.move,
		m.initial,
	)
}

var totalInvalidMoves int = 0

const MAX_TOTAL_INVALID_MOVES int = 20

func findInvalidMoves(t *testing.T, initialState InitialState, maxDepth int) []string {
	result := []string{}
	invalidMovesToSearch := []InvalidMovesToSearch{}

	g, err := GamestateFromFenString(initialState.startingFen)
	assert.Nil(t, err)
	b := g.CreateBitboards()

	for _, move := range initialState.moves {
		update := BoardUpdate{}
		err := g.PerformMove(move, &update, &b)
		if err != nil {
			t.Error(fmt.Errorf("perform %v => %v: %w", FenStringForGame(&g), move, err))
		}
	}

	assert.True(t, strings.HasPrefix(FenStringForGame(&g), initialState.exectedFenAfterMovesApplied))

	for i := 1; i <= maxDepth; i++ {
		incorrectMoves := computeIncorrectPerftMoves(t, &g, &b, i)
		if len(incorrectMoves) > 0 {
			fmt.Println("found invalid moves", incorrectMoves)
			for move, issue := range incorrectMoves {
				invalidMovesToSearch = append(invalidMovesToSearch, InvalidMovesToSearch{move, issue, initialState})
			}
			break
		}
	}

	for _, search := range invalidMovesToSearch {
		if totalInvalidMoves > MAX_TOTAL_INVALID_MOVES {
			break
		}
		if search.issue == MOVE_IS_INVALID || search.issue == MOVE_IS_MISSING {
			result = append(result, search.String())
			totalInvalidMoves++
		} else {
			move := g.MoveFromString(search.move)

			update := BoardUpdate{}
			err := g.PerformMove(move, &update, &b)
			if err != nil {
				t.Error(fmt.Errorf("perform %v => %v: %w", FenStringForGame(&g), move, err))
			}

			result = append(result, findInvalidMoves(t,
				InitialState{
					initialState.startingFen,
					append(initialState.moves, move),
					FenStringForGame(&g),
				}, maxDepth-1)...)

			err = g.UndoUpdate(&update, &b)
			if err != nil {
				t.Error(fmt.Errorf("undo %v => %v: %w", FenStringForGame(&g), move, err))
			}
		}
	}

	if len(result) == 0 && len(invalidMovesToSearch) > 0 && totalInvalidMoves < MAX_TOTAL_INVALID_MOVES {
		t.Error(fmt.Errorf("we weren't able to find the invalid move %v => %v", FenStringForGame(&g), invalidMovesToSearch))
	}
	return result
}

func TestIncorrectEnPassantOutOfBounds(t *testing.T) {
	s := "rnbqkb1r/1ppppppp/5n2/p7/6PP/8/PPPPPP2/RNBQKBNR w KQkq a6 2 2"
	invalidMoves := findInvalidMoves(t, InitialState{s, []Move{}, s}, 2)

	for _, move := range invalidMoves {
		assert.Equal(t, nil, move)
	}
}

func TestIncorrectUndoBoard(t *testing.T) {
	s := "rnbqkbnr/pp1p1ppp/2p5/4pP2/8/2P5/PP1PP1PP/RNBQKBNR b KQkq - 5 3"
	invalidMoves := findInvalidMoves(t, InitialState{s, []Move{}, s}, 3)

	for _, move := range invalidMoves {
		assert.Equal(t, nil, move)
	}
}

func TestFindIncorrectMoves(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	invalidMoves := findInvalidMoves(t, InitialState{s, []Move{}, s}, 3)

	for _, move := range invalidMoves {
		assert.Equal(t, nil, move)
	}
}

func TestMovesAtDepthForPawnOutOfBoundsCapture(t *testing.T) {
	s := "rnbqkbnr/1ppppppp/8/p7/8/7P/PPPPPPP1/RNBQKBNR w KQkq - 0 2"

	EXPECTED_COUNT := []int{
		1,
		19,
		399,
	}

	for depth, expectedCount := range EXPECTED_COUNT {
		g, err := GamestateFromFenString(s)
		assert.Nil(t, err)
		b := g.CreateBitboards()
		actualPerft, _ := CountAndPerftForDepthWithProgress(t, &g, &b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualPerft.leaves)
	}
}

func TestMovesAtDepth(t *testing.T) {
	s := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	EXPECTED_COUNT := []int{
		1,
		20,
		400,
		8902,
		// 197281,
		// 4865609,
		// 119060324,
		// 3195901860,
	}

	// defer profile.Start(profile.ProfilePath("../data/TestMovesAtDepth")).Stop()
	for depth, expectedCount := range EXPECTED_COUNT {
		g, err := GamestateFromFenString(s)
		assert.Nil(t, err)
		b := g.CreateBitboards()
		actualCount, _ := CountAndPerftForDepthWithProgress(t, &g, &b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualCount.leaves)
	}
}

func TestPosition2(t *testing.T) {
	s := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq -"

	EXPECTED_COUNT := []int{
		1,
		48,
		2039,
		97862,
		// 4085603,
	}

	for depth, expectedCount := range EXPECTED_COUNT {
		g, err := GamestateFromFenString(s)
		assert.Nil(t, err)
		b := g.CreateBitboards()
		actualCount, _ := CountAndPerftForDepthWithProgress(t, &g, &b, depth, expectedCount)

		assert.Equal(t, expectedCount, actualCount.leaves)
	}

	if t.Failed() {
		invalidMoves := findInvalidMoves(t, InitialState{s, []Move{}, s}, 3)

		for _, move := range invalidMoves {
			assert.Equal(t, nil, move)
		}
	}
}
