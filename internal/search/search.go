package search

import (
	"time"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/evaluation"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

var Inf int = 999999

func PlayerIsInCheck(g *GameState, b *Bitboards) bool {
	return KingIsInCheck(b, g.Player)
}

func IsLegal(g *GameState, b *Bitboards, move Move) (bool, Error) {
	var returnError Error

	player := g.Player

	var update BoardUpdate
	err := g.PerformMove(move, &update, b)
	defer func() {
		err = g.UndoUpdate(&update, b)
		returnError = Join(returnError, err)
	}()

	if !IsNil(err) {
		returnError = Join(returnError, err)
		return false, returnError
	}

	if KingIsInCheck(b, player) {
		returnError = Join(returnError, err)
		return false, returnError
	}

	returnError = Join(returnError, err)
	return true, returnError
}

func NoValidMoves(g *GameState, b *Bitboards) (bool, Error) {
	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GeneratePseudoMovesSkippingCastling(b, g, moves)

	for _, move := range *moves {
		legal, err := IsLegal(g, b, move)
		if !IsNil(err) {
			return legal, err
		}

		if legal {
			return false, NilError
		}
	}

	return true, NilError
}

func evaluateCapturesInner(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, Error) {
	if KingIsInCheck(b, g.Enemy()) {
		return SearchResult{Inf, 1, 1}, NilError
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoCaptures(b, g, moves)

	if len(*moves) == 0 {
		score := Evaluate(b, g.Player)
		return SearchResult{score, 1, 1}, NilError
	}

	totalSearched := 0

	for _, move := range *moves {
		if move.Evaluation.Value() < 100 {
			continue
		}

		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("perform evaluateCapturesInner %v: %w", move.String(), err)
		}

		result, err := evaluateCapturesInner(g, b,
			-enemyCanForceScore,
			-playerCanForceScore)
		if !IsNil(err) {
			return SearchResult{}, Errorf("recurse evaluateCapturesInner %v: %w", move.String(), err)
		}
		enemyScore := result.Score
		totalSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("undo evaluateCapturesInner %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, totalSearched}, NilError
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, totalSearched}, NilError
}

func evaluateCaptures(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int) (SearchResult, Error) {
	standPat := Evaluate(b, g.Player)
	if standPat > enemyCanForceScore {
		return SearchResult{enemyCanForceScore, 1, 1}, NilError
	} else if standPat > playerCanForceScore {
		playerCanForceScore = standPat
	}

	return evaluateCapturesInner(g, b, playerCanForceScore, enemyCanForceScore)
}

type SearchResult struct {
	Score              int
	TotalSearched      int
	QuiescenceSearched int
}

func evaluateSearch(g *GameState, b *Bitboards, playerCanForceScore int, enemyCanForceScore int, depth int) (SearchResult, Error) {
	if KingIsInCheck(b, g.Enemy()) {
		return SearchResult{Inf, 1, 0}, NilError
	}

	if depth == 0 {
		score := Evaluate(b, g.Player)
		return SearchResult{score, 1, 0}, NilError
	}

	moves := GetMovesBuffer()
	defer ReleaseMovesBuffer(moves)

	GenerateSortedPseudoMoves(b, g, moves)

	totalSearched := 0
	quiescenceSearched := 0

	for _, move := range *moves {
		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("perform evaluateSearch %v: %w", move.String(), err)
		}

		var result SearchResult
		if depth == 1 && move.MoveType == CaptureMove {
			result, err = evaluateCaptures(g, b,
				-enemyCanForceScore,
				-playerCanForceScore)
		} else {
			result, err = evaluateSearch(g, b,
				-enemyCanForceScore,
				-playerCanForceScore,
				depth-1)
		}
		if !IsNil(err) {
			return SearchResult{}, Errorf("%v %v: %w", move.String(), depth, err)
		}

		enemyScore := result.Score
		totalSearched += result.TotalSearched
		quiescenceSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if !IsNil(err) {
			return SearchResult{}, Errorf("undo evaluateSearch %v: %w", move.String(), err)
		}

		currentScore := -enemyScore
		if currentScore >= enemyCanForceScore {
			return SearchResult{enemyCanForceScore, totalSearched, quiescenceSearched}, NilError
		}

		if currentScore > playerCanForceScore {
			playerCanForceScore = currentScore
		}
	}

	return SearchResult{playerCanForceScore, totalSearched, quiescenceSearched}, NilError
}

func Search(g *GameState, b *Bitboards, depth int, logger Logger) (Optional[Move], Error) {
	// defer profile.Start(profile.ProfilePath("../data/Search")).Stop()

	moves := GetMovesBuffer()
	GenerateSortedPseudoMoves(b, g, moves)

	bestMoveSoFar := Empty[Move]()
	bestScoreSoFar := -Inf

	quiescenceSearched := 0
	totalSearched := 0

	startTime := time.Now()

	for i, move := range *moves {
		update := BoardUpdate{}
		err := g.PerformMove(move, &update, b)
		if !IsNil(err) {
			return Empty[Move](), Errorf("perform Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		result, err := evaluateSearch(g, b,
			-Inf, Inf, depth)
		if !IsNil(err) {
			return Empty[Move](), Errorf("evaluate Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		enemyScore := result.Score
		totalSearched += result.TotalSearched
		quiescenceSearched += result.QuiescenceSearched

		err = g.UndoUpdate(&update, b)
		if !IsNil(err) {
			return Empty[Move](), Errorf("undo Search %v => %v: %w", FenStringForGame(g), move.String(), err)
		}

		currentScore := -enemyScore

		bestComparisonStr := "worse"
		if currentScore > bestScoreSoFar {
			bestComparisonStr = "better"
		}
		logger.Println(IndentMany(".  ", i, "/", len(*moves), move, "searched", result.TotalSearched,
			"with initial search", result.TotalSearched-result.QuiescenceSearched,
			"and ending captures", result.QuiescenceSearched,
			"under", move.String(),
			"with score", currentScore,
			"which is", bestComparisonStr,
			"than the bestScoreSoFar", bestScoreSoFar))

		if currentScore > bestScoreSoFar {
			bestScoreSoFar = currentScore
			bestMoveSoFar = Some(move)
		}
	}

	PLY_COUNTS := []int{
		1,
		20,
		400,
		8902,
		197281,
		4865609,
		119060324,
		3195901860,
	}

	for i := 0; i < len(PLY_COUNTS); i++ {
		if totalSearched < PLY_COUNTS[i] {
			logger.Println("searched", totalSearched,
				"with initial search", totalSearched-quiescenceSearched, "and ending captures", quiescenceSearched,
				"nodes in", time.Since(startTime), ", ~ perft of ply", i, "(", PLY_COUNTS[i], ")")
			break
		}
	}

	return bestMoveSoFar, NilError
}
