package mc

import (
	"bytes"
  "fmt"
	"github.com/bmizerany/assert"
	"testing"
	"net"
	"runtime"
)

const mcAddr = "localhost:11211"

func TestMCSimple(t *testing.T) {
	nc, err := net.Dial("tcp", mcAddr)
	assert.Equalf(t, nil, err, "%v", err)

	cn := &Conn{rwc: nc, buf: new(bytes.Buffer)}

	if runtime.GOOS != "xxx-darwin" {
		println("Not on Darwin, testing auth")
		err = cn.Auth("user-1", "pass")
		assert.Equalf(t, nil, err, "%v", err)
	}

	err = cn.Del("foo")
	if err != ErrNotFound {
		assert.Equalf(t, nil, err, "%v", err)
	}

	_, _, _, err = cn.Get("foo")
	assert.Equalf(t, ErrNotFound, err, "%v", err)

	err = cn.Set("foo", "bar", 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)

	// unconditional SET
	err = cn.Set("foo", "bar", 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)

	err = cn.Set("foo", "bar", 1928, 0, 0)
	assert.Equalf(t, ErrKeyExists, err, "%v", err)

	v, _, _, err := cn.Get("foo")
	assert.Equalf(t, nil, err, "%v", err)
	assert.Equal(t, "bar", v)

	err = cn.Del("n")
	if err != ErrNotFound {
		assert.Equalf(t, nil, err, "%v", err)
	}

	n, cas, err := cn.Incr("n", 1, 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, 1, n)

	n, cas, err = cn.Incr("n", 1, 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, 2, n)

	n, cas, err = cn.Decr("n", 1, 0, 0, 0)
	assert.Equalf(t, nil, err, "%v", err)
	assert.NotEqual(t, 0, cas)
	assert.Equal(t, 1, n)
}

func TestEdges(t *testing.T) {
	nc, err := net.Dial("tcp", mcAddr)
	assert.Equalf(t, nil, err, "%v", err)

	cn := &Conn{rwc: nc, buf: new(bytes.Buffer)}

	if runtime.GOOS != "xxx-darwin" {
		println("Not on Darwin, testing auth")
		err = cn.Auth("user-1", "pass")
		assert.Equalf(t, nil, err, "%v", err)
	}

  // fmt.Println("generally see how CAS values behave...")

  // general

  err = cn.Set("foo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  _, cas1, _, err := cn.Get("foo")
  // fmt.Println("CAS (foo) =", cas1)

  err = cn.Set("goo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (goo):", err)
  }
  _, _, _, err = cn.Get("goo")
  // fmt.Println("CAS (goo) =", cas2)

  err = cn.Set("foo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  _, cas1, _, err = cn.Get("foo")
  // fmt.Println("CAS (foo) =", cas1)

  err = cn.Set("goo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (goo):", err)
  }
  _, _, _, err = cn.Get("goo")
  // fmt.Println("CAS (goo) =", cas2)

  // set

  // fmt.Println("\nwhat happens when I set a new value and give a CAS?...")

  // should fail
  cn.Del("hoo")
  err = cn.Set("hoo", "bar", 100, 0, 0)
  if err == nil {
    t.Errorf("no error (set hoo):", err)
  }
  v, _, _, err := cn.Get("hoo")
  if err == nil {
    t.Errorf("no error (get hoo):", err)
  }
  // fmt.Println("CAS (goo) =", cas2)

  // fmt.Println("\nsetting an existing value with bad CAS...")

  // should fail
  err = cn.Set("foo", "bar", 9090, 0, 0)
  if err == nil {
    t.Errorf("error (foo):", err)
  }
  _, cas1, _, err = cn.Get("foo")
  // fmt.Println("CAS (foo) =", cas1)

  // get

  // fmt.Println("\nretrieve value with 0 CAS...")
  v, cas1, _, err = cn.GetCAS("foo", 0)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  // fmt.Printf("CAS (foo) =", cas1, "value =", v)

  // fmt.Println("\nretrieve value with good CAS...")
  v, cas1, _, err = cn.GetCAS("foo", cas1)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  // fmt.Println("CAS (foo) =", cas1, "value =", v)

  // fmt.Println("\nretrieve value with bad CAS...")
  v, cas1, _, err = cn.GetCAS("foo", cas1 + 1)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  // fmt.Println("CAS (foo) =", cas1, "value =", v)

  // delete

  // fmt.Println("\ndelete existing key...")
  err = cn.Set("foo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (set foo):", err)
  }
  err = cn.Del("foo")
  if err != nil {
    t.Errorf("error (del foo):", err)
  }

  // fmt.Println("\ndelete non-existent key...")
  err = cn.Del("foo")
  if err == nil {
    t.Errorf("no error (del foo):", err)
  }

  // fmt.Println("\ndelete existing key with 0 CAS...")
  err = cn.Set("foo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (set foo):", err)
  }
  err = cn.DelCAS("foo", 0)
  if err != nil {
    t.Errorf("error (del foo):", err)
  }

  // fmt.Println("\ndelete existing key with good CAS...")
  err = cn.Set("foo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (set foo):", err)
  }
  _, cas1, _, err = cn.GetCAS("foo", cas1)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  err = cn.DelCAS("foo", cas1)
  if err != nil {
    t.Errorf("error (del foo):", err)
  }

  // fmt.Println("\ndelete existing key with bad CAS...")
  err = cn.Set("foo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (set foo):", err)
  }
  _, cas1, _, err = cn.GetCAS("foo", cas1)
  if err != nil {
    t.Errorf("error (foo):", err)
  }
  err = cn.DelCAS("foo", cas1 + 10)
  if err == nil {
    t.Errorf("no error (del foo):", err)
  }
  v, cas1, _, err = cn.Get("foo")
  if err != nil {
    t.Errorf("error (foo = %s): %v", v, err)
  } else {
    // fmt.Println("foo =", v)
  }


  // add

  // fmt.Println("\nTesting add...")

  cn.Del("igo")
  err = cn.Add("igo", "bar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (add igo):", err)
  }
  v, cas2, _, err := cn.Get("igo")
  if err != nil {
    t.Errorf("error (get igo):", err)
  } else if v != "bar" {
    t.Errorf("error (value igo != bar):", v)
  }
  // fmt.Println("CAS (igo) =", cas2)

  // fmt.Println("\nwhat happens when I add a new value and give a CAS?...")

  cn.Del("joo")
  err = cn.Add("joo", "bar", 100, 0, 0)
  if err == nil {
    t.Errorf("no error (add joo):", err)

    v, cas2, _, err = cn.Get("joo")
    if err != nil {
      t.Errorf("error (get joo):", err)
    } else if v != "bar" {
      t.Errorf("error (value joo != bar):", v)
    }
  } else {
    // fmt.Println("CAS (joo) =", cas2)
  }

  // fmt.Println("\nadd an existing value (should fail)...")

  cn.Add("joo", "bar", 0, 0, 0)
  err = cn.Add("joo", "bar", 0, 0, 0)
  if err == nil {
    t.Errorf("no error (add joo):", err)
  }
  _, cas1, _, err = cn.Get("joo")
  // fmt.Println("CAS (joo) =", cas1)


  // replace

  // fmt.Println("\nTesting replace...")

  cn.Set("loo", "bar", 0, 0, 0)
  err = cn.Rep("loo", "zar", 0, 0, 0)
  if err != nil {
    t.Errorf("error (rep loo):", err)
  }
  v, cas2, _, err = cn.Get("loo")
  if err != nil {
    t.Errorf("error (get loo):", err)
  } else if v != "zar" {
    t.Errorf("error (value loo != zar):", v)
  }
  // fmt.Println("CAS (loo) =", cas2)

  // fmt.Println("\nwhat happens when I replace a value and give a good CAS?...")

  err = cn.Rep("loo", "zar", cas2, 0, 0)
  if err != nil {
    t.Errorf("error (rep loo):", err)
  }
  v, cas2, _, err = cn.Get("loo")
  if err != nil {
    t.Errorf("error (get loo):", err)
  } else if v != "zar" {
    t.Errorf("error (value loo != zar):", v)
  }
  // fmt.Println("CAS (loo) =", cas2)

  // fmt.Println("\nwhat happens when I replace a value and give a bad CAS?...")

  err = cn.Rep("loo", "bar", 100, 0, 0)
  if err == nil {
    t.Errorf("no error (rep loo):", err)
    v, cas2, _, err = cn.Get("loo")
    if err != nil {
      t.Errorf("error (get loo):", err)
    } else if v != "bar" {
      t.Errorf("error (value loo != bar):", v)
    }
  } else {
    // fmt.Println("CAS (loo) =", cas2)
  }

  // fmt.Println("\nreplace an missing key (should fail)...")

  cn.Del("loo")
  err = cn.Rep("loo", "zar", 0, 0, 0)
  if err == nil {
    t.Errorf("no error (rep loo):", err)
  }


  // test incr/decr

  // fmt.Println("\ntesting incr/decr")

  cn.Del("moo")
  n, cas1, err := cn.Incr("moo", 10, 10, 0, 0)
  if err != nil {
    t.Errorf("error (incr moo):", err)
  } else {
    fmt.Println("should be 10, is", n, "(CAS: ", cas1, ")")
  }

  n, cas1, err = cn.Incr("moo", 2, 90, 0, 0)
  if err != nil {
    t.Errorf("error (incr moo):", err)
  } else {
    fmt.Println("should be 12, is", n, "(CAS: ", cas1, ")")
  }

  // screw up CAS a little
  cn.Set("a", "bb", 0, 0, 0)
  cn.Set("a", "bb", 0, 0, 0)
  cn.Set("a", "bb", 0, 0, 0)
  cn.Set("a", "bb", 0, 0, 0)

  n, cas1, err = cn.Decr("moo", 9, 90, 0, 0)
  if err != nil || n != 3 {
    t.Errorf("error (decr moo):", err)
  } else {
    fmt.Println("should be 3, is", n, "(CAS: ", cas1, ")")
  }

  n, cas1, err = cn.Decr("moo", 9, 90, 0, 0)
  if err != nil || n != 0 {
    t.Errorf("error (decr moo):", err)
  } else {
    fmt.Println("should be 0, is", n, "(CAS: ", cas1, ")")
  }

  n, cas1, err = cn.Incr("moo", 7, 90, 0, 0)
  if err != nil || n != 7 {
    t.Errorf("error (incr moo):", err)
  } else {
    fmt.Println("should be 7, is", n, "(CAS: ", cas1, ")")
  }

  n, cas1, err = cn.Incr("moo", 7, 90, 0, cas1)
  if err != nil || n != 14 {
    t.Errorf("error (incr moo):", err)
  } else {
    fmt.Println("should be 14, is", n, "(CAS: ", cas1, ")")
  }

  n, cas1, err = cn.Incr("moo", 7, 90, 0, cas1 + 99)
  if err == nil {
    t.Errorf("should have failed (bad CAS): error (incr moo):", err)
  } else {
    fmt.Println("should be 0, is", n, "(CAS: ", cas1, ")")
  }
}

