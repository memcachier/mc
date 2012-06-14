# figure out what GOROOT is supposed to be
GOROOT ?= $(shell printf 't:;@echo $$(GOROOT)\n' | go make -f -)
include $(GOROOT)/src/Make.inc

TARG=github.com/bmizerany/mc.go
GOFILES=\
	mc.go\

include $(GOROOT)/src/Make.pkg
