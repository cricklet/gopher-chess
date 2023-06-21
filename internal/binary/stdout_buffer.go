package binary

import (
	"sync"

	. "github.com/cricklet/chessgo/internal/helpers"
)

type StdOutBuffer struct {
	buffer []string
	read   int

	waitingLock sync.Mutex
	waiting     Optional[chan bool]

	noCopy NoCopy
}

func (u *StdOutBuffer) Update(line string) {
	u.buffer = append(u.buffer, line)

	u.waitingLock.Lock()
	defer u.waitingLock.Unlock()
	if u.waiting.HasValue() {
		u.waiting.Value() <- true
		u.waiting = Empty[chan bool]()
	}
}

func (u *StdOutBuffer) Flush(callback func(line string) Error) Error {
	for i := u.read; i < len(u.buffer); i++ {
		err := callback(u.buffer[i])
		if !IsNil(err) {
			break
		}
	}

	u.read = len(u.buffer)

	return NilError
}

func (u *StdOutBuffer) Wait() chan bool {
	u.waitingLock.Lock()
	defer u.waitingLock.Unlock()

	if u.waiting.HasValue() {
		return u.waiting.Value()
	}

	u.waiting = Some(make(chan bool))
	return u.waiting.Value()
}
