# 10. Cookbook & FAQ

## End-to-end walkthrough

The canonical example is
[`pkg/v1/example_app/integration_test/main.go`](../example_app/integration_test/main.go).
It exercises every part of the library. Its stage flow:

| Stage | What it demonstrates |
|-------|----------------------|
| `Setup` | build + start a service, connect DB, `SetupTable`, wait for readiness, seed data |
| `Success Case` | HTTP update + DB verification + read-back |
| `Complex Request (POST JSON)` | headers, JSON body, field assertions |
| `Fail Case` | manipulate state to force a server error, assert `500` |
| `Redis Hash Operations` | `HSet`/`HGet`/`HIncrement` |
| `Complex Request (POST XML)` | XML body + `ExpectXmlBodyField` |
| `Mock Server Example` | dynamic mock route with query-based case routing |
| `Mock Server XML Example` | XML request extraction + case routing |
| `Mock Server Path Params Example` | `/a/b/{ee}/c/{dd}` with params in conditions/templates |
| `Cleanup` | `app.Stop()` — **last** |

Run it:

```bash
cd pkg/v1 && go run ./example_app/integration_test -mode cli
```

## Patterns

### Setup / act / assert / cleanup

```go
t.Stage("Setup", func() { /* deps + data */ })
t.Stage("Act", func()   { /* exercise, assert */ })
t.Stage("Cleanup", func(){ /* always last */ })
```

### Wait for readiness

`RunAppServer` is async; poll a health endpoint before the first request. See
[Apps & mocks](07-app-and-mocks.md#running-an-external-app-appgo).

### Seed then assert side effects

```go
t.Stage("Insert", func() {
    v1.SendRESTRequest(url, v1.WithMethod(http.MethodPost), v1.WithJSONBody(payload))
    row := db.Fetch("SELECT status FROM users WHERE id = ?", 1).GetRow(0)
    row.ExpectCond("status", v1.ConditionEqual, "active")
})
```

### Mock a flaky/unavailable dependency

```go
client := v1.NewDynamicMockClient("http://localhost:9001")
v1.AssertNoError(client.RegisterRoute(9002, "GET", "/third-party/{id}", []v1.ResponseFuncConfig{
    v1.GenerateRandomInt(10, 50, "LATENCY"),
    v1.SetStatusCode("", 200),
    v1.SetJsonBody("", `{"id":"{{.id}}"}`),
    v1.SetHeader("", "Content-Type", "application/json"),
}))
```

### Assert data-driven cases with conditions

```go
v1.ExpectJsonBodyFieldCond(resp, "score", v1.ConditionGreaterThanOrEqual, 0)
v1.ExpectJsonBodyFieldCond(resp, "email", v1.ConditionMatches, `^[^@]+@[^@]+$`)
v1.ExpectJsonBodyFieldCond(resp, "tags", v1.ConditionNotEmpty, nil)
```

### Table-driven stages (dynamic registration)

```go
for _, c := range cases {
    c := c
    t.Stage("Case "+c.Name, func() {
        resp := v1.SendRequest(c.URL)
        v1.ExpectStatusCode(resp, c.Want)
    })
}
```

### Expect failure without failing the test

Put the expected error in its own stage and assert the error status:

```go
t.Stage("Invalid input returns 400", func() {
    resp := v1.SendRESTRequest(url, v1.WithMethod(http.MethodPost), v1.WithBodyString(`{bad}`))
    v1.ExpectStatusCode(resp, http.StatusBadRequest)
})
```

## Anti-patterns

| Anti-pattern | Why it hurts | Do instead |
|--------------|--------------|------------|
| Stopping a dependency in a middle stage | later stages fail | `Cleanup` last |
| Sending requests immediately after `RunAppServer` | race / `connection refused` | poll for readiness |
| Comparing DB/JSON numbers with `==` | driver type differences | `ExpectCond` / `AssertCondition` |
| Long unrelated work in one stage | hard to debug/re-run in UI/CLI | one concern per stage |
| Assuming dry-run does real work | discovery runs with side effects skipped | guard custom helpers with `IsDryRun()` |
| Running stages concurrently yourself | corrupts recorded actions | rely on `RunStageByName` serialization |

## FAQ

**Can two stages run at once?**
No. `RunStageByName` serializes; `TryRunStageByName` returns `ErrStageRunning`.
See [Run modes](08-run-modes.md#concurrency-all-modes).

**How do I run a single stage?**
UI: click it. Server: `POST /api/stage/run {"name":"..."}`. CLI-command:
`run <stage>`. Programmatically: `tester.RunStageByName("...")`.

**Why does `RunAppServer` return before the app is up?**
It starts the process asynchronously. Wait for readiness yourself.

**How do I run the same test in CI?**
`go test`-style: point your CI at the binary with `-mode cli` and check the exit
code (non-zero on failure).

**How do I know what a stage does without running it?**
Dry run: `tester.DryRunAll()` then `v1.GetStageActions(name)`. The UI's
"Refresh Actions" does this via `POST /api/discover`.

**Can stage names contain spaces or `:`?**
Yes. Names are matched exactly and are not delimited IDs.

**How do I write a custom helper?**
Record an action, skip side effects in dry-run:

```go
func provision() {
    v1.RecordAction("Provision tenant", func() { provision() })
    if v1.IsDryRun() {
        return
    }
    // ... real work, using v1.Assert/Expect to validate ...
}
```

**Where do assertions live?**
`assert.go` (Fail/Assert), `expect_condition.go` (condition engine + `AssertCondition`),
`request.go` (HTTP + JSON/XML `Expect*`), `db.go`/`redis.go` (data assertions).

**How is the condition engine shared?**
One implementation in `pkg/condition`, used by both `pkg/v1` assertions and the
dynamic mock server. See [Assertions & conditions](04-assertions-and-conditions.md).

## Where to look next

- Source: [`pkg/v1`](../) — each file has focused package comments.
- Condition engine: [`pkg/condition`](../../condition/).
- Dynamic mock server: [`pkg/dynamic-mock-server`](../../dynamic-mock-server/)
  and its runnable `client-example/`.
- Redis mock server: [`pkg/redis-mock-server`](../../redis-mock-server/).
