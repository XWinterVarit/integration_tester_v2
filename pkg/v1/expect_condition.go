package v1

import (
	"encoding/json"
	"fmt"

	cond "github.com/XWinterVarit/integrate_tester_v2/pkg/condition"
)

// Condition names, re-exported from pkg/condition so callers can keep using
// v1.ConditionEqual, v1.ConditionContains, etc.
const (
	ConditionEqual              = cond.Equal
	ConditionNotEqual           = cond.NotEqual
	ConditionContains           = cond.Contains
	ConditionNotContains        = cond.NotContains
	ConditionStartsWith         = cond.StartsWith
	ConditionEndsWith           = cond.EndsWith
	ConditionGreaterThan        = cond.GreaterThan
	ConditionLessThan           = cond.LessThan
	ConditionGreaterThanOrEqual = cond.GreaterThanOrEqual
	ConditionLessThanOrEqual    = cond.LessThanOrEqual
	ConditionMatches            = cond.Matches
	ConditionIn                 = cond.In
	ConditionNotIn              = cond.NotIn
	ConditionEmpty              = cond.Empty
	ConditionNotEmpty           = cond.NotEmpty
)

// ValidateCondition returns an error when condition is not recognised, listing
// the supported names. Use it to fail fast on typos instead of silently
// treating them as "no match".
func ValidateCondition(condition string) error {
	return cond.Validate(condition)
}

// evaluateCondition is the shared strict condition evaluator (pkg/condition).
func evaluateCondition(actual interface{}, condition string, expected interface{}) bool {
	return cond.Evaluate(actual, condition, expected)
}

func validateExpected(condition string, expected interface{}) error {
	return cond.ValidateExpected(condition, expected)
}

// AssertCondition asserts that actual satisfies condition against expected,
// using the same semantics as the Expect* helpers. format/args build the failure
// message; when omitted a default message is used.
func AssertCondition(actual interface{}, condition string, expected interface{}, format string, args ...interface{}) {
	if IsDryRun() {
		return
	}
	if err := ValidateCondition(condition); err != nil {
		Fail("AssertCondition failed: %v", err)
	}
	if err := validateExpected(condition, expected); err != nil {
		Fail("AssertCondition failed: %v", err)
	}
	if !evaluateCondition(actual, condition, expected) {
		msg := ""
		if format != "" {
			msg = fmt.Sprintf(format, args...)
		}
		if msg != "" {
			Fail("AssertCondition failed: %s\nActual: %v (%T)\nExpected: %v (%T) [%s]", msg, actual, actual, expected, expected, condition)
		}
		Fail("AssertCondition failed:\nActual: %v (%T)\nExpected: %v (%T) [%s]", actual, actual, expected, expected, condition)
	}
	Logf(LogTypeExpect, "Condition %s %v - PASSED", condition, expected)
}

// normalizeJSONValue round-trips v through JSON so that native Go values (ints,
// structs, ...) are converted to the same float64/map/slice representation that
// json.Unmarshal produces.
func normalizeJSONValue(v interface{}) interface{} {
	if _, isJSONNumber := v.(json.Number); isJSONNumber {
		return v
	}
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}
