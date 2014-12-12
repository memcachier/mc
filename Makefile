SHELL := /bin/bash
ifndef GO
GO=go
endif

.PHONY: all

all: mc

mc: src/mc/*.go
	@mkdir -p bin
	@cd bin && GOPATH=$(CURDIR) $(GO) build mc

test: mc
	@memcached -p 11289 & echo $$! > test.pids
	@GOPATH=$(CURDIR) $(GO) test -test.short -v mc; ST=$?; \
	cd $(CURDIR); cat test.pids | xargs kill; rm test.pids
	@exit ${ST}

test-full: mc
	@memcached -p 11289 & echo $$! > test.pids
	@GOPATH=$(CURDIR) $(GO) test -v mc; ST=$?; \
	cd $(CURDIR); cat test.pids | xargs kill; rm test.pids
	@exit ${ST}

clean:
	go clean
	rm -Rf bin

