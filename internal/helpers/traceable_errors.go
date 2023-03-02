package helpers

import (
	"errors"

	"github.com/ztrue/tracerr"
)

type Error struct {
	err   tracerr.Error
	other []tracerr.Error
}

var NilError = Error{nil, nil}

func IsNil(err error) bool {
	var traceableErr Error
	if errors.As(err, &traceableErr) {
		return traceableErr.First() == nil
	}
	return err == nil
}

var _errorNumber = 0
var _errorIndents = []string{
	".  ",
	"-  ",
}

func (e Error) Error() string {
	_errorNumber = (_errorNumber + 1) % len(_errorIndents)
	return Indent(tracerr.SprintSourceColor(e.err, 3), _errorIndents[_errorNumber])
}

func (e Error) String() string {
	return tracerr.SprintSourceColor(e.err, 3)
}

func (e Error) First() tracerr.Error {
	return e.err
}

func Wrap(err error) Error {
	return Error{tracerr.Wrap(err), nil}
}

func WrapReturn[T any](x T, err error) (T, Error) {
	return x, Wrap(err)
}

func Join(others ...Error) Error {
	hasError := false
	for _, o := range others {
		if !IsNil(o) {
			hasError = true
			break
		}
	}
	if !hasError {
		return NilError
	}

	others = FilterSlice(others, func(err Error) bool {
		return !IsNil(err)
	})
	if len(others) == 1 {
		return others[0]
	} else {
		return Error{others[0].err, MapSlice(others[1:], func(e Error) tracerr.Error { return e.err })}
	}
}

func Errorf(format string, args ...interface{}) Error {
	return Error{tracerr.Errorf(format, args...), nil}
}
