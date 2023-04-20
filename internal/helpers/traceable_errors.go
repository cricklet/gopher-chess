package helpers

import (
	"github.com/ztrue/tracerr"
)

type Error struct {
	errs []tracerr.Error
}

func (e *Error) IsNil() bool {
	return IsNil(e)
}

type ErrorRef struct {
	// only hold a single value so the internal accumulated value isn't copied
	// when ErrorAccumulator is passed by value
	reference []Error
}

func (errRef *ErrorRef) NumErrors() int {
	if errRef.IsNil() {
		return 0
	}
	return errRef.reference[0].NumErrors()
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

var NilError = Error{nil}

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
	result := ""
	for _, err := range e.errs {
		result += Indent(tracerr.Sprint(err), _errorIndents[_errorNumber]) + "\n"
	}
	return result
}

func (e Error) String() string {
	result := ""
	for _, err := range e.errs {
		result += "-------------------------------------------------------------------------------\n"
		result += tracerr.SprintSourceColor(err, 3) + "\n"
	}
	return result
}

func (e Error) First() tracerr.Error {
	if e.errs == nil {
		return nil
	} else {
		return e.errs[0]
	}
}

func Wrap(err error) Error {
	return Error{[]tracerr.Error{tracerr.Wrap(err)}}
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
		result := Error{}
		for _, o := range others {
			result.errs = append(result.errs, o.errs...)
		}
		return result
	}
}

func (err Error) NumErrors() int {
	if IsNil(err) {
		return 0
	}

	num := 0
	for _, e := range err.errs {
		if e != nil {
			num++
		}
	}
	return num
}

func Errorf(format string, args ...interface{}) Error {
	return Error{[]tracerr.Error{tracerr.Errorf(format, args...)}}
}
