package apm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopOption(t *testing.T) {
	asserter := assert.New(t)
	requirer := require.New(t)

	asserter.NotNil(Global())

	ins, err := New(t.Context(), &Options{})
	requirer.NoError(err)
	asserter.NotNil(ins)

	requirer.NoError(ins.Shutdown(t.Context()))
	assert.NotNil(t, Global().AppTracer())
	assert.NotNil(t, Global().AppMeter())
}
