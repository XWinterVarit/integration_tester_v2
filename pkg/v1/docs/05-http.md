# 5. HTTP testing

## Sending requests

```go
func SendRESTRequest(url string, opts ...RESTRequestOption) Response
func SendRequest(url string) Response // GET shortcut
```

`Response`:

```go
type Response struct {
    StatusCode int
    Body       string
    Header     map[string]string // first value per header
}
```

Both requests and responses are logged as actions (`Request: GET url`) and log
entries (`LogTypeRequest`), including pretty-printed bodies.

### Options

| Option | Effect |
|--------|--------|
| `WithMethod(method)` | HTTP method (default `GET`) |
| `WithHeader(key, value)` | add one header |
| `WithHeaders(map[string]string)` | merge headers |
| `WithJSONBody(v)` | `json.Marshal(v)`, sets `Content-Type: application/json` if unset |
| `WithXMLBody(v)` | `xml.Marshal(v)`, sets `Content-Type: application/xml` if unset |
| `WithBody([]byte)` | raw body |
| `WithBodyString(string)` | raw body from a string |
| `WithIgnoreServerSSL(bool)` | skip TLS verification (defaults to true for `https://`) |

## Assertions

| Function | Checks |
|----------|--------|
| `ExpectStatusCode(resp, code)` | status code (failure message includes body) |
| `ExpectHeader(resp, key, value)` | header value, **case-insensitive key** |
| `ExpectJsonBody(resp, expected)` | whole JSON body deep-equals expected |
| `ExpectJsonBodyField(resp, path, expected)` | one field by path |
| `ExpectJsonBodyFieldCond(resp, path, condition, expected)` | field satisfies condition |
| `ExpectXmlBody(resp, xml)` | whole XML body structurally equal |
| `ExpectXmlBodyField(resp, path, expected)` | one XML node/attribute |
| `ExpectXmlBodyFieldCond(resp, path, condition, expected)` | node satisfies condition |
| `PrettyXml(s)` | helper to pretty-print XML for logs |

All `Expect*` are no-ops in dry-run mode.

### Field paths

- JSON: `a`, `b.c`, `d[0]`, `users[0].name`
- XML: `response.user.name`, `response.items.item[1]`, `response.user.@id`

## Examples

### GET + JSON

```go
resp := v1.SendRequest("http://localhost:8080/read?id=1")
v1.ExpectStatusCode(resp, 200)
v1.ExpectHeader(resp, "Content-Type", "application/json")
v1.ExpectJsonBodyField(resp, "id", "1")
v1.ExpectJsonBodyField(resp, "status", "updated")
```

### POST JSON with headers

```go
requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

resp := v1.SendRESTRequest("http://localhost:8080/update-json",
    v1.WithMethod(http.MethodPost),
    v1.WithHeader("X-Request-ID", requestID),
    v1.WithHeaders(map[string]string{"X-Trace": "itest"}),
    v1.WithJSONBody(map[string]interface{}{
        "id":     "1",
        "status": "json-updated",
    }),
)

v1.ExpectStatusCode(resp, 200)
v1.ExpectJsonBodyField(resp, "request_id", requestID)
v1.ExpectJsonBodyFieldCond(resp, "status", v1.ConditionStartsWith, "json-")
```

### POST XML

```go
resp := v1.SendRESTRequest("http://localhost:8080/update-xml",
    v1.WithMethod(http.MethodPost),
    v1.WithXMLBody(struct {
        XMLName struct{} `xml:"request"`
        ID      string   `xml:"id"`
        Status  string   `xml:"status"`
    }{ID: "1", Status: "xml-updated"},
)

v1.ExpectStatusCode(resp, 200)
v1.ExpectHeader(resp, "Content-Type", "application/xml")
v1.ExpectXmlBodyField(resp, "response.status", "xml-updated")
```

### Expect an error response

```go
resp := v1.SendRequest("http://localhost:8080/read?id=1") // server sees invalid DB state
v1.ExpectStatusCode(resp, http.StatusInternalServerError)
```

### Authenticated request

```go
resp := v1.SendRESTRequest("http://localhost:8080/admin",
    v1.WithHeader("Authorization", "Bearer "+token),
)
v1.ExpectStatusCode(resp, http.StatusOK)
```

### Self-signed HTTPS

```go
resp := v1.SendRESTRequest("https://localhost:8443/health",
    v1.WithIgnoreServerSSL(true),
)
```

## Tips

- `ExpectStatusCode` includes the response body in the failure message, which
  is usually enough to debug a bad request without extra logging.
- Use `ExpectJsonBodyFieldCond` for values you cannot know exactly (timestamps,
  generated ids): `ConditionNotEmpty`, `ConditionMatches`, `ConditionGreaterThan`.
- Prefer the dynamic mock server (see [Apps & mocks](07-app-and-mocks.md)) over
  hard-coded fake endpoints for dependencies.

Next: [Database & Redis →](06-database-and-redis.md)
