package chess

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/cricklet/chessgo/internal/helpers"
)

func (p Player) fenString() string {
	if p == White {
		return "w"
	} else {
		return "b"
	}
}

var FEN_STRING_FOR_CASTLING = [2][2]string{
	{"K", "Q"},
	{"k", "q"},
}

func fenStringForCastlingAllowed(playerAndCastlingSideAllowed [2][2]bool) string {
	s := ""
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if playerAndCastlingSideAllowed[i][j] {
				s += FEN_STRING_FOR_CASTLING[i][j]
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

func (b *BoardArray) fenString() string {
	s := ""
	for rank := 7; rank >= 0; rank-- {
		numSpaces := 0
		for file := 0; file < 8; file++ {
			index := IndexFromFileRank(FileRank{File(file), Rank(rank)})
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

func (g *GameState) fenString() string {
	s := ""
	s += fmt.Sprintf("%v %v %v %v %v %v",
		g.Board.fenString(),
		g.player.fenString(),
		fenStringForCastlingAllowed(g.playerAndCastlingSideAllowed),
		fenStringForEnPassant(g.enPassantTarget),
		g.halfMoveClock,
		g.fullMoveClock)

	return s
}

func GamestateFromFenString(s string) (GameState, error) {
	ss := strings.Fields(s)
	if len(ss) != 6 {
		return GameState{}, fmt.Errorf("wrong num %v of fields in str '%v'", len(ss), s)
	}

	game := GameState{}

	boardStr, playerString, castlingRightsString, enPassantTargetString, halfMoveClockString, fullMoveClockString := ss[0], ss[1], ss[2], ss[3], ss[4], ss[5]

	var rankIndex Rank = 7
	var fileIndex File = 0
	for _, c := range boardStr {
		if c == '/' {
			if fileIndex != 8 {
				return GameState{}, fmt.Errorf("not enough squares in rank, '%v'", s)
			}
			rankIndex--
			fileIndex = 0
		} else if indicesToSkip, err := strconv.ParseInt(string(c), 10, 0); err == nil {
			fileIndex += File(indicesToSkip)
		} else if p, err := pieceFromString(c); err == nil {
			// note, we insert pieces into the board in inverse order so the 0th index refers to a1
			game.Board[IndexFromFileRank(FileRank{fileIndex, rankIndex})] = p
			fileIndex++
		} else {
			return GameState{}, fmt.Errorf("unknown character '%v' in '%v'", c, s)
		}
	}

	if player, err := playerFromString(playerString); err == nil {
		game.player = player
	} else {
		return GameState{}, fmt.Errorf("invalid player '%v' in '%v'", playerString, s)
	}

	for _, c := range castlingRightsString {
		switch c {
		case '-':
			continue
		case 'K':
			game.playerAndCastlingSideAllowed[White][Kingside] = true
		case 'Q':
			game.playerAndCastlingSideAllowed[White][Queenside] = true
		case 'k':
			game.playerAndCastlingSideAllowed[BLACK][Kingside] = true
		case 'q':
			game.playerAndCastlingSideAllowed[BLACK][Queenside] = true
		}
	}

	if enPassantTargetString == "-" {
		game.enPassantTarget = Empty[FileRank]()
	} else if enPassantTarget, err := FileRankFromString(enPassantTargetString); err == nil {
		game.enPassantTarget = Some(enPassantTarget)
	} else {
		return GameState{}, fmt.Errorf("invalid en-passant target '%v' in '%v'", enPassantTargetString, s)
	}

	if halfMoveClock, err := strconv.ParseInt(string(halfMoveClockString), 10, 0); err == nil {
		game.halfMoveClock = int(halfMoveClock)
	} else {
		return GameState{}, fmt.Errorf("invalid half move clock '%v' in '%v'", halfMoveClockString, s)
	}

	if fullMoveClock, err := strconv.ParseInt(string(fullMoveClockString), 10, 0); err == nil {
		game.fullMoveClock = int(fullMoveClock)
	} else {
		return GameState{}, fmt.Errorf("invalid full move clock '%v' in '%v'", fullMoveClockString, s)
	}

	return game, nil
}
