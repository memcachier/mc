package mc

import (
	"github.com/bmizerany/assert"
	"testing"
)

const mcAddr = "localhost:11211"

func TestMCSimple(t *testing.T) {
	cn, err := Dial(mcAddr)
	assert.Equalf(t, nil, err, "%v", err)

	_, _, err = cn.Get("foo")
	assert.Equalf(t, ErrNotFound, err, "%v", err)
}
