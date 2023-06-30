package search

import (
	. "github.com/cricklet/chessgo/internal/game"
	. "github.com/cricklet/chessgo/internal/helpers"
)

func PlayerIsInCheck(g *GameState) bool {
	return KingIsInCheck(g.Bitboards, g.Player)
}

func IsLegal(g *GameState, move Move) (bool, Error) {
	var returnError Error

	player := g.Player

	var update BoardUpdate
	err := g.PerformMove(move, &update)
	defer func() {
		err = g.UndoUpdate(&update)
		returnError = Join(returnError, err)
	}()

	if !IsNil(err) {
		returnError = Join(returnError, err)
		return false, returnError
	}

	if KingIsInCheck(g.Bitboards, player) {
		returnError = Join(returnError, err)
		return false, returnError
	}

	returnError = Join(returnError, err)
	return true, returnError
}

func NoValidMoves(g *GameState) (bool, Error) {
	foundValidMove := false
	returnError := NilError

	GeneratePseudoMovesSkippingCastling(func(move Move) {
		if foundValidMove || !IsNil(returnError) {
			return
		}

		legal, err := IsLegal(g, move)
		if !IsNil(err) {
			returnError = Join(returnError, err)
		} else if legal {
			foundValidMove = true
		}
	}, g)

	return !foundValidMove, returnError
}
