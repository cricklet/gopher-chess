package chessgo

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"math/rand"
	"os"
	"runtime"
	"strings"
)

type Bitboard uint64

type Success bool

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

func generateWalkMovesWithMagic(
	pieces Bitboard,
	allOccupied Bitboard,
	selfOccupied Bitboard,
	magicTable MagicMoveTable,
	output []Move,
) []Move {
	for _, startIndex := range pieces.eachIndexOfOne() {
		magicValues := magicTable.magics[startIndex]
		magicIndex := magicIndex(magicValues.Magic, allOccupied, magicValues.BitsInMagicIndex)

		potential := magicTable.moves[startIndex][magicIndex]
		potential = potential & ^selfOccupied

		for _, endIndex := range potential.eachIndexOfOne() {
			output = append(output, Move{startIndex, endIndex})
		}
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
		// moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, N, moves)
		// moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, S, moves)
		// moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, E, moves)
		// moves = generateWalkMoves(b.players[player].rooks, b.occupied, b.players[player.other()].occupied, W, moves)

		// moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, NE, moves)
		// moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, SE, moves)
		// moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, NW, moves)
		// moves = generateWalkMoves(b.players[player].bishops, b.occupied, b.players[player.other()].occupied, SW, moves)

		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, N, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, S, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, E, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, W, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, NE, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, SE, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, NW, moves)
		// moves = generateWalkMoves(b.players[player].queens, b.occupied, b.players[player.other()].occupied, SW, moves)

		moves = generateWalkMovesWithMagic(b.players[player].rooks, b.occupied, b.players[player].occupied, ROOK_MAGIC_TABLE, moves)
		moves = generateWalkMovesWithMagic(b.players[player].bishops, b.occupied, b.players[player].occupied, ROOK_MAGIC_TABLE, moves)
		moves = generateWalkMovesWithMagic(b.players[player].queens, b.occupied, b.players[player].occupied, ROOK_MAGIC_TABLE, moves)
		moves = generateWalkMovesWithMagic(b.players[player].queens, b.occupied, b.players[player].occupied, BISHOP_MAGIC_TABLE, moves)
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

type MoveBoardForBlockerBoard struct {
	moveBoard    Bitboard
	blockerBoard Bitboard
}

func generateMoveBoards(
	pieceIndex int, blockerMask Bitboard, dirs []Dir,
) [] /* OnesCount64(blockerMask) */ MoveBoardForBlockerBoard {
	numBits := bits.OnesCount64(uint64(blockerMask))
	numBlockerBoards := 1 << numBits

	blockerBoards := make([]Bitboard, numBlockerBoards)
	for seed := 0; seed < numBlockerBoards; seed++ {
		blockerBoards[seed] = generateBlockerBoard(blockerMask, seed)
	}

	pieceBoard := singleBitboard(pieceIndex)

	result := make([]MoveBoardForBlockerBoard, numBlockerBoards)
	for seed, blockerBoard := range blockerBoards {
		moves := Bitboard(0)
		for _, dir := range dirs {
			moves = generateWalkBitboard(pieceBoard, blockerBoard, dir, moves)
		}

		result[seed] = MoveBoardForBlockerBoard{moves, blockerBoard}
	}
	return result
}

func generateBlockerMasks(dirs []Dir) [64]Bitboard {
	result := [64]Bitboard{}

	for i := 0; i < 64; i++ {
		result[i] = generateBlockerMask(i, dirs)
	}

	return result
}

func rand64() uint64 {
	return uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
}

func mostlyZeroRand64() uint64 {
	return rand64() & rand64() & rand64()
}

func magicIndex(magic uint64, blockerBoard Bitboard, bitsInIndex int) int {
	return int((uint64(blockerBoard) * magic) >> (64 - bitsInIndex))
}

var tmpCache = [1 << 12]Bitboard{}
var tmpHit = [1 << 12]bool{}

func magicIndexWorks(magic uint64, moves []MoveBoardForBlockerBoard, bitsInIndex int) bool {
	for i := range tmpCache {
		tmpCache[i] = 0
	}
	for i := range tmpHit {
		tmpHit[i] = false
	}
	for _, move := range moves {
		i := magicIndex(magic, move.blockerBoard, bitsInIndex)
		if tmpHit[i] {
			if tmpCache[i] != move.moveBoard {
				return false
			}
		} else {
			tmpCache[i] = move.moveBoard
			tmpHit[i] = true
		}
	}

	return true
}

func bitsRequiredForMagicIndex(magic uint64, moves []MoveBoardForBlockerBoard) (int, Success) {
	success := Success(false)
	bestBitsInIndex := 0

	for bitsInIndex := 12; bitsInIndex > 0; bitsInIndex-- {
		if magicIndexWorks(magic, moves, bitsInIndex) {
			bestBitsInIndex = bitsInIndex
			success = true
		} else {
			break
		}
	}

	return bestBitsInIndex, success
}

func findBetterMagicValue(bestMagic MagicValue, moves []MoveBoardForBlockerBoard) MagicValue {
	for i := 0; i < 1000; i++ {
		magic := mostlyZeroRand64()
		bitsInIndex, currentSuccess := bitsRequiredForMagicIndex(magic, moves)
		if !currentSuccess {
			continue
		}

		if bitsInIndex < bestMagic.BitsInMagicIndex {
			bestMagic.Magic = magic
			bestMagic.BitsInMagicIndex = bitsInIndex
		}
	}

	return bestMagic
}

func generateMagicMoveTable(dirs []Dir, blockerMasks [64]Bitboard, bestMagics [64]MagicValue, label string) MagicMoveTable {
	result := MagicMoveTable{}

	// bar := progressbar.Default(64, label)

	for i := 0; i < 64; i++ {
		blockerMask := blockerMasks[i]
		moves := generateMoveBoards(i, blockerMask, dirs)

		betterMagic := findBetterMagicValue(bestMagics[i], moves)
		result.magics[i] = betterMagic

		result.moves[i] = make([]Bitboard, 1<<betterMagic.BitsInMagicIndex)
		for _, m := range moves {
			magicIndex := magicIndex(betterMagic.Magic, m.blockerBoard, betterMagic.BitsInMagicIndex)
			result.moves[i][magicIndex] = m.moveBoard
		}

		// bar.Add(1)
	}

	return result
}

type MagicValue struct {
	Magic            uint64
	BitsInMagicIndex int
}

type MagicMoveTable struct {
	// Each of the 64 indices in the board has a magic-lookup precomputed.
	// This is used to lookup a move based on the current occupancy of the
	// board, eg:
	// ROOK_MOVES[
	//   (
	//     ((occupancy & blockerMask) * magic)
	//     >> (64 - numBits)
	//   ) << previousBits
	//  ]
	magics [64]MagicValue
	moves  [64][]Bitboard
}

func (m MagicValue) string() string {
	return fmt.Sprintf("{%v, %v}", m.Magic, m.BitsInMagicIndex)
}

// We mask the occupancy with the blockerMask to get the blockerBoard.
// Then we generate a magic index that gives a unique index that we use
// to index the moves database.
//  where

var ROOK_BLOCKER_MASKS [64]Bitboard
var BISHOP_BLOCKER_MASKS [64]Bitboard

var ROOK_BEST_MAGICS = [64]MagicValue{
	{9331458498780872708, 12}, {4665729506550484992, 11}, {144126186415460480, 11}, {144124147393380420, 12}, {11565257037802111104, 11}, {144132788852099073, 11}, {360290736719004416, 11}, {72057871080096230, 12}, {4719913149124313312, 11}, {293156463157707144, 10}, {6917669902577307648, 10}, {140771923603456, 10}, {1162069475734979584, 10}, {9223935029758136344, 10}, {73465046232203520, 10}, {72198473260253312, 11}, {72207677412868132, 11}, {9160032444752128, 10}, {144256475856900105, 10}, {5193215519872860424, 10}, {159430394052612, 10}, {10523224031208014848, 10}, {864765895917076752, 10}, {600333755678852, 11}, {15832969587466384, 11}, {4503884168962050, 10}, {1161937501029400896, 10}, {5814147670840180754, 10}, {576645472412763136, 10}, {42786397639148544, 10}, {2315415374626029896, 10}, {10520549469173335296, 11}, {2317524495633481760, 11}, {360323223285399872, 10}, {9007474451424004, 10}, {5700005885121026, 10}, {10160261531204324352, 10}, {15016162516944359556, 10}, {17636813465603, 10}, {150026164885260370, 11}, {18015225290719265, 11}, {292736450217132032, 10}, {1333100674342224000, 10}, {1153484494829912080, 10}, {145243183935160356, 10}, {4648277800028340236, 10}, {18295882077241348, 10}, {148900299225235458, 11}, {2308517022067064960, 11}, {2666166164849787008, 10}, {10484947351389610496, 10}, {865113409641250944, 10}, {79164905423104, 10}, {598134445769894144, 10}, {8865384334336, 10}, {140741783341184, 11}, {11822236544142419985, 12}, {853358739210241, 11}, {2306689770606579907, 11}, {27305340485764105, 11}, {562958563547782, 12}, {576742261673689253, 11}, {563053041289474, 11}, {72061994248775234, 12},
}
var BISHOP_BEST_MAGICS = [64]MagicValue{
	{1171237203947823488, 6}, {2308412585671671873, 5}, {7569428664312397952, 5}, {1155182929459020040, 5}, {883849190865657860, 5}, {23791370577911968, 5}, {4936090344850063874, 5}, {146649013763063808, 6}, {936753137990238992, 5}, {2278222469285378, 5}, {1196989970411233792, 5}, {324720985242599456, 5}, {5764660884244799536, 5}, {2394762130760320, 5}, {621497027822370952, 5}, {13981425596434489600, 5}, {27065647490015380, 5}, {5190404141385548160, 5}, {9605402366906400, 7}, {579851818030354560, 7}, {1190076210669946880, 7}, {73606260729094176, 7}, {63472633420988992, 5}, {144191067330330882, 5}, {9296115726568935426, 5}, {1153494350270302208, 5}, {2594293288496408642, 7}, {288533842569070752, 9}, {282097763762178, 9}, {12682493891987964224, 7}, {3413158987827720, 5}, {144257574865338502, 5}, {9227880378178601482, 5}, {578723650582085891, 5}, {563226173772032, 7}, {4611688219602845825, 9}, {577596552386969664, 9}, {784805039544846344, 7}, {4512990774821376, 5}, {13856521630425031561, 5}, {36187162681018624, 5}, {81208298082213924, 5}, {563370994700560, 7}, {598417927602305, 7}, {1733894656929825796, 7}, {9223935605837201536, 7}, {83396204645406928, 5}, {2594638672888348928, 5}, {4575136872169504, 5}, {1443143505936385, 5}, {288232576282804224, 5}, {2199569041456, 5}, {1181772762902036736, 5}, {582517344230309892, 5}, {4616194085424742402, 5}, {78814110179000972, 5}, {380572319064539168, 6}, {4625202317049012226, 5}, {109354164517619712, 5}, {18256567021373440, 5}, {1154047404782782976, 5}, {586593868780142848, 5}, {9223566169653444672, 5}, {4508038484721921, 6},
}

var ROOK_MAGIC_TABLE MagicMoveTable
var BISHOP_MAGIC_TABLE MagicMoveTable

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v KB", bToKb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v KB", bToKb(m.TotalAlloc))
	fmt.Printf("\tSys = %v KB", bToKb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToKb(b uint64) uint64 {
	return b / 1024
}

func initMagicTables() {
	rookInput, err := os.ReadFile("magics-for-rook.json")
	if err == nil {
		json.Unmarshal(rookInput, &ROOK_BEST_MAGICS)
	}

	bishopInput, err := os.ReadFile("magics-for-bishop.json")
	if err == nil {
		json.Unmarshal(bishopInput, &BISHOP_BEST_MAGICS)
	}

	ROOK_BLOCKER_MASKS = generateBlockerMasks(ROOK_DIRS)
	BISHOP_BLOCKER_MASKS = generateBlockerMasks(BISHOP_DIRS)

	ROOK_MAGIC_TABLE = generateMagicMoveTable(ROOK_DIRS, ROOK_BLOCKER_MASKS, ROOK_BEST_MAGICS, "rook magics ")
	BISHOP_MAGIC_TABLE = generateMagicMoveTable(BISHOP_DIRS, BISHOP_BLOCKER_MASKS, BISHOP_BEST_MAGICS, "bishop magic")

	lowestRookBits := 12
	sumRookBits := 0
	for _, m := range ROOK_MAGIC_TABLE.magics {
		if m.BitsInMagicIndex < lowestRookBits {
			lowestRookBits = m.BitsInMagicIndex
		}
		sumRookBits += m.BitsInMagicIndex
	}

	lowestBishopBits := 12
	sumBishopBits := 0
	for _, m := range BISHOP_MAGIC_TABLE.magics {
		if m.BitsInMagicIndex < lowestBishopBits {
			lowestBishopBits = m.BitsInMagicIndex
		}
		sumBishopBits += m.BitsInMagicIndex
	}

	fmt.Println("rook bits for magic index: best", lowestRookBits, "average", sumRookBits/64.0)
	fmt.Println("bishop bits for magic index: best", lowestBishopBits, "average", sumBishopBits/64.0)

	if rookOutput, err := json.Marshal(ROOK_BEST_MAGICS); err == nil {
		os.WriteFile("magics-for-rook.json", rookOutput, 0777)
	} else {
		panic("couldn't marshal rook magics")
	}
	if bishopOutput, err := json.Marshal(BISHOP_BEST_MAGICS); err == nil {
		os.WriteFile("magics-for-bishop.json", bishopOutput, 0777)
	} else {
		panic("couldn't marshal bishop magics")
	}
}
