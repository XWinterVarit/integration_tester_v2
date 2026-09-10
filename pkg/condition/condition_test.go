package condition

import "testing"

func TestEvaluate(t *testing.T) {
	cases := []struct {
		name      string
		actual    interface{}
		condition string
		expected  interface{}
		want      bool
	}{
		{"equal int/float", 1, Equal, 1.0, true},
		{"equal nested numeric", []interface{}{1, 2}, Equal, []interface{}{1.0, 2.0}, true},
		{"equal nested map", map[string]interface{}{"a": 1}, Equal, map[string]interface{}{"a": 1.0}, true},
		{"strict string vs int", "1", Equal, 1, false},
		{"strict bool vs string", true, Equal, "true", false},
		{"strict nil vs string", nil, Equal, "<nil>", false},
		{"nil equal nil", nil, Equal, nil, true},
		{"nil not equal empty", nil, Equal, "", false},
		{"not equal", 1, NotEqual, 2, true},
		{"greater int", 10, GreaterThan, 5, true},
		{"greater numeric strings", "10", GreaterThan, 9, true},
		{"greater non numeric", "abc", GreaterThan, 5, false},
		{"less or equal", "5", LessThanOrEqual, 5, true},
		{"contains", "hello world", Contains, "world", true},
		{"not contains", "hello", NotContains, "xyz", true},
		{"starts with", "hello", StartsWith, "he", true},
		{"ends with", "hello", EndsWith, "lo", true},
		{"matches", "abc123", Matches, `^[a-z]+\d+$`, true},
		{"matches no match", "abc", Matches, `^\d+$`, false},
		{"matches invalid regex", "abc", Matches, `[`, false},
		{"in slice", "b", In, []string{"a", "b"}, true},
		{"in comma string", "b", In, "a, b, c", true},
		{"not in", "z", NotIn, []string{"a", "b"}, true},
		{"empty string", "", Empty, nil, true},
		{"empty slice", []interface{}{}, Empty, nil, true},
		{"not empty", "x", NotEmpty, nil, true},
		{"unknown condition", 1, "Nope", 1, false},
	}

	for _, tc := range cases {
		if got := Evaluate(tc.actual, tc.condition, tc.expected); got != tc.want {
			t.Errorf("%s: Evaluate(%v, %s, %v) = %v, want %v",
				tc.name, tc.actual, tc.condition, tc.expected, got, tc.want)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Validate(Equal); err != nil {
		t.Errorf("expected Equal to be valid, got %v", err)
	}
	if err := Validate("Nope"); err == nil {
		t.Error("expected unknown condition to be rejected")
	}
}

func TestValidateExpected(t *testing.T) {
	if err := ValidateExpected(Matches, `^a$`); err != nil {
		t.Errorf("valid regex rejected: %v", err)
	}
	if err := ValidateExpected(Matches, `[`); err == nil {
		t.Error("invalid regex accepted")
	}
	if err := ValidateExpected(In, []string{"a"}); err != nil {
		t.Errorf("valid In rejected: %v", err)
	}
	if err := ValidateExpected(In, 5); err == nil {
		t.Error("invalid In expected value accepted")
	}
}
