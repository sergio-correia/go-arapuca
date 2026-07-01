# go-arapuca — Project Overview

## Purpose
Go bindings for [arapuca](https://github.com/sergio-correia/arapuca), a Linux/macOS process sandbox.
Wraps the C API (which is itself a Rust static library) via cgo to provide idiomatic Go types for launching sandboxed subprocesses.

## Tech Stack
- **Language**: Go 1.25+ (single package `arapuca`, no sub-packages with code)
- **FFI**: cgo linking to `libarapuca.a` (Rust static library) via pkg-config
- **Dependencies**: Zero Go dependencies (only stdlib + cgo)
- **Build system**: GNU Make
- **CI**: GitHub Actions on Fedora container
- **Pre-commit**: go vet, gofmt, go build, go test

## What arapuca enforces
- **Linux**: Landlock filesystem restrictions, seccomp BPF, cgroups v2 (memory, CPU, PIDs), network namespace isolation, rlimits, pdeathsig, setsid, env sanitization
- **macOS**: sandbox-exec (Seatbelt) with deny-default profiles, rlimits, memory polling, parent-PID watchdog

## Codebase Structure
```
arapuca.go          — All public API: Sandbox, Process, Config, Profile, ResourceUsage, utility functions
microvm.go          — MicroVM functions (build tag: microvm && linux)
microvm_stub.go     — Stubs when built without microvm (build tag: !microvm || !linux)
arapuca_test.go     — Unit tests for sandbox lifecycle and utility probes
stdio_test.go       — Integration tests for stdio pipe redirection
stdio_go_test.go    — Integration tests launching Go binary through sandbox (needs /tmp/arapuca-probe)
Makefile            — Build, test, lint, setup targets
wtmcp/              — Empty directory scaffolding (placeholder)
.github/workflows/  — CI configuration
```

## Key Architecture Patterns
- Single-file cgo binding (`arapuca.go`) — all public types and methods in one file
- Build-tag based conditional compilation for micro-VM support (microvm.go vs microvm_stub.go)
- Thread-local error checking: every cgo call that can fail uses `runtime.LockOSThread()`/`runtime.UnlockOSThread()`
- GC safety: `runtime.KeepAlive()` on `*os.File` values after C launch call to prevent premature FD closure
- Context cancellation sends SIGKILL to the process group (`kill(-pid, SIGKILL)`)
- Both `Sandbox` and `Process` are mutex-protected for concurrent use
