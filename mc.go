package mc

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync"
)

// Errors
var (
	ErrNotFound = os.NewError("mc: not found")
	ErrKeyExists = os.NewError("mc: key exists")
	ErrValueTooLarge = os.NewError("mc: value to large")
	ErrInvalidArgs = os.NewError("mc: invalid arguments")
	ErrValueNotStored = os.NewError("mc: value not stored")
	ErrNonNumeric = os.NewError("mc: incr/decr called on non-numeric value")
	ErrAuthRequired = os.NewError("mc: authentication required")
	ErrUnknownCommand = os.NewError("mc: unknown command")
	ErrOutOfMemory = os.NewError("mc: out of memory")
)

var errMap = map[uint16]os.Error{
	0: nil,
	1: ErrNotFound,
	2: ErrKeyExists,
	3: ErrValueTooLarge,
	4: ErrInvalidArgs,
	5: ErrValueNotStored,
	6: ErrNonNumeric,
	0x20: ErrAuthRequired,
	0x81: ErrUnknownCommand,
	0x82: ErrOutOfMemory,
}

type header struct {
	Magic  uint8
	Op     uint8
	KeyLen   uint16
	ExtraLen   uint8
	DataType  uint8
	ResvOrStatus  uint16
	BodyLen   uint32
	Opaque uint32
	CAS    uint64
}

type Conn struct {
	rwc io.ReadWriteCloser
	l   sync.Mutex
}

func Dial(addr string) (*Conn, os.Error) {
	nc, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	cn := &Conn{rwc: nc}
	return cn, nil
}

func (cn *Conn) Get(key string) (string, int, os.Error) {
	cn.l.Lock()
	defer cn.l.Unlock()

	h := &header{
		Magic: 0x80,
		Op:  0x00,
		KeyLen: uint16(len(key)),
		BodyLen: uint32(len(key)),
	}

	err := binary.Write(cn.rwc, binary.BigEndian, h)
	if err != nil {
		return "", 0, err
	}

	_, err = io.WriteString(cn.rwc, key)
	if err != nil {
		return "", 0, err
	}

	err = cn.readHeader(h)
	if err != nil {
		return "", 0, err
	}

	val, err := cn.readString(h.KeyLen)
	if err != nil {
		return "", 0, err
	}

	return val, int(h.CAS), nil
}

func (cn *Conn) readString(n uint16) (string, os.Error) {
	b := make([]byte, n)
	_, err := io.ReadFull(cn.rwc, b)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (cn *Conn) readHeader(h *header) os.Error {
	err := binary.Read(cn.rwc, binary.BigEndian, h)
	if err != nil {
		return err
	}

	err = checkError(h)
	if err != nil {
		return err
	}

	return nil
}

func checkError(h *header) os.Error {
	err, ok := errMap[h.ResvOrStatus]
	if !ok {
		return os.NewError("mc: unknown error from server")
	}
	return err
}
