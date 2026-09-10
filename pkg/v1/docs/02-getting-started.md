# 2. Getting started

## Install

```bash
go get github.com/XWinterVarit/integrate_tester_v2/pkg/v1
```

The library imports the database drivers you choose, so you must import the
drivers yourself (for `database/sql`):

```go
import (
    _ "github.com/mattn/go-sqlite3"   // sqlite
    _ "github.com/sijms/go-ora/v2"    // oracle
)
```

## Minimal test

Create `cmd/itest/main.go` (or any `main` package):

```go
package main

import (
    "net/http"

    v1 "github.com/XWinterVarit/integrate_tester_v2/pkg/v1"
)

func main() {
    t := v1.NewTester()

    t.Stage("Health check", func() {
        resp := v1.SendRequest("http://localhost:8080/health")
        v1.ExpectStatusCode(resp, http.StatusOK)
        v1.ExpectJsonBodyField(resp, "status", "ok")
    })

    t.Stage("Cleanup", func() {
        // stop servers, close connections, ...
    })

    v1.Run(t)
}
```

## Pick the mode at launch

Register the `-mode` flag before `flag.Parse`, then call `v1.Run`:

```go
func main() {
    v1.RegisterModeFlag("")
    flag.Parse()

    t := v1.NewTester()
    // ... register stages ...

    v1.Run(t)
}
```

| Mode | Flag | Purpose |
|------|------|---------|
| `gui` (default) | `-mode gui` | Electron desktop UI for a human to click stages |
| `cli` | `-mode cli` | Run all stages sequentially (CI), exit `1` on failure |
| `cli-command` | `-mode cli-command` | Interactive stdin/stdout session for an AI/debugger |
| `server` | `-mode server` | HTTP API + browser UI |

The mode can also come from the environment:

```bash
INTEGRATION_TESTER_MODE=cli go run .   # IT_MODE also works
```

If you prefer your own flag, skip `RegisterModeFlag` and call
`v1.RunWithMode(t, mode)`.

## Run it

```bash
go run . -mode cli
```

```text
=== Integration Test (CLI Mode) ===

[STAGE] Health check
  PASSED (12ms)

[STAGE] Cleanup
  PASSED (1ms)

=== Results: 2/2 stages passed ===
```

## GUI mode

The GUI is an Electron + React app that talks to a small HTTP/SSE server started
by the Go process. Build the frontend once:

```bash
cd ui && npm install && npm run build
```

`v1.Run` (or `RunGUI`) finds the `ui/` folder automatically. Override with
`INTEGRATION_TESTER_UI=<path>` or point at a built bundle via `IT_UI_DIST`.
If Electron is unavailable, the same UI opens in the default browser from
`ui/dist`. See [Run modes](08-run-modes.md) for details.

## Running the shipped example

A complete end-to-end example lives in
[`pkg/v1/example_app/integration_test/main.go`](../example_app/integration_test/main.go).
It builds and starts a sample service, drives it over HTTP, checks a database
and Redis, and exercises the dynamic mock server.

```bash
# from the repo root
cd pkg/v1
go run ./example_app/integration_test -mode cli
```

Next: [Designing a test flow →](03-designing-a-test-flow.md)
