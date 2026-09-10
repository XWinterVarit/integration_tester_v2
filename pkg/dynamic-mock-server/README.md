### Overview of the Dynamic Mock Server (`pkg/dynamic-mock-server`)

This package implements a **dynamic HTTP mock server**. You can:

- Start a controller server.
- Configure mock routes at runtime via HTTP APIs or the Go client.
- Serve mock responses from separate HTTP ports.
- Reset mocks per port or all at once.

It is useful for integration tests where your system under test calls
external HTTP services that you want to simulate.

High‑level architecture:

```text
                  +-------------------------------+
                  |  MockController (control API) |
                  |  - manages mock instances     |
                  +-------------------------------+
                     ^                       ^
                     |                       |
   client.go         |                       |    server.go
  (your tests)       |                       |
+----------------+   |                       |  +---------------------+
| Client API     |---+  register/reset/etc.  +->| MockServerInstance  |
| (Register...)  |                              | (HTTP mock on port) |
+----------------+                              +---------------------+
```

---

### Core Components

#### `server.go`

Main types (names may be slightly simplified here for readability):

- `MockServerInstance` — represents a running HTTP mock server on a given port.
- `MockController` — owns multiple `MockServerInstance`s and exposes an HTTP API
  to configure them.

Key responsibilities:

- Start and stop the control server.
- Handle registration of mock routes.
- Start a per‑port mock server when needed.
- Reset mocks for a specific port or all ports.
- Route incoming HTTP requests on mock ports to the correct mock response.

Typical request flow:

```text
1. Test code calls controller endpoint /register-route
2. MockController parses the desired route and ensures a MockServerInstance
   exists for the target port.
3. The route is added to that instance's configuration.
4. When your application makes an HTTP call to that mock port,
   MockServerInstance matches the request to a configured route and
   returns the specified response.
```

#### `handler.go`

Defines HTTP handlers bound to the control server routes, such as:

- Registering a new route.
- Resetting a port.
- Resetting all ports.

Handlers decode HTTP requests into model structs and delegate to `MockController`.

#### `model.go`

Defines the data structures exchanged between client and server, for example:

- Route definitions (method, path, port).
- Match conditions (headers, body JSON fields, etc.).
- Response definitions (status, body, headers, delay, etc.).

These structs usually have JSON tags so they can be serialized/deserialized
on the wire.

#### `logger.go`

Implements a small logging helper used only inside this package. It is
separate from `pkg/v1/logger.go`.

---

### Go Client (`client.go`)

The client wraps the control API in a simple Go interface so you don't have
to build HTTP requests by hand.

Typical features:

- Create a client pointing at the controller base URL.
- Register mock routes with request/response definitions.
- Reset mocks for a port or all ports.

Conceptual example (exact types may differ slightly from this sketch):

```go
package main

import (
    "log"

    mockserver "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func main() {
    client := mockserver.NewClient("http://localhost:9000")

    err := client.RegisterRoute(mockserver.Route{
        Port:   8081,
        Method: "GET",
        Path:   "/users/123",
        Response: mockserver.Response{
            Status: 200,
            Body:   `{"id": 123, "name": "Alice"}`,
        },
    })
    if err != nil {
        log.Fatalf("failed to register route: %v", err)
    }

    // Now, any HTTP GET to http://localhost:8081/users/123 will
    // return the configured JSON response.
}
```

For concrete, up‑to‑date usage samples, see the files in:

- `pkg/dynamic-mock-server/client-example/`

---

### Client Examples (`client-example/`)

This directory contains runnable examples that show different ways to use
the dynamic mock server:

- **basic_examples.go** — simple route setup and reset.
- **advanced_examples.go** — more complex scenarios.
- **case_examples.go**, **complex_examples.go** — multiple routes and cases.
- **conditional_examples.go** — responses that depend on request body/headers.
- **extended_conditions_examples.go** — newer matching features.
- **generator_examples.go** — dynamic responses created at request time.
- **new_features_examples.go** — demonstrations of newly added capabilities.

Run these examples to see how the client and controller work together.

---

### How It Fits with `pkg/v1`

You can combine the dynamic mock server with the `pkg/v1` tester:

```text
Your Test
  |
  +--> dynamic-mock-server client
  |       +--> configure mocked HTTP dependencies
  |
  +--> v1.Tester stages
          +--> use SendRequest / Expect* helpers against those mocks
```

This lets you control all external HTTP dependencies of your system under
test while still using the convenient assertions and logging from `pkg/v1`.
