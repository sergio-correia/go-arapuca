GO_VERSION := 1.25

.PHONY: build test lint vet check clean update-lib

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

# Update the vendored static library from a local arapuca build.
# Usage: make update-lib ARAPUCA_DIR=../arapuca
ARAPUCA_DIR ?= ../arapuca

update-lib:
	cd $(ARAPUCA_DIR) && cargo build --release
	cp $(ARAPUCA_DIR)/target/release/libarapuca.a lib/linux_amd64/
	cp $(ARAPUCA_DIR)/include/arapuca.h lib/
	@echo "Updated lib from $(ARAPUCA_DIR)"
