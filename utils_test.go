package mc

import (
	"bytes"
	"compress/zlib"
	"io"
	"reflect"
	"strings"
	"testing"
)

// start connection
func testInitCompress(t *testing.T) *Client {
	config := DefaultConfig()
	config.Compression.compress = func(value string) (string, error) {
		var compressedValue bytes.Buffer
		zw, err := zlib.NewWriterLevel(&compressedValue, -1)
		if err != nil {
			return value, err
		}
		if _, err = zw.Write([]byte(value)); err != nil {
			return value, err
		}
		zw.Close()
		return compressedValue.String(), nil
	}

	config.Compression.decompress = func(value string) (string, error) {
		if value == "" {
			return value, nil
		}
		zr, err := zlib.NewReader(strings.NewReader(value))
		if err != nil {
			return value, err
		}
		defer zr.Close()
		var unCompressedValue bytes.Buffer
		_, err = io.Copy(&unCompressedValue, zr)
		if err != nil {
			return value, err
		}
		return unCompressedValue.String(), nil
	}

	c := NewMCwithConfig(mcAddr, user, pass, config)
	err := c.Flush(0)
	assertEqualf(t, nil, err, "unexpected error during initial flush: %v", err)
	return c
}

// start connection
func testInit(t *testing.T) *Client {
	c := NewMC(mcAddr, user, pass)
	err := c.Flush(0)
	assertEqualf(t, nil, err, "unexpected error during initial flush: %v", err)
	return c
}

// start connection
func testInitMultiNode(t *testing.T, servers, username, password string) *Client {
	c := NewMC(servers, user, pass)
	err := c.Flush(0)
	assertEqualf(t, nil, err, "unexpected error during initial flush: %v", err)
	return c
}

// TODO: asserts gives just the location of FailNow, more of a stack trace would be nice

func assertEqualf(t *testing.T, exp, got interface{}, format string, args ...interface{}) {
	if !reflect.DeepEqual(exp, got) {
		t.Errorf(format, args...)
		t.Errorf("expected: %v", exp)
		t.Errorf("got: %v", got)
		t.FailNow()
	}
}

func assertNotEqualf(t *testing.T, exp, got interface{}, format string, args ...interface{}) {
	if reflect.DeepEqual(exp, got) {
		t.Errorf(format, args...)
		t.Errorf("did not expect: %v", exp)
		t.FailNow()
	}
}

func assertTruef(t *testing.T, res bool, format string, args ...interface{}) {
	if !res {
		t.Fatalf(format, args...)
	}
}
