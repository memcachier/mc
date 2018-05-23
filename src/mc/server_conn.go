package mc

// Handles the connection between the client and memcached servers.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
	"strings"
)

// serverConn is a connection to a memcache server.
type serverConn struct {
	address   string
	username  string
	password  string
	config    *config
	conn      net.Conn
	buf       *bytes.Buffer
	opq       uint32
	backupMsg msg
}

func newServerConn(address, username, password string, config *config) *serverConn {
	serverConn := &serverConn{
		address: address,
		username: username,
		password: password,
		config: config,
		buf: new(bytes.Buffer),
	}
	return serverConn
}

func (sc *serverConn) perform(m *msg) error {
	// lazy connection
	if sc.conn == nil {
		err := sc.connect()
		if err != nil {
			return err
		}
	}
	return sc.sendRecv(m)
}

func (sc *serverConn) performStats(m *msg) (mcStats, error) {
	// lazy connection
	if sc.conn == nil {
		err := sc.connect()
		if err != nil {
			return nil, err
		}
	}
	return sc.sendRecvStats(m)
}

func (sc *serverConn) quit(m *msg) {
	if sc.conn != nil {
		sc.sendRecv(m)
		sc.conn.Close()
		sc.conn = nil
	}
}

func (sc *serverConn) connect() error {
	c, err := net.DialTimeout("tcp", sc.address, sc.config.ConnectionTimeout)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}
	sc.conn = c
	tcpConn, ok := c.(*net.TCPConn)
	if !ok {
		return &Error{StatusNetworkError, "Cannot convert into TCP connection", nil}
	}
	// TCP config TODO make configurable
	tcpConn.SetKeepAlive(true)
	tcpConn.SetKeepAlivePeriod(60 * time.Second)
	tcpConn.SetNoDelay(true)
	// authenticate
	err = sc.auth()
	if err != nil {
		// Error, except if the server doesn't support authentication
		mErr := err.(*Error)
		if mErr.Status != StatusUnknownCommand {
			sc.conn.Close()
			sc.conn = nil
			return err
		}
	}
	return nil
}

// Auth performs SASL authentication (using the PLAIN method) with the server.
func (sc *serverConn) auth() error {
	s, err := sc.authList()
	if err != nil {
		return err
	}

	switch {
	case strings.Index(s, "PLAIN") != -1:
		return sc.authPlain()
	}

	return &Error{StatusAuthUnknown, fmt.Sprintf("mc: unknown auth types %q", s), nil}
}

// authList runs the SASL authentication list command with the server to
// retrieve the list of support authentication mechansims.
func (sc *serverConn) authList() (string, error) {
	m := &msg{
		header: header{
			Op: opAuthList,
		},
	}

	err := sc.sendRecv(m)
	return m.val, err
}

// authPlain performs SASL authentication using the PLAIN method.
func (sc *serverConn) authPlain() error {
	m := &msg{
		header: header{
			Op: opAuthStart,
		},

		key: "PLAIN",
		val: fmt.Sprintf("\x00%s\x00%s", sc.username, sc.password),
	}

	return sc.sendRecv(m)
}

// sendRecv sends and receives a complete memcache request/response exchange.
func (sc *serverConn) sendRecv(m *msg) error {
	// fmt.Printf("sendRecv: %v, %v\n", m.header.Op, m.key)
	err := sc.send(m)
	if err != nil {
		sc.resetConn(err)
		return err
	}
	err = sc.recv(m)
	if err != nil {
		sc.resetConn(err)
		return err
	}
	return nil
}

// sendRecvStats
func (sc *serverConn) sendRecvStats(m *msg) (stats mcStats, err error) {
	err = sc.send(m)
	if err != nil {
		sc.resetConn(err)
		return
	}

	// collect all statistics
	stats = make(map[string]string)
	for {
		err = sc.recv(m)
		// error or termination message
		if err != nil || m.KeyLen == 0 {
			if err != nil {
				sc.resetConn(err)
			}
			return
		}
		stats[m.key] = m.val
	}
	return
}

// send sends a request to the memcache server.
func (sc *serverConn) send(m *msg) error {
	m.Magic = magicSend
	m.ExtraLen = sizeOfExtras(m.iextras)
	m.KeyLen = uint16(len(m.key))
	m.BodyLen = uint32(m.ExtraLen) + uint32(m.KeyLen) + uint32(len(m.val))
	m.Opaque = sc.opq
	sc.opq++

	// Request
	err := binary.Write(sc.buf, binary.BigEndian, m.header)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}

	for _, e := range m.iextras {
		err = binary.Write(sc.buf, binary.BigEndian, e)
		if err != nil {
			return wrapError(StatusNetworkError, err)
		}
	}

	_, err = io.WriteString(sc.buf, m.key)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}

	_, err = io.WriteString(sc.buf, m.val)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}

	// Make sure write does not block forever
	sc.conn.SetWriteDeadline(time.Now().Add(sc.config.ConnectionTimeout))
	_, err = sc.buf.WriteTo(sc.conn)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}

	return nil
}

// recv receives a memcached response. It takes a msg into which to store the
// response.
func (sc *serverConn) recv(m *msg) error {
	// Make sure read does not block forever
	sc.conn.SetReadDeadline(time.Now().Add(sc.config.ConnectionTimeout))

	err := binary.Read(sc.conn, binary.BigEndian, &m.header)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}

	bd := make([]byte, m.BodyLen)
	_, err = io.ReadFull(sc.conn, bd)
	if err != nil {
		return wrapError(StatusNetworkError, err)
	}

	buf := bytes.NewBuffer(bd)

	if m.ResvOrStatus == 0 && m.ExtraLen > 0 {
		for _, e := range m.oextras {
			err := binary.Read(buf, binary.BigEndian, e)
			if err != nil {
				return wrapError(StatusNetworkError, err)
			}
		}
	}

	m.key = string(buf.Next(int(m.KeyLen)))
	vlen := int(m.BodyLen) - int(m.ExtraLen) - int(m.KeyLen)
	m.val = string(buf.Next(int(vlen)))
	// fmt.Printf("recv return: %v\n", m.ResvOrStatus)
	return newError(m.ResvOrStatus)
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

// resetConn destroy connection if a network error ocurred. serverConn will
// reconnect on next usage.
func (sc *serverConn) resetConn(err error) {
	if  err.(*Error).Status == StatusNetworkError {
		sc.conn.Close()
		sc.conn = nil
	}
}

func (sc *serverConn) backup(m *msg) {
	sc.backupMsg.key = m.key
	sc.backupMsg.val = m.val
	sc.backupMsg.header.Magic = m.header.Magic
	sc.backupMsg.header.Op = m.header.Op
	sc.backupMsg.header.KeyLen = m.header.KeyLen
	sc.backupMsg.header.ExtraLen = m.header.ExtraLen
	sc.backupMsg.header.DataType = m.header.DataType
	sc.backupMsg.header.ResvOrStatus = m.header.ResvOrStatus
	sc.backupMsg.header.BodyLen = m.header.BodyLen
	sc.backupMsg.header.Opaque = m.header.Opaque
	sc.backupMsg.header.CAS = m.header.CAS
	sc.backupMsg.iextras = nil // go way of clearing a slice, this is just fucked up
	for _, v := range m.iextras {
		sc.backupMsg.iextras = append(sc.backupMsg.iextras, v)
	}
	sc.backupMsg.oextras = nil
	for _, v := range m.oextras {
		sc.backupMsg.oextras = append(sc.backupMsg.oextras, v)
	}
}

func (sc *serverConn) restore(m *msg) {
	m.key = sc.backupMsg.key
	m.val = sc.backupMsg.val
	m.header.Magic = sc.backupMsg.header.Magic
	m.header.Op = sc.backupMsg.header.Op
	m.header.KeyLen = sc.backupMsg.header.KeyLen
	m.header.ExtraLen = sc.backupMsg.header.ExtraLen
	m.header.DataType = sc.backupMsg.header.DataType
	m.header.ResvOrStatus = sc.backupMsg.header.ResvOrStatus
	m.header.BodyLen = sc.backupMsg.header.BodyLen
	m.header.Opaque = sc.backupMsg.header.Opaque
	m.header.CAS = sc.backupMsg.header.CAS
	m.iextras = nil // go way of clearing a slice, this is just fucked up
	for _, v := range sc.backupMsg.iextras {
		m.iextras = append(m.iextras, v)
	}
	m.oextras = nil
	for _, v := range sc.backupMsg.oextras {
		m.oextras = append(m.oextras, v)
	}
}
