GO_VERSION := 1.25

.PHONY: build test lint vet check clean setup

ARAPUCA_DIR ?= ../arapuca
PREFIX      ?= $(HOME)/.local

build:
	CGO_ENABLED=1 go build ./...

test:
	CGO_ENABLED=1 go test -v ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

check: vet test

clean:
	go clean -cache

# Build and install arapuca from a local checkout.
# After running this, set PKG_CONFIG_PATH=$(PREFIX)/lib/pkgconfig
setup:
	cd $(ARAPUCA_DIR) && make install PREFIX=$(PREFIX)
	@echo "Done. Run: export PKG_CONFIG_PATH=$(PREFIX)/lib/pkgconfig"
