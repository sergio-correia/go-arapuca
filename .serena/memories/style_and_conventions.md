# Style & Conventions

## Code Style
- **Formatter**: `gofmt` (standard Go formatting, enforced by CI and pre-commit)
- **Package**: Single package `arapuca` — no sub-packages with code
- **Naming**: Standard Go conventions (exported = PascalCase, unexported = camelCase)

## Error Handling
- All errors prefixed with `"arapuca: "` — maintain this convention
- Thread-local errors from C: always wrap cgo calls in `runtime.LockOSThread()` / `runtime.UnlockOSThread()` before calling `lastError()`

## Documentation
- Package-level doc comment at top of `arapuca.go`
- All exported functions/types have doc comments
- Comments are descriptive, not redundant

## Test Naming
- `Test<Feature>` — basic unit test
- `Test<Feature>WithWrapper` — variant exercising the Landlock wrapper binary path
- Tests that need external binaries call `t.Skip()` if not available

## Build Tags
- `microvm && linux` for full micro-VM code (`microvm.go`)
- `!microvm || !linux` for stubs (`microvm_stub.go`)
- Complementary constraints — one is always active

## CGo Patterns
- `C.CString()` + deferred `C.free()` for string arguments
- `runtime.KeepAlive()` on `*os.File` after launch to prevent GC closing FDs
- `setConfigStr()` helper for DRY string-field setting on C config struct
