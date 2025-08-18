# Code Quality Standards

This document establishes our lint rules, testing expectations, and contributor guidelines to keep the codebase robust, readable, and maintainable.

## Philosophy

- Prefer correctness and clarity over cleverness.
- Enforce high-signal issues as blocking; keep style/complexity signals as advisory.
- Gate new code in PRs; track existing debt in scheduled or advisory runs.
- Keep benches/tests useful by excluding noisy checks where appropriate.

## Lint Rules

We use `golangci-lint` v2.x with a two-tier approach.

- Error-tier (blocking):
  - `errcheck`, `staticcheck`, `govet` (without `fieldalignment`), `gosimple`
  - `ineffassign`, `unused`, `unparam`, `noctx`, `exportloopref`
  - `gosec` (with known false-positives silenced via path/text rules)
  - `nolintlint` (strict: explanations and specific linters required)

- Warning-tier (advisory):
  - `gofmt`, `gofumpt`, `goimports`
  - `exhaustive` (use `//exhaustive:ignore` where appropriate)
  - `gocritic` (diagnostic/perf/style; avoid shadow noise in benches)
  - `dupl`, `funlen`, `gocyclo`, `mnd` (raised thresholds; benches/tests excluded)

Additional rules:

- `depguard`: Only denies `github.com/sirupsen/logrus` (use our logger).
- `gomodguard`: Allowed/blocked modules can be added as needed.
- `nolint` hygiene:
  - Always include a specific linter name.
  - Always include a short, actionable reason.

See `.golangci.yml` for the exact configuration and exclusions.

## Test Coverage

- Target overall coverage: 80% (stretch goal: 85%).
- Current enforcement: per-file minimum 85% for `internal/digest` (critical path).
- We will expand package coverage gates incrementally to reach the target.

How to check locally:

```
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
# Per-file min for digest
go run ./internal/tools/covercheck -profile coverage.out -threshold 85 -include mitl/internal/digest
```

## Do’s and Don’ts

Do:

- Write small, focused functions; keep behavior obvious.
- Handle errors explicitly; check returns from I/O and OS calls.
- Add tests for new features and bug fixes.
- Use our logger and error utilities; prefer wrapped errors with context.
- Keep imports ordered; run `make fmt`.

Don’t:

- Introduce global state unnecessarily.
- Silence linters with bare `//nolint`; always specify the linter and a reason.
- Introduce new direct dependencies without discussion.
- Use `logrus` directly (blocked by `depguard`).

## Local Developer Workflow

```
make fmt
golangci-lint run --timeout=5m

# Blocking subset like CI (gate only new changes):
golangci-lint run --new-from-rev=origin/main \
  --enable-only=errcheck,staticcheck,govet,gosimple,ineffassign,unused,unparam,noctx,exportloopref,gosec,nolintlint

go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n1
go run ./internal/tools/covercheck -profile coverage.out -threshold 85 -include mitl/internal/digest
```

## CI Enforcement

- PRs: run a blocking subset (Error-tier) on new changes; run full set as advisory.
- Main: run full advisory lint; keep tests and build blocking.
- Preflight: ensure digest determinism and .mitlignore behavior remain correct.

