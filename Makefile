BINARY_NAME := schema-api
DIST_DIR    := dist
OS_LIST     := linux windows darwin
ARCH_LIST   := amd64 arm64
GOFLAGS     := -trimpath
LDFLAGS     := -s -w

.PHONY: all build test clean

all: build

build:
	@mkdir -p $(DIST_DIR)
	@for os in $(OS_LIST); do \
		for arch in $(ARCH_LIST); do \
			ext=""; \
			[ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
				-o "$(DIST_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext" .; \
		done; \
	done

test:
	go test ./...

clean:
	rm -rf $(DIST_DIR)
