# Development

## Requirements

- Go version declared in `go.mod`
- a supported terminal
- optional external tools required by example commands

## Build

```bash
go build -o cmdpeek ./cmd/cmdpeek
```

## Test

```bash
go test ./...
go test -race ./...
go vet ./...
```

## Formatting

```bash
gofmt -w .
```

Check without modifying files:

```bash
files="$(gofmt -l .)"
test -z "$files" || {
  echo "$files"
  exit 1
}
```

## Run locally

```bash
./cmdpeek --config examples/basic.yaml
```

## Project structure

```text
cmd/cmdpeek          CLI entry point
internal/catalog     YAML loading and validation
internal/template    Variable rendering and previews
internal/variable    Dynamic command option resolution
internal/tui         Catalog, variable and confirmation interfaces
internal/executor    Shell execution
examples             Example command catalogs
```

## Release snapshot

```bash
goreleaser release --snapshot --clean
```
