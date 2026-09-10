# v1 Integration Tester — Documentation

`pkg/v1` is a Go library for writing **integration tests as a sequence of named
stages**. It records the operations each stage performs (actions), gives you
assertions that stop a stage on failure, and can expose the whole thing through
a GUI, a web UI, or an AI-driven CLI.

This `docs/` folder is written for **both humans and AI agents**. Human readers
can follow it linearly; AI agents should jump to the [machine-readable quick
reference](09-ai-agent-guide.md#quick-reference).

---

## Reading guide

| # | Document | What you learn |
|---|----------|----------------|
| 1 | [Mental model](01-mental-model.md) | How `Tester`, stages, actions, dry-run, and logging fit together |
| 2 | [Getting started](02-getting-started.md) | A minimal runnable test and how to run it |
| 3 | [Designing a test flow](03-designing-a-test-flow.md) | Setup → checks → cleanup, ordering, error handling, best practices |
| 4 | [Assertions & conditions](04-assertions-and-conditions.md) | `Fail`/`Assert`, the strict condition engine, `AssertCondition` |
| 5 | [HTTP testing](05-http.md) | `SendRESTRequest`, JSON/XML assertions, field paths |
| 6 | [Database & Redis](06-database-and-redis.md) | `Connect`, `SetupTable`, `Fetch`, Redis helpers |
| 7 | [Apps & mocks](07-app-and-mocks.md) | `RunAppServer`, in-memory mocks, dynamic mock server, path params |
| 8 | [Run modes](08-run-modes.md) | `-mode gui\|cli\|cli-command\|server`, HTTP API, SSE |
| 9 | [AI agent guide](09-ai-agent-guide.md) | Driving `cli-command`, protocol, concurrency rules |
| 10 | [Cookbook & FAQ](10-cookbook-and-faq.md) | End-to-end examples, patterns, anti-patterns, FAQ |

---

## 60-second overview

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
    })

    v1.Run(t) // gui | cli | cli-command | server — chosen at launch
}
```

Run it in any mode:

```bash
go run . -mode cli            # run every stage, exit non-zero on failure
go run . -mode cli-command    # interactive session for an AI/debugger
go run . -mode server         # browser UI
go run . -mode gui            # Electron desktop UI (default)
```

---

## Core ideas (TL;DR)

- **Stage** — a named `func()` that performs part of the test.
- **Action** — an operation recorded inside a stage (HTTP call, DB write, …).
- **Failure** — a stage fails by panicking through `v1.Fail`; the runner catches
  it, marks the stage `FAILED`, and continues with the next stage.
- **Concurrency** — stages are serialized internally. Two stages never run at
  once, in any mode.
- **Dry run** — `DryRunAll` executes stages with side effects skipped, only to
  discover the list of actions for a UI. Assertions are skipped.
- **Conditions** — one strict engine in `pkg/condition`, shared by assertions
  and the dynamic mock server.

Continue with [the mental model →](01-mental-model.md)
