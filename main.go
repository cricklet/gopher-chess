package chessgo

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

type File int
type Rank int

func (f File) string() string {
	return [8]string{
		"a", "b", "c", "d", "e", "f", "g", "h",
	}[f]
}
func (r Rank) string() string {
	return [8]string{
		"1", "2", "3", "4", "5", "6", "7", "8",
	}[r]
}

func rankFromChar(c byte) (Rank, error) {
	rank := int(c - '1')
	if rank < 0 || rank >= 8 {
		return 0, errors.New(fmt.Sprintf("rank invalid %v", c))
	}
	return Rank(rank), nil
}

func fileFromChar(c byte) (File, error) {
	file := int(c - 'a')
	if file < 0 || file >= 8 {
		return 0, errors.New(fmt.Sprintf("file invalid %v", c))
	}
	return File(file), nil
}

type FileRank struct {
	file File
	rank Rank
}

func (v FileRank) string() string {
	return v.file.string() + v.rank.string()
}

func fileRankFromString(s string) (FileRank, error) {
	if len(s) != 2 {
		return FileRank{}, errors.New(fmt.Sprintf("invalid location %v", s))
	}

	file, fileErr := fileFromChar(s[0])
	rank, rankErr := rankFromChar(s[1])

	if fileErr != nil || rankErr != nil {
		return FileRank{}, errors.New(fmt.Sprintf("invalid location %v with errors %v, %v", s, fileErr, rankErr))
	}

	return FileRank{file, rank}, nil
}

type Player int

const (
	WHITE Player = iota
	BLACK
)

func playerFromString(c string) (Player, error) {
	switch c {
	case "b":
		return BLACK, nil
	case "w":
		return WHITE, nil
	default:
		return WHITE, errors.New(fmt.Sprintf("invalid player char %v", c))
	}
}

func (p Player) other() Player {
	return 1 - p
}

type Piece int

const (
	XX Piece = iota
	WR
	WN
	WB
	WK
	WQ
	WP
	BR
	BN
	BB
	BK
	BQ
	BP
)

func pieceFromString(c rune) (Piece, error) {
	switch c {
	case 'R':
		return WR, nil
	case 'N':
		return WN, nil
	case 'B':
		return WB, nil
	case 'K':
		return WK, nil
	case 'Q':
		return WQ, nil
	case 'P':
		return WP, nil
	case 'r':
		return BR, nil
	case 'n':
		return BN, nil
	case 'b':
		return BB, nil
	case 'k':
		return BK, nil
	case 'q':
		return BQ, nil
	case 'p':
		return BP, nil
	default:
		return XX, errors.New(fmt.Sprintf("invalid piece %v", c))
	}
}

func (p Piece) string() string {
	return []string{
		" ",
		"R",
		"N",
		"B",
		"K",
		"Q",
		"P",
		"r",
		"n",
		"b",
		"k",
		"q",
		"p",
	}[p]
}

func (p Piece) isWhite() bool {
	return p <= WP && p >= WR
}

func (p Piece) isBlack() bool {
	return p <= BP && p >= BR
}

func (p Piece) isEmpty() bool {
	return p == XX
}

type BoardArray [64]Piece

type NaturalBoardArray [64]Piece

func (n NaturalBoardArray) AsBoardArray() BoardArray {
	b := BoardArray{}

	for rank := 0; rank < 8; rank++ {
		index := rank * 8
		newIndex := (7 - rank) * 8
		copy(b[index:index+8], n[newIndex:newIndex+8])
	}

	return b
}

func (b BoardArray) string() string {
	result := ""
	for rank := 7; rank >= 0; rank-- {
		row := b[rank*8 : (rank+1)*8]
		for _, p := range row {
			result += p.string()
		}
		result += "\n"
	}
	return result
}

func pieceAtFileRank(board BoardArray, location FileRank) Piece {
	return board[boardIndexFromFileRank(location)]
}

func boardIndexFromFileRank(location FileRank) int {
	return int(location.rank)*8 + int(location.file)
}

func fileRankFromBoardIndex(index int) FileRank {
	f := File(index & 0b111)
	r := Rank(index >> 3)
	return FileRank{f, r}
}

func boardIndexFromString(s string) int {
	location, err := fileRankFromString(s)
	if err != nil {
		panic(err)
	}
	return boardIndexFromFileRank(location)
}

type GameState struct {
	board                   BoardArray
	player                  Player
	whiteCanCastleKingside  bool
	whiteCanCastleQueenside bool
	blackCanCastleKingside  bool
	blackCanCastleQueenside bool
	enPassantTarget         *FileRank
	halfMoveClock           int
	fullMoveClock           int
}

func gamestateFromString(s string) (GameState, error) {
	ss := strings.Fields(s)
	if len(ss) != 6 {
		return GameState{}, errors.New(fmt.Sprintf("wrong num %v of fields in str '%v'", len(ss), s))
	}

	game := GameState{}

	boardStr, playerString, castlingRightsString, enPassantTargetString, halfMoveClockString, fullMoveClockString := ss[0], ss[1], ss[2], ss[3], ss[4], ss[5]

	var rankIndex Rank = 7
	var fileIndex File = 0
	for _, c := range boardStr {
		if c == '/' {
			if fileIndex != 8 {
				return GameState{}, errors.New(fmt.Sprintf("not enough squares in rank, '%v'", s))
			}
			rankIndex--
			fileIndex = 0
		} else if indicesToSkip, err := strconv.ParseInt(string(c), 10, 0); err == nil {
			fileIndex += File(indicesToSkip)
		} else if p, err := pieceFromString(c); err == nil {
			// note, we insert pieces into the board in inverse order so the 0th index refers to a1
			game.board[boardIndexFromFileRank(FileRank{fileIndex, rankIndex})] = p
			fileIndex++
		} else {
			return GameState{}, errors.New(fmt.Sprintf("unknown character '%v' in '%v'", c, s))
		}
	}

	if player, err := playerFromString(playerString); err == nil {
		game.player = player
	} else {
		return GameState{}, errors.New(fmt.Sprintf("invalid player '%v' in '%v'", playerString, s))
	}

	for _, c := range castlingRightsString {
		switch c {
		case '-':
			continue
		case 'K':
			game.whiteCanCastleKingside = true
		case 'Q':
			game.whiteCanCastleQueenside = true
		case 'k':
			game.blackCanCastleKingside = true
		case 'q':
			game.blackCanCastleQueenside = true
		}
	}

	if enPassantTargetString == "-" {
		game.enPassantTarget = nil
	} else if enPassantTarget, err := fileRankFromString(enPassantTargetString); err == nil {
		game.enPassantTarget = &enPassantTarget
	} else {
		return GameState{}, errors.New(fmt.Sprintf("invalid en-passant target '%v' in '%v'", enPassantTargetString, s))
	}

	if halfMoveClock, err := strconv.ParseInt(string(halfMoveClockString), 10, 0); err == nil {
		game.halfMoveClock = int(halfMoveClock)
	} else {
		return GameState{}, errors.New(fmt.Sprintf("invalid half move clock '%v' in '%v'", halfMoveClockString, s))
	}

	if fullMoveClock, err := strconv.ParseInt(string(fullMoveClockString), 10, 0); err == nil {
		game.fullMoveClock = int(fullMoveClock)
	} else {
		return GameState{}, errors.New(fmt.Sprintf("invalid full move clock '%v' in '%v'", fullMoveClockString, s))
	}

	return game, nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(string(debug.Stack()))
		}
	}()

	var testFile File = 0
	var testRank Rank = 1

	scanner := bufio.NewScanner(os.Stdin)
	done := false
	for !done && scanner.Scan() {
		input := scanner.Text()
		if input == "uci" {
			fmt.Println("name chess-go")
			fmt.Println("id author Kenrick Rilee")
			fmt.Println("uciok")
		} else if input == "isready" {
			fmt.Println("readyok")
		} else if strings.HasPrefix(input, "go") {
			fmt.Printf("bestmove %v%v%v%v\n", testFile, testRank, testFile, (testRank + 1))
			testFile++
		} else if input == "quit" {
			done = true
		}
	}
}
