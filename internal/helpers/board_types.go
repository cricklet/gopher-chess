package helpers

import "fmt"

type File uint
type Rank uint

type FileRank struct {
	File File
	Rank Rank
}

type Player uint

const (
	White Player = iota
	Black
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
	Rook PieceType = iota
	Knight
	Bishop
	King
	Queen
	Pawn
	InvalidPiece
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

func RankFromChar(c byte) (Rank, error) {
	rank := int(c - '1')
	if rank < 0 || rank >= 8 {
		return 0, fmt.Errorf("rank invalid %v", c)
	}
	return Rank(rank), nil
}

func FileFromChar(c byte) (File, error) {
	file := int(c - 'a')
	if file < 0 || file >= 8 {
		return 0, fmt.Errorf("file invalid %v", c)
	}
	return File(file), nil
}

func StringFromBoardIndex(index int) string {
	return FileRankFromIndex(index).String()
}

func (v FileRank) String() string {
	return v.File.String() + v.Rank.String()
}

func FileRankFromString(s string) (FileRank, error) {
	if len(s) != 2 {
		return FileRank{}, fmt.Errorf("invalid location %v", s)
	}

	file, fileErr := FileFromChar(s[0])
	rank, rankErr := RankFromChar(s[1])

	if fileErr != nil || rankErr != nil {
		return FileRank{}, fmt.Errorf("invalid location %v with errors %w, %w", s, fileErr, rankErr)
	}

	return FileRank{file, rank}, nil
}

func PlayerFromString(c string) (Player, error) {
	switch c {
	case "b":
		return Black, nil
	case "w":
		return White, nil
	default:
		return White, fmt.Errorf("invalid player char %v", c)
	}
}

var PieceTypeLookup [16]PieceType = func() [16]PieceType {
	result := [16]PieceType{}
	result[XX] = InvalidPiece
	result[WR] = Rook
	result[WN] = Knight
	result[WB] = Bishop
	result[WK] = King
	result[WQ] = Queen
	result[WP] = Pawn
	result[BR] = Rook
	result[BN] = Knight
	result[BB] = Bishop
	result[BK] = King
	result[BQ] = Queen
	result[BP] = Pawn
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

func (p Piece) PieceType() PieceType {
	return PieceTypeLookup[p]
}

func (p PieceType) IsValid() bool {
	return p >= Rook && p <= Pawn
}

var PlayerForPiece [16]Player = func() [16]Player {
	result := [16]Player{}
	for i := WR; i <= WP; i++ {
		result[i] = White
	}
	for i := BR; i <= BP; i++ {
		result[i] = Black
	}
	return result
}()

func (p Piece) Player() Player {
	if p < BR {
		return White
	}
	return Black
}

func (p Piece) PlayerLookup() Player {
	return PlayerForPiece[p]
}

func (p Player) Other() Player {
	return 1 - p
}

var PieceForPlayer [2][8]Piece = func() [2][8]Piece {
	result := [2][8]Piece{}

	result[White][Rook] = WR
	result[White][Knight] = WN
	result[White][Bishop] = WB
	result[White][King] = WK
	result[White][Queen] = WQ
	result[White][Pawn] = WP

	result[Black][Rook] = BR
	result[Black][Knight] = BN
	result[Black][Bishop] = BB
	result[Black][King] = BK
	result[Black][Queen] = BQ
	result[Black][Pawn] = BP

	return result
}()

// func (p PieceType) forPlayer(player Player) Piece {
// 	return PIECE_FOR_PLAYER[player][p]
// }

func PieceFromString(c rune) (Piece, error) {
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

func (p Piece) IsWhite() bool {
	return p <= WP && p >= WR
}

func (p Piece) IsBlack() bool {
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

func PieceAtFileRank(board BoardArray, location FileRank) Piece {
	return board[IndexFromFileRank(location)]
}

func IndexFromFileRank(location FileRank) int {
	return int(location.Rank)*8 + int(location.File)
}

func FileRankFromIndex(index int) FileRank {
	f := File(index & 0b111)
	r := Rank(index >> 3)
	return FileRank{f, r}
}

func BoardIndexFromString(s string) int {
	location, err := FileRankFromString(s)
	if err != nil {
		panic(err)
	}
	return IndexFromFileRank(location)
}

type CastlingSide int

const (
	Kingside CastlingSide = iota
	Queenside
)

var AllCastlingSides = [2]CastlingSide{Kingside, Queenside}

type MoveType int

const (
	QuietMove MoveType = iota
	CaptureMove
	CastlingMove
	EnPassantMove
)

type Move struct {
	MoveType   MoveType
	StartIndex int
	EndIndex   int
	Evaluation Optional[int]
}

func MoveFromString(s string, m MoveType) Move {
	first := s[0:2]
	second := s[2:4]
	return Move{m, BoardIndexFromString(first), BoardIndexFromString(second), Empty[int]()}
}

func (m Move) String() string {
	return StringFromBoardIndex(m.StartIndex) + StringFromBoardIndex(m.EndIndex)
}

func (m Move) DebugString() string {
	return fmt.Sprintf("%v%v, %v", StringFromBoardIndex(m.StartIndex), StringFromBoardIndex(m.EndIndex), m.MoveType)
}

type BoardUpdate struct {
	Indices [4]int
	Pieces  [4]Piece
	Num     int

	PrevPieces                       [4]Piece
	PrevPlayer                       Player
	PrevPlayerAndCastlingSideAllowed [2][2]bool
	PrevEnPassantTarget              Optional[FileRank]
	PrevHalfMoveClock                int
	PrevFullMoveClock                int
}

func (u *BoardUpdate) Add(prevPiece Piece, index int, piece Piece) {
	u.Indices[u.Num] = index
	u.Pieces[u.Num] = piece
	u.PrevPieces[u.Num] = prevPiece
	u.Num++
}
