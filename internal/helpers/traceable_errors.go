package helpers

import (
	"github.com/ztrue/tracerr"
)

type Error struct {
	err   tracerr.Error
	other []tracerr.Error
}

func (e *Error) IsNil() bool {
	return IsNil(e)
}

type ErrorRef struct {
	// only hold a single value so the internal accumulated value isn't copied
	// when ErrorAccumulator is passed by value
	reference []Error
}

func (e *ErrorRef) Add(err Error) {
	if e.reference == nil {
		e.reference = []Error{err}
	} else {
		e.reference[0] = Join(e.reference[0], err)
	}
}

func (e *ErrorRef) IsNil() bool {
	return e.reference == nil || IsNil(e.reference[0])
}

func (e *ErrorRef) HasError() bool {
	return !e.IsNil()
}

func (e *ErrorRef) Error() Error {
	if e.reference == nil {
		return NilError
	} else {
		return e.reference[0]
	}
}

var NilError = Error{nil, nil}

func IsNil(err error) bool {
	if traceableErr, ok := err.(Error); ok {
		return traceableErr.First() == nil
	}
	if traceableErr, ok := err.(*Error); ok {
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
	return Indent(tracerr.Sprint(e.err), _errorIndents[_errorNumber])
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

func JoinReturn[T any](e Error, x T, err Error) (T, Error) {
	return x, Join(e, Join(err))
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
