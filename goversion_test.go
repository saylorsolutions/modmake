//go:build it

package modmake

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoTools_PinLatest(t *testing.T) {
	var (
		inst *GoTools
		buf  bytes.Buffer
	)
	assert.NoError(t, Go().Command("version").Output(&buf).Run(context.Background()))
	assert.NotContains(t, buf.String(), "1.6.4")

	buf.Reset()
	assert.NotPanics(t, func() {
		inst = Go().PinLatest(6)
	})
	assert.NoError(t, inst.Command("version").Output(&buf).Run(context.Background()))
	assert.Contains(t, buf.String(), "1.6.4")

	buf.Reset()
	Go().InvalidateCache()
	assert.NoError(t, Go().Command("version").Output(&buf).Run(context.Background()))
	assert.NotContains(t, buf.String(), "1.6.4")
}
