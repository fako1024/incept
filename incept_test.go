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
