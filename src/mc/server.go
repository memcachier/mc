package mc

// Handles the connection between the client and memcached servers.

import (
  "bytes"
  "encoding/binary"
  "io"
  "net"
  "sync"
  "fmt"
)

// Conn is a connection to a memcache server.
type Conn struct {
  rwc io.ReadWriteCloser
  l   sync.Mutex
  buf *bytes.Buffer
  opq uint32
}

// Dial establishes a connection to a memcache server.
func Dial(nett, addr string) (*Conn, error) {
  nc, err := net.Dial(nett, addr)
  if err != nil {
    return nil, err
  }

  cn := NewConnection(nc)
  return cn, nil
}

func NewConnection(conn io.ReadWriteCloser) *Conn {
  return &Conn{rwc: conn, buf: new(bytes.Buffer)}
}

// Close closes the memcache connection.
func (cn *Conn) Close() error {
  return cn.rwc.Close()
}

// sendRecv sends and receives a complete memcache request/response exchange.
//
// LOCK INVARIANT: protected by the Conn.l lock.
func (cn *Conn) sendRecv(m *msg) (err *MCError) {
  cn.l.Lock()
  defer cn.l.Unlock()

  err = cn.send(m)
  if err != nil {
    return
  }

  err = cn.recv(m)
  if err != nil {
    return
  }

  return nil
}

// send sends a request to the memcache server.
//
// LOCK INVARIANT: Unprotected.
func (cn *Conn) send(m *msg) *MCError {
  m.Magic = 0x80
  m.ExtraLen = sizeOfExtras(m.iextras)
  m.KeyLen = uint16(len(m.key))
  m.BodyLen = uint32(m.ExtraLen) + uint32(m.KeyLen) + uint32(len(m.val))
  m.Opaque = cn.opq
  cn.opq += 1

  // Request
  err := binary.Write(cn.buf, binary.BigEndian, m.header)
  if err != nil {
    return &MCError{0xffff, err.Error()}
  }

  for _, e := range m.iextras {
    err = binary.Write(cn.buf, binary.BigEndian, e)
    if err != nil {
      return &MCError{0xffff, err.Error()}
    }
  }

  _, err = io.WriteString(cn.buf, m.key)
  if err != nil {
    return &MCError{0xffff, err.Error()}
  }

  _, err = io.WriteString(cn.buf, m.val)
  if err != nil {
    return &MCError{0xffff, err.Error()}
  }

  _, err = cn.buf.WriteTo(cn.rwc)
  if err != nil {
    return &MCError{0xffff, err.Error()}
  }
  return nil
}

// recv receives a memcached response. It takes a msg into which to store the
// response.
//
// LOCK INVARIANT: Unprotected.
func (cn *Conn) recv(m *msg) *MCError {
  err := binary.Read(cn.rwc, binary.BigEndian, &m.header)
  if err != nil {
    return &MCError{0xffff, err.Error()}
  }

  bd := make([]byte, m.BodyLen)
  _, err = io.ReadFull(cn.rwc, bd)
  if err != nil {
    return &MCError{0xffff, err.Error()}
  }

  buf := bytes.NewBuffer(bd)

  if m.ResvOrStatus == 0 && m.ExtraLen > 0 {
    for _, e := range m.oextras {
      err := binary.Read(buf, binary.BigEndian, e)
      if err != nil {
        return &MCError{0xffff, err.Error()}
      }
    }
  }

  m.key = string(buf.Next(int(m.KeyLen)))
  vlen := int(m.BodyLen) - int(m.ExtraLen) - int(m.KeyLen)
  m.val = string(buf.Next(int(vlen)))

  return checkError(m)
}

// checkError checks if the received response is an error.
func checkError(m *msg) *MCError {
  return NewMCError(m.ResvOrStatus)
}

// sizeOfExtras returns the size of the extras field for the memcache request.
func sizeOfExtras(extras []interface{}) (l uint8) {
  for _, e := range extras {
    switch e.(type) {
    default:
      panic(fmt.Sprintf("mc: unknown extra type (%T)", e))
    case uint8:
      l += 8 / 8
    case uint16:
      l += 16 / 8
    case uint32:
      l += 32 / 8
    case uint64:
      l += 64 / 8
    }
  }
  return
}

