package helpers

import (
	"io"
	"strings"
	"sync"
)

type ReadableWriter struct {
	Writer   io.Writer
	ReadChan chan string
}

func (r *ReadableWriter) Write(p []byte) (n int, err error) {
	for _, line := range strings.Split(string(p), "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			r.ReadChan <- line
		}
	}
	return r.Writer.Write(p)
}

func AppendSafe[T any](m *sync.Mutex, slice []T, item T) []T {
	m.Lock()
	defer m.Unlock()
	return append(slice, item)
}
