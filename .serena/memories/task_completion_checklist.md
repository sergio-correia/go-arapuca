# Task Completion Checklist

After completing any code change, run:

1. **Format**: `gofmt -l -w .`
2. **Vet**: `CGO_ENABLED=1 go vet ./...`
3. **Build**: `CGO_ENABLED=1 go build ./...`
4. **Test**: `CGO_ENABLED=1 go test -v ./...`
5. **Module tidy**: `go mod tidy` (if dependencies changed)

If modifying micro-VM code, also run with `-tags microvm`:
- `CGO_ENABLED=1 go build -tags microvm ./...`
- `CGO_ENABLED=1 go test -v -tags microvm ./...`

## CI Checks (all must pass)
- Build
- Test
- go vet
- gofmt (no unformatted files)
- go mod tidy (go.mod unchanged)

## Notes
- `CGO_ENABLED=1` is mandatory for all Go commands
- `PKG_CONFIG_PATH` must include the path to `arapuca.pc`
- Pre-commit hooks will run vet, gofmt, build, and test automatically
