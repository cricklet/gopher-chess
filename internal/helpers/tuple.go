package helpers

type Pair[T, U any] struct {
	First  T
	Second U
}

type Triple[A, B, C any] struct {
	A A
	B B
	C C
}
