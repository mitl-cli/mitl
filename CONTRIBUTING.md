# Contributing to mitl

Thanks for your interest in improving mitl! This guide describes the basics to get set up, make changes, and submit them.

## Getting Started
- Prerequisites: Go 1.21+ (or newer), a container runtime (Docker/Podman/etc.).
- Clone and build:
  - `make build` (produces `bin/mitl`)
  - `make test` (runs the full test suite)

## Development Workflow
- Format and vet: `make fmt` and `go vet ./...`
- Fast loop: `make dev` (fmt + test + build)
- Run locally: `bin/mitl <command>` (or `go run cmd/mitl/main.go <command>`)
- Useful: `make preflight-light` to validate core UX quickly

## Code Quality & Lint
- See `docs/CODE_QUALITY.md` for our standards (linters, coverage, do's/don'ts).
- Run `make fmt` and `golangci-lint run` locally before pushing.
- For PRs, simulate CI's blocking subset:
  - `golangci-lint run --new-from-rev=origin/main --enable-only=errcheck,staticcheck,govet,gosimple,ineffassign,unused,unparam,noctx,exportloopref,gosec,nolintlint`

## Testing
- All packages: `make test`
- Coverage report: `make test-coverage`
- Benchmarks: `make test-bench` or `mitl bench ...`

## Code Style
- Go code must be formatted with `gofmt -s`.
- Keep functions small and focused; return wrapped errors with context.
- Follow existing naming conventions (exported: CamelCase, unexported: camelCase).

## Commits & Pull Requests
- Use clear, imperative commit messages (e.g., "add digest lockfile support").
- Keep PRs focused and well-described: purpose, changes, and how you validated.
- Include commands used to test locally (e.g., `make test`, `mitl run ...`).

## Issues & Feature Requests
- Open an issue with steps to reproduce, expected vs actual behavior, and environment details.
- For larger proposals, include a brief design summary.

## Security
- Never commit secrets.
- If you believe you’ve found a security issue, please email the maintainers or open a private security advisory.

## Releases (maintainers)
- Tag a release: `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`
- Our CI builds artifacts and can auto-update the Homebrew tap.

Thanks again for contributing!
