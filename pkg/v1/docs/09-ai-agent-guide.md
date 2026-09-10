# 9. AI agent guide

This page is optimized for an AI agent (or any automated driver) that needs to
**run and debug an integration test** through `pkg/v1`'s command mode. It is
written as a protocol spec.

## How to start it

Run the test binary with mode `cli-command`:

```bash
go run . -mode cli-command
# or, for a prebuilt binary:
./itest -mode cli-command
# or:
INTEGRATION_TESTER_MODE=cli-command ./itest
```

The process reads commands from **stdin**, one per line, and writes results to
**stdout** (application logs go to stderr via Go's `log`). Keep the process
alive for the whole session; it is stateful.

Handshake on startup:

```text
=== Integration Test (Command Mode) ===
Commands: list | run <stage> | status | exit
Ready.
                          <-- blank line terminates the handshake
```

## Commands

| Command | Effect |
|---------|--------|
| `list` | print all stage names, one per line, indented two spaces |
| `run <stage>` | start the stage (rejected if one is already running) |
| `status` | report the running/last stage and timings |
| `exit` / `quit` | wait for any active run, then end the session |

Stage names are matched **exactly** and may contain spaces and `:`.

## Framing rule

**Every command response ends with a blank line.** A `run` response is an event
stream that also ends with a blank line after `PASSED`/`FAILED`. Read until you
see a blank line to know a response is complete.

## Event tokens (stdout)

| Line | When |
|------|------|
| `RUNNING <stage>` | a stage just started |
| `HEARTBEAT <stage> elapsed=5s still-running` | periodic progress (default every 5s) |
| `SLOW <stage> elapsed=1m0s still-running slow-threshold=1m0s` | run exceeds the slow threshold |
| `PASSED <stage> duration=12.3s` | stage finished successfully |
| `FAILED <stage> duration=3.1s error=<message>` | stage failed |
| `BUSY running=<stage> elapsed=... requested=<stage>` | `run` rejected: one stage is active |
| `STATUS RUNNING stage=<stage> elapsed=...` | `status` while a stage runs |
| `STATUS IDLE` | `status` before any run |
| `STATUS IDLE last=<stage> result=<PASSED\|FAILED> duration=... [error=...]` | `status` after a run |
| `  <stage name>` | one line per stage for `list` |
| `ERROR unknown stage: <name>` | `run` with an unknown stage |
| `ERROR unknown command: <line>` | unrecognized command |
| `EXIT waiting for <stage> to finish...` | `exit` issued during a run |
| `Bye.` | session ended |

Durations use Go's `time.Duration` string form, rounded to milliseconds
(`345ms`, `1.2s`, `2m0s`).

## Concurrency contract (important)

- **Exactly one stage runs at a time.** There is no way to run two in parallel.
- A second `run` while busy returns `BUSY`; it does **not** queue and does not
  start.
- `status` is answered immediately, even while a stage runs.
- `exit` waits for the active stage to finish before returning.

Therefore: after sending `run X`, wait for `PASSED X` or `FAILED X` before
sending another `run`. You may send `status` at any time.

## Recommended control loop

```text
1. spawn: go run . -mode cli-command   (stdin=pipe, stdout=pipe)
2. read lines until a blank line          -> handshake done
3. send "list"                            -> collect stage names
4. for each stage:
     send "run <stage>"
     read lines until a blank line:
        - print/emit HEARTBEAT/SLOW as progress
        - on PASSED/FAILED: record result and stop reading
5. send "exit"
6. close stdin, wait for process exit (0)
```

Pseudocode for reading a response:

```python
def read_response(proc):
    lines = []
    for raw in proc.stdout:          # line-buffered
        line = raw.rstrip("\n")
        if line == "":
            return lines             # blank line = end of response
        lines.append(line)
    return lines
```

## Handling `BUSY`

`BUSY` means a stage is already running. Do not retry immediately in a tight
loop; either read the current run's terminal event, or poll `status` until it is
`IDLE`, then retry.

```text
send: run Slow
recv: RUNNING Slow
send: status
recv: STATUS RUNNING stage=Slow elapsed=1.2s
send: run Slow
recv: BUSY running=Slow elapsed=1.4s requested=Slow
...
recv: PASSED Slow duration=3.0s
```

## Detecting "too long"

- `HEARTBEAT` proves the process is alive.
- `SLOW` means the run passed the slow threshold; treat it as a warning.
- If you need a hard limit, set `INTEGRATION_TESTER_CLI_TIMEOUT` (e.g. `2m`).
  When exceeded the process prints `TIMEOUT ...` and exits with code `2`
  (a stuck stage cannot be cancelled, so the process exits to guarantee no
  overlap).

## Configuration

| Env var | Default | Meaning |
|---------|---------|---------|
| `INTEGRATION_TESTER_CLI_HEARTBEAT` | `5s` | progress interval (`<0` disables) |
| `INTEGRATION_TESTER_CLI_SLOW_AFTER` | `1m` | slow-warning threshold |
| `INTEGRATION_TESTER_CLI_TIMEOUT` | off | hard limit; `0` disables |

Programmatic equivalents via `v1.CLICommandOptions{Heartbeat, SlowAfter, Timeout}`.

## Quick reference

```text
LAUNCH     <binary> -mode cli-command
BREAK      blank line ends every response
RUN        run <stage>        -> RUNNING ... (HEARTBEAT|SLOW)* (PASSED|FAILED)
QUERY      status             -> STATUS RUNNING ... | STATUS IDLE ...
LIST       list               -> "  <name>" per line
EXIT       exit | quit        -> waits, then "Bye."
CONFLICT   run while running  -> BUSY running=<stage> requested=<stage>
```

## Non-interactive alternative

If you only need pass/fail for all stages once, use `-mode cli` instead. It runs
everything sequentially and exits `1` on any failure — no protocol, just
`[STAGE]`/`PASSED`/`FAILED` lines.

For programmatic embedding (e.g. you own the streams), use:

```go
v1.RunCLICommandIO(t, in io.Reader, out io.Writer, v1.CLICommandOptions{})
```

Next: [Cookbook & FAQ →](10-cookbook-and-faq.md)
