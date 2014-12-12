package mc

import (
  "fmt"
  "github.com/bmizerany/assert"
  "testing"
  "time"
  "math/rand"
  "regexp"
  "strconv"
)

const (
  mcAddr    = "localhost:11289"
  doAuth    = false
  authOnMac = true
  user      = "user-1"
  pass      = "pass"
)

// shared connection
var cn *Conn = nil

// Some basic tests that functions work
func TestMCSimple(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    VAL1 = "bar"
    VAL2 = "bar-bad"
    VAL3 = "bar-good"
  )

  _, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "expected missing key: %v", err)

  // unconditional SET
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  cas, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // make sure CAS works
  _, err = cn.Set(KEY1, VAL2, 0, 0, cas + 1)
  assert.Equalf(t, ErrKeyExists, err, "expected CAS mismatch: %v", err)

  // check SET actually set the correct value...
  v, _, cas2, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %s", v)
  assert.Equalf(t, cas, cas2, "CAS shouldn't have changed: %d, %d", cas, cas2)

  // use correct CAS...
  cas2, err = cn.Set(KEY1, VAL3, 0, 0, cas)
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
  v1, _, cas1, err := cn.getCAS(KEY1, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v1, "wrong value: %s", v1)

  // retrieve value with good CAS...
  v2, _, cas2, err := cn.getCAS(KEY1, cas1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, v1, v2, "value changed when it shouldn't: %s, %s", v1, v2)
  assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)

  // retrieve value with bad CAS...
  v3, _, cas1, err := cn.getCAS(KEY1, cas1 + 1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, v3, v2, "value changed when it shouldn't: %s, %s", v3, v2)
  assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)

  // really make sure CAS is bad (above could be an off by one bug...)
  v4, _, cas1, err := cn.getCAS(KEY1, cas1 + 992313128)
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
  v, _, cas2, err := cn.Get(KEY1)
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
  v, _, cas2, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)

  // make sure changing value works...
  _, err = cn.Set(KEY1, VAL2, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  v, _, cas1, err = cn.Get(KEY1)
  assert.Equalf(t, VAL2, v, "wrong value: %s", v)

  // Delete KEY1 and check it worked, needed for next test...
  err = cn.Del(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)

  // What happens when I set a new key and specify a CAS?
  // (should fail, bad CAS, can't specify a CAS for a non-existent key, it fails,
  // doesn't just ignore the CAS...)
  cas, err := cn.Set(KEY1, VAL1, 0, 0, 1)
  assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)
  assert.Equalf(t, uint64(0), cas, "CAS should be nil: %d", cas)

  // make sure it really didn't set it...
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)
  // TODO: On errors a human readable error description should be returned. So
  // could test that.

  // Setting an existing value with bad CAS... should fail
  _, err = cn.Set(KEY2, VAL2, 0, 0, cas2 + 1)
  assert.Equalf(t, ErrKeyExists, err, "wrong error: %v", err)
  v, _, cas1, err = cn.Get(KEY2)
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
    MAX_VAL_SIZE = 1024 * 1024 - 80
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
  } else {
    assert.Equalf(t, ErrNotFound, err, "expected no key: %v", err)
  }
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
  _, err := cn.Add(KEY1, VAL1, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error adding key: %v", err)

  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error getting key: %v", err)
  assert.Equalf(t, v, VAL1, "unexpected value for key: %v", v)

  // check add works... (key already present)
  _, err = cn.Add(KEY1, VAL1, 0, 0)
  assert.Equalf(t, ErrKeyExists, err,
    "expected an error adding existing key: %v", err)

  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error getting key: %v", err)
  assert.Equalf(t, v, VAL1, "unexpected value for key: %v", v)
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
  _, err := cn.Replace(KEY1, VAL1, 0, 0, 0)
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
  _, err = cn.Replace(KEY1, VAL2, 0, 0, 0)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)

  // check replace works [2nd take]... (key not already present)
  cn.Del(KEY1)
  _, err = cn.Replace(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, ErrNotFound, err,
    "expected an error replacing non-existent key: %v", err)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "expected error getting key: %v", err)

  // What happens when I replace a value and give a good CAS?...
  cas, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  cas, err = cn.Replace(KEY1, VAL1, 0, 0, cas)
  assert.Equalf(t, nil, err, "replace with good CAS failed: %v", err)

  // bad CAS
  _, err = cn.Replace(KEY1, VAL2, 0, 0, cas + 1)
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
  v, _, cas1, err := cn.Get(KEY1)
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
  v, _, cas1, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err,
    "delete with wrong CAS seems to have succeeded: %v", err)

  // delete existing key with 0 CAS...
  cas1, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  err = cn.DelCAS(KEY1, 0)
  assert.Equalf(t, nil, err,
    "unexpected error for deleting key with 0 CAS: %v", err)
  v, _, cas1, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err,
    "delete with wrong CAS seems to have succeeded: %v", err)
}


// Test behaviour of errors and cache removal.
// NOTE: calling incr/decr on a non-numeric returns an error BUT also seems to
//       remove it from the cache...
// NOTE: I think above may have been a bug present in memcache 1.4.12 but is
//       fixed in 1.4.13...
func TestIncrDecrNonNumeric(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "n"
    N_START uint64 = 10
    N_VAL = "11211"
    VAL = "nup"
  )

  _, err := cn.Set(KEY1, VAL, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, v, VAL, "wrong value: %v", v)

  _, _, err = cn.Incr(KEY1, 1, N_START, 0, 0)
  assert.Equalf(t, ErrNonNumeric, err, "unexpected error: %v", err)

  _, _, err = cn.Decr(KEY1, 1, N_START, 0, 0)
  assert.Equalf(t, ErrNonNumeric, err, "unexpected error: %v", err)

  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, v, VAL, "wrong value: %v", v)
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

  // no delta_only set before, so should incr
  exp = exp + 39
  n, _, err = cn.Incr(KEY2, 39, N_START, 1, 0)
  assert.Equalf(t, nil, err, "%v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
}


// Test Incr/Decr expiration field.
// This is a stupid name for the field as it has nothing to do with expiration /
// ttl. Instead its used to indicate that the incr/decr should fail if the key
// doesn't already exist in the cache. (i.e., that is since the incr/decr
// command takes both an initial value and a delta, the expiration field allows
// us to say that only the delta should be applied and rather than use the
// initial value when the key doesn't exist, throw an error).
//
// Only the value 0xffffffff is used to indicate that only the delta should be
// applied, all other values for expiration allow either the initial or delta to
// be used.
func TestIncrExpiration(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "n"
    N_START uint64 = 10
    ONLY_DELTA uint32 = 0xffffffff
  )

  // fail as we only allow applying the delta with that expiration value.
  cn.Del(KEY1)
  _, _, err := cn.Incr(KEY1, 10, N_START, ONLY_DELTA, 0)
  assert.Equalf(t, ErrNotFound, err, "unexpected error: %v", err)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "key shouldn't exist in cache: %v", err)

  // suceed this time. Any value but ONLY_DELTA should succeed.
  exp := N_START
  n, _, err := cn.Incr(KEY1, 10, N_START, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
  cn.Del(KEY1)

  // suceed this time. Any value but ONLY_DELTA should succeed.
  exp = N_START
  n, _, err = cn.Incr(KEY1, 10, N_START, 1, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
  cn.Del(KEY1)

  // suceed this time. Any value but ONLY_DELTA should succeed.
  exp = N_START
  n, _, err = cn.Incr(KEY1, 10, N_START, ONLY_DELTA - 1, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
  cn.Del(KEY1)
}


// Test Incr/Decr overflow...
func TestIncrDecrWrap(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "n"
    N_START uint64 = 10
    MAX_1 uint64 = 0xfffffffffffffffe
    MAX uint64 = 0xffffffffffffffff
  )

  // setup...
  exp := N_START
  n, _, err := cn.Decr(KEY1, N_START + 1, N_START, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // can't decr past 0...
  exp = 0
  n, _, err = cn.Decr(KEY1, N_START + 1, N_START, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // test limit of incr...
  exp = MAX_1
  n, _, err = cn.Incr(KEY1, MAX_1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  exp = MAX
  n, _, err = cn.Incr(KEY1, 1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)

  // overflow... wrap around
  exp = 0
  n, _, err = cn.Incr(KEY1, 1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, n, "wrong value: %d (expected %d)", n, exp)
}

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
  if err != ErrValueNotStored {
    t.Errorf("expected 'value not stored error', got: %v", err)
  }
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "expected not found error: %v", err)

  // check CAS works...
  v, _, cas, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = v
  _, err = cn.Append(KEY1, VAL2, cas + 1)
  assert.Equalf(t, ErrKeyExists, err, "expected key exists error: %v", err)
  v, _, cas2, err := cn.Get(KEY1)
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
  if err != ErrValueNotStored {
    t.Errorf("expected 'value not stored error', got: %v", err)
  }
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "expected not found error: %v", err)

  // check CAS works...
  v, _, cas, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = v
  _, err = cn.Prepend(KEY1, VAL2, cas + 1)
  assert.Equalf(t, ErrKeyExists, err, "expected key exists error: %v", err)
  v, _, cas2, err := cn.Get(KEY1)
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


// Test NoOp works... (by putting NoOps all between the prepend tests)
func TestNoOp(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "moo"
    VAL2 = "bar"
  )

  err := cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)
  err = cn.NoOp()
  err = cn.NoOp()
  err = cn.NoOp()
  err = cn.NoOp()
  err = cn.NoOp()
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)

  cn.Del(KEY1)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)
  cn.Del(KEY2)

  // normal append
  exp := VAL1
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)
  exp = VAL2 + exp
  _, err = cn.Prepend(KEY1, VAL2, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)

  // append to non-existent value
  exp = VAL1
  _, err = cn.Prepend(KEY2, VAL1, 0)
  if err != ErrValueNotStored {
    t.Errorf("expected 'value not stored error', got: %v", err)
  }
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "expected not found error: %v", err)

  // check CAS works...
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, cas, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = v
  _, err = cn.Prepend(KEY1, VAL2, cas + 1)
  assert.Equalf(t, ErrKeyExists, err, "expected key exists error: %v", err)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, cas2, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
  assert.Equalf(t, cas, cas2, "CAS shouldn't have changed: %d != %d", cas, cas2)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)
  exp = VAL2 + exp
  _, err = cn.Prepend(KEY1, VAL2, cas)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  exp = VAL1 + exp
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)

  // check 0 CAS...
  _, err = cn.Prepend(KEY1, VAL1, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, exp, v, "wrong value: %s", v)
  err = cn.NoOp()
  assert.Equalf(t, nil, err, "noop unexpected error: %v", err)
}


// Test Flush behaviour
func TestFlush(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    KEY3 = "hoo"
    VAL1 = "bar"
    VAL2 = "zar"
    VAL3 = "gar"
  )

  err := cn.Flush(0)
  assert.Equalf(t, nil, err, "flush produced error: %v", err)

  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)

  err = cn.Flush(0)
  assert.Equalf(t, nil, err, "flush produced error: %v", err)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)

  // do two sets of same key, make sure CAS changes...
  cas1, err := cn.Set(KEY2, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  cas2, err := cn.Set(KEY2, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.NotEqual(t, cas1, cas2, "CAS don't match: %d == %d", cas1, cas2)

  // try to get back the vals...
  err = cn.Flush(0)
  assert.Equalf(t, nil, err, "flush produced error: %v", err)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)

  err = cn.Del(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)
  err = cn.Del(KEY2)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)

  // do two sets
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  _, err = cn.Set(KEY2, VAL2, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // flush in future!
  err = cn.Flush(3)

  // set a key now, after sending flush in future command. Should this key be
  // included in flush when it applies?
  _, err = cn.Set(KEY3, VAL3, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // keys should still survive as the flush hasn't applied yet.
  time.Sleep(900 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should have found key as flushed in future!: %v", err)
  time.Sleep(100 * time.Millisecond)
  _, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "should have found key as flushed in future!: %v", err)

  // now keys should all be flushed
  time.Sleep(2200 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)
  _, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)
  _, _, _, err = cn.Get(KEY3)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)

  // do two sets
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  _, err = cn.Set(KEY2, VAL2, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // flush in future! (should overwrite old flush in futures...)
  err = cn.Flush(3)
  time.Sleep(900 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should have found key as flushed in future!: %v", err)
  time.Sleep(100 * time.Millisecond)
  _, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "should have found key as flushed in future!: %v", err)
  err = cn.Flush(4)
  time.Sleep(2000 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should have found key as flushed in future!: %v", err)
  _, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "should have found key as flushed in future!: %v", err)
  time.Sleep(2000 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key as flushed: %v", err)
}


// Test flush, flush future.
func TestFlushFuture(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "bar"
    VAL2 = "zar"
  )

  // clear cache
  err := cn.Flush(0)
  assert.Equalf(t, nil, err, "flush produced error: %v", err)

  // set KEY1, KEY2
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  _, err = cn.Set(KEY2, VAL2, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // wait two seconds
  time.Sleep(2000 * time.Millisecond)

  // flush cache (KEY1, KEY2)
  err = cn.Flush(0)
  assert.Equalf(t, nil, err, "flush produced error: %v", err)

  // get KEY1 -> null
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key! err: %v", err)

  // re-set KEY1
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // flush again, but in future
  err = cn.Flush(2)

  // XXX: Memcached is broken for this.
  // get KEY2 -- memcached bug where flush in future can resurrect items
  // _, _, _, err = cn.Get(KEY2)
  // assert.Equalf(t, ErrNotFound, err, "shouldn't have found key! err: %v", err)

  // get KEY1
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should have found key1! err: %v", err)

  // wait for flush to expire
  time.Sleep(2500 * time.Millisecond)

  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't have found key! err: %v", err)
}


// Test the version command works...
func TestVersion(t *testing.T) {
  testInit(t)

  ver, err := cn.Version()
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  good, err := regexp.MatchString("[0-9]+\\.[0-9]+\\.[0-9]+", ver)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, good, true, "version of unexcpected form: %s", ver)
}


// Test the quit command works...
func TestQuit(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "fooz"
    VAL1 = "barz"
  )

  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %s", v)

  err = cn.Quit()
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  _, _, _, err = cn.Get(KEY1)
  assert.NotEqual(t, nil, err, "expected an error (closed connection)")

  err = cn.Quit()
  assert.NotEqual(t, nil, err, "expected an error (closed connection)")

  cn = nil
}

// Test expiration works...
// See Note [Expiration] in mc.go for details of how expiration works.
// NOTE: Can't really test long expirations properly...
func TestExpiration(t *testing.T) {
  testInit(t)

  const (
    KEY0 = "zoo"
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "moo"
    VAL2 = "bar"
  )

  // no expiration, should last forever...
  _, err := cn.Set(KEY0, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  v, _, _, err := cn.Get(KEY0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)


  // 1 second expiration...
  _, err = cn.Set(KEY1, VAL1, 0, 1, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  time.Sleep(1100 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache anymore: %v", err)

  v, _, _, err  = cn.Get(KEY0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)
}

// Test expiration works...
// See Note [Expiration] in mc.go for details of how expiration works.
// NOTE: Can't really test long expirations properly...
func TestExpirationTouch(t *testing.T) {
  if testing.Short() {
    t.Skip("skipping test in short mode.")
  }

  testInit(t)

  const (
    KEY0 = "zoo"
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "moo"
    VAL2 = "bar"
  )

  // no expiration, should last forever...
  _, err := cn.Set(KEY0, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // 2 second expiration...
  _, err = cn.Set(KEY1, VAL2, 0, 2, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  time.Sleep(100 * time.Millisecond)
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 800 total...
  time.Sleep(700 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 900 total...
  time.Sleep(200 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 2000 total...
  time.Sleep(1100 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache anymore: %v", err)

  // Test Touch...
  // NOTE: This works for me with a memcached built from source but not with the
  // one installed via homebrew...
  // 2 second expiration...
  _, err = cn.Set(KEY1, VAL2, 0, 2, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  time.Sleep(100 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 800 total...
  time.Sleep(700 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)

  // make expiration 3 seconds from now (previously would expire 1 second from
  // now, so a 4 second expiration in total...)
  _, err = cn.Touch(KEY1, 3)
  assert.Equalf(t, nil, err, "touch failed: %v", err)
  // 1200 (2000 total)...
  time.Sleep(1200 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 1700 (2500 total)...
  time.Sleep(500 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 1900 (2700 total)...
  time.Sleep(200 * time.Millisecond)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 3500 (4300) total...
  time.Sleep(1600 * time.Millisecond)
  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache anymore: %v", err)

  // key0 still should be alive (no timeout)
  v, _, _, err = cn.Get(KEY0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)
}


// Test Touch command works...
func TestTouch(t *testing.T) {
  if testing.Short() {
    t.Skip("skipping test in short mode.")
  }

  testInit(t)

  const (
    KEY1 = "foo"
    VAL1 = "bar"
  )

  // no expiration, lets see if touch can set an expiration, not just extend...
  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  cn.Touch(KEY1, 2)

  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  time.Sleep(1000 * time.Millisecond)

  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  time.Sleep(1500 * time.Millisecond)

  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache: %v", err)

  // no expiration, let see if we can expire immediately with Touch...
  // NO, 0 = ignore, so the Touch is a noop really...
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  cn.Touch(KEY1, 0)

  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  time.Sleep(1000 * time.Millisecond)

  _, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
}


// Test GAT (get-and-touch) works...
// See Note [Expiration] in mc.go for details of how expiration works.
func TestGAT(t *testing.T) {
  if testing.Short() {
    t.Skip("skipping test in short mode.")
  }

  testInit(t)

  const (
    KEY1 = "foo"
    KEY2 = "goo"
    VAL1 = "moo"
    VAL2 = "bar"
    FLAGS uint32 = 921321
  )

  // no expiration, should last forever...
  _, err := cn.Set(KEY1, VAL1, FLAGS, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  v, f, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  // no expiration...
  _, err = cn.Set(KEY2, VAL2, FLAGS, 0, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)

  // get + set 1 second expiration...
  v, f, _, err = cn.GAT(KEY2, 1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  v, f, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  time.Sleep(1500 * time.Millisecond)

  _, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache anymore: %v", err)
  _, _, _, err = cn.GAT(KEY2, 1)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache anymore: %v", err)

  // Test GAT...
  // 2 second expiration...
  _, err = cn.Set(KEY2, VAL2, FLAGS, 2, 0)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  time.Sleep(100 * time.Millisecond)
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  // 800 total...
  time.Sleep(700 * time.Millisecond)
  v, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, nil, err, "should be in cache still: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)

  // make expiration 2 seconds from now (previously would expire 1 second from
  // now, so a 3 second expiration in total...)
  v, f, _, err = cn.GAT(KEY2, 2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  // 900...
  time.Sleep(900 * time.Millisecond)

  // reset ttl...
  v, f, _, err = cn.GAT(KEY2, 2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  // 900...
  time.Sleep(900 * time.Millisecond)

  // reset ttl...
  v, f, _, err = cn.GAT(KEY2, 2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  // 900...
  time.Sleep(800 * time.Millisecond)

  // reset ttl...
  v, f, _, err = cn.GAT(KEY2, 2)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL2, v, "wrong value: %v", v)
  assert.Equalf(t, FLAGS, f, "wrong flags: %v", f)

  // 2000...
  time.Sleep(2000 * time.Millisecond)

  _, _, _, err = cn.Get(KEY2)
  assert.Equalf(t, ErrNotFound, err, "shouldn't be in cache anymore: %v", err)

  // should be alive still (no expiration on this key)
  v, _, _, err = cn.Get(KEY1)
  assert.Equalf(t, nil, err, "shouldn't be an error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %v", v)
}


// Some basic tests that functions work
func testThread(t *testing.T, id int, ch chan bool) {
  const (
    KEY1 = "foo"
    VAL1 = "boo"
    KEY3 = "bar"
  )

  idx := strconv.Itoa(id)
  key2 := KEY1 + idx

  // lots of sets of this but should all be setting it to boo...
  _, err := cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // should be unique to a thread...
  cas2, err := cn.Set(key2, idx, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // contention but all setting same value...
  v, _, _, err := cn.Get(KEY1)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, VAL1, v, "wrong value: %s", v)

  // key is unique to thread, so even CAS shouldn't change...
  v, _, cas2x, err := cn.Get(key2)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.Equalf(t, idx, v, "wrong value: %s", v)
  assert.Equalf(t, cas2, cas2x, "CAS shouldn't have changed: %d, %d", cas2, cas2x)

  // lots of sets of this and with diff values...
  cas1, err := cn.Set(KEY3, idx, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // try getting straight away...
  v, _, cas1x, err := cn.Get(KEY3)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  // if cas didn't change our value should have been returned...
  if cas1 == cas1x {
    assert.Equalf(t, idx, v, "wrong value (cas didn't change): %s", v)
  }

  ch <- true
}

// Test threaded interaction...
func TestThreaded(t *testing.T) {
  testInit(t)

  ch := make(chan bool)

  for i := 0; i < 30; i++ {
    go testThread(t, i, ch)
  }

  for i := 0; i < 30; i++ {
    _ = <- ch
  }
}

func testAdvGet(t *testing.T, op opCode, key string, expKey string, opq uint32) *msg {
  var flags  uint32

  m := &msg{
    header: header{
      Op:  op,
      CAS: uint64(0),
      Opaque: uint32(opq),
    },
    oextras: []interface{}{&flags},
    key: key,
  }

  err := cn.sendRecv(m)

  assert.Equalf(t, nil, err, "Unexpected error! %s", err)
  // XXX: Issues here with new server send/recv split! Seems a golang bug to do
  // with lifting variables to heap perhaps and sharing?
  // assert.Equalf(t, op, m.header.Op, "Response has wrong op code! %d != %d", op, m.header.Op)
  // assert.Equalf(t, opq, m.header.Opaque, "Response has wrong opaque! %d != %d", opq, m.header.Opaque)
  // assert.Equalf(t, expKey, m.key, "Get returned key! %s", m.key)

  return m
}

// Test that the various get types work and that Opaque works... e.g., all the
// components needed for multi_get.
func TestGetExotic(t *testing.T) {
  const (
    KEY = "key"
    VAL = "bar"
  )

  testInit(t)

  _, err := cn.Set(KEY, VAL, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // TODO: Testing only when a key exists, need to also test functionality that
  // on key miss, getq doesn't return a response.

  // get
  testAdvGet(t, OpGet, KEY, "", 123)
  testAdvGet(t, OpGet, KEY, "", 0)
  testAdvGet(t, OpGet, KEY, "", 0xffffffff)
  testAdvGet(t, OpGet, KEY, "", 0xfffffff0)
  testAdvGet(t, OpGet, KEY, "", 0xf0f0f0f0)

  // getq
  testAdvGet(t, OpGetQ, KEY, "", 123)
  testAdvGet(t, OpGetQ, KEY, "", 0)
  testAdvGet(t, OpGetQ, KEY, "", 0xffffffff)
  testAdvGet(t, OpGetQ, KEY, "", 0xfffffff0)
  testAdvGet(t, OpGetQ, KEY, "", 0xf0f0f0f0)

  // getk
  testAdvGet(t, OpGetK, KEY, KEY, 123)
  testAdvGet(t, OpGetK, KEY, KEY, 0)
  testAdvGet(t, OpGetK, KEY, KEY, 0xffffffff)
  testAdvGet(t, OpGetK, KEY, KEY, 0xfffffff0)
  testAdvGet(t, OpGetK, KEY, KEY, 0xf0f0f0f0)

  // getkq
  testAdvGet(t, OpGetKQ, KEY, KEY, 123)
  testAdvGet(t, OpGetKQ, KEY, KEY, 0)
  testAdvGet(t, OpGetKQ, KEY, KEY, 0xffffffff)
  testAdvGet(t, OpGetKQ, KEY, KEY, 0xfffffff0)
  testAdvGet(t, OpGetKQ, KEY, KEY, 0xf0f0f0f0)
}

func testAdvGat(t *testing.T, op opCode, key string, expKey string, opq uint32) *msg {
  var exp uint32
  var flags uint32

  m := &msg{
    header: header{
      Op:  op,
      CAS: uint64(0),
      Opaque: uint32(opq),
    },
    iextras: []interface{}{exp},
    oextras: []interface{}{&flags},
    key: key,
  }

  err := cn.sendRecv(m)

  assert.Equalf(t, nil, err, "Unexpected error! %s", err)
  // XXX: Issues here with new server send/recv split! Seems a golang bug to do
  // with lifting variables to heap perhaps and sharing?
  // assert.Equalf(t, op, m.header.Op, "Response has wrong op code! %d != %d", op, m.header.Op)
  // assert.Equalf(t, opq, m.header.Opaque, "Response has wrong opaque! %d != %d", opq, m.header.Opaque)
  // assert.Equalf(t, expKey, m.key, "Get returned key! %s", m.key)

  return m
}

// Test that the various gat types work
func TestGatExotic(t *testing.T) {
  const (
    KEY = "key"
    VAL = "bar"
  )

  testInit(t)

  _, err := cn.Set(KEY, VAL, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  // TODO: Testing only when a key exists, need to also test functionality that
  // on key miss, getq doesn't return a response. And test that the 'touch'
  // aspect is functioning.

  // get
  testAdvGat(t, OpGAT, KEY, "", 123)
  testAdvGat(t, OpGAT, KEY, "", 0)
  testAdvGat(t, OpGAT, KEY, "", 0xffffffff)
  testAdvGat(t, OpGAT, KEY, "", 0xfffffff0)
  testAdvGat(t, OpGAT, KEY, "", 0xf0f0f0f0)

  // getq
  testAdvGat(t, OpGATQ, KEY, "", 123)
  testAdvGat(t, OpGATQ, KEY, "", 0)
  testAdvGat(t, OpGATQ, KEY, "", 0xffffffff)
  testAdvGat(t, OpGATQ, KEY, "", 0xfffffff0)
  testAdvGat(t, OpGATQ, KEY, "", 0xf0f0f0f0)

  // getk
  testAdvGat(t, OpGATK, KEY, KEY, 123)
  testAdvGat(t, OpGATK, KEY, KEY, 0)
  testAdvGat(t, OpGATK, KEY, KEY, 0xffffffff)
  testAdvGat(t, OpGATK, KEY, KEY, 0xfffffff0)
  testAdvGat(t, OpGATK, KEY, KEY, 0xf0f0f0f0)

  // getkq
  testAdvGat(t, OpGATKQ, KEY, KEY, 123)
  testAdvGat(t, OpGATKQ, KEY, KEY, 0)
  testAdvGat(t, OpGATKQ, KEY, KEY, 0xffffffff)
  testAdvGat(t, OpGATKQ, KEY, KEY, 0xfffffff0)
  testAdvGat(t, OpGATKQ, KEY, KEY, 0xf0f0f0f0)
}

func TestGetStats(t *testing.T) {
  testInit(t)

  const (
    KEY1 = "exists"
    VAL1 = "bar"
    KEY2 = "noexists"

    GET_HITS = 12348
    GET_MISSES = 1993
  )

  // wait for other tests to finish...
  time.Sleep(2000 * time.Millisecond)

  // clear cache and get starting point.
  cn.Flush(0)
  stats, err := cn.Stats()
  assert.Equalf(t, nil, err, "unexpected error: %v", err)
  assert.T(t, len(stats) > 0, "stats is empty! ", stats)
  startMisses, err := strconv.ParseUint(stats["get_misses"], 10, 64)
  assert.Equalf(t, nil, err, "unexpected error: %v, stats struct: %v",
    err, stats)
  startHits, err := strconv.ParseUint(stats["get_hits"], 10, 64)
  assert.Equalf(t, nil, err, "unexpected error: %v, stats struct: %v",
    err, stats)

  // setup key
  _, err = cn.Set(KEY1, VAL1, 0, 0, 0)
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  c := make(chan bool)

  // run get hit thread
  go func() {
    for i := 0; i < GET_HITS; i++ {
      _, _, _, err := cn.Get(KEY1)
      assert.Equalf(t, nil, err, "unexpected error: %v", err)
    }
    c <- true
  }()

  // run get miss thread
  go func() {
    for i := 0; i < GET_MISSES; i++ {
      _, _, _, err := cn.Get(KEY2)
      assert.Equalf(t, ErrNotFound, err, "expected 'not found' error: %v", err)
    }
    c <- true
  }()

  // wait on both threads
  _ = <-c
  _ = <-c
  stats, err = cn.Stats()
  assert.Equalf(t, nil, err, "unexpected error: %v", err)

  getMisses := strconv.FormatUint(GET_MISSES + startMisses, 10)
  if stats["get_misses"] != getMisses {
    t.Errorf("get_misses (%s) != expected (%s)\n", stats["get_misses"], getMisses)
  }

  getHits := strconv.FormatUint(GET_HITS + startHits, 10)
  if stats["get_hits"] != getHits {
    t.Errorf("get_hits (%s) != expected (%s)\n", stats["get_hits"], getHits)
  }
}

