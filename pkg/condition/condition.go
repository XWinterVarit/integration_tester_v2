// Package condition is the single, shared implementation of the condition
// vocabulary used by pkg/v1 assertions and the dynamic mock server's request
// matching. It is intentionally dependency-free (standard library only) so both
// packages can import it without creating an import cycle.
//
// Semantics are strict/typed:
//   - numeric equality normalizes int/uint/float/json.Number by value;
//   - ordering comparisons accept numeric strings (e.g. XML text values);
//   - equality does NOT stringify: "1" is not equal to 1;
//   - maps/slices are compared recursively with numeric normalization;
//   - nil only equals nil.
package condition

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Condition names.
const (
	Equal              = "Equal"
	NotEqual           = "NotEqual"
	Contains           = "Contains"
	NotContains        = "NotContains"
	StartsWith         = "StartsWith"
	EndsWith           = "EndsWith"
	GreaterThan        = "GreaterThan"
	LessThan           = "LessThan"
	GreaterThanOrEqual = "GreaterThanOrEqual"
	LessThanOrEqual    = "LessThanOrEqual"
	Matches            = "Matches"
	In                 = "In"
	NotIn              = "NotIn"
	Empty              = "Empty"
	NotEmpty           = "NotEmpty"
)

var supported = map[string]struct{}{
	Equal:              {},
	NotEqual:           {},
	Contains:           {},
	NotContains:        {},
	StartsWith:         {},
	EndsWith:           {},
	GreaterThan:        {},
	LessThan:           {},
	GreaterThanOrEqual: {},
	LessThanOrEqual:    {},
	Matches:            {},
	In:                 {},
	NotIn:              {},
	Empty:              {},
	NotEmpty:           {},
}

// Validate returns an error when condition is not recognised, listing the
// supported names.
func Validate(condition string) error {
	if _, ok := supported[condition]; ok {
		return nil
	}
	names := make([]string, 0, len(supported))
	for name := range supported {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Errorf("unknown condition %q (supported: %s)", condition, strings.Join(names, ", "))
}

// ValidateExpected checks that the expected value is usable with the condition
// (e.g. a compilable regex, or a collection for In/NotIn).
func ValidateExpected(condition string, expected interface{}) error {
	switch condition {
	case Matches:
		if _, err := regexp.Compile(fmt.Sprintf("%v", expected)); err != nil {
			return fmt.Errorf("condition %s has invalid regex %q: %v", condition, expected, err)
		}
	case In, NotIn:
		switch expected.(type) {
		case []interface{}, []string, string:
		default:
			return fmt.Errorf("condition %s expects a slice or comma-separated string, got %T", condition, expected)
		}
	}
	return nil
}

// Evaluate reports whether actual satisfies condition against expected. Unknown
// conditions return false; call Validate first to surface typos.
func Evaluate(actual interface{}, condition string, expected interface{}) bool {
	switch condition {
	case Equal:
		return valuesEqual(actual, expected)
	case NotEqual:
		return !valuesEqual(actual, expected)
	case GreaterThan:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a > b })
	case LessThan:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a < b })
	case GreaterThanOrEqual:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a >= b })
	case LessThanOrEqual:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a <= b })
	case Contains:
		return stringContains(actual, expected, func(a, b string) bool { return strings.Contains(a, b) })
	case NotContains:
		return stringContains(actual, expected, func(a, b string) bool { return !strings.Contains(a, b) })
	case StartsWith:
		return stringContains(actual, expected, func(a, b string) bool { return strings.HasPrefix(a, b) })
	case EndsWith:
		return stringContains(actual, expected, func(a, b string) bool { return strings.HasSuffix(a, b) })
	case Matches:
		re, err := regexp.Compile(fmt.Sprintf("%v", expected))
		if err != nil {
			return false
		}
		return re.MatchString(fmt.Sprintf("%v", actual))
	case In:
		return valueIn(actual, expected)
	case NotIn:
		return !valueIn(actual, expected)
	case Empty:
		return isEmptyValue(actual)
	case NotEmpty:
		return !isEmptyValue(actual)
	default:
		return false
	}
}

// valuesEqual reports deep equality with numeric normalization, so int/float
// (and json.Number) values compare by numeric value even inside maps/slices.
func valuesEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	if IsNumber(a) && IsNumber(b) {
		return ToFloat64(a) == ToFloat64(b)
	}

	switch av := a.(type) {
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !valuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bvVal, ok := bv[k]
			if !ok || !valuesEqual(v, bvVal) {
				return false
			}
		}
		return true
	}

	return reflect.DeepEqual(a, b)
}

// compareNumbers parses both operands as numbers (accepting numeric strings such
// as XML text values) and applies cmp. Returns false when either side is not
// numeric.
func compareNumbers(a, b interface{}, cmp func(float64, float64) bool) bool {
	af, ok1 := toNumberLoose(a)
	bf, ok2 := toNumberLoose(b)
	if !ok1 || !ok2 {
		return false
	}
	return cmp(af, bf)
}

func stringContains(a, b interface{}, cmp func(string, string) bool) bool {
	if a == nil || b == nil {
		return false
	}
	return cmp(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// valueIn reports whether actual equals any element of expected. expected may be
// a []interface{}, []string, or a comma-separated string.
func valueIn(actual, expected interface{}) bool {
	switch e := expected.(type) {
	case []interface{}:
		for _, item := range e {
			if valuesEqual(actual, item) {
				return true
			}
		}
	case []string:
		for _, item := range e {
			if valuesEqual(actual, item) {
				return true
			}
		}
	case string:
		for _, item := range strings.Split(e, ",") {
			if valuesEqual(actual, strings.TrimSpace(item)) {
				return true
			}
		}
	default:
		return valuesEqual(actual, expected)
	}
	return false
}

// isEmptyValue reports whether v is nil, an empty string, or an empty
// slice/array/map.
func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() == 0
	}
	return false
}

// toNumberLoose converts numeric types and numeric strings to float64.
func toNumberLoose(v interface{}) (float64, bool) {
	if IsNumber(v) {
		return ToFloat64(v), true
	}
	if s, ok := v.(string); ok {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// IsNumber reports whether v is one of the supported numeric types.
func IsNumber(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, json.Number:
		return true
	}
	return false
}

// ToFloat64 converts a supported numeric value to float64, returning 0 for
// non-numeric input.
func ToFloat64(v interface{}) float64 {
	if n, ok := v.(json.Number); ok {
		f, err := n.Float64()
		if err != nil {
			return 0
		}
		return f
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	}
	return 0
}
