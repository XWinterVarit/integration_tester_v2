package v1

import (
	"fmt"
	"reflect"
	"strings"
)

// evaluateCondition compares actual and expected according to the provided condition constant.
// It supports numeric comparisons, string comparisons (including contains/prefix/suffix),
// equality/non-equality, and nil (JSON null/DB NULL) handling.
func evaluateCondition(actual interface{}, condition string, expected interface{}) bool {
	switch condition {
	case ConditionEqual:
		return valuesEqual(actual, expected)
	case ConditionNotEqual:
		return !valuesEqual(actual, expected)
	case ConditionGreaterThan:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a > b })
	case ConditionLessThan:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a < b })
	case ConditionGreaterThanOrEqual:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a >= b })
	case ConditionLessThanOrEqual:
		return compareNumbers(actual, expected, func(a, b float64) bool { return a <= b })
	case ConditionContains:
		return stringContains(actual, expected, func(a, b string) bool { return strings.Contains(a, b) })
	case ConditionNotContains:
		return stringContains(actual, expected, func(a, b string) bool { return !strings.Contains(a, b) })
	case ConditionStartsWith:
		return stringContains(actual, expected, func(a, b string) bool { return strings.HasPrefix(a, b) })
	case ConditionEndsWith:
		return stringContains(actual, expected, func(a, b string) bool { return strings.HasSuffix(a, b) })
	default:
		return false
	}
}

func valuesEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	if isNumber(a) && isNumber(b) {
		return toFloat64(a) == toFloat64(b)
	}

	return reflect.DeepEqual(a, b)
}

func compareNumbers(a, b interface{}, cmp func(float64, float64) bool) bool {
	if a == nil || b == nil {
		return false
	}
	if isNumber(a) && isNumber(b) {
		return cmp(toFloat64(a), toFloat64(b))
	}
	return false
}

func stringContains(a, b interface{}, cmp func(string, string) bool) bool {
	if a == nil || b == nil {
		return false
	}
	return cmp(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}
