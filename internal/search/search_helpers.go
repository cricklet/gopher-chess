package search

import (
	"fmt"

	. "github.com/cricklet/chessgo/internal/bitboards"
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

var Inf int = 999999

func InitialBounds() int {
	return Inf + 1
}

func IsMate(score int) bool {
	return score > Inf-100 || score < -Inf+100
}

func IncrementMate(score int) (int, Error) {
	if !IsMate(score) {
		return score, Errorf("only increment mate scores)")
	}

	if score < 0 {
		return score + 1, NilError
	} else {
		return score - 1, NilError
	}
}

func MateNegative() int {
	return -Inf
}

func MateInNScore(n int) (int, Error) {
	if n == 0 {
		return Inf, Errorf("not sure which direction for mate")
	}
	if n < 0 {
		// mate in -1 should give -999998
		// mate in -2 should give -999997
		return -Inf + (-n), NilError
	} else {
		// mate in 1 should give 999998
		// mate in 2 should give 999997
		return Inf - n, NilError
	}
}

func ScoreString(score int) string {
	if score > Inf-100 {
		return fmt.Sprint("mate+", Inf-score)
	}
	if score < -Inf+100 {
		return fmt.Sprint("mate-", -(-Inf - score))
	}
	return fmt.Sprint(score)
}

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
	foundValidMove := false
	returnError := NilError

	GeneratePseudoMovesSkippingCastling(func(move Move) {
		if foundValidMove || !IsNil(returnError) {
			return
		}

		legal, err := IsLegal(g, b, move)
		if !IsNil(err) {
			returnError = Join(returnError, err)
		} else if legal {
			foundValidMove = true
		}
	}, b, g)

	return !foundValidMove, returnError
}
