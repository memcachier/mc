package mc

// Handles all server connections to a particular memcached servers.

import (
	"net"
	"sync"
	"time"
)

// server represents a server and contains all connections to that server
type server struct {
	address string
	config  *Config
	// NOTE: organizing the pool as a chan makes the usage of the containing
	// connections treadsafe
	pool    chan mcConn
	isAlive bool
	lock    sync.Mutex
}

func newServer(address, username, password string, config *Config, newMcConn connGen) *server {
	var hostport string
	host, port, err := net.SplitHostPort(address)
	if err == nil {
		hostport = net.JoinHostPort(host, port)
	} else {
		hostport = net.JoinHostPort(address, "11211")
	}
	server := &server{
		address: hostport,
		config:  config,
		pool:    make(chan mcConn, config.PoolSize),
		isAlive: true,
	}

	for i := 0; i < config.PoolSize; i++ {
		server.pool <- newMcConn(hostport, username, password, config)
	}

	return server
}

func (s *server) perform(m *msg) error {
	var err error
	for i := 0; ; {
		timeout := time.After(s.config.ConnectionTimeout)
		select {
		case c := <-s.pool:
			// NOTE: this serverConn is no longer available in the pool (equivalent to locking)
			if c == nil {
				return &Error{StatusUnknownError, "Client is closed (did you call Quit?)", nil}
			}

			// backup request if a retry might be possible
			if i+1 < s.config.Retries {
				c.backup(m)
			}

			err = c.perform(m)
			s.pool <- c
			if err == nil {
				return nil
			}
			// Return Memcached errors except network errors.
			mErr := err.(*Error)
			if mErr.Status != StatusNetworkError {
				return err
			}

			// check if retry needed
			i++
			if i < s.config.Retries {
				// restore request since m now contains the failed response
				c.restore(m)
				time.Sleep(s.config.RetryDelay)
			} else {
				return err
			}
		case <-timeout:
			// do not retry
			return &Error{StatusUnknownError,
				"Timed out while waiting for connection from pool. " +
					"Maybe increase your pool size?",
				nil}
		}
	}
	// return err
}

func (s *server) performStats(m *msg) (mcStats, error) {
	timeout := time.After(s.config.ConnectionTimeout)
	select {
	case c := <-s.pool:
		// NOTE: this serverConn is no longer available in the pool (equivalent to locking)
		if c == nil {
			return nil, &Error{StatusUnknownError, "Client is closed (did you call Quit?)", nil}
		}

		stats, err := c.performStats(m)
		s.pool <- c
		return stats, err

	case <-timeout:
		// do not retry
		return nil, &Error{StatusUnknownError,
			"Timed out while waiting for connection from pool. " +
				"Maybe increase your pool size?",
			nil}
	}
}

func (s *server) quit(m *msg) {
	for i := 0; i < s.config.PoolSize; i++ {
		c := <-s.pool
		if c == nil {
			// Do not double quit
			return
		}
		c.quit(m)
	}
	close(s.pool)
}

func (s *server) changeAlive(alive bool) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.isAlive != alive {
		s.isAlive = alive
		return true
	}
	return false
}
