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

type Conn struct {
	rwc io.ReadWriteCloser
	l   sync.Mutex
	buf *bytes.Buffer
}

func Dial(nett, addr string) (*Conn, error) {
	nc, err := net.Dial(nett, addr)
	if err != nil {
		return nil, err
	}

	cn := &Conn{rwc: nc, buf: new(bytes.Buffer)}
	return cn, nil
}

func (cn *Conn) Close() error {
	return cn.rwc.Close()
}

func (cn *Conn) sendRecv(m *msg) (err error) {
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

func (cn *Conn) send(m *msg) (err error) {
	m.Magic = 0x80
	m.ExtraLen = sizeOfExtras(m.iextras)
	m.KeyLen = uint16(len(m.key))
	m.BodyLen = uint32(m.ExtraLen) + uint32(m.KeyLen) + uint32(len(m.val))

	cn.l.Lock()
	defer cn.l.Unlock()

	// Request
	err = binary.Write(cn.buf, binary.BigEndian, m.header)
	if err != nil {
		return
	}

	for _, e := range m.iextras {
		err = binary.Write(cn.buf, binary.BigEndian, e)
		if err != nil {
			return
		}
	}

	_, err = io.WriteString(cn.buf, m.key)
	if err != nil {
		return
	}

	_, err = io.WriteString(cn.buf, m.val)
	if err != nil {
		return
	}

	_, err = cn.buf.WriteTo(cn.rwc)
  return
}

// recv receives a memcached response. It takes a msg into which to store the
// response.
func (cn *Conn) recv(m *msg) (err error) {
	err = binary.Read(cn.rwc, binary.BigEndian, &m.header)
	if err != nil {
		return
	}

	bd := make([]byte, m.BodyLen)
	_, err = io.ReadFull(cn.rwc, bd)
	if err != nil {
		return
	}

	buf := bytes.NewBuffer(bd)

  if m.ResvOrStatus == 0 && m.ExtraLen > 0 {
    for _, e := range m.oextras {
      err = binary.Read(buf, binary.BigEndian, e)
      if err != nil {
        return
      }
    }
  }

	m.key = string(buf.Next(int(m.KeyLen)))
	vlen := int(m.BodyLen) - int(m.ExtraLen) - int(m.KeyLen)
	m.val = string(buf.Next(int(vlen)))

	return checkError(m)
}

func checkError(m *msg) error {
	err, ok := errMap[m.ResvOrStatus]
	if !ok {
		return ErrUnknownError
	}
	return err
}

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

