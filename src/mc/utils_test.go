package mc

import (
  "bytes"
  "github.com/bmizerany/assert"
  "testing"
  "net"
  "runtime"
)

// start connection
func testInit(t *testing.T) *Conn {
  if cn == nil {
    nc, err := net.Dial("tcp", mcAddr)
    assert.Equalf(t, nil, err, "%v", err)
    cn = &Conn{rwc: nc, buf: new(bytes.Buffer)}
    testAuth(cn, t)
    cn.Flush(0)
    assert.Equalf(t, nil, err, "unexpected error: %v", err)
  }
  return cn
}

// login if required...
func testAuth(cn *Conn, t *testing.T) bool{
  if !doAuth {
    return false
  } else if runtime.GOOS == "darwin" {
    if !authOnMac {
      return false
    } else {
      println("On Darwin but testing auth anyway")
    }
  } else {
    println("Not on Darwin, testing auth")
  }

  err := cn.Auth(user, pass)
  assert.Equalf(t, nil, err, "authentication failed: %v", err)
  return true
}

// TODO: would be nice to use helpers but the assert gives just the location of
// assert, I need more of a stack trace...

func get(t *testing.T, key, val string, ecas uint64, eerr error) {
  v, ocas, _, err := cn.Get(key)
  assert.Equalf(t, eerr, err, "shouldn't be an error: %v", err)
  if eerr == nil && val != "" {
    assert.Equalf(t, val, v, "wrong value: %v", v)
  }
  if ecas != 0 {
    assert.Equalf(t, ecas, ocas, "wrong CAS for get: %d != %d", ocas, ecas)
  }
}

func set(t *testing.T, key, val string, ocas uint64, flags, exp uint32, ecas uint64, eerr error) {
  ocas, err := cn.Set(key, val, 0, 0, 0)
  assert.Equalf(t, eerr, err, "shouldn't be an error: %v", err)
  if ecas != 0 {
    assert.Equalf(t, ecas, ocas, "wrong CAS for get: %d != %d", ocas, ecas)
  }
}

func flush(t *testing.T, when uint32, eerr error) {
  err := cn.Flush(when)
  assert.Equalf(t, eerr, err, "shouldn't be an error: %v", err)
}

