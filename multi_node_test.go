// +build multinode

package mc

import (
	"fmt"
	"regexp"
	"testing"
)

func TestMultiNodeVersion(t *testing.T) {
	c := testInitMultiNode(t, "localhost:11289 localhost:11290 localhost:11291", "", "")
	vers, err := c.Version()
	assertEqualf(t, nil, err, "unexpected error: %v", err)
	for k, v := range vers {
		fmt.Printf("    host: %s  version: %s\n", k, v)
		good, errRegex := regexp.MatchString("[0-9]+\\.[0-9]+\\.[0-9]+", v)
		assertEqualf(t, nil, errRegex, "unexpected error: %v", errRegex)
		assertEqualf(t, good, true, "version of unexcpected form: %s", v)
	}
	c.Quit()
}

func TestMultiNodeNoOp(t *testing.T) {
	c := testInitMultiNode(t, "localhost:11289 localhost:11290 localhost:11291", "", "")
	err := c.NoOp()
	assertEqualf(t, nil, err, "unexpected error: %v", err)
	c.Quit()
}

func TestMultiNodeFlush(t *testing.T) {
	c := testInitMultiNode(t, "localhost:11289 localhost:11290 localhost:11291", "", "")
	err := c.Flush(0)
	assertEqualf(t, nil, err, "unexpected error: %v", err)
	c.Quit()
}
