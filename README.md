# mc.go: A Go client for Memcached

[![Build Status](https://img.shields.io/travis/memcachier/mc.svg?style=flat)](https://travis-ci.org/memcachier/mc)

This is a (pure) Go client for [Memcached](http://memcached.org). It supports
the binary Memcached protocol and SASL authentication. It's thread-safe.

## Install

		$ go get github.com/memcachier/mc

## Use

		import "github.com/memcachier/mc"

		func main() {
			// Error handling omitted for demo
			cn, err := mc.Dial("tcp", "localhost:11211")
			if err != nil {
				...
			}

			// Only PLAIN SASL auth supported right now
			// See: http://code.google.com/p/memcached/wiki/SASLHowto
			err = cn.Auth("foo", "bar")
			if err != nil {
				...
			}
			

			val, cas, err = cn.Get("foo")
			if err != nil {
				...
			}

			exp = 3600 // 2 hours
			err = cn.Set("foo", "bar", cas, exp)
			if err != nil {
				...
			}

			err = cn.Del("foo")
			if err != nil {
				...
			}
		}

## Missing Feature

There is nearly coverage of the Memcached protocol, but at the moment we only
support a single Memcached server. Support for a Memcached cluster using a
sharding / hashing method is still needed.

The biggest missing protocol feature is support for `multi_get` and other
batched operations.

The `Stats` call also doesn't support sending a key across, which is needed for
finer grained stats and resetting counters on the server.

There is also no support for asynchronous IO.

## Performance

Right now we use a single per-connection mutex and don't support pipe-lining any
operations.

## Get involved!

We are happy to receive bug reports, fixes, documentation enhancements,
and other improvements.

Please report bugs via the
[github issue tracker](http://github.com/memcachier/mc/issues).

Master [git repository](http://github.com/memcachier/mc):

* `git clone git://github.com/memcachier/mc.git`

## Licensing

This library is MIT-licensed.

## Authors

This library is written and maintained by David Terei (<code@davidterei.com>).
It was originally written by [Blake Mizerany](https://github.com/bmizerany/mc).

