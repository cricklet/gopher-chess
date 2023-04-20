package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNil(t *testing.T) {
	var err error
	assert.True(t, IsNil(err))

	var traceableErr Error = NilError
	assert.True(t, IsNil(traceableErr))
	assert.True(t, IsNil(&traceableErr))
}

func TestErrorRef(t *testing.T) {
	errRef := ErrorRef{}
	assert.True(t, errRef.IsNil())

	errRef.Add(NilError)
	assert.True(t, errRef.IsNil())

	errRef.Add(Errorf("asdf"))
	assert.False(t, errRef.IsNil())
	assert.Equal(t, 1, errRef.NumErrors())

	func(errRefCopy ErrorRef) {
		errRefCopy.Add(Errorf("qwerty"))
	}(errRef)
	assert.False(t, errRef.IsNil())
	assert.Equal(t, 2, errRef.NumErrors())

	foo := func(errRefCopy ErrorRef) {
		errRefCopy.Add(Errorf("bar"))
	}
	func(errRefCopy ErrorRef) {
		foo(errRefCopy)
	}(errRef)

	assert.False(t, errRef.IsNil())
	assert.Equal(t, 3, errRef.NumErrors())

	// fmt.Println(errRef.Error().String())
	// fmt.Println(errRef.Error().Error())
}
