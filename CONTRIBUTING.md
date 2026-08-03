# Contributing to cmdpeek

Thank you for your interest in contributing to `cmdpeek`.

The project aims to remain small, simple, and easy to use. Contributions should preserve that focus.

## Requirements

- The Go version declared in [`go.mod`](go.mod)
- A supported terminal
- Optional external tools required by the example commands you want to test

## Development setup

Clone the repository:

```bash
git clone https://github.com/pierinho13/cmdpeek.git
cd cmdpeek
```

Download dependencies:

```bash
go mod download
```

Run the tests:

```bash
go test ./...
```

Build the project:

```bash
go build -o cmdpeek ./cmd/cmdpeek
```

Run it locally:

```bash
./cmdpeek --config examples/basic.yaml
```

## Code quality

Before opening a pull request, run:

```bash
gofmt -w .
go vet ./...
go test ./...
```

You can also validate the release configuration with:

```bash
goreleaser release --snapshot --clean
```

## Branches

Create a branch from `main`:

```bash
git switch main
git pull --ff-only
git switch -c feat/my-change
```

Use a clear branch name, such as:

```text
feat/add-variable-source
fix/empty-command-options
docs/update-installation
```

## Commits

Use clear and focused commit messages.

Examples:

```text
feat: add command history
fix: handle empty command output
docs: update Homebrew installation
test: cover dependent variables
```

## Pull requests

A pull request should:

- explain the problem or feature
- describe the changes
- include tests when appropriate
- keep unrelated changes out of the same PR
- pass `go test ./...`
- preserve the tool's simple user experience

## Reporting bugs

Use the bug report template and include:

- `cmdpeek` version or commit
- operating system
- terminal and shell
- configuration sample with sensitive values removed
- command-line arguments
- expected behavior
- actual behavior
- steps to reproduce

Do not include tokens, credentials, private configuration contents or other sensitive information.

## Feature requests

Feature requests are welcome, especially when they improve usability without adding unnecessary complexity.

Please describe:

- the problem you are trying to solve
- the expected behavior
- why it fits the scope of `cmdpeek`

## Security issues

Do not report security vulnerabilities in public issues.

See [SECURITY.md](SECURITY.md) for reporting instructions.
