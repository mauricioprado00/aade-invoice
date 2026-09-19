COMMANDS := aade-invoice aade-list aade-read aade-template-from-mark aade-income aade-expenses aade-vat aade-e3

# go.mod asks for 1.22, and `go` on the PATH here is a wrapper that runs an old
# toolchain inside Docker, so prefer the real 1.22 install when it is present.
# Override on the command line with `make GO=/path/to/go`.
GO ?= $(firstword $(wildcard /usr/lib/go-1.22/bin/go /usr/lib/go/bin/go) go)

GO_VERSION := $(shell $(GO) version 2>/dev/null)
ifeq ($(filter go1.22% go1.23% go1.24% go1.25% go1.26%,$(word 3,$(GO_VERSION))),)
$(error $(GO) is $(if $(GO_VERSION),$(word 3,$(GO_VERSION)),not a working Go toolchain); this module needs go1.22 or newer — pass GO=/path/to/go)
endif

.PHONY: all clean test
all: $(addprefix bin/,$(COMMANDS))

bin/%: cmd/% $(wildcard internal/*/*.go)
	$(GO) build -o $@ ./$<

test:
	$(GO) vet ./...
	$(GO) test ./...

clean:
	rm -rf bin
