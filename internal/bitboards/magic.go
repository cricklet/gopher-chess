package bitboards

import (
	"encoding/json"
	"fmt"
	"log"
	"math/bits"
	"math/rand"
	"os"
	"runtime"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type MagicValue struct {
	Magic            uint64
	BitsInMagicIndex int
}

func (m MagicValue) String() string {
	return fmt.Sprintf("{%v, %v}", m.Magic, m.BitsInMagicIndex)
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
	Magics       [64]MagicValue
	BlockerMasks [64]Bitboard
	Moves        [64][]Bitboard
}

var RookBestMagics = [64]MagicValue{
	{9331458498780872708, 12}, {4665729506550484992, 11}, {144126186415460480, 11}, {144124147393380420, 12}, {11565257037802111104, 11}, {144132788852099073, 11}, {360290736719004416, 11}, {72057871080096230, 12}, {4719913149124313312, 11}, {293156463157707144, 10}, {6917669902577307648, 10}, {140771923603456, 10}, {1162069475734979584, 10}, {9223935029758136344, 10}, {73465046232203520, 10}, {72198473260253312, 11}, {72207677412868132, 11}, {9160032444752128, 10}, {144256475856900105, 10}, {5193215519872860424, 10}, {159430394052612, 10}, {10523224031208014848, 10}, {864765895917076752, 10}, {600333755678852, 11}, {15832969587466384, 11}, {4503884168962050, 10}, {1161937501029400896, 10}, {5814147670840180754, 10}, {576645472412763136, 10}, {42786397639148544, 10}, {2315415374626029896, 10}, {10520549469173335296, 11}, {2317524495633481760, 11}, {360323223285399872, 10}, {9007474451424004, 10}, {5700005885121026, 10}, {10160261531204324352, 10}, {15016162516944359556, 10}, {17636813465603, 10}, {150026164885260370, 11}, {18015225290719265, 11}, {292736450217132032, 10}, {1333100674342224000, 10}, {1153484494829912080, 10}, {145243183935160356, 10}, {4648277800028340236, 10}, {18295882077241348, 10}, {148900299225235458, 11}, {2308517022067064960, 11}, {2666166164849787008, 10}, {10484947351389610496, 10}, {865113409641250944, 10}, {79164905423104, 10}, {598134445769894144, 10}, {8865384334336, 10}, {140741783341184, 11}, {11822236544142419985, 12}, {853358739210241, 11}, {2306689770606579907, 11}, {27305340485764105, 11}, {562958563547782, 12}, {576742261673689253, 11}, {563053041289474, 11}, {72061994248775234, 12},
}
var BishopBestMagics = [64]MagicValue{
	{1171237203947823488, 6}, {2308412585671671873, 5}, {7569428664312397952, 5}, {1155182929459020040, 5}, {883849190865657860, 5}, {23791370577911968, 5}, {4936090344850063874, 5}, {146649013763063808, 6}, {936753137990238992, 5}, {2278222469285378, 5}, {1196989970411233792, 5}, {324720985242599456, 5}, {5764660884244799536, 5}, {2394762130760320, 5}, {621497027822370952, 5}, {13981425596434489600, 5}, {27065647490015380, 5}, {5190404141385548160, 5}, {9605402366906400, 7}, {579851818030354560, 7}, {1190076210669946880, 7}, {73606260729094176, 7}, {63472633420988992, 5}, {144191067330330882, 5}, {9296115726568935426, 5}, {1153494350270302208, 5}, {2594293288496408642, 7}, {288533842569070752, 9}, {282097763762178, 9}, {12682493891987964224, 7}, {3413158987827720, 5}, {144257574865338502, 5}, {9227880378178601482, 5}, {578723650582085891, 5}, {563226173772032, 7}, {4611688219602845825, 9}, {577596552386969664, 9}, {784805039544846344, 7}, {4512990774821376, 5}, {13856521630425031561, 5}, {36187162681018624, 5}, {81208298082213924, 5}, {563370994700560, 7}, {598417927602305, 7}, {1733894656929825796, 7}, {9223935605837201536, 7}, {83396204645406928, 5}, {2594638672888348928, 5}, {4575136872169504, 5}, {1443143505936385, 5}, {288232576282804224, 5}, {2199569041456, 5}, {1181772762902036736, 5}, {582517344230309892, 5}, {4616194085424742402, 5}, {78814110179000972, 5}, {380572319064539168, 6}, {4625202317049012226, 5}, {109354164517619712, 5}, {18256567021373440, 5}, {1154047404782782976, 5}, {586593868780142848, 5}, {9223566169653444672, 5}, {4508038484721921, 6},
}

var RookMagicTable MagicMoveTable
var BishopMagicTable MagicMoveTable

func unmarshalMagics(path string, magics *[64]MagicValue) Error {
	input, err := os.ReadFile(RootDir() + "/data/magics-for-rook.json")
	if !IsNil(err) {
		//lint:ignore nilerr reason it's fine if we haven't cached better magics. We'll compute new ones now.
		return NilError
	}
	err = json.Unmarshal(input, magics)
	if !IsNil(err) {
		return Wrap(err)
	}

	return NilError
}

func marshalMagics(path string, magics *[64]MagicValue) Error {
	output, err := json.Marshal(RookBestMagics)
	if !IsNil(err) {
		return Wrap(err)
	}
	err = os.WriteFile(path, output, 0600)
	return Wrap(err)
}

func initMagicTables() Error {
	err := unmarshalMagics(RootDir()+"/data/magics-for-rook.json", &RookBestMagics)
	if !IsNil(err) {
		return err
	}

	err = unmarshalMagics(RootDir()+"/data/magics-for-bishop.json", &BishopBestMagics)
	if !IsNil(err) {
		return err
	}

	RookMagicTable = generateMagicMoveTable(RookDirs, RookBestMagics, "rook magics ")
	BishopMagicTable = generateMagicMoveTable(BishopDirs, BishopBestMagics, "bishop magic")

	lowestRookBits := 12
	sumRookBits := 0
	for _, m := range RookMagicTable.Magics {
		if m.BitsInMagicIndex < lowestRookBits {
			lowestRookBits = m.BitsInMagicIndex
		}
		sumRookBits += m.BitsInMagicIndex
	}

	lowestBishopBits := 12
	sumBishopBits := 0
	for _, m := range BishopMagicTable.Magics {
		if m.BitsInMagicIndex < lowestBishopBits {
			lowestBishopBits = m.BitsInMagicIndex
		}
		sumBishopBits += m.BitsInMagicIndex
	}

	// log.Println("rook bits for magic index: best", lowestRookBits, "average", sumRookBits/64.0)
	// log.Println("bishop bits for magic index: best", lowestBishopBits, "average", sumBishopBits/64.0)

	err = marshalMagics(RootDir()+"/data/magics-for-rook.json", &RookBestMagics)
	if !IsNil(err) {
		return err
	}

	err = marshalMagics(RootDir()+"/data/magics-for-bishop.json", &BishopBestMagics)
	if !IsNil(err) {
		return err
	}

	return NilError
}

func rand64() uint64 {
	return uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
}

func mostlyZeroRand64() uint64 {
	x := rand64()
	y := rand64()
	z := rand64()
	return x & y & z
}

func MagicIndex(magic uint64, blockerBoard Bitboard, bitsInIndex int) int {
	mult := uint64(blockerBoard) * magic
	shift := 64 - bitsInIndex
	result := mult >> shift
	return int(result)
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
		i := MagicIndex(magic, move.blockerBoard, bitsInIndex)
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

func generateWalkBitboard(
	pieceBoard Bitboard,
	blockerBoard Bitboard,
	dir Dir,
	output Bitboard,
) Bitboard {
	mask := PreMoveMasks[dir]
	offset := Offsets[dir]

	totalOffset := 0
	potential := pieceBoard

	for potential != 0 {
		potential = RotateTowardsIndex64(potential&mask, offset)
		totalOffset += offset

		quiet := potential & ^blockerBoard
		capture := potential & blockerBoard

		output |= quiet | capture

		potential = quiet
	}

	return output
}

func generateBlockerMask(startIndex int, dirs []Dir) Bitboard {
	result := Bitboard(0)
	for _, dir := range dirs {
		walk := generateWalkBitboard(SingleBitboard(startIndex), Bitboard(0), dir, result)
		result |= walk & PreMoveMasks[dir]
	}

	result &= ^SingleBitboard(startIndex)

	return result
}

func generateBlockerBoard(blockerMask Bitboard, seed int) Bitboard {
	result := Bitboard(0)

	buffer := GetIndicesBuffer()
	numBits := bits.OnesCount64(uint64(blockerMask))
	for i := 0; i < numBits; i++ {
		// If the bit at i is 1 in the seed...
		if seed&(1<<i) != 0 {
			// Find the ith one bit in blockerMask and set the corresponding bit to one in result.
			for oneIndex, indexInBitboard := range *blockerMask.EachIndexOfOne(buffer) {
				if oneIndex == i {
					result |= SingleBitboard(indexInBitboard)
				}
			}
		}
	}
	ReleaseIndicesBuffer(buffer)

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

	pieceBoard := SingleBitboard(pieceIndex)

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

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Printf("Alloc = %v KB\n", bToKb(m.Alloc))
	log.Printf("\tTotalAlloc = %v KB\n", bToKb(m.TotalAlloc))
	log.Printf("\tSys = %v KB\n", bToKb(m.Sys))
	log.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToKb(b uint64) uint64 {
	return b / 1024
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

func generateMagicMoveTable(dirs []Dir, bestMagics [64]MagicValue, label string) MagicMoveTable {
	result := MagicMoveTable{}

	// bar := progressbar.Default(64, label)

	for i := 0; i < 64; i++ {
		blockerMask := generateBlockerMask(i, dirs)
		result.BlockerMasks[i] = blockerMask

		moves := generateMoveBoards(i, blockerMask, dirs)

		betterMagic := findBetterMagicValue(bestMagics[i], moves)
		result.Magics[i] = betterMagic

		result.Moves[i] = make([]Bitboard, 1<<betterMagic.BitsInMagicIndex)
		for _, m := range moves {
			magicIndex := MagicIndex(betterMagic.Magic, m.blockerBoard, betterMagic.BitsInMagicIndex)
			result.Moves[i][magicIndex] = m.moveBoard
		}

		// bar.Add(1)
	}

	return result
}

func init() {
	// defer profile.Start(profile.ProfilePath(RootDir() + "/data")).Stop()
	// defer profile.Start(profile.MemProfile, profile.ProfilePath(RootDir() + "/data")).Stop()
	err := initMagicTables()
	if !IsNil(err) {
		fmt.Println("Error initializing magic tables: ", err)
	}
}
