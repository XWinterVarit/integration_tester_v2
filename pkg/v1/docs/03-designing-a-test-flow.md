# 3. Designing a test flow

This is the most important page for newcomers: **how to structure an integration
test with `pkg/v1`.**

## The golden rule

> One stage = one meaningful step that can fail on its own.

Stages run top to bottom. A failure marks the stage `FAILED` and moves on; it
does **not** stop the process (in `cli` mode `RunCLI` only exits after all
stages finish). Because of that, later stages must not assume earlier ones
succeeded unless you design them that way.

## Recommended shape

```text
Setup            -> start dependencies, prepare data, start the app
Happy path       -> the main scenario, asserts success + side effects
Edge / error case-> expected failures and validation
Cleanup (LAST)   -> stop processes, delete fixtures
```

Keep `Cleanup` as the **last** stage. If you stop a server in the middle, every
later stage that needs it will fail.

## A full, runnable example

```go
package main

import (
    "net/http"

    v1 "github.com/XWinterVarit/integrate_tester_v2/pkg/v1"
)

func main() {
    t := v1.NewTester()

    var app *v1.AppServer
    var db *v1.DBClient

    t.Stage("Setup", func() {
        db = v1.Connect("sqlite3", ":memory:")
        db.SetupTable("users", true, []v1.Field{
            {"id", "INTEGER PRIMARY KEY AUTOINCREMENT"},
            {"name", "TEXT"},
            {"status", "TEXT"},
        }, nil)

        app = v1.RunAppServer("./example_app_bin", "-dsn", "...")
        // RunAppServer returns immediately; wait until the app is ready.
        v1.Assert(waitFor("http://localhost:8080/health", 15*time.Second) == nil,
            "app did not become ready")
    })

    t.Stage("Create user", func() {
        resp := v1.SendRESTRequest("http://localhost:8080/insert?id=1&name=alice&status=active",
            v1.WithMethod(http.MethodPost))
        v1.ExpectStatusCode(resp, http.StatusCreated)

        row := db.Fetch("SELECT name, status FROM users WHERE id = ?", 1).GetRow(0)
        row.Expect("name", "alice")
        row.Expect("status", "active")
    })

    t.Stage("Expected failure", func() {
        db.Update("users", map[string]interface{}{"status": "bad"}, "id = ?", 1)
        resp := v1.SendRequest("http://localhost:8080/read?id=1")
        v1.ExpectStatusCode(resp, http.StatusInternalServerError)
    })

    t.Stage("Cleanup", func() {
        if app != nil {
            app.Stop()
        }
    })

    v1.Run(t)
}
```

`waitFor` is a small helper you write yourself (the library intentionally does
not guess your app's readiness endpoint):

```go
func waitFor(url string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if resp, err := http.Get(url); err == nil {
            resp.Body.Close()
            return nil
        }
        time.Sleep(200 * time.Millisecond)
    }
    return fmt.Errorf("server at %s not ready after %s", url, timeout)
}
```

## Stage naming

- Use short, unique, human-readable names — they appear in the UI and in the CLI.
- Names may contain any character, including `:`. They are map keys, not
  delimited IDs, so `"Setup: DB"` is fine.
- `RunStageByName` matches the name **exactly**.

## Recording actions (optional but recommended for GUI)

Actions are the units the UI displays and lets a human re-run. `RecordAction`
does not execute anything on its own; helpers call it with a closure that
repeats the work:

```go
t.Stage("Setup", func() {
    v1.RecordAction("Create users table", func() {
        db.SetupTable("users", true, fields, nil) // real work
    })
})
```

Most built-in helpers (`SendRESTRequest`, `Connect`, `SetupTable`, …) already
record an action and perform the work. Use `RecordAction` directly only for your
own custom operations.

## Dry run / discovery

The GUI discovers actions without executing them:

```go
tester.DryRunAll()
for _, a := range v1.GetStageActions("Setup") {
    fmt.Println(a.Summary)
}
```

Helpers skip side effects when `v1.IsDryRun()` is true. If you write custom
helpers, follow the same pattern:

```go
func myOperation() {
    v1.RecordAction("My operation", func() { myOperation() })
    if v1.IsDryRun() {
        return
    }
    // ... real work ...
}
```

## Error handling patterns

```go
// hard fail the stage
v1.Assert(err == nil, "unexpected: %v", err)
v1.AssertNoError(err)
v1.Fail("expected X but got %v", got)

// assert a condition with the shared engine
v1.AssertCondition(resp.StatusCode, v1.ConditionIn, []interface{}{200, 201}, "unexpected status")
```

Because `Fail` panics, any code after it in the stage does not run. That is
usually what you want: a stage is all-or-nothing.

## Best practices

1. **Idempotent setup.** `SetupTable(table, true, ...)` recreates the table, so
   re-running a stage is safe.
2. **Cleanup last.** Never stop a dependency before stages that use it.
3. **Wait for readiness.** `RunAppServer` is asynchronous; poll a health
   endpoint before sending requests.
4. **One concern per stage.** Makes CLI debugging and UI re-runs much easier.
5. **Use conditions, not `==`, for cross-driver values.** DB drivers return
   different numeric types; `RowResult.ExpectCond`/`AssertCondition` normalize.
6. **Keep secrets out of logs.** The logger prints request bodies/headers.

Next: [Assertions & conditions →](04-assertions-and-conditions.md)
