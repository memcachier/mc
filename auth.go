package mc

// Handles SASL authentication.

import (
	"fmt"
	"strings"
)

func (cn *Conn) Auth(user, pass string) error {
	s, err := cn.authList()
	if err != nil {
		return err
	}

	switch {
	case strings.Index(s, "PLAIN") != -1:
		return cn.authPlain(user, pass)
	}

	return fmt.Errorf("mc: unknown auth types %q", s)
}

func (cn *Conn) authList() (s string, err error) {
	m := &msg{
		header: header{
			Op: OpAuthList,
		},
	}

	err = cn.send(m)
	return m.val, err
}

func (cn *Conn) authPlain(user, pass string) error {
	m := &msg{
		header: header{
			Op: OpAuthStart,
		},

		key: "PLAIN",
		val: fmt.Sprintf("\x00%s\x00%s", user, pass),
	}

	return cn.send(m)
}

