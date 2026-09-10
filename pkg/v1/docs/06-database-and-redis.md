# 6. Database & Redis

## Database (`db.go`)

`DBClient` wraps `database/sql`. Import the driver in your `main` package.

```go
import _ "github.com/mattn/go-sqlite3" // or go-ora, etc.
```

### Connect

```go
db := v1.Connect("sqlite3", ":memory:")
db := v1.Connect("oracle", "oracle://user:pass@localhost:1521/XE")
```

`Connect` pings the database and fails the stage if unreachable. In dry-run it
returns an unconnected client and does nothing.

### Schema

```go
type Field struct { Name string; Type string }
type Index struct { Columns []string }

db.SetupTable("users", true,
    []v1.Field{
        {"id", "INTEGER PRIMARY KEY AUTOINCREMENT"},
        {"name", "TEXT"},
        {"status", "TEXT"},
    },
    []v1.Index{{Columns: []string{"name"}}},
)
```

- `isReplace = true` drops and recreates the table (idempotent setup).
- Column types are passed straight to the driver, so use the syntax of your DB
  (e.g. Oracle `NUMBER PRIMARY KEY`, `VARCHAR2(100)`).
- `SetupTableFromAnother(destClient, destTable, srcClient, srcTable, true)`
  copies schema from another client.

Other schema helpers: `DropTable(table)`, `CleanTable(table)` (delete all rows).

### Writing data

```go
type InsertField struct { Key string; Value interface{} }

db.InsertOne("users", []v1.InsertField{
    {Key: "name", Value: "alice"},
    {Key: "status", Value: "active"},
})

db.ReplaceData("users", []interface{}{1, "alice", "active"})

db.Update("users", map[string]interface{}{"status": "updated"}, "id = ?", 1)

db.DeleteOne("users", "id = ?", 1)
db.DeleteWithLimit("users", "status = ?", 10, "bad")
```

> Write placeholders as `?`. The library translates them to the right syntax for
> the driver (e.g. `:1` for Oracle) automatically.

### Reading & asserting

```go
result := db.Fetch("SELECT name, status FROM users WHERE id = ?", 1)

result.ExpectCount(1)

row := result.GetRow(0)
name := row.Get("name")                 // interface{}; fails if field missing
row.Expect("status", "updated")          // equality with string fallback
row.ExpectCond("score", v1.ConditionGreaterThanOrEqual, 10)
```

- `Fetch(query, args...)` returns `*QueryResult` with `Rows []RowResult`.
- `QueryResult`: `Count() int`, `GetRow(i) *RowResult`, `ExpectCount(n)`.
- `RowResult`: `Get(field)`, `GetTo(field, &target)`, `Expect(field, value)`,
  `ExpectCond(field, condition, value)`.
- Column names are matched **case-insensitively** (stored lowercased).
- `Get`/`GetRow` fail the stage (panic) on missing field/index.

`GetTo` scans into a pointer using `fmt.Sscan`:

```go
var score int
row.GetTo("score", &score)
```

## Redis (`redis.go`)

`RedisClient` talks to the **redis-mock-server** over HTTP (server address +
access key), which proxies to a real Redis.

```go
redis := v1.ConnectRedis("http://localhost:9100", "welcome")
```

### Basic keys

```go
redis.Set("foo", "bar", 0)
redis.ExpectValue("foo", "bar")
redis.ExpectFound("foo")
redis.ExpectNotFound("missing")

value := redis.Get("foo")
redis.Del("foo")
redis.FlushAll()
```

### Hashes

```go
redis.HSet("profile:1", "name", "Alice")
redis.HSet("profile:1", "email", "alice@example.com")

name := redis.HGet("profile:1", "name")
v1.Assert(name == "Alice", "unexpected name: %s", name)

newViews := redis.HIncrement("stats:page", "views", 10) // int64
v1.Assert(newViews == 10, "unexpected views: %d", newViews)
```

### JSON fields

For values stored as JSON strings, you can get/set/assert nested paths:

```go
redis.SetJsonField("user:1", "profile.name", "Alice")
redis.ExpectJsonField("user:1", "profile.name", "Alice")

redis.HSetJsonField("user:1", "data", "score", 10)
redis.HExpectJsonField("user:1", "data", "score", 10)
```

JSON paths use dot notation and array indexes (`items[0].id`).

## Dry-run behavior

Every DB/Redis helper records an action first and returns early when
`IsDryRun()` is true, so discovery never touches real infrastructure.

## Common pitfalls

- Forgetting to import the driver in `main` → `sql: unknown driver`.
- Using the wrong placeholder syntax → just use `?`.
- Assuming `Get` returns a concrete type; DB drivers may return `[]byte`
  (converted to `string` for you) or varied numeric types — prefer `ExpectCond`.
- A missing column in `Get`/`Expect` fails the stage; check `Fetch`'s SELECT list.

Next: [Apps & mocks →](07-app-and-mocks.md)
