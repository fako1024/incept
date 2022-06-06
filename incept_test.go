package incept

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	i, err := New(
		WithExitFn(func(code int) {
			if code != 0 {
				t.Fatalf("Unexpected return code %d", code)
			}
		}),
	)
	assert.Nil(t, err)
	_ = i
}

func TestUpdate(t *testing.T) {
	i, err := New(
		WithExitFn(func(code int) {
			if code != 0 {
				t.Fatalf("Unexpected return code %d", code)
			}
		}),
	)
	assert.Nil(t, err)

	binaryData, err := i.Binary()
	assert.Nil(t, err)

	assert.Nil(t, i.Update(binaryData))
}
