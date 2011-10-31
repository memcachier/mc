package mc

import (
	"fmt"
	"io"
	"os"
)

type lrwc struct {
	rwc io.ReadWriteCloser
}

func (l *lrwc) Write(b []byte) (int, os.Error) {
	fmt.Printf(">> %q\n", b)
	return l.rwc.Write(b)
}

func (l *lrwc) Read(b []byte) (int, os.Error) {
	n, err := l.rwc.Read(b)
	fmt.Printf("<< %q\n", b)
	return n, err
}

func (l *lrwc) Close() os.Error {
	fmt.Println("<closed>")
	return l.rwc.Close()
}
