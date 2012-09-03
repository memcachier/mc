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
  assert.Equalf(t, nil, err, "%v", err)
  return true
}


// Some basic tests that functions work
func TestMCSimple(t *testing.T) {
  testInit(t)

  err := cn.Del("foo")
	if err != ErrNotFound {
		assert.Equalf(t, nil, err, "%v", err)
	}

	_, _, _, err = cn.Get("foo")
	assert.Equalf(t, ErrNotFound, err, "%v", err)

	// unconditional SET
	_, err = cn.Set("foo", "bar", 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
  cas, err := cn.Set("foo", "bar", 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)

  // make sure CAS works
	_, err = cn.Set("foo", "bar-bad", cas + 999, 0, 0)
	assert.Equalf(t, ErrKeyExists, err, "%v", err)

  // check SET actually set the correct value...
	v, _, _, err := cn.Get("foo")
	assert.Equalf(t, nil, err, "%v", err)
	assert.Equal(t, "bar", v)

  // use correct CAS...
  _, err = cn.Set("foo", "bar-good", cas, 0, 0)
  assert.Equalf(t, nil, err, "%v", err)

  // check DEL of non-existing key fails...
	err = cn.Del("n")
	if err != ErrNotFound {
		assert.Equalf(t, nil, err, "%v", err)
	}
	err = cn.Del("n")
  assert.Equalf(t, ErrNotFound, err, "%v", err)

  // test INCR/DECR...
	n, cas, err := cn.Incr("n", 1, 10, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, uint64(10), n)

	n, cas, err = cn.Incr("n", 1, 99, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, uint64(11), n)

	n, cas, err = cn.Decr("n", 1, 97, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, uint64(10), n)

  // test CAS works... (should match)
  n, cas, err = cn.Decr("n", 1, 93, 0, cas)
  assert.Equal(t, nil, err, "%v", err)
  assert.NotEqual(t, 0, cas)
  assert.Equal(t, uint64(9), n)

  // test CAS works... (should fail, doesn't match)
  n, cas, err = cn.Decr("n", 1, 87, 0, cas + 97)
  assert.Equal(t, ErrKeyExists, err, "%v", err)
  assert.Equal(t, uint64(0), n, "%d", n)
  assert.Equal(t, uint64(0), cas, "%d", cas)
}


// Test expiration works...
func TestIncrTimeouts(t *testing.T) {
  testInit(t)

  cn.Del("n")

  // Incr (key, delta, initial, ttl, cas)
  n, _, err := cn.Incr("n", 1, 9, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.Equal(t, uint64(9), n)

  time.Sleep(2000 * time.Millisecond)

	n, _, err = cn.Incr("n", 1, 3, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.Equal(t, uint64(10), n)
}


// Testing MAX SIZE of values...
// Testing if when you set a key/value with a bad value (e.g > 1MB) does that
// remove the existing key/value still or leave it intact?
func TestSetBadRemovePrevious(t *testing.T) {
  // XXX: larger than this memcached doesn't like for key 'foo'
  const MAX_VAL_SIZE = 1024 * 1024 - 74
  const KEY = "foo"

  testInit(t)

  // check basic get/set works first
  val := "bar"
  _, err := cn.Set(KEY, val, 0, 0, 0)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  v, _, _, err := cn.Get(KEY)
  if v != val {
    t.Errorf("error [GET] (foo) not equal", val, ":", err)
  }

  // MAX GOOD VALUE

  // generate random bytes
  data := make([]byte, MAX_VAL_SIZE)
  for i := 0; i < MAX_VAL_SIZE; i++ {
    data[i] = byte(rand.Int())
  }

  val = string(data)
  _, err = cn.Set(KEY, val, 0, 0, 0)
  if err != nil {
    t.Errorf("error (foo): %s\n", err)
  }
  v, _, _, err = cn.Get(KEY)
  if v != val {
    t.Errorf("error [GET] (foo) not equal: %s\n", err)
  }

  // MAX GOOD VALUE * 2

  // generate random bytes
  data = make([]byte, 2 * MAX_VAL_SIZE)
  for i := 0; i < 2 * MAX_VAL_SIZE; i++ {
    data[i] = byte(rand.Int())
  }

  val2 := string(data)
  _, err = cn.Set(KEY, val2, 0, 0, 0)
  if err == nil {
    t.Errorf("expecting error for value too large!")
  }
  v, _, _, err = cn.Get(KEY)
  if err == nil {
    fmt.Println("\tmemcached removes the old value... so expecting no key")
    fmt.Println("\tnot an error but just a different semantics than memcached")
  }
}

// Test some edge cases of memcached. This was originally done to better
// understand the protocol but servers as a good test for the client and
// server...


// Test SET behaviour with CAS...
func TestSetEdges(t *testing.T) {
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


// Test GET, does it care about CAS?
// NOTE: No it shouldn't, memcached mainline doesn't...
func TestGetEdges(t *testing.T) {
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


func TestEdges(t *testing.T) {
  testInit(t)

  // DELETE

  // fmt.Println("\ndelete existing key...")
  // _, err = cn.Set("foo", "bar", 0, 0, 0)
  // if err != nil {
  //   t.Errorf("error (set foo):", err)
  // }
  // err = cn.Del("foo")
  // if err != nil {
  //   t.Errorf("error (del foo):", err)
  // }

  // // fmt.Println("\ndelete non-existent key...")
  // err = cn.Del("foo")
  // if err == nil {
  //   t.Errorf("no error (del foo):", err)
  // }

  // // fmt.Println("\ndelete existing key with 0 CAS...")
  // err = cn.Set("foo", "bar", 0, 0, 0)
  // if err != nil {
  //   t.Errorf("error (set foo):", err)
  // }
  // err = cn.DelCAS("foo", 0)
  // if err != nil {
  //   t.Errorf("error (del foo):", err)
  // }

  // // fmt.Println("\ndelete existing key with good CAS...")
  // err = cn.Set("foo", "bar", 0, 0, 0)
  // if err != nil {
  //   t.Errorf("error (set foo):", err)
  // }
  // _, cas1, _, err = cn.GetCAS("foo", cas1)
  // if err != nil {
  //   t.Errorf("error (foo):", err)
  // }
  // err = cn.DelCAS("foo", cas1)
  // if err != nil {
  //   t.Errorf("error (del foo):", err)
  // }

  // // fmt.Println("\ndelete existing key with bad CAS...")
  // err = cn.Set("foo", "bar", 0, 0, 0)
  // if err != nil {
  //   t.Errorf("error (set foo):", err)
  // }
  // _, cas1, _, err = cn.GetCAS("foo", cas1)
  // if err != nil {
  //   t.Errorf("error (foo):", err)
  // }
  // err = cn.DelCAS("foo", cas1 + 10)
  // if err == nil {
  //   t.Errorf("no error (del foo):", err)
  // }
  // v, cas1, _, err = cn.Get("foo")
  // if err != nil {
  //   t.Errorf("error (foo = %s): %v", v, err)
  // } else {
  //   // fmt.Println("foo =", v)
  // }


  // // add

  // // fmt.Println("\nTesting add...")

  // cn.Del("igo")
  // err = cn.Add("igo", "bar", 0, 0, 0)
  // if err != nil {
  //   t.Errorf("error (add igo):", err)
  // }
  // v, cas2, _, err := cn.Get("igo")
  // if err != nil {
  //   t.Errorf("error (get igo):", err)
  // } else if v != "bar" {
  //   t.Errorf("error (value igo != bar):", v)
  // }
  // // fmt.Println("CAS (igo) =", cas2)

  // // fmt.Println("\nwhat happens when I add a new value and give a CAS?...")

  // cn.Del("joo")
  // err = cn.Add("joo", "bar", 100, 0, 0)
  // if err == nil {
  //   t.Errorf("no error (add joo):", err)

  //   v, cas2, _, err = cn.Get("joo")
  //   if err != nil {
  //     t.Errorf("error (get joo):", err)
  //   } else if v != "bar" {
  //     t.Errorf("error (value joo != bar):", v)
  //   }
  // } else {
  //   // fmt.Println("CAS (joo) =", cas2)
  // }

  // // fmt.Println("\nadd an existing value (should fail)...")

  // cn.Add("joo", "bar", 0, 0, 0)
  // err = cn.Add("joo", "bar", 0, 0, 0)
  // if err == nil {
  //   t.Errorf("no error (add joo):", err)
  // }
  // _, cas1, _, err = cn.Get("joo")
  // // fmt.Println("CAS (joo) =", cas1)


  // // replace

  // // fmt.Println("\nTesting replace...")

  // cn.Set("loo", "bar", 0, 0, 0)
  // err = cn.Rep("loo", "zar", 0, 0, 0)
  // if err != nil {
  //   t.Errorf("error (rep loo):", err)
  // }
  // v, cas2, _, err = cn.Get("loo")
  // if err != nil {
  //   t.Errorf("error (get loo):", err)
  // } else if v != "zar" {
  //   t.Errorf("error (value loo != zar):", v)
  // }
  // // fmt.Println("CAS (loo) =", cas2)

  // // fmt.Println("\nwhat happens when I replace a value and give a good CAS?...")

  // err = cn.Rep("loo", "zar", cas2, 0, 0)
  // if err != nil {
  //   t.Errorf("error (rep loo):", err)
  // }
  // v, cas2, _, err = cn.Get("loo")
  // if err != nil {
  //   t.Errorf("error (get loo):", err)
  // } else if v != "zar" {
  //   t.Errorf("error (value loo != zar):", v)
  // }
  // // fmt.Println("CAS (loo) =", cas2)

  // // fmt.Println("\nwhat happens when I replace a value and give a bad CAS?...")

  // err = cn.Rep("loo", "bar", 100, 0, 0)
  // if err == nil {
  //   t.Errorf("no error (rep loo):", err)
  //   v, cas2, _, err = cn.Get("loo")
  //   if err != nil {
  //     t.Errorf("error (get loo):", err)
  //   } else if v != "bar" {
  //     t.Errorf("error (value loo != bar):", v)
  //   }
  // } else {
  //   // fmt.Println("CAS (loo) =", cas2)
  // }

  // // fmt.Println("\nreplace an missing key (should fail)...")

  // cn.Del("loo")
  // err = cn.Rep("loo", "zar", 0, 0, 0)
  // if err == nil {
  //   t.Errorf("no error (rep loo):", err)
  // }


  // // test incr/decr

  // // fmt.Println("\ntesting incr/decr")

  // cn.Del("moo")
  // n, cas1, err := cn.Incr("moo", 10, 10, 0, 0)
  // if err != nil {
  //   t.Errorf("error (incr moo):", err)
  // } else {
  //   fmt.Println("should be 10, is", n, "(CAS: ", cas1, ")")
  // }

  // n, cas1, err = cn.Incr("moo", 2, 90, 0, 0)
  // if err != nil {
  //   t.Errorf("error (incr moo):", err)
  // } else {
  //   fmt.Println("should be 12, is", n, "(CAS: ", cas1, ")")
  // }

  // // screw up CAS a little
  // cn.Set("a", "bb", 0, 0, 0)
  // cn.Set("a", "bb", 0, 0, 0)
  // cn.Set("a", "bb", 0, 0, 0)
  // cn.Set("a", "bb", 0, 0, 0)

  // n, cas1, err = cn.Decr("moo", 9, 90, 0, 0)
  // if err != nil || n != 3 {
  //   t.Errorf("error (decr moo):", err)
  // } else {
  //   fmt.Println("should be 3, is", n, "(CAS: ", cas1, ")")
  // }

  // n, cas1, err = cn.Decr("moo", 9, 90, 0, 0)
  // if err != nil || n != 0 {
  //   t.Errorf("error (decr moo):", err)
  // } else {
  //   fmt.Println("should be 0, is", n, "(CAS: ", cas1, ")")
  // }

  // n, cas1, err = cn.Incr("moo", 7, 90, 0, 0)
  // if err != nil || n != 7 {
  //   t.Errorf("error (incr moo):", err)
  // } else {
  //   fmt.Println("should be 7, is", n, "(CAS: ", cas1, ")")
  // }

  // n, cas1, err = cn.Incr("moo", 7, 90, 0, cas1)
  // if err != nil || n != 14 {
  //   t.Errorf("error (incr moo):", err)
  // } else {
  //   fmt.Println("should be 14, is", n, "(CAS: ", cas1, ")")
  // }

  // n, cas1, err = cn.Incr("moo", 7, 90, 0, cas1 + 99)
  // if err == nil {
  //   t.Errorf("should have failed (bad CAS): error (incr moo):", err)
  // } else {
  //   fmt.Println("should be 0, is", n, "(CAS: ", cas1, ")")
  // }
}

