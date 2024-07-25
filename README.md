# mc.go: A Go client for Memcached

[![godoc](https://godoc.org/github.com/memcachier/mc?status.svg)](http://godoc.org/github.com/memcachier/mc)
[![Build Status](https://img.shields.io/travis/memcachier/mc.svg?style=flat)](https://travis-ci.org/memcachier/mc)

This is a (pure) Go client for [Memcached](http://memcached.org). It supports
the binary Memcached protocol, SASL authentication and Compression. It's thread-safe.
It allows connections to entire Memcached clusters and supports connection
pools, timeouts, and failover.

## Install

Module-aware mode:

```
$ go get github.com/memcachier/mc/v3
```

Legacy GOPATH mode:

```
$ go get github.com/memcachier/mc
```

## Use

```go
import "github.com/memcachier/mc/v3"
// Legacy GOPATH mode:
// import "github.com/memcachier/mc"

func main() {
	// Error handling omitted for demo

	// Only PLAIN SASL auth supported right now
	c := mc.NewMC("localhost:11211", "username", "password")
	defer c.Quit()

	exp := 3600 // 2 hours
	cas, err = c.Set("foo", "bar", flags, exp, cas)
	if err != nil {
		...
	}

	val, flags, cas, err = c.Get("foo")
	if err != nil {
		...
	}

	err = c.Del("foo")
	if err != nil {
		...
	}
}
```

## Using Compression

```go
import (
  "github.com/memcachier/mc/v3"
  "compress/zlib"
)
// Legacy GOPATH mode:
// import "github.com/memcachier/mc"

func main() {
	// Error handling omitted for demo

	// Only PLAIN SASL auth supported right now
  config := mc.DefaultConfig()

  // You have to set the functions to deflate and unzip
	// At this example we are using zlib

  config.Compression.decompress = func(value string) (string, error) {
		var compressedValue bytes.Buffer
		zw, err := zlib.NewWriterLevel(&compressedValue, -1)
		if err != nil {
			return value, err
		}
		if _, err = zw.Write([]byte(value)); err != nil {
			return value, err
		}
		zw.Close()
		return compressedValue.String(), nil
	}

	config.Compression.compress = func(value string) (string, error) {
		if value == "" {
			return value, nil
		}
		zr, err := zlib.NewReader(strings.NewReader(value))
		if err != nil {
			return value, nil // Does not return error, the value could be not compressed
		}
		defer zr.Close()
		var unCompressedValue bytes.Buffer
		_, err = io.Copy(&unCompressedValue, zr)
		if err != nil {
			return value, nil
		}
		return unCompressedValue.String(), nil
	}

	c := mc.NewMCwithConfig("localhost:11211", "username", "password", config)
	defer c.Quit()

	exp := 3600 // 2 hours
	cas, err = c.Set("foo", "bar", flags, exp, cas)
	if err != nil {
		...
	}

	val, flags, cas, err = c.Get("foo")
	if err != nil {
		...
	}

	err = c.Del("foo")
	if err != nil {
		...
	}
}
```

## Missing Feature

There is nearly coverage of the Memcached protocol.
The biggest missing protocol feature is support for `multi_get` and other
batched operations.

There is also no support for asynchronous IO.

## Performance

Right now we use a single per-connection mutex and don't support pipe-lining any
operations. There is however support for connection pools which should make up
for it.

## Get involved!

We are happy to receive bug reports, fixes, documentation enhancements,
and other improvements.

Please report bugs via the
[github issue tracker](http://github.com/memcachier/mc/issues).

Master [git repository](http://github.com/memcachier/mc):

- `git clone git://github.com/memcachier/mc.git`

## Licensing

This library is MIT-licensed.

## Authors

This library is written and maintained by MemCachier.
It was originally written by [Blake Mizerany](https://github.com/bmizerany/mc).
