# 1. Mental model

## The pieces

```text
                 +-------------------------------------------------+
 your test       |                    Tester                       |
 (main.go)  ---> |  Stages: [{ Name, Func }, ...]                  |
                 |                                                 |
                 |  RunStageByName(name)                           |
                 |    - serialize (only one stage at a time)       |
                 |    - set currentStage + recording               |
                 |    - run Func(); recover panics -> error        |
                 |    - clear recording                            |
                 +---------------------+---------------------------+
                                       |
                                       v
   helpers called inside a stage:      RecordAction("...", fn)   --> Actions
   SendRESTRequest / DB / Redis / ...  Log(type, summary, detail) --> Logs
```

- **`Tester`** owns the ordered list of stages and runs them.
- **`StageFunc`** (`func()`) is the body of a stage.
- **`Action`** is a replayable operation recorded while a stage runs. It is used
  by the UI to show what a stage does, and to let a human re-run one action.
- **`LogEntry`** is a structured event (`Stage`, `DB`, `Request`, `Expect`,
  `Error`, …) streamed to the UI and the console.

## Object model

```go
type Tester struct {
    Stages []StageDef // ordered
    // unexported: mutexes for stage list + serialized execution
}

type StageDef struct {
    Name string
    Func StageFunc
}

type StageFunc func()

type Action struct {
    Summary string // e.g. "Request: POST http://host/update"
    Func    func() // the recorded operation (used by UI "Run" buttons)
}
```

## Lifecycle of a stage

```text
RunStageByName("Setup")
  |
  |-- lock runMu                      (serialize: no two stages at once)
  |-- wait for any in-progress dry run
  |-- find StageDef by exact name
  |-- currentStage = "Setup"; isRecording = true
  |-- Log(Stage, "Running Stage: Setup")
  |-- execute Func()
  |     |-- helpers call RecordAction(...)   -> recorded
  |     |-- helpers call Log(...)            -> streamed
  |     |-- assertions call Fail(...)        -> panic(TestError)
  |
  |-- recover:
  |     TestError          -> stage FAILED, return error "failed: <msg>"
  |     other panic        -> stage FAILED (Crash), return error "panic: <v>"
  |     no panic           -> stage PASSED
  |-- currentStage = ""; isRecording = false
  |-- unlock runMu
```

A **stage is a boundary**: the `Func` is responsible for doing whatever it
needs (HTTP, DB, starting processes). Helpers both *do* the work and *record* it.

## Failure model

- `v1.Fail(format, args...)` logs an error and panics with `TestError`.
- `v1.Assert(cond, ...)` calls `Fail` when `cond` is false.
- `v1.AssertNoError(err)` calls `Fail` when `err != nil`.
- The stage runner recovers the panic, so **a failing stage never kills the
  process**; it becomes `FAILED` and the caller decides what to do next.
- In **dry-run** mode `Fail` does not panic (it logs an "Assertion skipped"
  info line), so discovery can proceed without dependencies.

## Dry run (action discovery)

`DryRunAll()` runs every stage with `isDryRun = true`. Helpers check
`IsDryRun()` and skip side effects while still calling `RecordAction`. This is
how the UI can list each stage's actions without executing them.

```go
tester.DryRunAll()
actions := v1.GetStageActions("Setup") // []Action
```

## Concurrency guarantees

Stage execution mutates shared, global state (`currentStage`, recorded actions).
The library therefore **serializes** runs:

- `(*Tester).RunStageByName(name)` blocks until any running stage finishes.
- `(*Tester).TryRunStageByName(name)` returns `v1.ErrStageRunning` if busy.
- `(*Tester).RunningStage()` returns the active stage name, or `""`.

This holds for every mode (GUI, server, CLI, CLI-command). You never need to
worry about two stages overlapping.

## Observability hooks

```go
v1.RegisterLogHandler(func(e v1.LogEntry) { /* every log event */ })
v1.RegisterActionUpdateHandler(func() { /* recorded actions changed */ })
```

The built-in UI server uses these to push logs/status over SSE. You can use them
for custom reporting.

Next: [Getting started →](02-getting-started.md)
