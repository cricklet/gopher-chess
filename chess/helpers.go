package chess

type Success bool

func mapSlice[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func reduceSlice[T, U any](ts []T, initial U, f func(U, T) U) U {
	u := initial
	for _, t := range ts {
		u = f(u, t)
	}
	return u
}

func reverseBits(n uint8) uint8 {
	return ReverseBitsCache[n]
}

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

func absDiff(x int, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

func flipArray(array [8][8]int) [8][8]int {
	result := [8][8]int{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			result[i][j] = array[7-i][j]
		}
	}
	return result
}
