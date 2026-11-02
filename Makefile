# mirrorctl Makefile
# Go build system. To add a new type module, create a package under internal/ and register it in main.go.

BINARY  ?= mirrorctl
BIN_DIR ?= bin

# Go toolchain
GO      ?= go
GOFLAGS ?= -trimpath
LDFLAGS ?= -s -w

# Cross-compilation: pass GOOS/GOARCH (command line or environment), e.g.
#   make GOOS=linux GOARCH=arm64
# When provided, the artifact is named $(BINARY)-$(GOOS)-$(GOARCH)[.exe];
# a plain `make` builds for the host as $(BIN_DIR)/$(BINARY).
GOOS   ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

ifeq ($(GOOS),windows)
EXT := .exe
endif

ifneq (,$(strip $(filter-out undefined default file,$(origin GOOS) $(origin GOARCH))))
TARGET := $(BIN_DIR)/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT)
else
TARGET := $(BIN_DIR)/$(BINARY)$(EXT)
endif

# Platforms built by `make dist` (override to customize).
PLATFORMS ?= \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: all build dist run test vet fmt clean help

all: build

build: $(TARGET)

$(TARGET): $(shell find . -name '*.go') go.mod | $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ .

# Build all release platforms into bin/ (used by CI and local release builds).
# Tries upx compression for linux binaries when upx is available; non-fatal.
dist:
	@set -e; for p in $(PLATFORMS); do \
		os=$${p%%/*}; arch=$${p##*/}; \
		echo "==> build $$os/$$arch"; \
		$(MAKE) --no-print-directory GOOS=$$os GOARCH=$$arch build; \
		out=$(BIN_DIR)/$(BINARY)-$$os-$$arch; \
		if [ "$$os" = windows ]; then out=$$out.exe; fi; \
		if [ "$$os" = linux ] && command -v upx >/dev/null 2>&1; then \
			upx --best --no-color -q -o "$$out.packed" "$$out" && mv "$$out.packed" "$$out" || rm -f "$$out.packed"; \
		fi; \
	done

$(BIN_DIR):
	@mkdir -p $@

# Quick run: make run ARGS="pypi list"
run: build
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
	@echo "make                    Build ./bin/$(BINARY) for the host"
	@echo "make GOOS=.. GOARCH=..  Cross-compile (e.g. make GOOS=linux GOARCH=arm64)"
	@echo "make dist               Build all release platforms into bin/"
	@echo "make run ARGS=..        Run (e.g. make run ARGS=\"pypi list\")"
	@echo "make test               Run unit tests"
	@echo "make vet                Static analysis"
	@echo "make fmt                Format source code"
	@echo "make clean              Clean build artifacts and cache"
