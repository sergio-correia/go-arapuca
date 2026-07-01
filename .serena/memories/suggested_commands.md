# Suggested Commands

## Prerequisites
The C library `libarapuca.a` MUST be installed before any Go command works.
```bash
# Install C library (needs Rust/cargo, gcc, make, cbindgen, pkg-config)
make setup ARAPUCA_DIR=../arapuca
export PKG_CONFIG_PATH=$HOME/.local/lib/pkgconfig

# With micro-VM support (also needs libkrun-devel, openssl-devel)
make setup-microvm ARAPUCA_DIR=../arapuca
```

## Build
```bash
make build                     # CGO_ENABLED=1 go build ./...
make build-microvm             # CGO_ENABLED=1 go build -tags microvm ./...
```

## Test
```bash
make test                      # CGO_ENABLED=1 go test -v ./...
make test-microvm              # CGO_ENABLED=1 go test -v -tags microvm ./...
```

## Lint & Format
```bash
make lint                      # golangci-lint run ./...
make vet                       # go vet ./...
gofmt -l -w .                  # format all Go files
```

## Combined Check
```bash
make check                     # vet + test
```

## Clean
```bash
make clean                     # go clean -cache
```

## Important Notes
- `CGO_ENABLED=1` is **always required** — the Makefile sets it, but bare `go` commands may not
- CI checks: build, test, vet, gofmt, go mod tidy
- Pre-commit hooks run: go vet, gofmt, go build, go test
