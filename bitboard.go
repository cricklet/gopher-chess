package chessgo

import (
	"fmt"
	"math/bits"
	"strings"
)

type Bitboard uint64

func SingleUint8(indexFromTheRight int) uint8 {
	return 1 << indexFromTheRight
}

var ALL_ZEROS Bitboard = Bitboard(0)
var ALL_ONES Bitboard = ^ALL_ZEROS

func zerosForRange(fs []int, rs []int) Bitboard {
	if len(fs) != len(rs) {
		panic("slices have different length")
	}

	result := ALL_ONES
	for i := 0; i < len(fs); i++ {
		result &= ^singleBitboard(boardIndexFromFileRank(FileRank{File(fs[i]), Rank(rs[i])}))
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

var ZEROS = []int{0, 0, 0, 0, 0, 0, 0, 0}
var ONES = []int{1, 1, 1, 1, 1, 1, 1, 1}
var SIXES = []int{6, 6, 6, 6, 6, 6, 6, 6}
var SEVENS = []int{7, 7, 7, 7, 7, 7, 7, 7}
var ZERO_TO_SEVEN = []int{0, 1, 2, 3, 4, 5, 6, 7}

var (
	MASK_WHITE_STARTING_PAWNS = ^zerosForRange(ZERO_TO_SEVEN, ONES)
	MASK_BLACK_STARTING_PAWNS = ^zerosForRange(ZERO_TO_SEVEN, SIXES)
)

func maskStartingPawnsForPlayer(player Player) Bitboard {
	switch player {
	case WHITE:
		return MASK_WHITE_STARTING_PAWNS
	case BLACK:
		return MASK_BLACK_STARTING_PAWNS
	}
	panic(fmt.Sprintf("invalid player %v", player))
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

func reverseBits(n uint8) uint8 {
	return ReverseBitsCache[n]
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

func singleBitboard(index int) Bitboard {
	return shiftTowardsIndex64(1, index)
}

func singleBitboardAllowingNegativeIndex(index int) Bitboard {
	return rotateTowardsIndex64(1, index)
}

func (b Bitboard) string() string {
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
				index := boardIndexFromFileRank(FileRank{File(file), Rank(7 - inverseRank)})
				b |= singleBitboard(index)
			}
		}
	}
	return b
}

type PlayerBitboards struct {
	occupied Bitboard
	rooks    Bitboard
	knights  Bitboard
	bishops  Bitboard
	queens   Bitboard
	king     Bitboard
	pawns    Bitboard
}

type Bitboards struct {
	occupied Bitboard
	players  [2]PlayerBitboards
}

func setupBitboards(g GameState) Bitboards {
	result := Bitboards{}
	for i, piece := range g.board {
		switch piece {
		case WR:
			result.players[WHITE].rooks |= singleBitboard(i)
		case WN:
			result.players[WHITE].knights |= singleBitboard(i)
		case WB:
			result.players[WHITE].bishops |= singleBitboard(i)
		case WK:
			result.players[WHITE].king |= singleBitboard(i)
		case WQ:
			result.players[WHITE].queens |= singleBitboard(i)
		case WP:
			result.players[WHITE].pawns |= singleBitboard(i)
		case BR:
			result.players[BLACK].rooks |= singleBitboard(i)
		case BN:
			result.players[BLACK].knights |= singleBitboard(i)
		case BB:
			result.players[BLACK].bishops |= singleBitboard(i)
		case BK:
			result.players[BLACK].king |= singleBitboard(i)
		case BQ:
			result.players[BLACK].queens |= singleBitboard(i)
		case BP:
			result.players[BLACK].pawns |= singleBitboard(i)
		}
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

type Move struct {
	startIndex int
	endIndex   int
}

func (b Bitboard) leastSignificantOne() Bitboard {
	return b & -b
}

func (b Bitboard) eachIndexOfOne() []int {
	result := make([]int, 0, 64)

	temp := b
	for temp != 0 {
		ls1 := temp.leastSignificantOne()
		index := bits.OnesCount64(uint64(ls1 - 1))
		result = append(result, int(index))
		temp = temp ^ ls1
	}

	return result
}

func generateWalkMoves(
	pieces Bitboard,
	allOccupied Bitboard,
	enemyOccupied Bitboard,
	dir Dir,
	output []Move,
) []Move {
	mask := PRE_MOVE_MASKS[dir]
	offset := OFFSETS[dir]

	totalOffset := 0
	potential := pieces

	for potential != 0 {
		potential = rotateTowardsIndex64(potential&mask, offset)
		totalOffset += offset

		quiet := potential & ^allOccupied
		capture := potential & enemyOccupied

		for _, index := range (quiet | capture).eachIndexOfOne() {
			output = append(output, Move{index - totalOffset, index})
		}

		potential = quiet
	}

	return output
}

func generateWalkBitboard(
	pieceBoard Bitboard,
	blockerBoard Bitboard,
	dir Dir,
	output Bitboard,
) Bitboard {
	mask := PRE_MOVE_MASKS[dir]
	offset := OFFSETS[dir]

	totalOffset := 0
	potential := pieceBoard

	for potential != 0 {
		potential = rotateTowardsIndex64(potential&mask, offset)
		totalOffset += offset

		quiet := potential & ^blockerBoard
		capture := potential & blockerBoard

		output |= quiet | capture

		potential = quiet
	}

	return output
}

func generateJumpMoves(
	pieces Bitboard,
	allOccupied Bitboard,
	enemyOccupied Bitboard,
	dirs []Dir,
	output []Move,
) []Move {
	for _, dir := range dirs {
		mask := PRE_MOVE_MASKS[dir]
		offset := OFFSETS[dir]

		potential := pieces
		potential = rotateTowardsIndex64(potential&mask, offset)

		quiet := potential & ^allOccupied
		capture := potential & enemyOccupied

		for _, index := range (quiet | capture).eachIndexOfOne() {
			output = append(output, Move{index - offset, index})
		}

		potential = quiet
	}
	return output
}

func (b Bitboards) generatePseudoMoves(player Player) []Move {
	moves := make([]Move, 0, 256)

	{
		// generate pawn pushes
		dir := S
		if player == WHITE {
			dir = N
		}

		// generate one step
		{
			potential := rotateTowardsIndex64(b.players[player].pawns, OFFSETS[dir])
			potential = potential & ^b.occupied
			for _, index := range potential.eachIndexOfOne() {
				moves = append(moves, Move{index - OFFSETS[dir], index})
			}
		}

		// generate skip step
		{
			potential := b.players[player].pawns
			potential = potential & maskStartingPawnsForPlayer(player)
			potential = rotateTowardsIndex64(potential, OFFSETS[dir])
			potential = potential & ^b.occupied
			potential = rotateTowardsIndex64(potential, OFFSETS[dir])
			potential = potential & ^b.occupied

			for _, index := range potential.eachIndexOfOne() {
				moves = append(moves, Move{index - 2*OFFSETS[dir], index})
			}
		}

		// generate captures
		{
			for _, dir := range []Dir{NE, NW} {
				potential := b.players[player].pawns
				potential = rotateTowardsIndex64(potential, OFFSETS[dir])
				potential = potential & b.players[player.other()].occupied

				for _, index := range potential.eachIndexOfOne() {
					moves = append(moves, Move{index - OFFSETS[dir], index})
				}
			}
		}
	}

	{
		// generate rook / bishop / queen moves
		moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, N, moves)
		moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, S, moves)
		moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, E, moves)
		moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, W, moves)

		moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, NE, moves)
		moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, SE, moves)
		moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, NW, moves)
		moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, SW, moves)

		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, N, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, S, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, E, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, W, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, NE, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, SE, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, NW, moves)
		moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, SW, moves)
	}

	{
		// generate knight moves
		moves = generateJumpMoves(b.players[player].knights, b.occupied, b.players[player.other()].occupied, KNIGHT_DIRS, moves)

		// generate king moves
		moves = generateJumpMoves(b.players[player].king, b.occupied, b.players[player.other()].occupied, KING_DIRS, moves)
	}

	return moves
}

func moveFromString(s string) Move {
	first := s[0:2]
	second := s[2:4]
	return Move{boardIndexFromString(first), boardIndexFromString(second)}
}

func (m Move) string() string {
	return stringFromBoardIndex(m.startIndex) + stringFromBoardIndex(m.endIndex)
}

func stringFromBoardIndex(index int) string {
	return fileRankFromBoardIndex(index).string()
}

func generateBlockerMask(startIndex int, dirs []Dir) Bitboard {
	result := Bitboard(0)
	for _, dir := range dirs {
		walk := generateWalkBitboard(singleBitboard(startIndex), Bitboard(0), dir, result)
		result |= walk & PRE_MOVE_MASKS[dir]
	}

	result &= ^singleBitboard(startIndex)

	return result
}

func generateBlockerBoard(blockerMask Bitboard, seed int) Bitboard {
	result := Bitboard(0)

	numBits := bits.OnesCount64(uint64(blockerMask))
	for i := 0; i < numBits; i++ {
		// If the bit at i is 1 in the seed...
		if seed&(1<<i) != 0 {
			// Find the ith one bit in blockerMask and set the corresponding bit to one in result.
			for oneIndex, indexInBitboard := range blockerMask.eachIndexOfOne() {
				if oneIndex == i {
					result |= singleBitboard(indexInBitboard)
				}
			}
		}
	}

	return result
}

func generateMoveBoards(pieceIndex int, blockerMask Bitboard, dirs []Dir) [] /* OnesCount64(blockerMask) */ Bitboard {
	numBits := bits.OnesCount64(uint64(blockerMask))
	numBlockerBoards := 1 << numBits

	blockerBoards := make([]Bitboard, numBlockerBoards)
	for seed := 0; seed < numBlockerBoards; seed++ {
		blockerBoards[seed] = generateBlockerBoard(blockerMask, seed)
	}

	pieceBoard := singleBitboard(pieceIndex)

	result := make([]Bitboard, numBlockerBoards)
	for seed, blockerBoard := range blockerBoards {
		moves := Bitboard(0)
		for _, dir := range dirs {
			moves = generateWalkBitboard(pieceBoard, blockerBoard, dir, moves)
		}

		result[seed] = moves
	}
	return result
}

func generateMagicLookupValues(dirs []Dir) [64]MagicLookupValues {
	result := [64]MagicLookupValues{}

	previousBits := 0

	for i := 0; i < 64; i++ {
		blockerMask := generateBlockerMask(i, dirs)
		numBits := bits.OnesCount64(uint64(blockerMask))

		result[i].previousBits = uint64(previousBits)
		result[i].blockerMask = blockerMask
		result[i].numBits = numBits

		previousBits += numBits
	}

	return result
}

type MagicLookupValues struct {
	// Each of the 64 indices in the board has a magic-lookup precomputed.
	// This is used to lookup a move based on the current occupancy of the
	// board, eg:
	// ROOK_MOVES[
	//   offsetIndex +
	//   (
	//     ((occupancy & blockerMask) * magic)
	//     >> (64 - numBits)
	//   )
	//  ]
	previousBits uint64
	blockerMask  Bitboard
	magic        uint64
	numBits      int
}

// We mask the occupancy with the blockerMask to get the blockerBoard.
// Then we generate a magic index that gives a unique index that we use
// to index the moves database.
//  where

var ROOK_MAGICS [64]MagicLookupValues = generateMagicLookupValues(ROOK_DIRS)

var BISHOP_MAGICS [64]MagicLookupValues = generateMagicLookupValues(BISHOP_DIRS)
