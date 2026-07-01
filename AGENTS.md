# AGENTS.md

## Overview

Go bindings for [arapuca](https://github.com/sergio-correia/arapuca), a Linux/macOS process sandbox. This is a **cgo wrapper library** — all runtime functionality lives in a Rust static library (`libarapuca.a`) linked via pkg-config. There are zero Go dependencies beyond the standard library.

## Commands

```bash
# Build (CGO_ENABLED=1 is mandatory — this is a cgo project)
make build                    # core sandbox
make build-microvm            # with micro-VM support (-tags microvm)

# Test
make test                     # core tests
make test-microvm             # with micro-VM tests

# Lint / format
make lint                     # golangci-lint
make vet                      # go vet
gofmt -l -w .                 # format (CI enforces this)

# Full check (vet + test)
make check
```

## Build Prerequisites — Read This First

The C library **must be installed** before any Go command works. Without `libarapuca.a` + `arapuca.pc` on the pkg-config path, `go build`, `go test`, `go vet`, and even `gopls` will all fail with "Package arapuca was not found." This is the single most common issue.

```bash
# Install the C library (needs Rust/cargo, gcc, make, cbindgen, pkg-config)
make setup ARAPUCA_DIR=/path/to/arapuca   # defaults to ../arapuca
export PKG_CONFIG_PATH=$HOME/.local/lib/pkgconfig

# For micro-VM support (also needs libkrun-devel, openssl-devel)
make setup-microvm ARAPUCA_DIR=/path/to/arapuca
```

CI clones and builds arapuca from source in a Fedora container (see `.github/workflows/ci.yaml`).

## Architecture

### File Layout

| File | Purpose |
|---|---|
| `arapuca.go` | All public API: `Sandbox`, `Process`, types (`Config`, `Profile`, `ResourceUsage`), utility functions. Single-file cgo binding. |
| `microvm.go` | Micro-VM functions (`MicroVmAvailable`, `ImagePull`, `applyIsolation`). Build tag: `microvm && linux`. |
| `microvm_stub.go` | Stub returning errors/false when built without `microvm` tag. Build tag: `!microvm \|\| !linux`. |
| `arapuca_test.go` | Unit tests for sandbox lifecycle, utility probes. |
| `stdio_test.go` | Integration tests for stdio pipe redirection (echo, cat relay, with/without Landlock wrapper). |
| `stdio_go_test.go` | Integration tests launching a Go binary through the sandbox. Requires a pre-built probe binary at `/tmp/arapuca-probe`; tests skip if absent. |
| `wtmcp/` | Empty directory scaffolding (placeholder for future work). |

### Control Flow

1. `New()` → creates `arapuca_ArapucaSandbox` handle via C FFI
2. `sb.Launch(ctx, cfg, cmd, args, extraFiles)` → builds C profile/config structs, calls `arapuca_launch()`, returns `*Process`
3. A goroutine watches `ctx.Done()` and sends `SIGKILL` to the process group on cancellation
4. `proc.Wait()` → blocks on `arapuca_process_wait()`, signals the cancellation goroutine via `done` channel
5. `proc.ResourceStats()` / `proc.OOMCount()` → read cgroup stats (must call before `Cleanup`)
6. `proc.Cleanup()` → releases C resources (cgroup, temp dir)

### Thread-Local Error Handling

The C library uses thread-local storage for error messages (`arapuca_last_error()`). Every cgo call that can fail is wrapped in `runtime.LockOSThread()` / `runtime.UnlockOSThread()` to prevent the Go scheduler from migrating the goroutine between the call and the error check. **Any new cgo call that checks errors must follow this pattern.**

### GC / File Descriptor Safety

After `arapuca_launch()`, `runtime.KeepAlive()` is called on all `*os.File` values (`Stdin`, `Stdout`, `Stderr`, `extraFiles`) to prevent the GC from finalizing them (and closing the underlying FDs) before the C code has duplicated them via `F_DUPFD_CLOEXEC`.

## Conventions

- **Single package**: everything is in package `arapuca` (no sub-packages with code yet)
- **Build tags** control micro-VM support: `microvm.go` vs `microvm_stub.go` use complementary build constraints
- **No Go dependencies**: `go.mod` has zero `require` directives; all functionality comes from the C library
- **Error style**: errors are prefixed with `"arapuca: "` — maintain this convention
- **Test naming**: `Test<Feature>` for unit tests, `Test<Feature>WithWrapper` for variants that exercise the Landlock wrapper binary path
- **Formatting**: `gofmt` enforced by CI and pre-commit hooks — no custom formatter config
- **CI checks**: build, test, vet, gofmt, `go mod tidy` — all must pass

## Gotchas

- **`CGO_ENABLED=1` is always required.** The Makefile sets it explicitly; bare `go build ./...` will fail if your environment defaults to `CGO_ENABLED=0`.
- **Wrapper-dependent tests**: tests with `WithWrapper` in the name call `WrapperPath()` and skip if the `arapuca` binary isn't on `$PATH`. The wrapper is only needed when `ReadPaths` or `WritePaths` are set (triggers Landlock).
- **`stdio_go_test.go` tests require a pre-built probe binary** at `/tmp/arapuca-probe`. They skip gracefully if it's missing, but will always skip in CI unless you build it first.
- **Context cancellation kills the process group** (`kill(-pid, SIGKILL)`), not just the process. This is intentional for sandbox cleanup.
- **`ResourceStats()` and `OOMCount()` must be called before `Cleanup()`** — after cleanup the C process handle is nil and they return zero values silently.
- **`go.sum` is absent** — this is expected since there are no Go dependencies. CI runs `go mod tidy` and diffs `go.mod` only.
- **The `wtmcp/` directory** is empty scaffolding — no code there yet.
