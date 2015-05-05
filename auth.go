package mc

// Handles SASL authentication.

import (
	"fmt"
	"strings"
)

// Auth performs SASL authentication (using the PLAIN method) with the server.
func (cn *Conn) Auth(user, pass string) *Error {
	s, err := cn.authList()
	if err != nil {
		return err
	}

	switch {
	case strings.Index(s, "PLAIN") != -1:
		return cn.authPlain(user, pass)
	}

	return &Error{StatusAuthUnknown, fmt.Sprintf("mc: unknown auth types %q", s), nil}
}

// authList runs the SASL authentication list command with the server to
// retrieve the list of support authentication mechansims.
func (cn *Conn) authList() (s string, err *Error) {
	m := &msg{
		header: header{
			Op: OpAuthList,
		},
	}

	err = cn.sendRecv(m)
	return m.val, err
}

// authPlain performs SASL authentication using the PLAIN method.
func (cn *Conn) authPlain(user, pass string) *Error {
	m := &msg{
		header: header{
			Op: OpAuthStart,
		},

		key: "PLAIN",
		val: fmt.Sprintf("\x00%s\x00%s", user, pass),
	}

	return cn.sendRecv(m)
}
