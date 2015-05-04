SHELL:=/bin/bash
ifndef GO
GO:=go
endif

.PHONY: all deps clean fmt vet lint help

all: build

build: *.go
	@$(GO) build

deps:
	git submodule init
	git submodule update

test: build
	@memcached -p 11289 & echo $$! > test.pids
	@GOPATH=$(CURDIR)/deps $(GO) test -test.short -v; ST=$?; \
	cd $(CURDIR); cat test.pids | xargs kill; rm test.pids
	@exit ${ST}

test-full: build
	@memcached -p 11289 & echo $$! > test.pids
	@GOPATH=$(CURDIR)/deps $(GO) test -v; ST=$?; \
	cd $(CURDIR); cat test.pids | xargs kill; rm test.pids
	@exit ${ST}

clean:
	@$(GO) clean

fmt:
	@$(GO) fmt

vet:
	@$(GO) vet

lint:
	@command -v golint >/dev/null 2>&1 \
		|| { echo >&2 "The 'golint' tool is required, please install"; exit 1;  }
	@golint

help:
	@echo "Build Targets"
	@echo "   build      - Build mc"
	@echo "   deps       - Git checkout dependencies"
	@echo "   test       - Quick test of mc"
	@echo "   test-full  - Longer test of mc against a real memcached process"
	@echo "   clean      - Remove built sources"
	@echo "   fmt        - Format the source code using 'go fmt'"
	@echo "   vet        - Analyze the source code for potential errors"
	@echo "   lint       - Analyze the source code for style mistakes"
	@echo ""

