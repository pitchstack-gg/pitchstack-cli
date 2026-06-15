BIN ?= pitchstack
PKG ?= ./cmd/pitchstack

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)

LDFLAGS = -s -w \
	-X github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo.Version=$(VERSION) \
	-X github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo.Commit=$(COMMIT)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o ./bin/$(BIN) $(PKG)

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: test
test:
	go test ./...

.PHONY: release-check
release-check:
	go run github.com/goreleaser/goreleaser/v2@latest check

.PHONY: release-snapshot
release-snapshot:
	go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish
