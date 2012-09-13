package mc

import (
	"bytes"
  "fmt"
	"github.com/bmizerany/assert"
	"testing"
  "time"
	"net"
  "math/rand"
	"runtime"
  "strconv"
)

const (
  mcAddr    = "localhost:11211"
  doAuth    = false
  authOnMac = true
  user      = "user-1"
  pass      = "pass"
)

// shared connection
var cn *Conn = nil

// start connection
func testInit(t *testing.T) *Conn {
  if cn == nil {
    nc, err := net.Dial("tcp", mcAddr)
    assert.Equalf(t, nil, err, "%v", err)
    cn = &Conn{rwc: nc, buf: new(bytes.Buffer)}
    testAuth(cn, t)
    // TODO: Should start with a flush...
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


// Some basic tests that functions work
func TestMCSimple(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    VAL1 = "bar"
    VAL2 = "bar-bad"
    VAL3 = "bar-good"
  )

  err := cn.Del(KEY1)
  // TODO: Should be clearer once we have flush...
	if err != ErrNotFound {
    assert.Equalf(t, nil, err, "unexpected error: %v", err)
	}

	_, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "expected missing key: %v", err)

	// unconditional SET
	_, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  cas, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // make sure CAS works
	_, err = cn.Set(KEY1, VAL2, cas + 1, 0, 0)
  assert.Equalf(t, ErrKeyExists, err, "expected CAS mismatch: %v", err)

  // check SET actually set the correct value...
	v, cas2, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %s", v)
  assert.Equalf(t, cas, cas2, "CAS shouldn't have changed: %d, %d", cas, cas2)

  // use correct CAS...
  cas2, err = cn.Set(KEY1, VAL3, cas, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.NotEqual(t, cas, cas2)
}


// Test GET, does it care about CAS?
// NOTE: No it shouldn't, memcached mainline doesn't...
func TestGet(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "fab"
    VAL1 = "faz"
  )

  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // retrieve value with 0 CAS...
  v1, cas1, _, err := cn.getCAS(KEY1, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v1, "wrong value: %s", v1)

  // retrieve value with good CAS...
  v2, cas2, _, err := cn.getCAS(KEY1, cas1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, v1, v2, "value changed when it shouldn't: %s, %s", v1, v2)
  assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)

  // retrieve value with bad CAS...
  v3, cas1, _, err := cn.getCAS(KEY1, cas1 + 1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, v3, v2, "value changed when it shouldn't: %s, %s", v3, v2)
  assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)

  // really make sure CAS is bad (above could be an off by one bug...)
  v4, cas1, _, err := cn.getCAS(KEY1, cas1 + 992313128)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, v4, v2, "value changed when it shouldn't: %s, %s", v4, v2)
  assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)
}


// Test some edge cases of memcached. This was originally done to better
// understand the protocol but servers as a good test for the client and
// server...

// Test SET behaviour with CAS...
func TestSet(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "bar"
    VAL2 = "zar"
  )

  cas1, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  v, cas2, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)
  assert.Equal(t, cas1, cas2, "CAS don't match: %d != %d", cas1, cas2)

  // do two sets of same key, make sure CAS changes...
  cas1, err = cn.Set(KEY2, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  cas2, err = cn.Set(KEY2, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.NotEqual(t, cas1, cas2, "CAS don't match: %d == %d", cas1, cas2)

  // get back the val from KEY2...
  v, cas2, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)

  // make sure changing value works...
  _, err = cn.Set(KEY1, VAL2, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  v, cas1, _, err = cn.Get(KEY1)
  assert.Equalf(t, VAL2, v, "wrong value: %s", v)

  // Delete KEY1 and check it worked, needed for next test...
  err = cn.Del(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)

  // What happens when I set a new key and specify a CAS?
  // (should fail, bad CAS, can't specify a CAS for a non-existent key, it fails,
  // doesn't just ignore the CAS...)
  cas, err := cn.Set(KEY1, VAL1, 1, 0, 0)
  assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)
  assert.Equalf(t, uint64(0), cas, "CAS should be nil: %d", cas)

  // make sure it really didn't set it...
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)
  // no value is the error string from the server...
  // assert.Equalf(t, nil, v, "string should be empty: %s", v)

  // Setting an existing value with bad CAS... should fail
  _, err = cn.Set(KEY2, VAL2, cas2 + 1, 0, 0)
  assert.Equalf(t, ErrKeyExists, err, "wrong error: %v", err)
  v, cas1, _, err = cn.Get(KEY2)
  assert.Equalf(t, VAL1, v, "value shouldn't have changed: %s", v)
  assert.Equalf(t, cas1, cas2, "CAS shouldn't have changed: %d, %d", cas1, cas2)
}


// Testing MAX SIZE of values...
// Testing if when you set a key/value with a bad value (e.g > 1MB) does that
// remove the existing key/value still or leave it intact?
func TestSetBadRemovePrevious(t *testing.T) {
  testInit(t)

  const (
    // Larger than this memcached doesn't like for key 'foo' (with defaults)
    MAX_VAL_SIZE = 1024 * 1024 - 74
    KEY = "foo"
    VAL = "bar"
  )

  // check basic get/set works first
  _, err := cn.Set(KEY, VAL, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err := cn.Get(KEY)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, VAL, v, "wrong value: %s", v)

  // MAX GOOD VALUE

  // generate random bytes
  data := make([]byte, MAX_VAL_SIZE)
  for i := 0; i < MAX_VAL_SIZE; i++ {
    data[i] = byte(rand.Int())
  }

  val := string(data)
  _, err = cn.Set(KEY, val, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err = cn.Get(KEY)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, val, v, "wrong value: (too big to print)")

  // MAX GOOD VALUE * 2

  // generate random bytes
  data = make([]byte, 2 * MAX_VAL_SIZE)
  for i := 0; i < 2 * MAX_VAL_SIZE; i++ {
    data[i] = byte(rand.Int())
  }

  val2 := string(data)
  _, err = cn.Set(KEY, val2, 0, 0, 0)
  assert.Equalf(t, ErrValueTooLarge, err, "expected too large error: %v", err)
  v, _, _, err = cn.Get(KEY)
  if err == nil {
    fmt.Println("\tmemcached removes the old value... so expecting no key")
    fmt.Println("\tnot an error but just a different semantics than memcached")
    // well it should at least be the old value stil..
    assert.Equalf(t, val, v, "wrong value: (too big to print)")
  }
  assert.Equalf(t, ErrNotFound, err, "expected no key: %v", err)
}


// Test ADD.
func TestAdd(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    VAL1 = "bar"
  )

  cn.Del(KEY1)

  // check add works... (key not already present)
  _, err := cn.Add(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error adding key: %v", err)

  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error getting key: %v", err)
  assert.Equalf(t, v, VAL1, "unexpected value for key: %v", v)

  // check add works... (key already present)
  _, err = cn.Add(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, ErrKeyExists, err,
    "expected an error adding existing key: %v", err)

  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error getting key: %v", err)
  assert.Equalf(t, v, VAL1, "unexpected value for key: %v", v)

  // what happens when I add a new value and give a CAS?...
  cn.Del(KEY1)
  _, err = cn.Add(KEY1, VAL1, 100, 0, 0)
  assert.Equalf(t, ErrNotFound, err,
    "expected an error adding new key with a CAS: %v", err)
}


// Test Replace.
func TestReplace(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    VAL1 = "bar"
    VAL2 = "car"
  )

  cn.Del(KEY1)

  // check replace works... (key not already present)
  _, err := cn.Rep(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, ErrNotFound, err,
    "expected an error replacing non-existent key: %v", err)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "expected error getting key: %v", err)

  // check replace works...(key already present)
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)
  _, err = cn.Rep(KEY1, VAL2, 0, 0, 0)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)

  // check replace works [2nd take]... (key not already present)
  cn.Del(KEY1)
  _, err = cn.Rep(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, ErrNotFound, err,
    "expected an error replacing non-existent key: %v", err)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "expected error getting key: %v", err)

  // What happens when I replace a value and give a good CAS?...
  cas, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  cas, err = cn.Rep(KEY1, VAL1, cas, 0, 0)
  assert.Equalf(t, nil, err, "replace with good CAS failed: %v", err)

  // bad CAS
  _, err = cn.Rep(KEY1, VAL2, cas + 1, 0, 0)
  assert.Equalf(t, ErrKeyExists, err, "replace with bad CAS failed: %v", err)
}


// Test Delete.
func TestDelete(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    VAL1 = "bar"
  )

  // delete existing key...
  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  err = cn.Del(KEY1)
  assert.Equalf(t, nil, err, "error deleting key: %v", err)

  // delete non-existent key...
  err = cn.Del(KEY1)
  assert.Equalf(t, ErrNotFound, err,
    "no error deleting non-existent key: %v", err)

  // delete existing key with 0 CAS...
  cas1, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  err = cn.DelCAS(KEY1, cas1 + 1)
  assert.Equalf(t, ErrKeyExists, err,
    "expected an error for deleting key with wrong CAS: %v", err)

  // confirm it isn't gone...
  v, cas1, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err,
    "delete with wrong CAS seems to have succeeded: %v", err)
  assert.Equalf(t, v, VAL1, "corrupted value in cache: %v", v)

  // now delete with good CAS...
  err = cn.DelCAS(KEY1, cas1)
  assert.Equalf(t, nil, err,
    "unexpected error for deleting key with correct CAS: %v", err)

  // delete existing key with good CAS...
  cas1, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  err = cn.DelCAS(KEY1, cas1)
  assert.Equalf(t, nil, err,
    "unexpected error for deleting key with correct CAS: %v", err)
  v, cas1, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err,
    "delete with wrong CAS seems to have succeeded: %v", err)

  // delete existing key with 0 CAS...
  cas1, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  err = cn.DelCAS(KEY1, 0)
  assert.Equalf(t, nil, err,
    "unexpected error for deleting key with 0 CAS: %v", err)
  v, cas1, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err,
    "delete with wrong CAS seems to have succeeded: %v", err)
}


// Test Incr/Decr works...
func TestIncrDecr(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "n"
    N_START uint64 = 10
    N_VAL = "11211"
  )

  // check DEL of non-existing key fails...
  err := cn.Del(KEY1)
	if err != ErrNotFound {
    assert.Equalf(t, nil, err, "unexpected error: %v", err)
	}
	err = cn.Del(KEY1)
  assert.Equalf(t, ErrNotFound, err, "expected missing key: %v", err)

  // test INCR/DECR...

  exp := N_START // track what we expect
	n, cas, err := cn.Incr(KEY1, 1, N_START, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  exp = exp + 1
	n, cas, err = cn.Incr(KEY1, 1, 99, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  exp = exp - 1
	n, cas, err = cn.Decr(KEY1, 1, 97, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // test big addition
  exp = exp + 1123139
	n, cas, err = cn.Incr(KEY1, 1123139, 97, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // test zero addition
  exp = exp + 0
	n, cas, err = cn.Incr(KEY1, 0, 97, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // test CAS works... (should match)
  exp = exp - 1
  n, cas, err = cn.Decr(KEY1, 1, 93, 0, cas)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // test CAS works... (should fail, doesn't match)
  exp = exp
  n, cas, err = cn.Decr(KEY1, 1, 87, 0, cas + 97)
  assert.Equal(t, ErrKeyExists, err, "expected CAS mismatch: %v", err)
  assert.Equal(t, uint64(0), n, "expected 0 due to CAS mismatch: %d", n)
  assert.Equal(t, uint64(0), cas, "expected 0 due to CAS mismatch: %d", cas)

  // test that get on a counter works...
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  vn := strconv.FormatUint(exp, 10)
  assert.Equalf(t, vn, v, "wrong value: %s (expected %s)", n, vn)

  // test that set on a counter works...
  _, err = cn.Set(KEY1, N_VAL, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, N_VAL, v, "wrong value: %s (expected %s)", v, N_VAL)
  exp, err = strconv.ParseUint(N_VAL, 10, 64)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = exp + 1123139
	n, cas, err = cn.Incr(KEY1, 1123139, 97, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	assert.NotEqual(t, 0, cas)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
}


// Test expiration works...
func TestIncrTimeouts(t *testing.T) {
  testInit(t)

  const (
    KEY2 = "n"
    N_START uint64 = 10
  )

  cn.Del(KEY2)

  // Incr (key, delta, initial, ttl, cas)
  exp := N_START
  n, _, err := cn.Incr(KEY2, 1, N_START, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  time.Sleep(1200 * time.Millisecond)

  // no expiration set before, so should incr
  exp = exp + 39
	n, _, err = cn.Incr(KEY2, 39, N_START, 1, 0)
	assert.Equalf(t, nil, err, "%v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  time.Sleep(1200 * time.Millisecond)

  // expiration set before, should have expired the key now...
  // TODO: Below fails, not sure who is wrong...
  // exp = N_START
	// n, _, err = cn.Incr(KEY2, 2, N_START, 0, 0)
	// assert.Equalf(t, nil, err, "%v", err)
  // assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
}

// TODO: Test Touch, GAT, Flush, NoOp, Version, Quit

// Test Append works...
func TestAppend(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "moo"
    VAL2 = "bar"
  )

  cn.Del(KEY1)
  cn.Del(KEY2)

	// normal append
  exp := VAL1
  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = exp + VAL2
  _, err = cn.Append(KEY1, VAL2, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)

  // append to non-existent value
  exp = VAL1
  _, err = cn.Append(KEY2, VAL1, 0)
  assert.Equalf(t, ErrValueNotStored, err,
    "expected value not stored error: %v", err)
	v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "expected not found error: %v", err)

  // check CAS works...
	v, cas, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = v
  _, err = cn.Append(KEY1, VAL2, cas + 1)
  assert.Equalf(t, ErrKeyExists, err, "expected key exists error: %v", err)
  v, cas2, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
  assert.Equalf(t, cas, cas2, "CAS shouldn't have changed: %d != %d", cas, cas2)
  exp = exp + VAL2
  _, err = cn.Append(KEY1, VAL2, cas)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = exp + VAL1

  // check 0 CAS...
  _, err = cn.Append(KEY1, VAL1, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
}


// Test Prepend works...
func TestPrepend(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "moo"
    VAL2 = "bar"
  )

  cn.Del(KEY1)
  cn.Del(KEY2)

	// normal append
  exp := VAL1
  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = VAL2 + exp
  _, err = cn.Prepend(KEY1, VAL2, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
	v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)

  // append to non-existent value
  exp = VAL1
  _, err = cn.Prepend(KEY2, VAL1, 0)
  assert.Equalf(t, ErrValueNotStored, err,
    "expected value not stored error: %v", err)
	v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "expected not found error: %v", err)

  // check CAS works...
	v, cas, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = v
  _, err = cn.Prepend(KEY1, VAL2, cas + 1)
  assert.Equalf(t, ErrKeyExists, err, "expected key exists error: %v", err)
  v, cas2, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
  assert.Equalf(t, cas, cas2, "CAS shouldn't have changed: %d != %d", cas, cas2)
  exp = VAL2 + exp
  _, err = cn.Prepend(KEY1, VAL2, cas)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = VAL1 + exp

  // check 0 CAS...
  _, err = cn.Prepend(KEY1, VAL1, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
}

