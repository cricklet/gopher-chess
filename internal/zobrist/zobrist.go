package zobrist

import (
	"math/rand"

	. "github.com/cricklet/chessgo/internal/helpers"
)

var ZobristPieceAtSquare [13] /*includes empty*/ [64]uint64
var ZobristSideToMove uint64
var ZobristCastlingRights [4]uint64
var ZobristEnPassant [8]uint64

func init() {
	r := rand.New(rand.NewSource(32879419))
	ZobristSideToMove = r.Uint64()
	for i := 0; i < 4; i++ {
		ZobristCastlingRights[i] = r.Uint64()
	}
	for i := 0; i < 8; i++ {
		ZobristEnPassant[i] = r.Uint64()
	}
	for piece := 1; /* skip empty */ piece < 13; piece++ {
		for boardIndex := 0; boardIndex < 64; boardIndex++ {
			ZobristPieceAtSquare[piece][boardIndex] = r.Uint64()
		}
	}
}

func HashForBoardPosition(
	board *BoardArray,
	player Player,
	playerAndCastlingSideAllowed *[2][2]bool,
	enPassantTarget Optional[FileRank],
) uint64 {
	hash := uint64(0)
	for boardIndex := 0; boardIndex < 64; boardIndex++ {
		piece := board[boardIndex]
		hash ^= ZobristPieceAtSquare[piece][boardIndex]

		// fmt.Printf("^ piece at square %v,%v %v\n", int(piece), boardIndex, ZobristPieceAtSquare[piece][boardIndex])
	}
	if player == Black {
		hash ^= ZobristSideToMove
		// fmt.Printf("^ side to move <black> %v\n", ZobristSideToMove)

	}
	for player := 0; player < 2; player++ {
		for side := 0; side < 2; side++ {
			if playerAndCastlingSideAllowed[player][side] {
				hash ^= ZobristCastlingRights[2*player+side]

				// fmt.Printf("^ castling rights %v%v %v\n", player, side, ZobristCastlingRights[2*player+side])
			}
		}
	}
	if enPassantTarget.HasValue() {
		hash ^= ZobristEnPassant[enPassantTarget.Value().File]
		// fmt.Printf("^ en passant %v %v\n", enPassantTarget.Value().File, ZobristEnPassant[enPassantTarget.Value().File])
	}
	return hash
}

func UpdateHash(hash uint64, update *BoardUpdate, newCastlingRights *[2][2]bool, newEnPassant Optional[FileRank]) uint64 {
	for i := 0; i < update.Num; i++ {
		index := update.Indices[i]

		newPiece := update.Pieces[i]
		hash ^= ZobristPieceAtSquare[newPiece][index]
		// fmt.Printf("^ piece at square %v,%v %v\n", int(newPiece), index, ZobristPieceAtSquare[newPiece][index])

		prevPiece := update.PrevPieces[i]
		hash ^= ZobristPieceAtSquare[prevPiece][index]
		// fmt.Printf("^ piece at square %v,%v %v\n", int(prevPiece), index, ZobristPieceAtSquare[prevPiece][index])
	}

	hash ^= ZobristSideToMove

	for player := 0; player < 2; player++ {
		for side := 0; side < 2; side++ {
			if newCastlingRights[player][side] != update.PreviousCastlingRights[player][side] {
				hash ^= ZobristCastlingRights[2*player+side]
				// fmt.Printf("^ castling rights %v%v %v\n", player, side, ZobristCastlingRights[2*player+side])
			}
		}
	}

	if newEnPassant != update.PrevEnPassantTarget {
		if newEnPassant.HasValue() {
			hash ^= ZobristEnPassant[newEnPassant.Value().File]
			// fmt.Printf("^ en passant %v %v\n", newEnPassant.Value().File, ZobristEnPassant[newEnPassant.Value().File])
		}
		if update.PrevEnPassantTarget.HasValue() {
			hash ^= ZobristEnPassant[update.PrevEnPassantTarget.Value().File]
			// fmt.Printf("^ en passant %v %v\n", newEnPassant.Value().File, ZobristEnPassant[update.PrevEnPassantTarget.Value().File])
		}
	}

	return hash
}
