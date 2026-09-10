package v1

import "testing"

func expectConditionPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("%s expected to panic", name)
			return
		}
		if _, ok := r.(TestError); !ok {
			t.Errorf("%s panicked with unexpected type: %T", name, r)
		}
	}()
	f()
}

func TestEvaluateCondition(t *testing.T) {
	cases := []struct {
		name      string
		actual    interface{}
		condition string
		expected  interface{}
		want      bool
	}{
		{"equal int/float", 1, ConditionEqual, 1.0, true},
		{"equal nested numeric", []interface{}{1, 2}, ConditionEqual, []interface{}{1.0, 2.0}, true},
		{"equal nested map", map[string]interface{}{"a": 1}, ConditionEqual, map[string]interface{}{"a": 1.0}, true},
		{"not equal", 1, ConditionNotEqual, 2, true},
		{"greater on numeric strings", "10", ConditionGreaterThan, 5, true},
		{"greater on numeric strings both", "10", ConditionGreaterThan, "9", true},
		{"less or equal numeric string", "5", ConditionLessThanOrEqual, 5, true},
		{"greater non numeric", "abc", ConditionGreaterThan, 5, false},
		{"contains", "hello world", ConditionContains, "world", true},
		{"matches", "abc123", ConditionMatches, `^[a-z]+\d+$`, true},
		{"matches no match", "abc", ConditionMatches, `^\d+$`, false},
		{"in slice", "b", ConditionIn, []string{"a", "b", "c"}, true},
		{"in comma string", "b", ConditionIn, "a, b, c", true},
		{"not in", "z", ConditionNotIn, []string{"a", "b"}, true},
		{"empty string", "", ConditionEmpty, nil, true},
		{"empty slice", []interface{}{}, ConditionEmpty, nil, true},
		{"not empty", "x", ConditionNotEmpty, nil, true},
		{"nil equal nil", nil, ConditionEqual, nil, true},
		{"nil not equal empty", nil, ConditionEqual, "", false},
	}

	for _, tc := range cases {
		if got := evaluateCondition(tc.actual, tc.condition, tc.expected); got != tc.want {
			t.Errorf("%s: evaluateCondition(%v, %s, %v) = %v, want %v",
				tc.name, tc.actual, tc.condition, tc.expected, got, tc.want)
		}
	}
}

func TestValidateCondition(t *testing.T) {
	if err := ValidateCondition(ConditionEqual); err != nil {
		t.Errorf("expected ConditionEqual to be valid, got %v", err)
	}
	if err := ValidateCondition("Nope"); err == nil {
		t.Error("expected unknown condition to be rejected")
	}
}

func TestAssertCondition(t *testing.T) {
	// Success (should not panic).
	AssertCondition(10, ConditionGreaterThan, 5, "ten > five")
	AssertCondition("b", ConditionIn, []string{"a", "b"}, "b in set")
	AssertCondition("", ConditionEmpty, nil, "empty")

	// Failure: mismatch.
	expectConditionPanic(t, "mismatch", func() {
		AssertCondition(1, ConditionGreaterThan, 5, "one > five")
	})

	// Failure: unknown condition.
	expectConditionPanic(t, "unknown condition", func() {
		AssertCondition(1, "Nope", 1, "")
	})

	// Failure: invalid regex.
	expectConditionPanic(t, "invalid regex", func() {
		AssertCondition("x", ConditionMatches, "[", "")
	})
}

func TestExpectJsonBodyNativeValues(t *testing.T) {
	// Native map with ints should normalize to the JSON float64 representation.
	ExpectJsonBody(Response{Body: `{"a":1,"b":[1,2],"nested":{"x":2}}`},
		map[string]interface{}{
			"a":      1,
			"b":      []interface{}{1, 2},
			"nested": map[string]interface{}{"x": 2},
		})

	// Expected struct should be normalized via JSON.
	type sample struct {
		A int   `json:"a"`
		B []int `json:"b"`
	}
	ExpectJsonBody(Response{Body: `{"a":1,"b":[1,2]}`}, sample{A: 1, B: []int{1, 2}})

	// Type mismatch still fails.
	expectConditionPanic(t, "ExpectJsonBody mismatch", func() {
		ExpectJsonBody(Response{Body: `{"a":1}`}, map[string]interface{}{"a": 2})
	})
}

func TestExpectXmlBodyFieldCondNumeric(t *testing.T) {
	resp := Response{Body: `<response><count>10</count></response>`}

	// Numeric ordering on XML text values (previously always failed).
	ExpectXmlBodyFieldCond(resp, "response.count", ConditionGreaterThan, "5")
	ExpectXmlBodyFieldCond(resp, "response.count", ConditionLessThan, "20")
	ExpectXmlBodyFieldCond(resp, "response.count", ConditionEqual, "10")

	expectConditionPanic(t, "xml numeric mismatch", func() {
		ExpectXmlBodyFieldCond(resp, "response.count", ConditionGreaterThan, "100")
	})
	expectConditionPanic(t, "xml unknown condition", func() {
		ExpectXmlBodyFieldCond(resp, "response.count", "Nope", "5")
	})
}

func TestExpectHeaderCaseInsensitive(t *testing.T) {
	resp := Response{Header: map[string]string{"Content-Type": "application/json"}}
	ExpectHeader(resp, "content-type", "application/json")

	expectConditionPanic(t, "missing header", func() {
		ExpectHeader(resp, "X-Missing", "value")
	})
}
