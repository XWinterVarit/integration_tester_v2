### Overview of the `pkg/v1` package

> **New to the library?** Start with the task-oriented guide in
> [`docs/`](docs/README.md) — it covers designing a test flow, assertions,
> HTTP/DB/Redis helpers, mocks, run modes, and an AI-agent guide for the CLI
> command mode. This README is the API-level reference.

This package provides building blocks for writing integration tests in Go. It helps you:

- Organize tests into **stages**.
- Record and display **actions** for each stage (especially useful for GUIs or CLIs).
- Use consistent **assertions** and **logging**.
- Work with **HTTP**, **databases**, and **external apps** in a test‑friendly way.
- Integrate with **mocks**, **dynamic mock server**, and an **Electron + React UI**.

High‑level architecture:

```text
Your Tests  -->  Tester  -->  StageFuncs
    |            |            |
    |            |            +--> HTTP / DB / App helpers
    |            |
    |            +--> Assertions & Logging
    |
    +--> (optionally) Dynamic Mock Server & GUI
```

---

### Stages, Actions, and Dry‑Run (`tester.go`)

Core types:

- `type StageFunc func()` — function to execute for a stage.
- `type StageDef struct { Name string; Func StageFunc }` — named stage definition.
- `type Action struct { Summary string; Func func() }` — recorded step inside a stage.
- `type Tester struct { Stages []StageDef }` — orchestrates stages.

Key functions:

- `NewTester()` — create a new tester.
- `(*Tester) Stage(name string, fn StageFunc)` — register a stage.
- `(*Tester) RunStageByName(name string) (err error)` — run a specific stage.
- `(*Tester) DryRunAll()` — dry‑run all stages.
- `(*Tester) DryRunStage(s StageDef)` — dry‑run a single stage.
- `RecordAction(summary string, fn func())` — record an action for the current stage.
- `GetStageActions(stageName string) []Action` — retrieve recorded actions.
- `RegisterActionUpdateHandler(fn func())` — subscribe to action updates (for UIs).
- `IsDryRun() bool` — check whether we are in dry‑run mode.

Stage execution flow:

```text
RunStageByName("SetupDB")
  |
  |-- find StageDef("SetupDB")
  |
  |-- set currentStage, enable recording, clear previous actions
  |
  |-- log: [Stage] Running Stage: SetupDB
  |
  |-- run user StageFunc
  |      (helpers may call RecordAction(...))
  |
  |-- defer+recover:
         - panic(TestError)  -> stage FAILED with message
         - other panic       -> stage FAILED (Crash)
         - no panic          -> stage PASSED
  |
  |-- disable recording, clear currentStage
```

Dry‑run flow:

```text
DryRunStage(stage)
  |
  |-- currentStage = stage.Name
  |-- isRecording = true
  |-- isDryRun = true
  |-- clear actions for this stage
  |
  |-- run StageFunc
  |      (helpers see IsDryRun()==true and usually
         record actions but skip side‑effects)
  |
  |-- recover from any panics
  |-- reset flags
```

Example:

```go
tester := v1.NewTester()

tester.Stage("Setup", func() {
    db := v1.Connect("sqlite3", ":memory:")
    _ = db

    v1.RecordAction("Create users table", func() {
        // real DB work here
    })
})

if err := tester.RunStageByName("Setup"); err != nil {
    t.Fatalf("stage failed: %v", err)
}

// Discover actions without executing side‑effects
tester.DryRunAll()
actions := v1.GetStageActions("Setup")
```

---

### Assertions and Failures (`assert.go`)

Core pieces:

- `type TestError struct { Message string }` — represents a controlled test failure.
- `func Fail(format string, args ...interface{})` — log and panic with `TestError`.
- `func Assert(condition bool, format string, args ...interface{})` — `Fail` if condition is false.
- `func AssertNoError(err error)` — `Fail` if `err != nil`.

Error flow:

```text
Assert(cond, msg)
  if !cond
    -> Fail(msg)
         |
         |-- Log(LogTypeError, "Assertion FAILED", msg)
         |-- panic(TestError{Message: msg})

RunStageByName
  |
  |-- defer recover()
         - if TestError -> log stage FAILED and return error
         - else         -> log stage FAILED (Crash) and return error
```

Usage:

```go
resp := v1.SendRequest(server.URL)
v1.Assert(resp.StatusCode == 200, "health check failed, got %d", resp.StatusCode)

err := doSomething()
v1.AssertNoError(err)
```

#### Conditions and `AssertCondition`

Condition-based assertions (`ExpectJsonBodyFieldCond`, `ExpectXmlBodyFieldCond`,
`RowResult.ExpectCond`, ...) and `AssertCondition` all use one strict evaluator in
`pkg/condition`, which is also used by the dynamic mock server's request matching.
Supported conditions:

- equality: `ConditionEqual`, `ConditionNotEqual`
- ordering: `ConditionGreaterThan`, `ConditionLessThan`, `ConditionGreaterThanOrEqual`, `ConditionLessThanOrEqual`
- text: `ConditionContains`, `ConditionNotContains`, `ConditionStartsWith`, `ConditionEndsWith`, `ConditionMatches` (regex)
- collections: `ConditionIn`, `ConditionNotIn`
- presence: `ConditionEmpty`, `ConditionNotEmpty`

Semantics are typed/strict: numeric values (int/uint/float/`json.Number`) compare
by value and maps/slices compare recursively; numeric strings are accepted for
ordering; equality does **not** stringify (`"1"` is not equal to `1`). Unknown
condition names fail fast.

```go
v1.AssertCondition(resp.StatusCode, v1.ConditionIn, []interface{}{200, 201}, "unexpected status")
```

---

### Central Logging (`logger.go`)

All helpers log through a simple central logger.

Types and constants:

- `type LogType string`.
- `LogTypeStage`, `LogTypeDB`, `LogTypeRequest`, `LogTypeMock`,
  `LogTypeApp`, `LogTypeExpect`, `LogTypeError`, `LogTypeInfo`.
- `type LogEntry struct { Type LogType; Summary, Detail string }`.
- `type LogHandler func(entry LogEntry)` — callback for log consumers.

Functions:

- `RegisterLogHandler(h LogHandler)` — add a handler.
- `Log(t LogType, summary, detail string)` — log an event and notify handlers.
- `Logf(t LogType, format string, v ...interface{})` — formatted logging helper.

Data flow:

```text
Helper (e.g. ExpectStatusCode)
   -> Logf(LogTypeExpect, "Status Code %d == %d - PASSED", ...)
        |
        |-- log.Printf("[Expect] ...")
        |
        |-- for each h in logHandlers:
               h(LogEntry{Type: Expect, Summary: ..., Detail: ...})
```

Example handler:

```go
v1.RegisterLogHandler(func(e v1.LogEntry) {
    fmt.Printf("UI: [%s] %s -- %s\n", e.Type, e.Summary, e.Detail)
})

v1.Log(v1.LogTypeInfo, "Starting integration tests", "")
```

---

### HTTP Requests and JSON Expectations (`request.go`)

The `request.go` helpers make HTTP checks concise and test‑friendly.

Core response type:

- `type Response struct { StatusCode int; Body string; Header map[string]string }`

Key functions:

- `SendRequest(url string) Response`
- `ExpectStatusCode(resp Response, expected int)`
- `ExpectHeader(resp Response, key, value string)`
- `ExpectJsonBody(resp Response, expectedJson interface{})`
- `ExpectJsonBodyField(resp Response, field string, expectedValue interface{})`

Internal helpers (for JSON paths):

- `getValueByPath(body interface{}, path string) (interface{}, error)`
- `isNumber(v interface{}) bool`
- `toFloat64(v interface{}) (float64, bool)`

`SendRequest` flow:

```text
SendRequest(url)
  |
  |-- RecordAction("Request: <url>", func() { SendRequest(url) })
  |-- if IsDryRun(): return empty Response
  |-- Logf(LogTypeRequest, "Sending GET request to: %s", url)
  |-- http.Get(url)
       - on error -> Fail("Request failed: %v", err)
  |-- read body and headers
  |-- Log(LogTypeRequest, "Received status ...", "Body: ... Headers: ...")
  |-- return Response{StatusCode, Body, Header}
```

`ExpectStatusCode` and `ExpectHeader`:

- Skip in dry‑run mode (`IsDryRun()`).
- On mismatch, call `Fail(...)` with helpful detail.
- On success, log `LogTypeExpect` entries.

`ExpectJsonBody`:

- Unmarshals `resp.Body` and `expectedJson` (if string) to `interface{}`.
- Compares with `reflect.DeepEqual`.
- On mismatch, calls `Fail` and includes both values in the message.

`ExpectJsonBodyField`:

- Parses `resp.Body` as JSON into `interface{}`.
- Uses a path like `"a"`, `"b.c"`, `"d[0]"`, or `"users[0].name"`.
- Extracts the value via `getValueByPath` and compares to `expectedValue`
  (with numeric type normalization).

JSON path examples:

```text
Body JSON:
{
  "a": 1,
  "b": { "c": 2 },
  "d": [3, 4]
}

Paths:
  "a"    -> 1
  "b.c"  -> 2
  "d[0]" -> 3
  "d[1]" -> 4
```

Example usage:

```go
resp := v1.SendRequest(server.URL)

v1.ExpectStatusCode(resp, 200)
v1.ExpectHeader(resp, "Content-Type", "application/json")
v1.ExpectJsonBodyField(resp, "data.user.name", "alice")
```

---

### Database Helpers (`db.go`)

The DB helpers wrap a SQL database connection with simple operations for tests.

Key concepts:

- `Connect(driver, dsn string) *DBClient` — connect to a DB (e.g. SQLite).
- `type Field struct { Name, Type string }` — table column definition.
- `(*DBClient) SetupTable(table string, autoIncrement bool, fields []Field, ...)` — create a table.
- `(*DBClient) ReplaceData(table string, values []interface{})` — insert or replace rows.
- `(*DBClient) Update(table string, set map[string]interface{}, where string, args ...interface{})` — update rows.
- `(*DBClient) CleanTable(table string)` — delete all rows.
- `(*DBClient) DeleteOne(table, where string, args ...interface{})` — delete a single matching row (safety requires WHERE).
- `(*DBClient) DeleteWithLimit(table, where string, limit int, args ...interface{})` — delete up to `limit` matching rows (limit<=0 deletes all matches, still requires WHERE). Handles Oracle/Postgres/SQLite differences internally.
- `(*DBClient) DropTable(table string)` — drop the table.
- `(*DBClient) Fetch(query string, args ...interface{}) QueryResult` — run a `SELECT` query.

Redis helpers (`redis.go`):

- `ConnectRedis(serverAddr, accessKey string) *RedisClient`
- `(*RedisClient) Set(key string, value interface{}, ttl time.Duration)`
- `(*RedisClient) Get(key string) string`
- `(*RedisClient) Del(keys ...string)`
- `(*RedisClient) ExpectValue(key, expected string)`
- `(*RedisClient) FlushAll()`

Redis usage:

```go
rc := v1.ConnectRedis("http://localhost:9100", "my-access-key")
rc.Set("foo", "bar", 0)
rc.ExpectValue("foo", "bar")
rc.Del("foo")
rc.FlushAll()
```

Result wrappers:

- `type QueryResult` — collection of rows
  - `Count() int`
  - `GetRow(i int) RowResult`
- `type RowResult` — single row
  - `Get(column string) interface{}`
  - `Expect(column string, expected interface{})` — assert value.

Typical usage:

```go
db := v1.Connect("sqlite3", ":memory:")
fields := []v1.Field{{"id", "INTEGER PRIMARY KEY AUTOINCREMENT"}, {"name", "TEXT"}}
db.SetupTable("users", true, fields, nil)

db.ReplaceData("users", []interface{}{1, "Alice"})

result := db.Fetch("SELECT name FROM users WHERE id = ?", 1)
row := result.GetRow(0)
row.Expect("name", "Alice")

db.CleanTable("users")
db.DeleteOne("users", "id = ?", 1)
db.DeleteWithLimit("users", "name = ?", 10, "Alice")
db.DropTable("users")
```

Errors from the underlying DB usually trigger `Fail(...)`, which panics and is
then caught at a higher level (for example by `RunStageByName`).

---

### External Application Runner (`app.go`)

`app.go` lets you start and stop external processes (services under test).

Types and functions:

- `type AppServer struct { cmd *exec.Cmd }` — wraps a running process.
- `func RunAppServer(path string, args ...string) *AppServer`
- `func (s *AppServer) Stop()`

Flow:

```text
RunAppServer(path, args...)
  |
  |-- RecordAction("App Run: path", func() { RunAppServer(path, args...) })
  |-- if IsDryRun(): return &AppServer{}
  |-- exec.Command(path, args...)
  |-- pipe stdout/stderr to os.Stdout/os.Stderr
  |-- Logf(LogTypeApp, "Starting Server: ...")
  |-- if cmd.Start() fails -> Fail("Failed to start server: %v", err)
  |-- return &AppServer{cmd}

AppServer.Stop()
  |
  |-- if cmd and cmd.Process are non‑nil:
        - Log(LogTypeApp, "Stopping Server", "")
        - Kill process
        - Wait for it (release resources)
```

Example:

```go
app := v1.RunAppServer("./my_service", "--port", "8080")
// ... run checks against the service ...
app.Stop()
```

In dry‑run mode this only records the action, it does not actually start a process.

---

### Mocks, Dynamic Mocks, Models, and the UI (`mock.go`, `dynamic_mock.go`, `model.go`, `serve.go`)

These files connect the core tester with mocks and the UI integrations.

At a high level they:

- Provide in‑memory mock behaviors (e.g. for services you call during stages).
- Bridge to the **dynamic mock server** from `pkg/dynamic-mock-server`.
- Define simple data models for stages, actions, and logs that the UI can display.
- Register log and action handlers that keep the UI in sync with test execution.

Conceptual diagram:

```text
 Tester            Logger             Stage/Actions        Electron UI
   |                |                     |                    |
   | RunStage       | Log()               | RecordAction()     |
   |--------------->|-------------------->|------------------->|
   |                |                     |                    |
   |                | RegisterLogHandler  | RegisterAction...  |
   |<---------------------------------------------------------|
           (server pushes logs/status over SSE; UI redraws)
```

#### Desktop / Web UI (`serve.go` + `ui/`)

The UI is an **Electron + React** app that talks to a small HTTP/SSE server
implemented in `serve.go`. The Go test process stays in charge of running
stages; the UI only sends commands and renders state.

- `NewUIServer(t) *UIServer` — create the server for a `*Tester`.
- `(*UIServer) Start() error` / `Stop()` — bind a free localhost port and serve.
- `(*UIServer) URL() string` — base URL (e.g. `http://127.0.0.1:54321`).
- `(*UIServer) Token() string` / `URLWithToken() string` — per-process auth token.
- `RunGUI(t)` — start the server and launch Electron (falls back to the browser).
- `RunServer(t)` — start the server and open the browser only.

The API is **authenticated**: every request except `GET /api/health` must carry
the per-process token, either as `Authorization: Bearer <token>`, as an
`X-IT-Token` header, or as a `?token=` query parameter (needed for `EventSource`).
The token is generated on startup and injected into the UI automatically, so
`RunGUI` / `RunServer` remain zero-config. To open the UI manually, use
`URLWithToken()`.

HTTP API:

```text
GET  /api/health        -> { status: "ok" }   (no token required)
GET  /api/state         -> { stages: [{name, status, actions}], logs: [...] }
GET  /api/events        -> server-sent events: state | log | stage | refresh | error
POST /api/stage/run     -> { name, runPrerequisites }
POST /api/action/run    -> { stage, index }
POST /api/run-all       -> {}
POST /api/discover      -> {}   (dry-run to populate actions)
```

The React sources live in `ui/src`; Electron entry points in `ui/electron`.
Build the frontend once with:

```bash
cd ui && npm install && npm run build
```

`RunGUI` then finds `ui/` automatically (override with `INTEGRATION_TESTER_UI`,
or point at a built bundle with `IT_UI_DIST`). If Electron is unavailable the
same UI is served from `ui/dist` in the browser.

#### Choosing the run mode at launch (`run.go`)

Instead of hard-coding `RunGUI(t)`, apps can let the operator pick the mode when
the binary starts. Register the `-mode` flag before `flag.Parse`, then call
`Run`:

```go
func main() {
    v1.RegisterModeFlag("") // registers -mode (gui | cli | cli-command | server)
    flag.Parse()

    t := v1.NewTester()
    // ... register stages ...

    v1.Run(t) // dispatches on CurrentMode()
}
```

```bash
go run . -mode cli            # run every stage, exit non-zero on failure
go run . -mode cli-command    # interactive stdin command session
go run . -mode server         # HTTP API + web UI in the browser
go run . -mode gui            # Electron desktop UI (default)

INTEGRATION_TESTER_MODE=cli go run .   # env var also works (IT_MODE alias)
```

Helpers:

- `RegisterModeFlag(usage string)` — registers `-mode` on `flag.CommandLine`
  (default from env, else `gui`). Safe to call more than once.
- `CurrentMode() string` — resolves the mode from the flag or env, default `gui`.
- `Run(t)` — dispatch using `CurrentMode()`.
- `RunWithMode(t, mode)` — dispatch an explicit mode (if the app defines its own flag).

Modes: `ModeGUI`, `ModeCLI`, `ModeCLICommand`, `ModeServer`.

#### CLI command mode for AI/debugging (`cli.go`)

`-mode cli-command` starts an interactive, line-oriented session on stdin/stdout
designed for an external driver (e.g. an AI agent). Stages run one at a time; the
session reports progress and rejects overlapping runs.

Commands: `list`, `run <stage>`, `status`, `exit`/`quit`. Every response ends
with a blank line; a `run` streams events and ends with `PASSED`/`FAILED`.

```text
RUNNING <stage>
HEARTBEAT <stage> elapsed=5s still-running
SLOW <stage> elapsed=1m0s still-running slow-threshold=1m0s
PASSED <stage> duration=12.3s
FAILED <stage> duration=3.1s error=...
BUSY running=<stage> elapsed=... requested=<stage>   # one stage already running
STATUS RUNNING stage=<stage> elapsed=...
STATUS IDLE last=<stage> result=PASSED duration=...
```

Options / environment:

- `RunCLICommandWithOptions(t, CLICommandOptions{Heartbeat, SlowAfter, Timeout})`.
- `INTEGRATION_TESTER_CLI_HEARTBEAT` — progress interval (default `5s`, `<0` disables).
- `INTEGRATION_TESTER_CLI_SLOW_AFTER` — slow warning threshold (default `1m`).
- `INTEGRATION_TESTER_CLI_TIMEOUT` — hard limit (default off; when exceeded the
  process exits so a stuck stage can never overlap the next one).

Concurrency safety: `Tester.RunStageByName` serializes runs with an internal
lock, `TryRunStageByName` returns `ErrStageRunning` instead of blocking, and
`RunningStage()` reports the active stage — so two stages never run at the same
time in any mode (CLI or UI).

---

### How This Package Fits into Integration Tests

Typical usage pattern:

```text
Your integration tests
  |
  +--> v1.Tester stages
  |      +--> HTTP, DB, App helpers
  |
  +--> (optionally) dynamic mock server
          +--> configure mock HTTP endpoints for dependencies
```

Use this package when you want structured stages, consistent logging, and clear
assertions around HTTP, DB, and external processes in your Go integration tests.
