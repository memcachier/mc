ifndef GO
GO=go
endif

.PHONY: all

all: mc

mc: src/mc/*.go
	@mkdir -p bin
	@cd bin && GOPATH=$(CURDIR) $(GO) build mc

test: mc
	@GOPATH=$(CURDIR) $(GO) test -v mc

clean:
	go clean
	rm -Rf bin

