package chess

import "fmt"

type File uint
type Rank uint

type FileRank struct {
	file File
	rank Rank
}

type Player uint

const (
	WHITE Player = iota
	BLACK
)

type Piece uint

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

type PieceType uint

const (
	ROOK PieceType = iota
	KNIGHT
	BISHOP
	KING
	QUEEN
	PAWN
	INVALID
)

func (f File) String() string {
	return [8]string{
		"a", "b", "c", "d", "e", "f", "g", "h",
	}[f]
}
func (r Rank) String() string {
	return [8]string{
		"1", "2", "3", "4", "5", "6", "7", "8",
	}[r]
}

func rankFromChar(c byte) (Rank, error) {
	rank := int(c - '1')
	if rank < 0 || rank >= 8 {
		return 0, fmt.Errorf("rank invalid %v", c)
	}
	return Rank(rank), nil
}

func fileFromChar(c byte) (File, error) {
	file := int(c - 'a')
	if file < 0 || file >= 8 {
		return 0, fmt.Errorf("file invalid %v", c)
	}
	return File(file), nil
}

func stringFromBoardIndex(index int) string {
	return FileRankFromIndex(index).String()
}

func (v FileRank) String() string {
	return v.file.String() + v.rank.String()
}

func FileRankFromString(s string) (FileRank, error) {
	if len(s) != 2 {
		return FileRank{}, fmt.Errorf("invalid location %v", s)
	}

	file, fileErr := fileFromChar(s[0])
	rank, rankErr := rankFromChar(s[1])

	if fileErr != nil || rankErr != nil {
		return FileRank{}, fmt.Errorf("invalid location %v with errors %w, %w", s, fileErr, rankErr)
	}

	return FileRank{file, rank}, nil
}

func playerFromString(c string) (Player, error) {
	switch c {
	case "b":
		return BLACK, nil
	case "w":
		return WHITE, nil
	default:
		return WHITE, fmt.Errorf("invalid player char %v", c)
	}
}

var PIECE_TYPE_LOOKUP [16]PieceType = func() [16]PieceType {
	result := [16]PieceType{}
	result[XX] = INVALID
	result[WR] = ROOK
	result[WN] = KNIGHT
	result[WB] = BISHOP
	result[WK] = KING
	result[WQ] = QUEEN
	result[WP] = PAWN
	result[BR] = ROOK
	result[BN] = KNIGHT
	result[BB] = BISHOP
	result[BK] = KING
	result[BQ] = QUEEN
	result[BP] = PAWN
	return result
}()

// func (p Piece) pieceType3() PieceType {
// 	if p < BR {
// 		return PieceType(p - WR)
// 	}
// 	return PieceType(p - BR)
// }

// func (p Piece) pieceType2() PieceType {
// 	return PieceType((p - 1) % 6)
// }

func (p Piece) pieceType() PieceType {
	return PIECE_TYPE_LOOKUP[p]
}

func (p PieceType) isValid() bool {
	return p >= ROOK && p <= PAWN
}

var PLAYER_FOR_PIECE [16]Player = func() [16]Player {
	result := [16]Player{}
	for i := WR; i <= WP; i++ {
		result[i] = WHITE
	}
	for i := BR; i <= BP; i++ {
		result[i] = BLACK
	}
	return result
}()

func (p Piece) player() Player {
	if p < BR {
		return WHITE
	}
	return BLACK
}

func (p Piece) player2() Player {
	return PLAYER_FOR_PIECE[p]
}

func (p Player) Other() Player {
	return 1 - p
}

var PIECE_FOR_PLAYER [2][8]Piece = func() [2][8]Piece {
	result := [2][8]Piece{}

	result[WHITE][ROOK] = WR
	result[WHITE][KNIGHT] = WN
	result[WHITE][BISHOP] = WB
	result[WHITE][KING] = WK
	result[WHITE][QUEEN] = WQ
	result[WHITE][PAWN] = WP

	result[BLACK][ROOK] = BR
	result[BLACK][KNIGHT] = BN
	result[BLACK][BISHOP] = BB
	result[BLACK][KING] = BK
	result[BLACK][QUEEN] = BQ
	result[BLACK][PAWN] = BP

	return result
}()

// func (p PieceType) forPlayer(player Player) Piece {
// 	return PIECE_FOR_PLAYER[player][p]
// }

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
		return XX, fmt.Errorf("invalid piece %v", c)
	}
}

func (p Piece) String() string {
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

// func (p Piece) isEmpty() bool {
// 	return p == XX
// }

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

func (b BoardArray) String() string {
	result := ""
	for rank := 7; rank >= 0; rank-- {
		row := b[rank*8 : (rank+1)*8]
		for _, p := range row {
			result += p.String()
		}
		if rank != 0 {
			result += "\n"
		}
	}
	return result
}

func pieceAtFileRank(board BoardArray, location FileRank) Piece {
	return board[IndexFromFileRank(location)]
}

func IndexFromFileRank(location FileRank) int {
	return int(location.rank)*8 + int(location.file)
}

func FileRankFromIndex(index int) FileRank {
	f := File(index & 0b111)
	r := Rank(index >> 3)
	return FileRank{f, r}
}

func boardIndexFromString(s string) int {
	location, err := FileRankFromString(s)
	if err != nil {
		panic(err)
	}
	return IndexFromFileRank(location)
}

type CastlingSide int

const (
	KINGSIDE CastlingSide = iota
	QUEENSIDE
)

var CASTLING_SIDES = [2]CastlingSide{KINGSIDE, QUEENSIDE}
