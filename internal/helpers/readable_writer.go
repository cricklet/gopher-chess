package helpers

import (
	"sync"
)

func AppendSafe[T any](m *sync.Mutex, slice []T, item T) []T {
	m.Lock()
	defer m.Unlock()
	return append(slice, item)
}
