# 8. Run modes

The same test can be driven in four ways. Pick at launch (`-mode`) or in code.

| Mode | Constant | Flag | Use case |
|------|----------|------|----------|
| GUI | `v1.ModeGUI` | `-mode gui` | Human clicks stages in the Electron app (default) |
| CLI | `v1.ModeCLI` | `-mode cli` | Run all stages for CI, exit non-zero on failure |
| CLI command | `v1.ModeCLICommand` | `-mode cli-command` | Interactive session for an AI/debugger |
| Server | `v1.ModeServer` | `-mode server` | HTTP API + browser UI |

## Selecting the mode

```go
func main() {
    v1.RegisterModeFlag("") // registers -mode on flag.CommandLine
    flag.Parse()

    t := v1.NewTester()
    // ... stages ...

    v1.Run(t) // dispatch on CurrentMode()
}
```

| Function | Purpose |
|----------|---------|
| `RegisterModeFlag(usage string)` | register `-mode` (default from env, else `gui`) |
| `CurrentMode() string` | resolve from flag or env |
| `Run(t *Tester)` | dispatch using `CurrentMode()` |
| `RunWithMode(t *Tester, mode string)` | dispatch an explicit mode |

Environment (used when no flag is given, or as the flag default):
`INTEGRATION_TESTER_MODE` or `IT_MODE`.

```bash
go run . -mode cli
INTEGRATION_TESTER_MODE=server go run .
```

## GUI mode

- Starts the HTTP server (below), then launches Electron pointing at `ui/dist`.
- Falls back to the default browser if Electron is not found.
- Locates the UI by searching upward for `ui/electron/main.cjs`; override with
  `INTEGRATION_TESTER_UI`, or set `IT_UI_DIST` to a built `index.html`.
- Requires the frontend to be built once: `cd ui && npm install && npm run build`.
- Blocks until the window closes.

Direct entry points: `RunGUI(t)`, `RunElectron(t)`.

## Server mode

Starts only the HTTP API (+ serves `ui/dist` at `/`) and opens the browser.
Blocks until interrupted.

```go
import "log"

srv := v1.NewUIServer(t)
if err := srv.Start(); err != nil {
    log.Fatal(err)
}
defer srv.Stop()
log.Println("UI at", srv.URL()) // e.g. http://127.0.0.1:54321
```

### HTTP API

| Method + path | Body | Result |
|---------------|------|--------|
| `GET /api/health` | — | `{"status":"ok"}` |
| `GET /api/state` | — | full snapshot (see below) |
| `GET /api/events` | — | SSE stream |
| `POST /api/stage/run` | `{"name":"Setup","runPrerequisites":true}` | `202` accepted |
| `POST /api/action/run` | `{"stage":"Setup","index":0}` | `202` accepted |
| `POST /api/run-all` | `{}` | `202` accepted |
| `POST /api/discover` | `{}` | `202` accepted (dry-run) |

Running endpoints return immediately (`202`); results arrive over SSE. Uses
`TryRunStageByName`-style serialization under the hood, so requests never
overlap.

`GET /api/state`:

```json
{
  "stages": [
    { "name": "Setup", "status": "PASSED", "actions": [{ "index": 0, "summary": "DB SetupTable: users" }] }
  ],
  "logs": [
    { "type": "Stage", "summary": "Running Stage: Setup", "detail": "" }
  ]
}
```

### SSE events (`GET /api/events`)

Each `data:` line is a JSON object with a `type` field:

| `type` | Extra fields | Meaning |
|--------|--------------|---------|
| `state` | `state` | full snapshot, sent on connect |
| `log` | `log` | a new `LogEntry` |
| `stage` | `stage`, `status` | a stage's status changed |
| `refresh` | — | actions changed; re-fetch `/api/state` |
| `error` | `stage`, `message` | a run failed |

Status values: `Not Run`, `Running...`, `PASSED`, `FAILED`.

## CLI mode (run all)

```go
v1.RunCLI(t)
```

Runs every stage in order, prints `[STAGE] name` + `PASSED/FAILED (duration)`,
and exits `1` if any stage failed. Ideal for CI.

```text
=== Integration Test (CLI Mode) ===

[STAGE] Setup
  PASSED (1.2s)

[STAGE] Success Case
  PASSED (345ms)

=== Results: 2/2 stages passed ===
```

## CLI command mode (AI / debugging)

An interactive, line-oriented stdin/stdout session. See the
[AI agent guide](09-ai-agent-guide.md) for the full protocol. Summary:

- Commands: `list`, `run <stage>`, `status`, `exit`/`quit`.
- Streams `RUNNING` / `HEARTBEAT` / `SLOW` / `PASSED` / `FAILED` events, and
  answers `status` while a stage runs.
- Rejects a second concurrent `run` with `BUSY`.
- Every response ends with a blank line.

```go
v1.RunCLICommand(t)                                  // defaults
v1.RunCLICommandWithOptions(t, v1.CLICommandOptions{ // timing control
    Heartbeat: 5 * time.Second,
    SlowAfter: time.Minute,
    Timeout:   0, // 0 = disabled
})
```

Env overrides: `INTEGRATION_TESTER_CLI_HEARTBEAT`,
`INTEGRATION_TESTER_CLI_SLOW_AFTER`, `INTEGRATION_TESTER_CLI_TIMEOUT`.

## Concurrency (all modes)

Stages are serialized by the tester itself:

- `RunStageByName` blocks until any current stage finishes.
- `TryRunStageByName` returns `v1.ErrStageRunning` instead of waiting.
- `RunningStage()` returns the active stage name or `""`.

So no mode can ever run two stages at the same time.

Next: [AI agent guide →](09-ai-agent-guide.md)
