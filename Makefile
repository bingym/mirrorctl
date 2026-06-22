# mirrorctl Makefile
# Go build system. To add a new type module, create a package under internal/ and register it in main.go.

BINARY  := mirrorctl
BIN_DIR := bin
TARGET  := $(BIN_DIR)/$(BINARY)
MODULE  := github.com/mirrorctl

# Go toolchain
GO      ?= go
GOFLAGS ?= -trimpath
LDFLAGS ?= -s -w

.PHONY: all clean run test vet fmt help

all: $(TARGET)

$(TARGET): $(shell find . -name '*.go') go.mod | $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ .
	@if command -v upx >/dev/null 2>&1; then \
		upx --best --no-color -q -o $@.packed $@ && mv $@.packed $@; \
	fi

$(BIN_DIR):
	@mkdir -p $@

# Quick run: make run ARGS="pypi list"
run: $(TARGET)
	./$(TARGET) $(ARGS)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BIN_DIR)
	$(GO) clean -cache -testcache

help:
	@echo "make            Build ./bin/mirrorctl"
	@echo "make run ARGS=.. Run (e.g. make run ARGS=\"pypi list\")"
	@echo "make test       Run unit tests"
	@echo "make vet        Static analysis"
	@echo "make fmt        Format source code"
	@echo "make clean      Clean build artifacts and cache"
