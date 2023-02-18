package chess

import (
	"math/bits"

	. "github.com/cricklet/chessgo/internal/helpers"
)

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
