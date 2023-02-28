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
}
