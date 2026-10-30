---
description: Run every check this repository has — Go and both apps
allowed-tools: Bash(make check), Bash(make lint-go), Bash(make lint-web), Bash(make test-go), Bash(make test-web), Bash(make test-integration), Bash(make fmt), Bash(gofmt:*), Bash(bunx prettier:*), Read, Edit, Glob, Grep
---

Run the checks, in this order, and fix what they find:

1. `make check` — formatting, go vet, staticcheck, both apps' lint and
   type-check, and every unit test. When it fails, `make lint-go`,
   `lint-web`, `test-go` or `test-web` reruns just that part.
2. `make test-integration` — the `TestLive…` tests, on throwaway Postgres
   databases. Run it when the change touched a migration, the store, or the
   route table in `internal/interfaces/api/server.go`; say so if you skip it.

Formatting is not a judgement call: `make fmt` formats everything, or
`gofmt -w <file>` and, inside the app the file belongs to,
`bunx prettier --write <file>`.

Report what passed and what did not. If something fails, fix the cause rather
than the test, unless the test is the thing that is wrong — and say which you
decided it was.
