package bitboards

import (
	"fmt"
	"math/bits"
	"strings"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type Bitboard uint64

type PlayerBitboards struct {
	Occupied Bitboard
	Pieces   [6]Bitboard // indexed via PieceType
}

type Bitboards struct {
	Occupied Bitboard
	Players  [2]PlayerBitboards
}

type IndicesBuffer []int

var GetIndicesBuffer, ReleaseIndicesBuffer, StatsIndicesBuffer = CreatePool(
	func() IndicesBuffer {
		return make(IndicesBuffer, 0, 64)
	},
	func(x *IndicesBuffer) {
		*x = (*x)[:0]
	},
)

func (b Bitboard) EachIndexOfOne(buffer *IndicesBuffer) *IndicesBuffer {
	*buffer = (*buffer)[:0]

	temp := b
	for temp != 0 {
		ls1 := temp.LeastSignificantOne()
		index := bits.OnesCount64(uint64(ls1 - 1))
		*buffer = append(*buffer, int(index))
		temp = temp ^ ls1
	}

	return buffer
}

func (b Bitboard) EachIndexOfOneCallback(callback func(int)) {
	temp := b
	for temp != 0 {
		ls1 := temp.LeastSignificantOne()
		index := bits.OnesCount64(uint64(ls1 - 1))
		callback(index)
		temp = temp ^ ls1
	}
}

func (b Bitboard) NextIndexOfOne() (int, Bitboard) {
	ls1 := b.LeastSignificantOne()
	index := bits.OnesCount64(uint64(ls1 - 1))
	b = b ^ ls1

	return index, b
}

type ReusableIndicesBuffers struct {
	startBuffer *IndicesBuffer
	endBuffer   *IndicesBuffer
}

func SetupBuffers() ReusableIndicesBuffers {
	return ReusableIndicesBuffers{GetIndicesBuffer(), GetIndicesBuffer()}
}

func (r ReusableIndicesBuffers) Release() {
	ReleaseIndicesBuffer(r.startBuffer)
	ReleaseIndicesBuffer(r.endBuffer)
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

	NumDirs
)

var KnightDirs = []Dir{
	NNE,
	NNW,
	SSE,
	SSW,
	ENE,
	ESE,
	WNW,
	WSW,
}

var RookDirs = []Dir{
	N,
	S,
	E,
	W,
}

var BishopDirs = []Dir{
	NE,
	NW,
	SE,
	SW,
}

var KingDirs = []Dir{
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
	OffsetN int = 8
	OffsetS int = -8
	OffsetE int = 1
	OffsetW int = -1
)

var Offsets = [NumDirs]int{
	OffsetN,
	OffsetS,
	OffsetE,
	OffsetW,

	OffsetN + OffsetE,
	OffsetN + OffsetW,
	OffsetS + OffsetE,
	OffsetS + OffsetW,

	OffsetN + OffsetN + OffsetE,
	OffsetN + OffsetN + OffsetW,
	OffsetS + OffsetS + OffsetE,
	OffsetS + OffsetS + OffsetW,
	OffsetE + OffsetN + OffsetE,
	OffsetE + OffsetS + OffsetE,
	OffsetW + OffsetN + OffsetW,
	OffsetW + OffsetS + OffsetW,
}

var PawnPushOffsets = [2]int{
	OffsetN,
	OffsetS,
}

var PawnPromotionBitboard = BitboardFromStrings([8]string{
	"11111111",
	"00000000",
	"00000000",
	"00000000",
	"00000000",
	"00000000",
	"00000000",
	"11111111",
})

var PawnCaptureOffsets = [2][2]int{
	{ // WHITE
		OffsetN + OffsetE, OffsetN + OffsetW,
	},
	{
		OffsetS + OffsetE, OffsetS + OffsetW,
	},
}

var AllZeros Bitboard = Bitboard(0)
var AllOnes Bitboard = ^AllZeros

var Zeros = []int{0, 0, 0, 0, 0, 0, 0, 0}
var Ones = []int{1, 1, 1, 1, 1, 1, 1, 1}
var Sixes = []int{6, 6, 6, 6, 6, 6, 6, 6}
var Sevens = []int{7, 7, 7, 7, 7, 7, 7, 7}
var ZeroToSeven = []int{0, 1, 2, 3, 4, 5, 6, 7}

var (
	MaskWhiteStartingPawns = ^ZerosForRange(ZeroToSeven, Ones)
	MaskBlackStartingPawns = ^ZerosForRange(ZeroToSeven, Sixes)
)

func IsPromotionIndex(index int, player Player) bool {
	if player == White {
		return index >= 56
	} else {
		return index < 8
	}
}

var StartingPawnsForPlayer = [2]Bitboard{
	MaskWhiteStartingPawns,
	MaskBlackStartingPawns,
}

func MaskStartingPawnsForPlayer(player Player) Bitboard {
	return StartingPawnsForPlayer[player]
}

var (
	MaskN Bitboard = ZerosForRange(ZeroToSeven, Sevens)
	MaskS Bitboard = ZerosForRange(ZeroToSeven, Zeros)
	MaskE Bitboard = ZerosForRange(Sevens, ZeroToSeven)
	MaskW Bitboard = ZerosForRange(Zeros, ZeroToSeven)

	MaskNN Bitboard = ZerosForRange(ZeroToSeven, Sixes)
	MaskSS Bitboard = ZerosForRange(ZeroToSeven, Ones)
	MaskEE Bitboard = ZerosForRange(Sixes, ZeroToSeven)
	MaskWW Bitboard = ZerosForRange(Ones, ZeroToSeven)

	MaskAllEdges Bitboard = MaskN & MaskS & MaskE & MaskW
)

var PreMoveMasks = [NumDirs]Bitboard{
	MaskN,
	MaskS,
	MaskE,
	MaskW,

	MaskN & MaskE,
	MaskN & MaskW,
	MaskS & MaskE,
	MaskS & MaskW,

	MaskNN & MaskN & MaskE,
	MaskNN & MaskN & MaskW,
	MaskSS & MaskS & MaskE,
	MaskSS & MaskS & MaskW,
	MaskEE & MaskN & MaskE,
	MaskEE & MaskS & MaskE,
	MaskWW & MaskN & MaskW,
	MaskWW & MaskS & MaskW,
}

const _forcePositiveOffset = 32

var PreMoveMaskFromOffset [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}
	result[_forcePositiveOffset+OffsetN] = PreMoveMasks[0]
	result[_forcePositiveOffset+OffsetS] = PreMoveMasks[1]
	result[_forcePositiveOffset+OffsetE] = PreMoveMasks[2]
	result[_forcePositiveOffset+OffsetW] = PreMoveMasks[3]

	result[_forcePositiveOffset+OffsetN+OffsetE] = PreMoveMasks[4]
	result[_forcePositiveOffset+OffsetN+OffsetW] = PreMoveMasks[5]
	result[_forcePositiveOffset+OffsetS+OffsetE] = PreMoveMasks[6]
	result[_forcePositiveOffset+OffsetS+OffsetW] = PreMoveMasks[7]

	result[_forcePositiveOffset+OffsetN+OffsetN+OffsetE] = PreMoveMasks[8]
	result[_forcePositiveOffset+OffsetN+OffsetN+OffsetW] = PreMoveMasks[9]
	result[_forcePositiveOffset+OffsetS+OffsetS+OffsetE] = PreMoveMasks[10]
	result[_forcePositiveOffset+OffsetS+OffsetS+OffsetW] = PreMoveMasks[11]
	result[_forcePositiveOffset+OffsetE+OffsetN+OffsetE] = PreMoveMasks[12]
	result[_forcePositiveOffset+OffsetE+OffsetS+OffsetE] = PreMoveMasks[13]
	result[_forcePositiveOffset+OffsetW+OffsetN+OffsetW] = PreMoveMasks[14]
	result[_forcePositiveOffset+OffsetW+OffsetS+OffsetW] = PreMoveMasks[15]
	return result
}()

func PremoveMaskFromOffset(offset int) Bitboard {
	return PreMoveMaskFromOffset[_forcePositiveOffset+offset]
}

var KnightAttackMasks [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}

	for i := 0; i < 64; i++ {
		pieceBoard := SingleBitboard(i)
		for _, dir := range KnightDirs {
			potential := pieceBoard & PreMoveMasks[dir]
			potential = RotateTowardsIndex64(potential, Offsets[dir])

			result[i] |= potential
		}
	}
	return result
}()

var KingAttackMasks [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}

	for i := 0; i < 64; i++ {
		pieceBoard := SingleBitboard(i)
		for _, dir := range KingDirs {
			potential := pieceBoard & PreMoveMasks[dir]
			potential = RotateTowardsIndex64(potential, Offsets[dir])

			result[i] |= potential
		}
	}
	return result
}()

var AllCastlingRequirements = func() [2][2]CastlingRequirements {
	result := [2][2]CastlingRequirements{}
	result[White][Kingside] = CastlingRequirements{
		Safe:   MapSlice([]string{"e1", "f1", "g1"}, BoardIndexFromString),
		Empty:  BitboardWithAllLocationsSet(([]string{"f1", "g1"})),
		Move:   MoveFromString("e1g1", CastlingMove),
		Pieces: BitboardWithAllLocationsSet([]string{"e1", "h1"}),
	}
	result[White][Queenside] = CastlingRequirements{
		Safe:   MapSlice([]string{"e1", "d1", "c1"}, BoardIndexFromString),
		Empty:  BitboardWithAllLocationsSet(([]string{"b1", "c1", "d1"})),
		Move:   MoveFromString("e1c1", CastlingMove),
		Pieces: BitboardWithAllLocationsSet([]string{"e1", "a1"}),
	}
	result[Black][Kingside] = CastlingRequirements{
		Safe:   MapSlice([]string{"e8", "f8", "g8"}, BoardIndexFromString),
		Empty:  BitboardWithAllLocationsSet(([]string{"f8", "g8"})),
		Move:   MoveFromString("e8g8", CastlingMove),
		Pieces: BitboardWithAllLocationsSet([]string{"e8", "h8"}),
	}
	result[Black][Queenside] = CastlingRequirements{
		Safe:   MapSlice([]string{"e8", "d8", "c8"}, BoardIndexFromString),
		Empty:  BitboardWithAllLocationsSet(([]string{"b8", "c8", "d8"})),
		Move:   MoveFromString("e8c8", CastlingMove),
		Pieces: BitboardWithAllLocationsSet([]string{"e8", "a8"}),
	}
	return result
}()

var A1 int = BoardIndexFromString("a1")
var B1 int = BoardIndexFromString("b1")
var C1 int = BoardIndexFromString("c1")
var D1 int = BoardIndexFromString("d1")
var E1 int = BoardIndexFromString("e1")
var F1 int = BoardIndexFromString("f1")
var G1 int = BoardIndexFromString("g1")
var H1 int = BoardIndexFromString("h1")
var A8 int = BoardIndexFromString("a8")
var B8 int = BoardIndexFromString("b8")
var C8 int = BoardIndexFromString("c8")
var D8 int = BoardIndexFromString("d8")
var E8 int = BoardIndexFromString("e8")
var F8 int = BoardIndexFromString("f8")
var G8 int = BoardIndexFromString("g8")
var H8 int = BoardIndexFromString("h8")

func RookMoveForCastle(startIndex int, endIndex int) (int, int, error) {
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
	return 0, 0, fmt.Errorf("unknown castling move %v %v", StringFromBoardIndex(startIndex), StringFromBoardIndex(endIndex))
}

var SingleBitboards [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}
	for i := 0; i < 64; i++ {
		result[i] = ShiftTowardsIndex64(1, i)
	}
	return result
}()

func SingleBitboard(index int) Bitboard {
	return SingleBitboards[index]
}

var SingleBitboardsAllowingNegativeIndex [64]Bitboard = func() [64]Bitboard {
	result := [64]Bitboard{}
	for i := 0; i < 64; i++ {
		result[i] = RotateTowardsIndex64(1, i)
	}
	return result
}()

func ZerosForRange(fs []int, rs []int) Bitboard {
	if len(fs) != len(rs) {
		panic("slices have different length")
	}

	result := AllOnes
	for i := 0; i < len(fs); i++ {
		result &= ^SingleBitboard(IndexFromFileRank(FileRank{File: File(fs[i]), Rank: Rank(rs[i])}))
	}
	return result
}

type CastlingRequirements struct {
	Empty  Bitboard
	Safe   []int
	Move   Move
	Pieces Bitboard
}

func OnesCount(b Bitboard) int {
	return bits.OnesCount64(uint64(b))
}

func BitboardWithAllLocationsSet(locations []string) Bitboard {
	return ReduceSlice(
		MapSlice(locations, BoardIndexFromString),
		0,
		func(result Bitboard, index int) Bitboard {
			return result | SingleBitboard(index)
		},
	)
}

func ShiftTowardIndex0(b Bitboard, n int) Bitboard {
	return b >> n
}

func ShiftTowardsIndex64(b Bitboard, n int) Bitboard {
	return b << n
}

func RotateTowardsIndex0(b Bitboard, n int) Bitboard {
	return Bitboard(bits.RotateLeft64(uint64(b), -n))
}

func RotateTowardsIndex64(b Bitboard, n int) Bitboard {
	return Bitboard(bits.RotateLeft64(uint64(b), n))
}

func (b Bitboard) String() string {
	ranks := [8]string{}
	for rank := 0; rank < 8; rank++ {
		bitsBefore := rank * 8
		bitsAfter := 64 - bitsBefore - 8

		r := b

		// clip everything above this rank
		r = ShiftTowardsIndex64(r, bitsAfter)
		// clip everything before this rank
		r = ShiftTowardIndex0(r, bitsBefore+bitsAfter)

		// mirror the bits so we're printing in a natural order
		// (10000000 for the top left / lowest index instead of 00000001)
		ranks[7-rank] = fmt.Sprintf("%08b", ReverseBits(uint8(r)))
	}

	return strings.Join(ranks[0:], "\n")
}

func BitboardFromStrings(strings [8]string) Bitboard {
	b := Bitboard(0)
	for inverseRank, line := range strings {
		for file, c := range line {
			if c == '1' {
				index := IndexFromFileRank(FileRank{File: File(file), Rank: Rank(7 - inverseRank)})
				b |= SingleBitboard(index)
			}
		}
	}
	return b
}

func (b Bitboard) LeastSignificantOne() Bitboard {
	return b & -b
}

func (b Bitboard) FirstIndexOfOne() int {
	ls1 := b.LeastSignificantOne()
	return bits.OnesCount64(uint64(ls1 - 1))
}

func (b *Bitboards) ClearSquare(index int, piece Piece) error {
	player := piece.Player()
	pieceType := piece.PieceType()
	if !pieceType.IsValid() {
		return fmt.Errorf("pieceType %v is not valid", piece.String())
	}
	oneBitboard := SingleBitboard(index)
	zeroBitboard := ^oneBitboard

	b.Occupied &= zeroBitboard
	b.Players[player].Occupied &= zeroBitboard
	b.Players[player].Pieces[pieceType] &= zeroBitboard

	return nil
}

func (b *Bitboards) SetSquare(index int, piece Piece) {
	player := piece.Player()
	pieceType := piece.PieceType()
	oneBitboard := SingleBitboard(index)

	b.Occupied |= oneBitboard
	b.Players[player].Occupied |= oneBitboard
	b.Players[player].Pieces[pieceType] |= oneBitboard
}
