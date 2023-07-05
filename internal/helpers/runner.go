package helpers

import "time"

type Position struct {
	Fen   string
	Moves []string
}

type SearchParams struct {
	Depth    Optional[int]
	Duration Optional[time.Duration]
}

type Runner interface {
	PerformMoveFromString(s string) Error
	SetupPosition(position Position) Error
	PerformMoves(startPos string, moves []string) Error
	MovesForSelection(s string) ([]string, Error)
	Rewind(num int) Error
	Reset()
	Search(SearchParams) (Optional[string], Optional[int], int, Error)
	IsNew() bool
}
