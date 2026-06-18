# Top-level Makefile for the cpa-usage-stats plugin. Local builds; the
# release pipeline lives in .github/workflows/release.yml.

PLUGIN_ID := cpa-usage-stats

# Default to the host platform unless overridden.
GOOS  ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
EXT := dll
else ifeq ($(GOOS),darwin)
EXT := dylib
else
EXT := so
endif

OUT := dist/$(PLUGIN_ID).$(EXT)

.PHONY: build clean tidy test fmt

build:
	@mkdir -p dist
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -buildmode=c-shared -o $(OUT) .
	@echo "built $(OUT)"

tidy:
	go mod tidy

test:
	go test ./...

fmt:
	gofmt -w .

clean:
	rm -rf dist
