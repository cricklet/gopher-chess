package binary

import (
	. "github.com/cricklet/chessgo/internal/helpers"
)

type StdOutBuffer struct {
	buffer  []string
	updated chan bool
	read    int

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

func (u *StdOutBuffer) Flush(callback func(line string) Error) Error {
	var err Error

	for i := u.read; i < len(u.buffer); i++ {
		err = callback(u.buffer[i])
		if !IsNil(err) {
			break
		}
	}

	u.read = len(u.buffer)

	return err
}

func (u *StdOutBuffer) Wait() chan bool {
	return u.updated
}
