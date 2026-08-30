BINARY_NAME := schema-api
DIST_DIR    := dist
OS_LIST     := linux windows darwin
ARCH_LIST   := amd64 arm64
GOFLAGS     := -trimpath
PKG         := github.com/arias9306/schema-api
DOC_FILES   := README.md LICENSE

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -s -w \
	-X $(PKG)/version.Version=$(VERSION) \
	-X $(PKG)/version.Commit=$(COMMIT) \
	-X $(PKG)/version.BuildDate=$(BUILD_DATE)

.PHONY: all build test test-race coverage bench lint govulncheck clean changelog changelog-unreleased

all: build

build:
	@mkdir -p $(DIST_DIR)
	@for os in $(OS_LIST); do \
		for arch in $(ARCH_LIST); do \
			ext=""; \
			[ "$$os" = "windows" ] && ext=".exe"; \
			staging="$(DIST_DIR)/staging/$$os-$$arch"; \
			archive="$(CURDIR)/$(DIST_DIR)/$(BINARY_NAME)-$$os-$$arch-$(VERSION)"; \
			echo "Building $$os/$$arch (version $(VERSION))..."; \
			mkdir -p "$$staging"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
				-o "$$staging/$(BINARY_NAME)$$ext" .; \
			cp $(DOC_FILES) "$$staging/"; \
			if [ "$$os" = "windows" ]; then \
				(cd "$$staging" && zip -q "$$archive.zip" "$(BINARY_NAME).exe" $(DOC_FILES)); \
			else \
				tar -czf "$$archive.tar.gz" -C "$$staging" "$(BINARY_NAME)" $(DOC_FILES); \
			fi; \
			rm -rf "$$staging"; \
		done; \
	done
	@rm -rf $(DIST_DIR)/staging

test:
	go test ./...

test-race:
	go test -race ./...

bench:
	go test -run xxx -bench . -benchmem -benchtime=1s ./...

coverage:
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out ./...
	@go tool cover -func=coverage/coverage.out | tail -n 1
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "HTML coverage report written to coverage/coverage.html"

lint:
	golangci-lint run ./...

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

clean:
	rm -rf $(DIST_DIR) coverage

changelog:
	git cliff -o CHANGELOG.md

changelog-unreleased:
	git cliff --unreleased
