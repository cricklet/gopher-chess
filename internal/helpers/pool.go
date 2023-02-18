package helpers

import (
	"fmt"
	"sync"
)

type PoolStats struct {
	creates int
	resets  int
	hits    int
}

func (s PoolStats) String() string {
	return fmt.Sprint("creates: ", s.creates, ", resets: ", s.resets, ", hits: ", s.hits)
}

func CreatePool[T any](create func() T, reset func(*T)) (func() *T, func(*T), func() PoolStats) {
	availableBuffer := [256]*T{}
	startIndex := 0
	endIndex := 0

	lock := sync.Mutex{}

	creates := 0
	resets := 0
	hits := 0

	var get = func() *T {
		lock.Lock()

		if endIndex != startIndex {
			result := availableBuffer[startIndex]
			startIndex = (startIndex + 1) % 256

			lock.Unlock()

			hits++
			return result
		}

		lock.Unlock()

		creates++
		result := create()
		return &result
	}

	var release = func(t *T) {
		resets++
		reset(t)

		lock.Lock()
		availableBuffer[endIndex] = t
		endIndex = (endIndex + 1) % 256
		lock.Unlock()
	}

	var stats = func() PoolStats {
		return PoolStats{creates, resets, hits}
	}

	return get, release, stats
}
