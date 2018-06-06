package mc

import (
	"strconv"
	"testing"
	"time"
)

// Test successful retries
func TestRetrySuccess(t *testing.T) {
	for i := 1; i < 4; i++ {
		config := DefaultConfig()
		config.Retries = i
		i_str := strconv.Itoa(i)
		c := newMockableMC("s1-"+i_str, "", "", config, newMockConn)

		val, flags, cs, err := c.Get("k1")
		if err != nil {
			t.Errorf("val: %v, flags: %v, cas: %v", val, flags, cs)
			t.Fatalf("expected no error: %v", err)
		}
		expectedVal := "k1,s1," + i_str
		if val != expectedVal {
			t.Fatalf("got wrong value: %v, expected: %v", val, expectedVal)
		}
	}
}

// Test failed retries
func TestRetryFailure(t *testing.T) {
	for i := 2; i < 4; i++ {
		config := DefaultConfig()
		config.Retries = i - 1
		i_str := strconv.Itoa(i)
		c := newMockableMC("s1-"+i_str, "", "", config, newMockConn)

		val, flags, cs, err := c.Get("k1")
		if err == nil {
			t.Errorf("val: %v, flags: %v, cas: %v", val, flags, cs)
			t.Fatal("expected error but got none")
		}
	}
}

// Test successful failover
func TestFailoverSuccess(t *testing.T) {
	config := DefaultConfig()
	config.DownRetryDelay = 2 * time.Second
	c := newMockableMC("s1-3,s2-1", "", "", config, newMockConn)

	key := "k2" // this key hashes to s1
	res := [3]string{key + ",s2,1", key + ",s2,2", key + ",s1,3"}
	// Expected behavior
	// 1st loop: try to get twice from s1, fails, marks s1 down, tries s2, succeeds
	// 2nd loop: get from s2 since s1 is still down, succeeds
	// 3rd loop: get from s1 since it is up again, succeeds
	for i := 0; i < len(res); i++ {
		val, flags, cs, err := c.Get(key)
		if err != nil {
			t.Errorf("val: %v, flags: %v, cas: %v", val, flags, cs)
			t.Fatalf("expected no error: %v", err)
		}
		expectedVal := res[i]
		if val != expectedVal {
			t.Fatalf("got wrong value: %v, expected: %v", val, expectedVal)
		}
		time.Sleep(1 * time.Second)
	}
}
