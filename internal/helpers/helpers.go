package helpers

type Success bool

func Ignore(t any) {
}

func MapSlice[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func FilterSlice[T any](ts []T, f func(T) bool) []T {
	filtered := []T{}
	for i := range ts {
		if f(ts[i]) {
			filtered = append(filtered, ts[i])
		}
	}
	return filtered
}
func FindInSlice[T any](ts []T, f func(T) bool) Optional[T] {
	for i := range ts {
		if f(ts[i]) {
			return Some(ts[i])
		}
	}
	return Empty[T]()
}

func ReduceSlice[T, U any](ts []T, initial U, f func(U, T) U) U {
	u := initial
	for _, t := range ts {
		u = f(u, t)
	}
	return u
}

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

func AbsDiff(x int, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

func MinInt(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func FlipArray(array [8][8]int) [8][8]int {
	result := [8][8]int{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			result[i][j] = array[7-i][j]
		}
	}
	return result
}
