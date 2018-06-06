package mc

import (
	"reflect"
	"testing"
)

// start connection
func testInit(t *testing.T) *Client {
	c := NewMC(mcAddr, user, pass)
	err := c.Flush(0)
	assertEqualf(t, nil, err, "unexpected error during initial flush: %v", err)
	return c
}

// TODO: asserts gives just the location of FailNow, more of a stack trace would be nice

func assertEqualf(t *testing.T, exp, got interface{}, format string, args ...interface{}) {
	if !reflect.DeepEqual(exp, got) {
		t.Errorf(format, args)
		t.Errorf("expected: %v", exp)
		t.Errorf("got: %v", got)
		t.FailNow()
	}
}

func assertNotEqualf(t *testing.T, exp, got interface{}, format string, args ...interface{}) {
	if reflect.DeepEqual(exp, got) {
		t.Errorf(format, args)
		t.Errorf("did not expect: %v", exp)
		t.FailNow()
	}
}

func assertTruef(t *testing.T, res bool, format string, args ...interface{}) {
	if !res {
		t.Fatalf(format, args)
	}
}
