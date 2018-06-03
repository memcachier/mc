package mc

import (
	"testing"
)

func BenchmarkSet(b *testing.B) {
	b.StopTimer()
	c := NewMC(mcAddr, user, pass)
	// Lazy connection. Make sure it connects before starting benchmark.
	_, err := c.Set("foo", "bar", 0, 0, 0)
	if err != nil {
		panic(err)
	}

	b.StartTimer()
	defer b.StopTimer()

	for i := 0; i < b.N; i++ {
		_, err := c.Set("foo", "bar", 0, 0, 0)
		if err != nil {
			panic(err)
		}
	}
}
