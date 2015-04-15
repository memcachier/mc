// Memcache client for Go.
package mc

import (
  "fmt"
)

// Protocol:
// Contains the actual memcache commands a user cares about.
// We document the protocol a little with each command, you can find the
// official documentation at:
// * https://github.com/memcached/memcached/blob/master/doc/protocol-binary.xml
// * https://github.com/memcached/memcached/blob/master/doc/protocol.txt
// * http://code.google.com/p/memcached/wiki/SASLAuthProtocol
// * http://tools.ietf.org/html/rfc4422 (SASL)
// However, sadly none of these are 100% accurate and you have to look at the
// memcached source code to find any missing cases or mismatches.
//
// * Protocol uses standard network order for bytes (big-endian)

// Command Variants:
// One quick note on memcache commands, many of them support the following
// variants [R, Q, K, KQ], e.g., GET can be GETK or GETQ...
// * <cmd>K : Include the KEY in the response.
// * <cmd>Q : Quiet version of a command. This means if the key doesn't exist,
//            no response is sent.
// * R<cmd> : Ranged version of the command. Not actually implemented by
//            memcached, just there for future extension if needed. So we
//            ignore in this client.

// Multi-Get:
// Simply implemented using GETQ. It's used for 'pipelining' requests, where the
// client sends many GETQ's without checking the response until the very end
// (batching). The memcached server doesn't do anything special, it sends
// response straight away, so it relies on the clients socket queuing up the
// responses on its buffer.

// Response:
// In addition to the key, value & extras we always get back in a response the
// status, CAS and opaque (although a user of a memcache client probably never
// cares about opaque, only we, the implementer of a memcache client, may care
// as it can be used for matching request with responses...)

// Expiration.
// * In seconds - when value of int is less than or equal
//   to: 60 * 60 * 24 * 30 (e.g., seconds in a month).
// * As a unix timestamp - otherwise.
//
// * Memcached accounts for time passing with a single global counter updated
//   once a second, so it therefore has an error margin of 1 second (as you
//   may do a set with expiration 0.5 seconds before it does a global time
//   update, and that 0.5 seconds will expire your key by 1 whole second).
// * Error margin is always under time, not over. E.g., a expiration of 4
//   seconds will actually expire somewhere in the range of (3,4) seconds.

// Retrieve a value from the cache.
func (cn *Conn) Get(key string) (val string, flags uint32, cas uint64, err *MCError) {
  // Variants: [R] Get [Q, K, KQ]
  // Request : MUST key; MUST NOT value, extras
  // Response: MAY key, value, extras ([0..3] flags)
  return cn.getCAS(key, 0)
}

// Retrieve a value in the cache but only if the CAS specified matches the CAS
// argument.
//
// NOTE: GET doesn't actually care about CAS, but we want this internally for
// testing purposes, to be able to test that a memcache server obeys the proper
// semantics of ignoring CAS with GETs.
func (cn *Conn) getCAS(key string, ocas uint64) (val string, flags uint32, cas uint64, err *MCError) {
  m := &msg{
    header: header{
      Op:  OpGet,
      CAS: uint64(ocas),
    },
    oextras: []interface{}{&flags},
    key: key,
  }

  err = cn.sendRecv(m)
  return m.val, flags, m.CAS, err
}

// Get and Touch. Both get the value associated with the key and update its
// expiration time.
func (cn *Conn) GAT(key string, exp uint32) (val string, flags uint32, cas uint64, err *MCError) {
  // Variants: GAT [Q, K, KQ]
  // Request : MUST key, extras; MUST NOT value
  // Response: MAY key, value, extras ([0..3] flags)
  m := &msg{
    header: header{
      Op: OpGAT,
    },
    iextras: []interface{}{exp},
    oextras: []interface{}{&flags},
    key: key,
  }

  err = cn.sendRecv(m)
  return m.val, flags, m.CAS, err
}

func (cn *Conn) Touch(key string, exp uint32) (cas uint64, err *MCError) {
  // Variants: Touch
  // Request : MUST key, extras; MUST NOT value
  // Response: MUST NOT key, value, extras
  m := &msg{
    header: header{
      Op: OpTouch,
    },
    iextras: []interface{}{exp},
    key: key,
  }

  err = cn.sendRecv(m)
  return m.CAS, err
}

// Set a key/value pair in the cache.
func (cn *Conn) Set(key, val string, flags, exp uint32, ocas uint64) (cas uint64, err *MCError){
  // Variants: [R] Set [Q]
  return cn.setGeneric(OpSet, key, val, ocas, flags, exp)
}

// Replace an existing key/value in the cache. Fails if key doesn't already
// exist in cache.
func (cn *Conn) Replace(key, val string, flags, exp uint32, ocas uint64) (cas uint64, err *MCError){
  // Variants: Replace [Q]
  return cn.setGeneric(OpReplace, key, val, ocas, flags, exp)
}

// Add a new key/value to the cache. Fails if the key already exists in the
// cache.
func (cn *Conn) Add(key, val string, flags, exp uint32) (cas uint64, err *MCError) {
  // Variants: Add [Q]
  return cn.setGeneric(OpAdd, key, val, 0, flags, exp)
}

// Set/Add/Replace a key/value pair in the cache.
func (cn *Conn) setGeneric(op opCode, key, val string, ocas uint64, flags, exp uint32) (cas uint64, err *MCError) {
  // Request : MUST key, value, extras ([0..3] flags, [4..7] expiration)
  // Response: MUST NOT key, value, extras
  // CAS: If a CAS is specified (non-zero), all sets only succeed if the key
  //      exists and has the CAS specified. Otherwise, an error is returned.
  m := &msg{
    header: header{
      Op:  op,
      CAS: ocas,
    },
    iextras: []interface{}{flags, exp},
    key:     key,
    val:     val,
  }

  err = cn.sendRecv(m)
  return m.CAS, err
}

// Increment a value in the cache. The value must be an unsigned 64bit integer
// stored as an ASCII string. It will wrap when incremented outside the range.
func (cn *Conn) Incr(key string, delta, init uint64, exp uint32, ocas uint64) (n, cas uint64, err *MCError) {
  return cn.incrdecr(OpIncrement, key, delta, init, exp, ocas)
}

// Decrement a value in the cache. The value must be an unsigned 64bit integer
// stored as an ASCII string. It can't be decremented below 0.
func (cn *Conn) Decr(key string, delta, init uint64, exp uint32, ocas uint64) (n, cas uint64, err *MCError) {
  return cn.incrdecr(OpDecrement, key, delta, init, exp, ocas)
}

// Incr/Decr a key/value pair in the cache.
func (cn *Conn) incrdecr(op opCode, key string, delta, init uint64, exp uint32, ocas uint64) (n, cas uint64, err *MCError) {
  // Variants: [R] Incr [Q], [R] Decr [Q]
  // Request : MUST key, extras; MUST NOT value
  //   Extras: [ 0.. 7] Amount to add/sub
  //           [ 8..15] Initial value for counter (if key doesn't exist)
  //           [16..20] Expiration
  // Response: MUST value; MUST NOT key, extras

  // * response value is 64 bit unsigned binary number.
  // * if the key doesn't exist and the expiration is all 1's (0xffffffff) then
  //   the operation will fail with NOT_FOUND.
  m := &msg{
    header: header{
      Op:  op,
      CAS: ocas,
    },
    iextras: []interface{}{delta, init, exp},
    key:     key,
  }

  err = cn.sendRecv(m)
  if err != nil {
    return
  }
  // value is returned as an unsigned 64bit integer (i.e., not as a string)
  return readInt(m.val), m.CAS, nil
}

// Convert string stored to an uint64 (where no actual byte changes are needed).
func readInt(b string) uint64 {
  switch len(b) {
  case 8: // 64 bit
    return uint64(uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
      uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56)
  }

  panic(fmt.Sprintf("mc: don't know how to parse string with %d bytes", len(b)))
}

// Append the value to the existing value for the key specified. An error is
// thrown if the key doesn't exist.
func (cn *Conn) Append(key, val string, ocas uint64) (cas uint64, err *MCError) {
  // Variants: [R] Append [Q]
  // Request : MUST key, value; MUST NOT extras
  // Response: MUST NOT key, value, extras
  m := &msg{
    header: header{
      Op: OpAppend,
      CAS: ocas,
    },
    key: key,
    val: val,
  }

  err = cn.sendRecv(m)
  return m.CAS, err
}

// Prepend the value to the existing value for the key specified. An error is
// thrown if the key doesn't exist.
func (cn *Conn) Prepend(key, val string, ocas uint64) (cas uint64, err *MCError) {
  // Variants: [R] Append [Q]
  // Request : MUST key, value; MUST NOT extras
  // Response: MUST NOT key, value, extras
  m := &msg{
    header: header{
      Op: OpPrepend,
      CAS: ocas,
    },
    key: key,
    val: val,
  }

  err = cn.sendRecv(m)
  return m.CAS, err
}

// Delete a key/value from the cache.
func (cn *Conn) Del(key string) error {
  return cn.DelCAS(key, 0)
}

// Delete a key/value from the cache but only if the CAS specified matches the
// CAS in the cache.
func (cn *Conn) DelCAS(key string, cas uint64) error {
  // Variants: [R] Del [Q]
  // Request : MUST key; MUST NOT value, extras
  // Response: MUST NOT key, value, extras
  m := &msg{
    header: header{
      Op:  OpDelete,
      CAS: cas,
    },
    key: key,
  }

  return cn.sendRecv(m)
}

// Flush the cache, that is, invalidate all keys. Note, this doesn't typically
// free memory on a memcache server (doing so compromises the O(1) nature of
// memcache). Instead nearly all servers do lazy expiration, where they don't
// free memory but won't return any keys to you that have expired.
func (cn *Conn) Flush(when uint32) (err *MCError) {
  // Variants: Flush [Q]
  // Request : MUST NOT key, value; MAY extras ([0..3] expiration)
  // Response: MUST NOT key, value, extras

  // optional expiration means that the flush won't become active until that
  // point in time, hence why the argument is called 'when' as that is more
  // descriptive of its function.
  m := &msg{
    header: header{
      Op: OpFlush,
    },
    iextras: []interface{}{when},
  }

  return cn.sendRecv(m)
}

// Send a No-Op message to the memcache server. This can be used as a heartbeat
// for the server to check it's functioning fine still.
func (cn *Conn) NoOp() (err *MCError) {
  // Variants: NoOp
  // Request : MUST NOT key, value, extras
  // Response: MUST NOT key, value, extras
  m := &msg{
    header: header{
      Op: OpNoop,
    },
  }

  return cn.sendRecv(m)
}

// Get the version of the memcached server connected to.
func (cn *Conn) Version() (ver string, err *MCError) {
  // Variants: Version
  // Request : MUST NOT key, value, extras
  // Response: MUST NOT key, extras; MUST value

  // value is the version as a string in form "X.Y.Z"
  m := &msg{
    header: header{
      Op: OpVersion,
    },
  }

  err = cn.sendRecv(m)
  return m.val, err
}

// Close connection with memcache server (nicely).
func (cn *Conn) Quit() (err *MCError) {
  // Variants: Quit [Q]
  // Request : MUST NOT key, value, extras
  // Response: MUST NOT key, value, extras
  m := &msg{
    header: header{
      Op: OpQuit,
    },
  }

  err = cn.sendRecv(m)
  cn.Close();
  return
}

// Stats returns some statistics about the memcached server.
func (cn *Conn) Stats() (stats map[string]string, err *MCError) {
  // Variants: Stats
  // Request : MAY HAVE key, MUST NOT value, extra
  // Response: Serries of responses that MUST HAVE key, value; followed by one
  //           response that MUST NOT have key, value. ALL MUST NOT extras.
  m := &msg{
    header: header{
      Op: OpStat,
    },
  }

  cn.l.Lock()
  defer cn.l.Unlock()

  err = cn.send(m)
  if err != nil {
    return
  }

  // collect all statistics
  stats = make(map[string]string)
  for {
    err = cn.recv(m)
    // error or termination message
    if err != nil || m.KeyLen == 0 {
      return
    }
    stats[m.key] = m.val
  }

  return
}

