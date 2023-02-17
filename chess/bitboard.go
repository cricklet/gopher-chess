package chess

import (
	"fmt"
	"math/bits"
	"strings"
)

type Bitboard uint64

type PlayerBitboards struct {
	occupied Bitboard
	pieces   [6]Bitboard // indexed via PieceType
}

type Bitboards struct {
	occupied Bitboard
	players  [2]PlayerBitboards
}

type Dir int

const (
	N Dir = iota
	S
	E
	W

	NE
	NW
	SE
	SW

	NNE
	NNW
	SSE
	SSW
	ENE
	ESE
	WNW
	WSW

	NUM_DIRS
)

var KNIGHT_DIRS = []Dir{
	NNE,
	NNW,
	SSE,
	SSW,
	ENE,
	ESE,
	WNW,
	WSW,
}

var ROOK_DIRS = []Dir{
	N,
	S,
	E,
	W,
}

var BISHOP_DIRS = []Dir{
	NE,
	NW,
	SE,
	SW,
}

var KING_DIRS = []Dir{
	N,
	S,
	E,
	W,
	NE,
	NW,
	SE,
	SW,
}

const (
	OFFSET_N int = 8
	OFFSET_S int = -8
	OFFSET_E int = 1
	OFFSET_W int = -1
)

var OFFSETS = [NUM_DIRS]int{
	OFFSET_N,
	OFFSET_S,
	OFFSET_E,
	OFFSET_W,

	OFFSET_N + OFFSET_E,
	OFFSET_N + OFFSET_W,
	OFFSET_S + OFFSET_E,
	OFFSET_S + OFFSET_W,

	OFFSET_N + OFFSET_N + OFFSET_E,
	OFFSET_N + OFFSET_N + OFFSET_W,
	OFFSET_S + OFFSET_S + OFFSET_E,
	OFFSET_S + OFFSET_S + OFFSET_W,
	OFFSET_E + OFFSET_N + OFFSET_E,
	OFFSET_E + OFFSET_S + OFFSET_E,
	OFFSET_W + OFFSET_N + OFFSET_W,
	OFFSET_W + OFFSET_S + OFFSET_W,
}

var PAWN_PUSH_OFFSETS = [2]int{
	OFFSET_N,
	OFFSET_S,
}

var PAWN_PROMOTION_BITBOARD = bitboardFromStrings([8]string{
	"11111111",
	"00000000",
	"00000000",
	"00000000",
	"00000000",
	"00000000",
	"00000000",
	"11111111",
})

var PAWN_CAPTURE_OFFSETS = [2][2]int{
	{ // WHITE
		OFFSET_N + OFFSET_E, OFFSET_N + OFFSET_W,
	},
	{
		OFFSET_S + OFFSET_E, OFFSET_S + OFFSET_W,
	},
}

var ALL_ZEROS Bitboard = Bitboard(0)
var ALL_ONES Bitboard = ^ALL_ZEROS

var ZEROS = []int{0, 0, 0, 0, 0, 0, 0, 0}
var ONES = []int{1, 1, 1, 1, 1, 1, 1, 1}
var SIXES = []int{6, 6, 6, 6, 6, 6, 6, 6}
var SEVENS = []int{7, 7, 7, 7, 7, 7, 7, 7}
var ZERO_TO_SEVEN = []int{0, 1, 2, 3, 4, 5, 6, 7}

var (
	MASK_WHITE_STARTING_PAWNS = ^zerosForRange(ZERO_TO_SEVEN, ONES)
	MASK_BLACK_STARTING_PAWNS = ^zerosForRange(ZERO_TO_SEVEN, SIXES)
)

var STARTING_PAWNS_FOR_PLAYER = [2]Bitboard{
	MASK_WHITE_STARTING_PAWNS,
	MASK_BLACK_STARTING_PAWNS,
}

func maskStartingPawnsForPlayer(player Player) Bitboard {
	return STARTING_PAWNS_FOR_PLAYER[player]
}

var (
	MASK_N Bitboard = zerosForRange(ZERO_TO_SEVEN, SEVENS)
	MASK_S Bitboard = zerosForRange(ZERO_TO_SEVEN, ZEROS)
	MASK_E Bitboard = zerosForRange(SEVENS, ZERO_TO_SEVEN)
	MASK_W Bitboard = zerosForRange(ZEROS, ZERO_TO_SEVEN)

	MASK_NN Bitboard = zerosForRange(ZERO_TO_SEVEN, SIXES)
	MASK_SS Bitboard = zerosForRange(ZERO_TO_SEVEN, ONES)
	MASK_EE Bitboard = zerosForRange(SIXES, ZERO_TO_SEVEN)
	MASK_WW Bitboard = zerosForRange(ONES, ZERO_TO_SEVEN)

	MASK_ALL_EDGES Bitboard = MASK_N & MASK_S & MASK_E & MASK_W
)

var PRE_MOVE_MASKS = [NUM_DIRS]Bitboard{
	MASK_N,
	MASK_S,
	MASK_E,
	MASK_W,

	MASK_N & MASK_E,
	MASK_N & MASK_W,
	MASK_S & MASK_E,
	MASK_S & MASK_W,

	MASK_NN & MASK_N & MASK_E,
	MASK_NN & MASK_N & MASK_W,
	MASK_SS & MASK_S & MASK_E,
	MASK_SS & MASK_S & MASK_W,
	MASK_EE & MASK_N & MASK_E,
	MASK_EE & MASK_S & MASK_E,
	MASK_WW & MASK_N & MASK_W,
	MASK_WW & MASK_S & MASK_W,
}

const FORCE_POSITIVE_OFFSET = 32

var PRE_MOVE_MASK_FROM_OFFSET [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}
	result[FORCE_POSITIVE_OFFSET+OFFSET_N] = PRE_MOVE_MASKS[0]
	result[FORCE_POSITIVE_OFFSET+OFFSET_S] = PRE_MOVE_MASKS[1]
	result[FORCE_POSITIVE_OFFSET+OFFSET_E] = PRE_MOVE_MASKS[2]
	result[FORCE_POSITIVE_OFFSET+OFFSET_W] = PRE_MOVE_MASKS[3]

	result[FORCE_POSITIVE_OFFSET+OFFSET_N+OFFSET_E] = PRE_MOVE_MASKS[4]
	result[FORCE_POSITIVE_OFFSET+OFFSET_N+OFFSET_W] = PRE_MOVE_MASKS[5]
	result[FORCE_POSITIVE_OFFSET+OFFSET_S+OFFSET_E] = PRE_MOVE_MASKS[6]
	result[FORCE_POSITIVE_OFFSET+OFFSET_S+OFFSET_W] = PRE_MOVE_MASKS[7]

	result[FORCE_POSITIVE_OFFSET+OFFSET_N+OFFSET_N+OFFSET_E] = PRE_MOVE_MASKS[8]
	result[FORCE_POSITIVE_OFFSET+OFFSET_N+OFFSET_N+OFFSET_W] = PRE_MOVE_MASKS[9]
	result[FORCE_POSITIVE_OFFSET+OFFSET_S+OFFSET_S+OFFSET_E] = PRE_MOVE_MASKS[10]
	result[FORCE_POSITIVE_OFFSET+OFFSET_S+OFFSET_S+OFFSET_W] = PRE_MOVE_MASKS[11]
	result[FORCE_POSITIVE_OFFSET+OFFSET_E+OFFSET_N+OFFSET_E] = PRE_MOVE_MASKS[12]
	result[FORCE_POSITIVE_OFFSET+OFFSET_E+OFFSET_S+OFFSET_E] = PRE_MOVE_MASKS[13]
	result[FORCE_POSITIVE_OFFSET+OFFSET_W+OFFSET_N+OFFSET_W] = PRE_MOVE_MASKS[14]
	result[FORCE_POSITIVE_OFFSET+OFFSET_W+OFFSET_S+OFFSET_W] = PRE_MOVE_MASKS[15]
	return result
}()

func PremoveMaskFromOffset(offset int) Bitboard {
	return PRE_MOVE_MASK_FROM_OFFSET[FORCE_POSITIVE_OFFSET+offset]
}

var KNIGHT_ATTACK_MASKS [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}

	for i := 0; i < 64; i++ {
		pieceBoard := singleBitboard(i)
		for _, dir := range KNIGHT_DIRS {
			potential := pieceBoard & PRE_MOVE_MASKS[dir]
			potential = rotateTowardsIndex64(potential, OFFSETS[dir])

			result[i] |= potential
		}
	}
	return result
}()

var KING_ATTACK_MASKS [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}

	for i := 0; i < 64; i++ {
		pieceBoard := singleBitboard(i)
		for _, dir := range KING_DIRS {
			potential := pieceBoard & PRE_MOVE_MASKS[dir]
			potential = rotateTowardsIndex64(potential, OFFSETS[dir])

			result[i] |= potential
		}
	}
	return result
}()

var CASTLING_REQUIREMENTS = func() [2][2]CastlingRequirements {
	result := [2][2]CastlingRequirements{}
	result[WHITE][KINGSIDE] = CastlingRequirements{
		safe:   MapSlice([]string{"e1", "f1", "g1"}, boardIndexFromString),
		empty:  bitboardWithAllLocationsSet(([]string{"f1", "g1"})),
		move:   moveFromString("e1g1", CASTLING_MOVE),
		pieces: bitboardWithAllLocationsSet([]string{"e1", "h1"}),
	}
	result[WHITE][QUEENSIDE] = CastlingRequirements{
		safe:   MapSlice([]string{"e1", "d1", "c1"}, boardIndexFromString),
		empty:  bitboardWithAllLocationsSet(([]string{"b1", "c1", "d1"})),
		move:   moveFromString("e1c1", CASTLING_MOVE),
		pieces: bitboardWithAllLocationsSet([]string{"e1", "a1"}),
	}
	result[BLACK][KINGSIDE] = CastlingRequirements{
		safe:   MapSlice([]string{"e8", "f8", "g8"}, boardIndexFromString),
		empty:  bitboardWithAllLocationsSet(([]string{"f8", "g8"})),
		move:   moveFromString("e8g8", CASTLING_MOVE),
		pieces: bitboardWithAllLocationsSet([]string{"e8", "h8"}),
	}
	result[BLACK][QUEENSIDE] = CastlingRequirements{
		safe:   MapSlice([]string{"e8", "d8", "c8"}, boardIndexFromString),
		empty:  bitboardWithAllLocationsSet(([]string{"b8", "c8", "d8"})),
		move:   moveFromString("e8c8", CASTLING_MOVE),
		pieces: bitboardWithAllLocationsSet([]string{"e8", "a8"}),
	}
	return result
}()

var A1 int = boardIndexFromString("a1")
var B1 int = boardIndexFromString("b1")
var C1 int = boardIndexFromString("c1")
var D1 int = boardIndexFromString("d1")
var E1 int = boardIndexFromString("e1")
var F1 int = boardIndexFromString("f1")
var G1 int = boardIndexFromString("g1")
var H1 int = boardIndexFromString("h1")
var A8 int = boardIndexFromString("a8")
var B8 int = boardIndexFromString("b8")
var C8 int = boardIndexFromString("c8")
var D8 int = boardIndexFromString("d8")
var E8 int = boardIndexFromString("e8")
var F8 int = boardIndexFromString("f8")
var G8 int = boardIndexFromString("g8")
var H8 int = boardIndexFromString("h8")

func rookMoveForCastle(startIndex int, endIndex int) (int, int, error) {
	switch startIndex {
	case E1:
		switch endIndex {
		case C1:
			return A1, D1, nil
		case G1:
			return H1, F1, nil
		}
	case E8:
		switch endIndex {
		case C8:
			return A8, D8, nil
		case G8:
			return H8, F8, nil
		}
	}
	return 0, 0, fmt.Errorf("unknown castling move %v %v", stringFromBoardIndex(startIndex), stringFromBoardIndex(endIndex))
}

var SINGLE_BITBOARDS [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}
	for i := 0; i < 64; i++ {
		result[i] = shiftTowardsIndex64(1, i)
	}
	return result
}()

func singleBitboard(index int) Bitboard {
	return SINGLE_BITBOARDS[index]
}

var SINGLE_BITBOARDS_ALLOWING_NEGATIVE_INDEX [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}
	for i := 0; i < 64; i++ {
		result[i] = rotateTowardsIndex64(1, i)
	}
	return result
}()

func SingleUint8(indexFromTheRight int) uint8 {
	return 1 << indexFromTheRight
}

func zerosForRange(fs []int, rs []int) Bitboard {
	if len(fs) != len(rs) {
		panic("slices have different length")
	}

	result := ALL_ONES
	for i := 0; i < len(fs); i++ {
		result &= ^singleBitboard(IndexFromFileRank(FileRank{File(fs[i]), Rank(rs[i])}))
	}
	return result
}

var ReverseBitsCache = func() [256]uint8 {
	result := [256]uint8{}
	for i := uint8(0); ; i++ {
		reversed := uint8(0)
		for bit := 0; bit < 8; bit++ {
			if i&SingleUint8(bit) > 0 {
				reversed |= SingleUint8(7 - bit)
			}
		}
		result[i] = reversed

		if i == uint8(255) {
			break
		}
	}
	return result
}()

type CastlingRequirements struct {
	empty  Bitboard
	safe   []int
	move   Move
	pieces Bitboard
}

func OnesCount(b Bitboard) int {
	return bits.OnesCount64(uint64(b))
}

func bitboardWithAllLocationsSet(locations []string) Bitboard {
	return ReduceSlice(
		MapSlice(locations, boardIndexFromString),
		0,
		func(result Bitboard, index int) Bitboard {
			return result | singleBitboard(index)
		},
	)
}

func shiftTowardsIndex0(b Bitboard, n int) Bitboard {
	return b >> n
}

func shiftTowardsIndex64(b Bitboard, n int) Bitboard {
	return b << n
}

func rotateTowardsIndex0(b Bitboard, n int) Bitboard {
	return Bitboard(bits.RotateLeft64(uint64(b), -n))
}

func rotateTowardsIndex64(b Bitboard, n int) Bitboard {
	return Bitboard(bits.RotateLeft64(uint64(b), n))
}

func (b Bitboard) String() string {
	ranks := [8]string{}
	for rank := 0; rank < 8; rank++ {
		bitsBefore := rank * 8
		bitsAfter := 64 - bitsBefore - 8

		r := b

		// clip everything above this rank
		r = shiftTowardsIndex64(r, bitsAfter)
		// clip everything before this rank
		r = shiftTowardsIndex0(r, bitsBefore+bitsAfter)

		// mirror the bits so we're printing in a natural order
		// (10000000 for the top left / lowest index instead of 00000001)
		ranks[7-rank] = fmt.Sprintf("%08b", reverseBits(uint8(r)))
	}

	return strings.Join(ranks[0:], "\n")
}

func bitboardFromStrings(strings [8]string) Bitboard {
	b := Bitboard(0)
	for inverseRank, line := range strings {
		for file, c := range line {
			if c == '1' {
				index := IndexFromFileRank(FileRank{File(file), Rank(7 - inverseRank)})
				b |= singleBitboard(index)
			}
		}
	}
	return b
}

func SetupBitboards(g *GameState) Bitboards {
	result := Bitboards{}
	for i, piece := range g.Board {
		if piece == XX {
			continue
		}
		pieceType := piece.pieceType()
		player := piece.player()
		result.players[player].pieces[pieceType] |= singleBitboard(i)

		if piece.isWhite() {
			result.occupied |= singleBitboard(i)
			result.players[WHITE].occupied |= singleBitboard(i)
		}
		if piece.isBlack() {
			result.occupied |= singleBitboard(i)
			result.players[BLACK].occupied |= singleBitboard(i)
		}
	}
	return result
}

func (b Bitboard) leastSignificantOne() Bitboard {
	return b & -b
}

func (b Bitboard) firstIndexOfOne() int {
	ls1 := b.leastSignificantOne()
	return bits.OnesCount64(uint64(ls1 - 1))
}

type IndicesBuffer []int

var GetIndicesBuffer, ReleaseIndicesBuffer, StatsIndicesBuffer = createPool(
	func() IndicesBuffer {
		return make(IndicesBuffer, 0, 64)
	},
	func(x *IndicesBuffer) {
		*x = (*x)[:0]
	},
)

func (b Bitboard) eachIndexOfOne(buffer *IndicesBuffer) *IndicesBuffer {
	*buffer = (*buffer)[:0]

	temp := b
	for temp != 0 {
		ls1 := temp.leastSignificantOne()
		index := bits.OnesCount64(uint64(ls1 - 1))
		*buffer = append(*buffer, int(index))
		temp = temp ^ ls1
	}

	return buffer
}

func (b Bitboard) eachIndexOfOne2(callback func(int)) {
	temp := b
	for temp != 0 {
		ls1 := temp.leastSignificantOne()
		index := bits.OnesCount64(uint64(ls1 - 1))
		callback(index)
		temp = temp ^ ls1
	}
}

func (b Bitboard) nextIndexOfOne() (int, Bitboard) {
	ls1 := b.leastSignificantOne()
	index := bits.OnesCount64(uint64(ls1 - 1))
	b = b ^ ls1

	return index, b
}

func (b *Bitboards) clearSquare(index int, piece Piece) {
	player := piece.player()
	pieceType := piece.pieceType()
	oneBitboard := singleBitboard(index)
	zeroBitboard := ^oneBitboard

	b.occupied &= zeroBitboard
	b.players[player].occupied &= zeroBitboard
	b.players[player].pieces[pieceType] &= zeroBitboard
}

func (b *Bitboards) setSquare(index int, piece Piece) {
	player := piece.player()
	pieceType := piece.pieceType()
	oneBitboard := singleBitboard(index)

	b.occupied |= oneBitboard
	b.players[player].occupied |= oneBitboard
	b.players[player].pieces[pieceType] |= oneBitboard
}

func (b *Bitboards) performMove(originalState *GameState, move Move) error {
	startIndex := move.startIndex
	endIndex := move.endIndex

	startPiece := originalState.Board[startIndex]

	switch move.moveType {
	case QUIET_MOVE:
		{
			b.clearSquare(startIndex, startPiece)
			b.setSquare(endIndex, startPiece)
		}
	case CAPTURE_MOVE:
		{
			// Remove captured piece
			endPiece := originalState.Board[endIndex]
			b.clearSquare(endIndex, endPiece)

			// Move the capturing piece
			b.clearSquare(startIndex, startPiece)
			b.setSquare(endIndex, startPiece)
		}
	case EN_PASSANT_MOVE:
		{
			capturedPlayer := startPiece.player().Other()
			capturedBackwards := N
			if capturedPlayer == BLACK {
				capturedBackwards = S
			}

			captureIndex := endIndex + OFFSETS[capturedBackwards]
			capturePiece := originalState.Board[captureIndex]

			b.clearSquare(captureIndex, capturePiece)
			b.clearSquare(startIndex, startPiece)
			b.setSquare(endIndex, startPiece)
		}
	case CASTLING_MOVE:
		{
			rookStartIndex, rookEndIndex, err := rookMoveForCastle(startIndex, endIndex)
			if err != nil {
				return err
			}
			rookPiece := originalState.Board[rookStartIndex]

			b.clearSquare(startIndex, startPiece)
			b.setSquare(endIndex, startPiece)

			b.clearSquare(rookStartIndex, rookPiece)
			b.clearSquare(rookEndIndex, rookPiece)
		}
	}

	return nil
}

func (b *Bitboards) undoUpdate(update BoardUpdate) {
	for i := update.num - 1; i >= 0; i-- {
		index := update.indices[i]
		current := update.pieces[i]
		previous := update.old[i]

		if current == XX {
			if previous == XX {
			} else {
				b.setSquare(index, previous)
			}
		} else {
			if previous == XX {
				b.clearSquare(index, current)
			} else {
				b.clearSquare(index, current)
				b.setSquare(index, previous)
			}
		}
	}
}
