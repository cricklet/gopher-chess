package chessgo

import (
	"fmt"
	"strconv"
	"strings"
)

type File uint
type Rank uint

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

type FileRank struct {
	file File
	rank Rank
}

func (v FileRank) String() string {
	return v.file.String() + v.rank.String()
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

type Player uint

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

func (p Piece) pieceType3() PieceType {
	if p < BR {
		return PieceType(p - WR)
	}
	return PieceType(p - BR)
}

func (p Piece) pieceType2() PieceType {
	return PieceType((p - 1) % 6)
}

func (p Piece) pieceType() PieceType {
	return PIECE_TYPE_LOOKUP[p]
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

func (p PieceType) forPlayer(player Player) Piece {
	return PIECE_FOR_PLAYER[player][p]
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

var CASTLING_SIDES = [2]CastlingSide{KINGSIDE, QUEENSIDE}

type Optional[T any] struct {
	_hasValue bool
	_t        T
}

func Some[T any](t T) Optional[T] {
	return Optional[T]{true, t}
}

func Empty[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) IsEmpty() bool {
	return !o._hasValue
}

func (o Optional[T]) HasValue() bool {
	return !o.IsEmpty()
}

func (o Optional[T]) Value() T {
	return o._t
}

type GameState struct {
	Board                        BoardArray
	player                       Player
	playerAndCastlingSideAllowed [2][2]bool
	enPassantTarget              Optional[FileRank]
	halfMoveClock                int
	fullMoveClock                int
}

func absDiff(x int, y int) int {
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

func (g *GameState) moveFromString(s string) Move {
	start := boardIndexFromString(s[0:2])
	end := boardIndexFromString(s[2:4])

	var moveType MoveType
	if g.Board[end] == XX {
		startPieceType := g.Board[start].pieceType()
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

type OldGameState struct {
	player                       Player
	playerAndCastlingSideAllowed [2][2]bool
	enPassantTarget              Optional[FileRank]
	halfMoveClock                int
	fullMoveClock                int
}

type BoardUpdate struct {
	indices [4]int
	pieces  [4]Piece
	num     int

	old [4]Piece
}

func (u *BoardUpdate) Add(g *GameState, index int, piece Piece) {
	u.indices[u.num] = index
	u.pieces[u.num] = piece
	u.old[u.num] = g.Board[index]
	u.num++
}

func SetupBoardUpdate(g *GameState, move Move, output *BoardUpdate) {
	startPiece := g.Board[move.startIndex]

	switch move.moveType {
	case QUIET_MOVE:
		{
			output.Add(g, move.startIndex, XX)
			output.Add(g, move.endIndex, startPiece)
		}
	case CAPTURE_MOVE:
		{
			output.Add(g, move.startIndex, XX)
			output.Add(g, move.endIndex, startPiece)
		}
	case EN_PASSANT_MOVE:
		{
			startPlayer := startPiece.player()
			backwardsDir := S
			if startPlayer == BLACK {
				backwardsDir = N
			}

			captureIndex := move.endIndex + OFFSETS[backwardsDir]
			output.Add(g, captureIndex, XX)
			output.Add(g, move.startIndex, XX)
			output.Add(g, move.endIndex, startPiece)
		}
	case CASTLING_MOVE:
		{
			rookStartIndex, rookEndIndex := rookMoveForCastle(move.startIndex, move.endIndex)
			rookPiece := g.Board[rookStartIndex]

			output.Add(g, move.startIndex, XX)
			output.Add(g, rookStartIndex, XX)
			output.Add(g, move.endIndex, startPiece)
			output.Add(g, rookEndIndex, rookPiece)
		}
	}
}

func RecordCurrentState(g *GameState, output *OldGameState) {
	output.player = g.player
	output.playerAndCastlingSideAllowed = g.playerAndCastlingSideAllowed
	output.enPassantTarget = g.enPassantTarget
	output.fullMoveClock = g.fullMoveClock
	output.halfMoveClock = g.halfMoveClock
}

func (g *GameState) performMove(move Move, update BoardUpdate) {
	startPiece := g.Board[move.startIndex]

	g.enPassantTarget = Empty[FileRank]()
	if move.moveType == QUIET_MOVE && isPawnSkip(startPiece, move) {
		g.enPassantTarget = Some(fileRankFromBoardIndex(enPassantTarget(move)))
	}

	for i := 0; i < update.num; i++ {
		g.Board[update.indices[i]] = update.pieces[i]
	}

	g.halfMoveClock++
	if g.player == BLACK {
		g.fullMoveClock++
	}
	g.player = g.player.other()
}

func (g *GameState) undoUpdate(undo OldGameState, update BoardUpdate) {
	g.player = undo.player
	g.playerAndCastlingSideAllowed = undo.playerAndCastlingSideAllowed
	g.enPassantTarget = undo.enPassantTarget
	g.fullMoveClock = undo.fullMoveClock
	g.halfMoveClock = undo.halfMoveClock

	for i := update.num - 1; i >= 0; i-- {
		index := update.indices[i]
		piece := update.old[i]

		g.Board[index] = piece
	}
}
func (g *GameState) enemy() Player {
	return g.player.other()
}

func (g *GameState) whiteCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[WHITE][KINGSIDE]
}
func (g *GameState) whiteCanCastleQueenside() bool {
	return g.playerAndCastlingSideAllowed[BLACK][QUEENSIDE]
}
func (g *GameState) blackCanCastleKingside() bool {
	return g.playerAndCastlingSideAllowed[WHITE][KINGSIDE]
}
func (g *GameState) blackCanCastleQueenside() bool {
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
	return enPassant.Value().String()
}

func (g *GameState) fenString() string {
	s := ""
	for rank := 7; rank >= 0; rank-- {
		numSpaces := 0
		for file := 0; file < 8; file++ {
			index := boardIndexFromFileRank(FileRank{File(file), Rank(rank)})
			piece := g.Board[index]
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
			game.Board[boardIndexFromFileRank(FileRank{fileIndex, rankIndex})] = p
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
func init() {
	// defer profile.Start(profile.ProfilePath(".")).Stop()
	// defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	initMagicTables()
}

type Runner struct {
	g *GameState
	b *Bitboards
}

func (r *Runner) HandleInputAndReturnDone(input string) bool {
	if input == "uci" {
		fmt.Println("name chess-go")
		fmt.Println("id author Kenrick Rilee")
		fmt.Println("uciok")
	} else if input == "isready" {
		fmt.Println("readyok")
	} else if strings.HasPrefix(input, "position fen ") {
		s := strings.TrimPrefix(input, "position fen ")
		game, err := GamestateFromFenString(s)
		if err != nil {
			panic(fmt.Errorf("couldn't create game from %v", s))
		}
		r.g = &game

		bitboards := SetupBitboards(r.g)
		r.b = &bitboards
	} else if strings.HasPrefix(input, "go") {
		move := Search(r.g, r.b, 6)
		if move.IsEmpty() {
			panic(fmt.Errorf("failed to find move for %v ", r.g.Board.String()))
		}
		fmt.Printf("bestmove %v\n", move.Value().String())
	} else if input == "quit" {
		return true
	}
	return false
}
