package binary

import (
	. "github.com/cricklet/chessgo/internal/helpers"
)

type StdOutBuffer struct {
	buffer  []string
	updated chan bool

	noCopy NoCopy
}

func (u *StdOutBuffer) Update(line string) {
	u.buffer = append(u.buffer, line)
	select {
	case u.updated <- true:
		{
		}
	default:
		{
		}
	}
}

func (u *StdOutBuffer) Flush(callback func(line string)) {
	for _, line := range u.buffer {
		callback(line)
	}

	u.buffer = []string{}
}

func (u *StdOutBuffer) Wait() chan bool {
	return u.updated
}
