package mc

import (
	"bytes"
	"github.com/bmizerany/assert"
	"testing"
	"net"
)

const mcAddr = "localhost:11211"

func TestMCSimple(t *testing.T) {
	nc, err := net.Dial("tcp", mcAddr)
	assert.Equalf(t, nil, err, "%v", err)

	cn := &Conn{rwc: nc, buf: new(bytes.Buffer)}

	err = cn.Auth("mcgo", "foo")
	assert.Equalf(t, nil, err, "%v", err)

	err = cn.Del("foo")
	if err != ErrNotFound {
		assert.Equalf(t, nil, err, "%v", err)
	}

	_, _, err = cn.Get("foo")
	assert.Equalf(t, ErrNotFound, err, "%v", err)

	err = cn.Set("foo", "bar", 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)

	err = cn.Set("foo", "bar", 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)

	err = cn.Del("n")
	if err != ErrNotFound {
		assert.Equalf(t, nil, err, "%v", err)
	}

	n, cas, err := cn.Incr("n", 1, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, 1, n)

	n, cas, err = cn.Incr("n", 1, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, 2, n)

	n, cas, err = cn.Decr("n", 1, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, 1, n)
}
