package game

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/cricklet/chessgo/internal/helpers"
)

func FenStringForPlayer(p Player) string {
	if p == White {
		return "w"
	} else {
		return "b"
	}
}

var fenStringForCastling = [2][2]string{
	{"K", "Q"},
	{"k", "q"},
}

func fenStringForCastlingAllowed(playerAndCastlingSideAllowed [2][2]bool) string {
	s := ""
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if playerAndCastlingSideAllowed[i][j] {
				s += fenStringForCastling[i][j]
			}
		}
	}
	if len(s) == 0 {
		s += "-"
	}
	return s
}

func fenStringForEnPassant(enPassant Optional[FileRank]) string {
	if enPassant.IsEmpty() {
		return "-"
	}
	return enPassant.Value().String()
}

func fenStringForBoard(b *BoardArray) string {
	s := ""
	for rank := 7; rank >= 0; rank-- {
		numSpaces := 0
		for file := 0; file < 8; file++ {
			index := IndexFromFileRank(FileRank{File: File(file), Rank: Rank(rank)})
			piece := b[index]
			if piece == XX {
				numSpaces++
				continue
			}
			if numSpaces > 0 {
				s += fmt.Sprint(numSpaces)
				numSpaces = 0
			}
			s += piece.String()
		}
		if numSpaces > 0 {
			s += fmt.Sprint(numSpaces)
		}
		if rank != 0 {
			s += "/"
		}
	}
	return s
}

func FenStringForGame(g *GameState) string {
	s := ""
	s += fmt.Sprintf("%v %v %v %v %v %v",
		fenStringForBoard(&g.Board),
		FenStringForPlayer(g.Player),
		fenStringForCastlingAllowed(g.PlayerAndCastlingSideAllowed),
		fenStringForEnPassant(g.EnPassantTarget),
		g.HalfMoveClock,
		g.FullMoveClock)

	return s
}

func GamestateFromFenString(s string) (GameState, Error) {
	ss := strings.Fields(s)
	if len(ss) != 6 && len(ss) != 4 && len(ss) != 2 {
		return GameState{}, Errorf("wrong num %v of fields in str '%v'", len(ss), s)
	}

	game := GameState{}

	boardStr, playerString := ss[0], ss[1]

	var rankIndex Rank = 7
	var fileIndex File = 0
	for _, c := range boardStr {
		if c == '/' {
			if fileIndex != 8 {
				return GameState{}, Errorf("not enough squares in rank, '%v'", s)
			}
			rankIndex--
			fileIndex = 0
		} else if indicesToSkip, err := strconv.ParseInt(string(c), 10, 0); IsNil(err) {
			fileIndex += File(indicesToSkip)
		} else if p, err := PieceFromString(c); IsNil(err) {
			// note, we insert pieces into the board in inverse order so the 0th index refers to a1
			game.Board[IndexFromFileRank(FileRank{File: fileIndex, Rank: rankIndex})] = p
			fileIndex++
		} else {
			return GameState{}, Errorf("unknown character '%v' in '%v'", c, s)
		}
	}

	if player, err := PlayerFromString(playerString); IsNil(err) {
		game.Player = player
	} else {
		return GameState{}, Errorf("invalid player '%v' in '%v'", playerString, s)
	}

	castlingRightsString, enPassantTargetString := "-", "-"
	if len(ss) >= 4 {
		castlingRightsString, enPassantTargetString = ss[2], ss[3]
	}

	halfMoveClockString, fullMoveClockString := "0", "1"
	if len(ss) == 6 {
		halfMoveClockString, fullMoveClockString = ss[4], ss[5]
	}

	for _, c := range castlingRightsString {
		switch c {
		case '-':
			continue
		case 'K':
			game.PlayerAndCastlingSideAllowed[White][Kingside] = true
		case 'Q':
			game.PlayerAndCastlingSideAllowed[White][Queenside] = true
		case 'k':
			game.PlayerAndCastlingSideAllowed[Black][Kingside] = true
		case 'q':
			game.PlayerAndCastlingSideAllowed[Black][Queenside] = true
		}
	}

	if enPassantTargetString == "-" {
		game.EnPassantTarget = Empty[FileRank]()
	} else if enPassantTarget, err := FileRankFromString(enPassantTargetString); IsNil(err) {
		game.EnPassantTarget = Some(enPassantTarget)
	} else {
		return GameState{}, Errorf("invalid en-passant target '%v' in '%v'", enPassantTargetString, s)
	}

	if halfMoveClock, err := strconv.ParseInt(string(halfMoveClockString), 10, 0); IsNil(err) {
		game.HalfMoveClock = int(halfMoveClock)
	} else {
		return GameState{}, Errorf("invalid half move clock '%v' in '%v'", halfMoveClockString, s)
	}

	if fullMoveClock, err := strconv.ParseInt(string(fullMoveClockString), 10, 0); IsNil(err) {
		game.FullMoveClock = int(fullMoveClock)
	} else {
		return GameState{}, Errorf("invalid full move clock '%v' in '%v'", fullMoveClockString, s)
	}

	return game, NilError
}
