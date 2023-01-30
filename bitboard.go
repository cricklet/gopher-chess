package chessgo

import (
	"fmt"
	"strings"
)

type Bitboard uint64

func SingleUint8(indexFromTheRight int) uint8 {
	return 1 << indexFromTheRight
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

func ReverseBits(n uint8) uint8 {
	return ReverseBitsCache[n]
}

func ShiftTowardsIndex0(b Bitboard, n int) Bitboard {
	return b >> n
}

func ShiftTowardsIndex64(b Bitboard, n int) Bitboard {
	return b << n
}

func SingleBitboard(index int) Bitboard {
	return ShiftTowardsIndex64(1, index)
}

func (b Bitboard) string() string {
	ranks := [8]string{}
	for rank := 0; rank < 8; rank++ {
		bitsBefore := rank * 8
		bitsAfter := 64 - bitsBefore - 8

		r := b

		// clip everything above this rank
		r = ShiftTowardsIndex64(r, bitsAfter)
		// clip everything before this rank
		r = ShiftTowardsIndex0(r, bitsBefore+bitsAfter)

		// mirror the bits so we're printing in a natural order
		// (10000000 for the top left / lowest index instead of 00000001)
		ranks[7-rank] = fmt.Sprintf("%08b", ReverseBits(uint8(r)))
	}

	return strings.Join(ranks[0:], "\n")
}
