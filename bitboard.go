package chessgo

type Bitboard uint64

func SingleBoard(index int) uint64 {
	return 2 << index
}

// func (b Bitboard) string() {
// 	for i := 0; i < 64; i += 8 {
// 		c := b << i
// 	}
// }
