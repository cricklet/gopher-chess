package search

import (
	. "github.com/cricklet/chessgo/internal/bitboards"
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
