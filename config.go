package mc

//

import (
	"time"
)

// Config holds the Memcache client configuration. Use DefaultConfig to get
// an initialized version.
type Config struct {
	Hasher     hasher
	Retries    int
	RetryDelay time.Duration
	Failover   bool
	// ConnectionTimeout is currently used to timeout getting connections from
	// pool, as a sending deadline and as a reading deadline. Worst case this
	// means a request can take 3 times the ConnectionTimeout.
	ConnectionTimeout  time.Duration
	DownRetryDelay     time.Duration
	PoolSize           int
	TcpKeepAlive       bool
	TcpKeepAlivePeriod time.Duration
	TcpNoDelay         bool
	Compression        struct {
		unzip   func(value string) (string, error)
		deflate func(value string) (string, error)
	}
}

/*
DefaultConfig returns a config object populated with the default values.
The default values currently are:

	config{
		Hasher:             NewModuloHasher(),
		Retries:            2,
		RetryDelay:         200 * time.Millisecond,
		Failover:           true,
		ConnectionTimeout:  2 * time.Second,
		DownRetryDelay:     60 * time.Second,
		PoolSize:           1,
		TcpKeepAlive:       true,
		TcpKeepAlivePeriod: 60 * time.Second,
		TcpNoDelay:         true,
		Compression        struct {
			unzip   nil
			deflate nil
		}
	}
*/
func DefaultConfig() *Config {
	return &Config{
		Hasher:             NewModuloHasher(),
		Retries:            2,
		RetryDelay:         200 * time.Millisecond,
		Failover:           true,
		ConnectionTimeout:  2 * time.Second,
		DownRetryDelay:     60 * time.Second,
		PoolSize:           1,
		TcpKeepAlive:       true,
		TcpKeepAlivePeriod: 60 * time.Second,
		TcpNoDelay:         true,
		Compression: struct {
			unzip   func(value string) (string, error)
			deflate func(value string) (string, error)
		}{unzip: nil, deflate: nil},
	}
}
