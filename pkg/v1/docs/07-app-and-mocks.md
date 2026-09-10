# 7. Apps & mocks

## Running an external app (`app.go`)

`RunAppServer` starts a child process (your service under test) and pipes its
stdout/stderr to the test's.

```go
type AppServer struct { /* ... */ }

func RunAppServer(path string, args ...string) *AppServer
func (s *AppServer) Stop()
```

```go
app := v1.RunAppServer("./example_app_bin", "-driver", "oracle", "-dsn", dsn)
defer app.Stop() // or call Stop in your Cleanup stage
```

> **`RunAppServer` returns immediately** — it does not wait for the app to bind
> its port. Always wait for readiness before sending requests:

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

app := v1.RunAppServer("./bin/app")
v1.AssertNoError(waitFor("http://localhost:8080/health", 15*time.Second))
```

In dry-run, `RunAppServer` records the action and returns an empty `AppServer`.

## In-memory mock server (`mock.go`)

For simple, code-defined fakes (no separate process):

```go
type MockHandlerFunc func(Request) Response

type Request  struct { Method, URL string; Header http.Header; Body string }
type Response struct { StatusCode int; Body string; Header map[string]string }

func RunMockServer(port string, handlers map[string]MockHandlerFunc) *MockServer
func UpdateMockServer(ms *MockServer, handlers map[string]MockHandlerFunc)
func (ms *MockServer) Stop()
```

```go
ms := v1.RunMockServer(":9099", map[string]v1.MockHandlerFunc{
    "/users/1": func(r v1.Request) v1.Response {
        return v1.Response{
            StatusCode: 200,
            Body:       `{"id":"1","name":"alice"}`,
            Header:     map[string]string{"Content-Type": "application/json"},
        }
    },
})
defer ms.Stop()
```

Handlers are matched by **exact URL path**; the method is available in
`Request.Method` if you need to branch inside a handler.

## Dynamic mock server (`dynamic_mock.go`)

For runtime-configurable HTTP dependencies. It has a **controller** (a separate
process or embedded `MockController`) and a **client** you use from tests.

Start the controller binary:

```bash
go run ./pkg/dynamic-mock-server/cmd -port 9001
```

Then in the test:

```go
client := v1.NewDynamicMockClient("http://localhost:9001")

err := client.RegisterRoute(9002, "GET", "/products/{id}", []v1.ResponseFuncConfig{
    // Extract a path parameter into USER-visible variables.
    v1.ExtractRequestPathParam("id", "PRODUCT_ID"),

    // Route to a case based on a captured path param.
    v1.IfRequestPathParamSetCase("id", v1.ConditionEqual, "42", "Found"),

    // Default response
    v1.SetStatusCode("", 404),
    v1.SetJsonBody("", `{"error":"not found"}`),

    // Found response
    v1.SetStatusCode("Found", 200),
    v1.SetHeader("Found", "Content-Type", "application/json"),
    v1.SetJsonBody("Found", `{"id":"{{.PRODUCT_ID}}","name":"Widget"}`),
})
v1.AssertNoError(err)
```

Bodies are Go `text/template` strings; variables come from `Extract*`,
generators, captured path params, and conditions. In dry-run, all client calls
are no-ops.

### Helper categories

| Category | Examples |
|----------|----------|
| Conditions | `IfRequestHeader`, `IfRequestJsonBody`, `IfRequestXmlBody`, `IfRequestPath`, `IfRequestPathParam`, `IfRequestQuery`, `IfDynamicVariable`, `IfRequestJsonType`, `IfRequestJsonArrayLength`, `IfRequestJsonObjectLength` |
| Condition + case | every condition has a `...SetCase` variant, e.g. `IfRequestQuerySetCase`, `IfRequestPathParamSetCase` |
| Extract | `ExtractRequestHeader`, `ExtractRequestJsonBody`, `ExtractRequestXmlBody`, `ExtractRequestPath`, `ExtractRequestPathParam`, `ExtractRequestQuery` |
| Generators | `GenerateRandomString`, `GenerateRandomInt`, `GenerateRandomIntFixLength`, `GenerateRandomDecimal`, `HashedString` |
| Variable ops | `ConvertToString`, `ConvertToInt`, `DynamicVarSubstring`, `DynamicVarJoin`, `Delete` |
| Response | `SetStatusCode`, `SetHeader`, `SetJsonBody`, `SetXmlBody`, `SetWait`, `SetRandomWait`, `CopyHeaderFromRequest` |

### Case routing

Every `...SetCase` helper sets the *active case* when its condition matches.
Response helpers with `caseStr == ""` apply to the default case; helpers with a
matching case name apply only when that case is active.

```go
v1.IfRequestQuerySetCase("tier", v1.ConditionEqual, "vip", "VIP"),
v1.SetStatusCode("", 200),
v1.SetJsonBody("", `{"tier":"basic"}`),
v1.SetStatusCode("VIP", 200),
v1.SetJsonBody("VIP", `{"tier":"vip"}`),
```

### Path parameters

Route paths may contain `{name}` placeholders. Each matches one non-empty path
segment and is injected into the template variables:

```go
// POST /a/b/{ee}/c/{dd}
v1.IfRequestPathParam("ee", v1.ConditionEqual, "hello", "EE_OK", "yes"),
v1.IfRequestPathParamSetCase("dd", v1.ConditionEqual, "admin", "AdminCase"),
v1.ExtractRequestPathParam("ee", "EE_COPY"),
v1.SetJsonBody("", `{"ee":"{{.ee}}","dd":"{{.dd}}"}`),
```

Matching rules: exact paths win over patterns; `{name}` matches any single
segment; literal segments must match exactly. Captured values are strings, so
use `ConditionEqual`/string conditions (or numeric ordering, which accepts
numeric strings).

### Reset between tests

```go
client.ResetPort(9002) // drop routes and stop the mock server on that port
client.ResetAll()      // drop everything
```

Next: [Run modes →](08-run-modes.md)
