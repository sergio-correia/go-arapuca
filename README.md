# go-arapuca

Go bindings for [arapuca](https://github.com/sergio-correia/arapuca),
a Linux/macOS process sandbox. Wraps the C API via cgo to provide
idiomatic Go types for launching sandboxed subprocesses.

## What arapuca enforces

On **Linux**: Landlock filesystem restrictions, seccomp BPF syscall
filtering, cgroups v2 resource limits (memory, CPU, PIDs), network
namespace isolation, rlimits, pdeathsig, setsid, environment sanitization.

On **macOS**: Apple's sandbox-exec (Seatbelt) with deny-default profiles,
rlimits, memory polling, parent-PID watchdog.

## Prerequisites

- Go 1.25+ with `CGO_ENABLED=1`
- C compiler (gcc or clang)
- pkg-config
- `libarapuca.a`, `arapuca.h`, and `arapuca.pc` installed where
  pkg-config can find them (see [Building the C library](#building-the-c-library))

## Install

```bash
go get github.com/sergio-correia/go-arapuca
```

## Versioning

go-arapuca links against arapuca's C ABI via pkg-config. There is no
automatic version coupling — you must keep them in sync. Each
go-arapuca release documents the minimum arapuca version it requires:

| go-arapuca | arapuca (min) |
|------------|---------------|
| v0.2.0+    | v0.1.1        |
| v0.1.x     | v0.1.0        |

If you see link errors or crashes, rebuild and reinstall the C
library first.

## Usage

### Process-level sandbox

```go
package main

import (
    "context"
    "fmt"
    "log"

    arapuca "github.com/sergio-correia/go-arapuca"
)

func main() {
    sb, err := arapuca.New()
    if err != nil {
        log.Fatal(err)
    }
    defer sb.Close()

    cfg := arapuca.Config{
        Profile: arapuca.Profile{
            ReadPaths:   []string{"/usr", "/lib", "/lib64", "/bin", "/etc", "/dev"},
            WritePaths:  []string{"/tmp/workspace"},
            MaxMemoryMB: 2048,
            MaxPIDs:     256,
            MaxCPUPct:   200, // 2 cores
            UseNetNS:    true,
        },
        TaskID:  "task-42",
        Phase:   "execute",
        WorkDir: "/tmp/workspace",
    }

    proc, err := sb.Launch(context.Background(), cfg, "/usr/bin/python3", []string{"agent.py"}, nil)
    if err != nil {
        log.Fatal(err)
    }

    exitCode, err := proc.Wait()
    if err != nil {
        log.Printf("process error: %v", err)
    }

    stats := proc.ResourceStats()
    fmt.Printf("exit=%d peak_memory=%d bytes\n", exitCode, stats.MemoryPeakBytes)

    proc.Cleanup()
}
```

### Micro-VM isolation (Linux only, requires `-tags microvm`)

```go
package main

import (
    "context"
    "fmt"
    "log"

    arapuca "github.com/sergio-correia/go-arapuca"
)

func main() {
    if !arapuca.MicroVmAvailable() {
        log.Fatal("micro-VM isolation not available (no KVM?)")
    }

    // Pull image (~500MB first time, cached for reuse).
    imagePath, err := arapuca.ImagePull("fedora", "42")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("image:", imagePath)

    sb, err := arapuca.New()
    if err != nil {
        log.Fatal(err)
    }
    defer sb.Close()

    cfg := arapuca.Config{
        Profile: arapuca.Profile{
            ReadPaths:  []string{"/home/user/project"},
            WritePaths: []string{"/tmp/workspace"},
            Isolation: &arapuca.MicroVmIsolation{
                Distro:  "fedora",
                Version: "42",
                CPUs:    2,
                MemMB:   2048,
            },
        },
        TaskID: "vm-task-1",
        Phase:  "execute",
    }

    proc, err := sb.Launch(context.Background(), cfg, "make", []string{"test"}, nil)
    if err != nil {
        log.Fatal(err)
    }

    exitCode, _ := proc.Wait()
    fmt.Printf("exit=%d\n", exitCode)
    proc.Cleanup()
}
```

## API

### Sandbox

```go
arapuca.New() (*Sandbox, error)           // create sandbox handle
sb.Launch(ctx, cfg, cmd, args, extraFiles)  // launch sandboxed subprocess
sb.CgroupsAvailable() bool                 // probe cgroups v2
sb.Close()                                 // release handle
```

### Process

```go
proc.PID() int                     // subprocess PID
proc.Wait() (int, error)           // wait for exit (code, error)
proc.ResourceStats() ResourceUsage // cgroup stats (before Cleanup)
proc.OOMCount() int                // OOM kill count (before Cleanup)
proc.Cleanup()                     // release resources
```

### Micro-VM (requires `-tags microvm`)

```go
arapuca.MicroVmAvailable() bool                  // probe KVM + qemu-img
arapuca.ImagePull(distro, version) (string, error) // download and cache image
```

Without the build tag, `MicroVmAvailable()` returns false and
`ImagePull()` returns an error.

### Utilities

```go
arapuca.MakeSocketDir() (string, error)  // temp dir for Unix sockets
arapuca.MakeTmpDir(taskID) (string, error)
arapuca.WrapperPath() string             // find arapuca binary
arapuca.LandlockABIVersion() int         // 0 if unavailable
arapuca.NetNSAvailable() bool
arapuca.DiskUsageMB(path) uint64
```

### Types

```go
type Profile struct {
    ReadPaths, WritePaths []string
    MaxMemoryMB           uint64             // 0 = no limit
    MaxCPUPct             uint32             // 0 = no limit; 200 = 2 cores
    MaxPIDs               uint32
    MaxFileSizeMB         uint64
    UseNetNS              bool
    Isolation             *MicroVmIsolation  // nil = process-level sandbox
}

type MicroVmIsolation struct {
    Distro    string  // e.g. "fedora", "centos" (or use ImagePath)
    Version   string  // e.g. "42", "9"
    ImagePath string  // explicit qcow2 path (alternative to Distro/Version)
    CPUs      uint32
    MemMB     uint32
}

type Config struct {
    Profile            Profile
    SocketDir          string
    TaskID             string
    Phase              string
    WorkDir            string
    Stdin              *os.File          // nil = inherit
    Stdout, Stderr     *os.File          // nil = inherit
    NetworkProxySocket string
    Env                map[string]string // extra env vars for subprocess
}

type ResourceUsage struct {
    MemoryCurrentBytes, MemoryPeakBytes int64
    CPUUsageSeconds                     float64
    PIDCount                            int64
    IOReadBytes, IOWriteBytes           int64
}
```

## Thread Safety

- `Sandbox` is safe for concurrent use (mutex-protected).
- `Process` methods are safe for concurrent use (mutex-protected).
- Error checking uses `runtime.LockOSThread()` to pin goroutines
  to OS threads (arapuca uses thread-local error storage).
- Context cancellation sends SIGKILL to the process group.

## Building the C library

go-arapuca links against `libarapuca.a` (a Rust static library)
discovered via pkg-config. Two build modes are available:

### Core sandbox (all platforms)

```bash
make setup
```

Installs process-level sandbox support only. No libkrun or OpenSSL
development packages required.

### With micro-VM support (Linux only)

```bash
make setup-microvm
```

Installs with micro-VM isolation via libkrun. Requires `libkrun-devel`
and `openssl-devel` (or equivalent) packages. Build Go code with
`-tags microvm`:

```bash
go build -tags microvm ./...
```

### Build matrix

| arapuca install      | go-arapuca build                | Result                      |
|----------------------|---------------------------------|-----------------------------|
| `make setup`         | `go build`                      | core sandbox only           |
| `make setup`         | `go build -tags microvm`        | **link error**              |
| `make setup-microvm` | `go build`                      | core sandbox only           |
| `make setup-microvm` | `go build -tags microvm`        | full micro-VM support       |

### Custom paths

```bash
# Custom arapuca checkout and install prefix
make setup ARAPUCA_DIR=/path/to/arapuca PREFIX=/opt/arapuca

# Ensure pkg-config finds the installed library
export PKG_CONFIG_PATH=$HOME/.local/lib/pkgconfig
```

Re-run `make setup` (or `make setup-microvm`) whenever the arapuca
library changes.

## License

Apache-2.0
