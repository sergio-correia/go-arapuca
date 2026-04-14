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

## Install

```bash
go get github.com/sergio-correia/go-arapuca
```

Requires `CGO_ENABLED=1` and a C compiler (gcc or clang). The static
library (`libarapuca.a`) is vendored via git-lfs — no Rust toolchain
needed.

## Usage

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
    MaxMemoryMB           uint64  // 0 = no limit
    MaxCPUPct             uint32  // 0 = no limit; 200 = 2 cores
    MaxPIDs               uint32
    MaxFileSizeMB         uint64
    UseNetNS              bool
}

type Config struct {
    Profile            Profile
    SocketDir          string
    TaskID             string
    Phase              string
    WorkDir            string
    Stdin              *os.File  // nil = inherit
    Stdout, Stderr     *os.File  // nil = inherit
    NetworkProxySocket string
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

## Building from source

To update the vendored `libarapuca.a` from a local arapuca build:

```bash
make update-lib ARAPUCA_DIR=../arapuca
```

## License

Apache-2.0
