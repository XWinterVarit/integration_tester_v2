# 4. Assertions & conditions

## Failure primitives

```go
func Fail(format string, args ...interface{})   // log error + panic(TestError)
func Assert(cond bool, format string, args ...interface{})
func AssertNoError(err error)
```

- `Fail` is the single failure primitive. It logs `Assertion FAILED` and panics
  with `v1.TestError`; the stage runner recovers it and returns an error.
- In **dry-run** mode `Fail` logs "Assertion skipped in dry-run" and returns
  instead of panicking.
- `Assert` / `AssertNoError` are thin wrappers around `Fail`.

```go
v1.Assert(resp.StatusCode == 200, "health failed: %d", resp.StatusCode)
v1.AssertNoError(err)
```

## The condition engine (`pkg/condition`)

All condition-based assertions share one strict evaluator:

```go
import cond "github.com/XWinterVarit/integrate_tester_v2/pkg/condition"

cond.Evaluate(actual, condition, expected) bool
cond.Validate(condition) error
cond.ValidateExpected(condition, expected) error
cond.IsNumber(v) bool
cond.ToFloat64(v) float64
```

`v1` re-exports the condition names as `v1.ConditionXxx`, and offers:

```go
func AssertCondition(actual interface{}, condition string, expected interface{},
    format string, args ...interface{})
```

```go
v1.AssertCondition(resp.StatusCode, v1.ConditionIn, []interface{}{200, 201}, "unexpected status")
v1.AssertCondition(row.Get("name"), v1.ConditionMatches, `^ali`, "bad name")
```

## Condition reference

| Constant | Value | Meaning |
|----------|-------|---------|
| `ConditionEqual` | `Equal` | strict deep equality |
| `ConditionNotEqual` | `NotEqual` | negation of Equal |
| `ConditionGreaterThan` | `GreaterThan` | numeric `>` |
| `ConditionLessThan` | `LessThan` | numeric `<` |
| `ConditionGreaterThanOrEqual` | `GreaterThanOrEqual` | numeric `>=` |
| `ConditionLessThanOrEqual` | `LessThanOrEqual` | numeric `<=` |
| `ConditionContains` | `Contains` | string contains |
| `ConditionNotContains` | `NotContains` | string does not contain |
| `ConditionStartsWith` | `StartsWith` | string prefix |
| `ConditionEndsWith` | `EndsWith` | string suffix |
| `ConditionMatches` | `Matches` | regex match on string form |
| `ConditionIn` | `In` | equals any element of a collection |
| `ConditionNotIn` | `NotIn` | not in collection |
| `ConditionEmpty` | `Empty` | nil / `""` / empty slice/map |
| `ConditionNotEmpty` | `NotEmpty` | negation of Empty |

### Strict semantics

- **Equality is typed.** It does **not** stringify values: `"1" != 1`,
  `true != "true"`, `nil != "<nil>"`. `nil` only equals `nil`.
- **Numeric normalization.** `int`, `uint`, `float*` and `json.Number` compare
  by numeric value, including inside maps and slices (recursively).
- **Ordering accepts numeric strings.** `"10" > 9` is true. This makes
  comparisons work on XML text and query/path values.
- **Maps/slices are compared recursively** with the rules above.
- **Collections** for `In`/`NotIn` may be `[]interface{}`, `[]string`, or a
  comma-separated string (`"a, b, c"`).
- **Unknown condition names fail fast** when validated (`Validate`), rather than
  silently returning false.

## Using conditions in assertions

### JSON field + condition

```go
v1.ExpectJsonBodyFieldCond(resp, "user.age", v1.ConditionGreaterThanOrEqual, 18)
v1.ExpectJsonBodyFieldCond(resp, "user.role", v1.ConditionIn, []string{"admin", "owner"})
v1.ExpectJsonBodyFieldCond(resp, "items[0].name", v1.ConditionStartsWith, "PROD-")
```

`field` supports dot paths and array indexes: `data.users[0].name`.

### XML field + condition

```go
v1.ExpectXmlBodyFieldCond(resp, "response.user.name", v1.ConditionStartsWith, "Ali")
v1.ExpectXmlBodyFieldCond(resp, "response.count", v1.ConditionGreaterThan, "5")
```

### Database field + condition

```go
row := db.Fetch("SELECT status, score FROM users WHERE id = ?", 1).GetRow(0)
row.ExpectCond("status", v1.ConditionEqual, "updated")
row.ExpectCond("score", v1.ConditionGreaterThanOrEqual, 10)
```

## Plain JSON / XML assertions

```go
v1.ExpectJsonBody(resp, `{"a":1,"b":[1,2]}`)         // JSON string
v1.ExpectJsonBody(resp, map[string]interface{}{...}) // native map/struct — normalized
v1.ExpectJsonBodyField(resp, "a", 1)
v1.ExpectXmlBody(resp, `<response><id>1</id></response>`)
v1.ExpectXmlBodyField(resp, "response.user.@id", "42")
```

`ExpectJsonBody` normalizes native Go values through JSON first, so `int` vs
`float64` differences do not cause false failures.

## Choosing between `==` and conditions

Use `AssertCondition` / `ExpectCond` when values may come from different types
(JSON numbers are `float64`, DB drivers vary). Use `==` only for values you
control and know the exact type of.

Next: [HTTP testing →](05-http.md)
