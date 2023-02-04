package chessgo

import (
	"bufio"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

type File uint8
type Rank uint8

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

type FileRank struct {
	file File
	rank Rank
}

func (v FileRank) string() string {
	return v.file.string() + v.rank.string()
}

func fileRankFromString(s string) (FileRank, error) {
	if len(s) != 2 {
		return FileRank{}, fmt.Errorf("invalid location %v", s)
	}

	file, fileErr := fileFromChar(s[0])
	rank, rankErr := rankFromChar(s[1])

	if fileErr != nil || rankErr != nil {
		return FileRank{}, fmt.Errorf("invalid location %v with errors %v, %v", s, fileErr, rankErr)
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
		return WHITE, fmt.Errorf("invalid player char %v", c)
	}
}

func (p Player) other() Player {
	return 1 - p
}

type Piece uint8

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

func (p Piece) pieceType() PieceType {
	switch p {
	case WR:
		return ROOK
	case WN:
		return KNIGHT
	case WB:
		return BISHOP
	case WK:
		return KING
	case WQ:
		return QUEEN
	case WP:
		return PAWN
	case BR:
		return ROOK
	case BN:
		return KNIGHT
	case BB:
		return BISHOP
	case BK:
		return KING
	case BQ:
		return QUEEN
	case BP:
		return PAWN
	}
	return EMPTY
}

func (p Piece) player() Player {
	if p >= WR && p <= WP {
		return WHITE
	}
	if p >= BR && p <= BP {
		return BLACK
	}

	panic("only call player() on a non-empty piece")
}

type PieceType uint8

const (
	ROOK PieceType = iota
	KNIGHT
	BISHOP
	KING
	QUEEN
	PAWN
	EMPTY
)

func (p PieceType) forPlayer(player Player) Piece {
	switch player {
	case WHITE:
		switch p {
		case ROOK:
			return WR
		case KNIGHT:
			return WN
		case BISHOP:
			return WB
		case KING:
			return WK
		case QUEEN:
			return WQ
		case PAWN:
			return WP
		}
	case BLACK:
		switch p {
		case ROOK:
			return BR
		case KNIGHT:
			return BN
		case BISHOP:
			return BB
		case KING:
			return BK
		case QUEEN:
			return BQ
		case PAWN:
			return BP
		}
	}

	panic(fmt.Sprint("could not determine piece based on", player, p))
}

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
		if rank != 0 {
			result += "\n"
		}
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

type CastlingSide int

const (
	KINGSIDE CastlingSide = iota
	QUEENSIDE
)

type Optional[T any] []T

func Some[T any](t T) Optional[T] {
	return Optional[T]{t}
}

func Empty[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) IsEmpty() bool {
	return len(o) == 0
}

func (o Optional[T]) HasValue() bool {
	return !o.IsEmpty()
}

func (o Optional[T]) Value() T {
	return o[0]
}

type GameState struct {
	board                        BoardArray
	player                       Player
	playerAndCastlingSideAllowed [2][2]bool
	enPassantTarget              Optional[FileRank]
	halfMoveClock                int
	fullMoveClock                int
}

func absDiff[T int](x T, y T) T {
	if x < y {
		return y - x
	}
	return x - y
}

func isPawnCapture(startPieceType PieceType, startIndex int, endIndex int) bool {
	if startPieceType != PAWN {
		return false
	}

	start := fileRankFromBoardIndex(startIndex)
	end := fileRankFromBoardIndex(endIndex)

	return absDiff(int(start.file), int(end.file)) == 1 && absDiff(int(start.rank), int(end.rank)) == 1
}

func (g GameState) moveFromString(s string) Move {
	start := boardIndexFromString(s[0:2])
	end := boardIndexFromString(s[2:4])

	var moveType MoveType
	if g.board[end] == XX {
		startPieceType := g.board[start].pieceType()
		// either a quiet, castle, or en passant
		if startPieceType == KING && absDiff(start, end) == 2 {
			moveType = CASTLING_MOVE
		} else if isPawnCapture(startPieceType, start, end) {
			moveType = EN_PASSANT_MOVE
		} else {
			moveType = QUIET_MOVE
		}
	} else {
		moveType = CAPTURE_MOVE
	}
	return Move{moveType, start, end}
}

func isPawnSkip(startPiece Piece, move Move) bool {
	if move.moveType != QUIET_MOVE || startPiece.pieceType() != PAWN {
		return false
	}

	return absDiff(move.startIndex, move.endIndex) == OFFSET_N+OFFSET_N
}

func enPassantTarget(move Move) int {
	if move.endIndex > move.startIndex {
		return move.startIndex + OFFSET_N
	} else {
		return move.startIndex + OFFSET_S
	}
}

func (g *GameState) performMove(move Move) {
	startPiece := g.board[move.startIndex]
	g.enPassantTarget = Empty[FileRank]()
	switch move.moveType {
	case QUIET_MOVE:
		{
			g.board[move.startIndex] = XX
			g.board[move.endIndex] = startPiece

			if isPawnSkip(startPiece, move) {
				g.enPassantTarget = Some(fileRankFromBoardIndex(enPassantTarget(move)))
			}
		}
	case CAPTURE_MOVE:
		{
			g.board[move.startIndex] = XX
			g.board[move.endIndex] = startPiece
		}
	case EN_PASSANT_MOVE:
		{
			startPlayer := startPiece.player()
			backwardsDir := S
			if startPlayer == BLACK {
				backwardsDir = N
			}

			captureIndex := move.endIndex - OFFSETS[backwardsDir]
			g.board[captureIndex] = XX
			g.board[move.startIndex] = XX
			g.board[move.endIndex] = startPiece
		}
	case CASTLING_MOVE:
		{
			rookStartIndex, rookEndIndex := rookMoveForCastle(move.startIndex, move.endIndex)
			rookPiece := g.board[rookStartIndex]

			g.board[move.startIndex] = XX
			g.board[rookStartIndex] = XX
			g.board[move.endIndex] = startPiece
			g.board[rookEndIndex] = rookPiece
		}
	}
	g.halfMoveClock++
	if g.player == BLACK {
		g.fullMoveClock++
	}
	g.player = g.player.other()
}

func (g GameState) enemy() Player {
	return g.player.other()
}

func (g GameState) whiteCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[WHITE][KINGSIDE]
}
func (g GameState) whiteCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[BLACK][QUEENSIDE]
}
func (g GameState) blackCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[WHITE][KINGSIDE]
}
func (g GameState) blackCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[BLACK][QUEENSIDE]
}

func (p Player) fenString() string {
	if p == WHITE {
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
	return enPassant.Value().string()
}

func (g GameState) fenString() string {
	s := ""
	for rank := 7; rank >= 0; rank-- {
		numSpaces := 0
		for file := 0; file < 8; file++ {
			index := boardIndexFromFileRank(FileRank{File(file), Rank(rank)})
			piece := g.board[index]
			if piece == XX {
				numSpaces++
				continue
			}
			if numSpaces > 0 {
				s += fmt.Sprint(numSpaces)
				numSpaces = 0
			}
			s += piece.string()
		}
		if numSpaces > 0 {
			s += fmt.Sprint(numSpaces)
		}
		s += "/"
	}
	s += fmt.Sprintf(" %v %v %v %v %v",
		g.player.fenString(),
		fenStringForCastlingAllowed(g.playerAndCastlingSideAllowed),
		fenStringForEnPassant(g.enPassantTarget),
		g.halfMoveClock,
		g.fullMoveClock)

	return s
}

func fenStringWithMoveApplied(s string, m string) (string, error) {
	g, err := gamestateFromFenString(s)
	if err != nil {
		return "", err
	}

	move := g.moveFromString(m)
	g.performMove(move)

	return g.fenString(), nil
}

func gamestateFromFenString(s string) (GameState, error) {
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
			game.board[boardIndexFromFileRank(FileRank{fileIndex, rankIndex})] = p
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
			game.playerAndCastlingSideAllowed[WHITE][KINGSIDE] = true
		case 'Q':
			game.playerAndCastlingSideAllowed[WHITE][QUEENSIDE] = true
		case 'k':
			game.playerAndCastlingSideAllowed[BLACK][KINGSIDE] = true
		case 'q':
			game.playerAndCastlingSideAllowed[BLACK][QUEENSIDE] = true
		}
	}

	if enPassantTargetString == "-" {
		game.enPassantTarget = Empty[FileRank]()
	} else if enPassantTarget, err := fileRankFromString(enPassantTargetString); err == nil {
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

func init() {
	// defer profile.Start(profile.ProfilePath(".")).Stop()
	// defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	initMagicTables()
}
