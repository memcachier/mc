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

# Yes, this is a hack, but Go's packing is also just stupid.
work-tree: src
	ln -s vendor src

test: build work-tree
	@memcached -p 11289 & echo $$! > test.pids
	@GOPATH=$(CURDIR) $(GO) test -test.short -v; ST=$?; \
	cd $(CURDIR); cat test.pids | xargs kill; rm test.pids
	@exit ${ST}

test-full: build work-tree
	@memcached -p 11289 & echo $$! > test.pids
	@GOPATH=$(CURDIR) GO15VENDOREXPERIMENT=1 $(GO) test -v; ST=$?; \
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
	@echo "   test       - Quick test of mc (against memcached on port 11289)"
	@echo "   test-full  - Longer test of mc (against memcached on port 11289)"
	@echo "   clean      - Remove built sources"
	@echo "   fmt        - Format the source code using 'go fmt'"
	@echo "   vet        - Analyze the source code for potential errors"
	@echo "   lint       - Analyze the source code for style mistakes"
	@echo ""

