GO_VERSION := 1.25

.PHONY: build test lint vet check clean setup setup-microvm build-microvm test-microvm

ARAPUCA_DIR ?= ../arapuca
PREFIX      ?= $(HOME)/.local

build:
	CGO_ENABLED=1 go build ./...

build-microvm:
	CGO_ENABLED=1 go build -tags microvm ./...

test:
	CGO_ENABLED=1 go test -v ./...

test-microvm:
	CGO_ENABLED=1 go test -v -tags microvm ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

check: vet test

clean:
	go clean -cache

# Build and install core arapuca (no microvm deps).
setup:
	cd $(ARAPUCA_DIR) && make install PREFIX=$(PREFIX)
	@echo "Done. Run: export PKG_CONFIG_PATH=$(PREFIX)/lib/pkgconfig"

# Build and install arapuca with micro-VM support (requires libkrun, openssl).
setup-microvm:
	cd $(ARAPUCA_DIR) && make install INSTALL_FEATURES=microvm PREFIX=$(PREFIX)
	@echo "Done. Run: export PKG_CONFIG_PATH=$(PREFIX)/lib/pkgconfig"
	@echo "Build with: go build -tags microvm ./..."
