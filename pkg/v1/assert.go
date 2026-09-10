package v1

import "fmt"

// TestError represents a controlled test failure.
type TestError struct {
	Message string
}

func (e TestError) Error() string {
	return e.Message
}

// Fail fails the current test stage with a message.
// It uses panic with TestError to stop execution, which is caught by the Stage runner.
func Fail(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	// In dry-run mode we skip panicking and avoid emitting error logs so that
	// discovery does not surface false failures when dependencies are absent.
	if IsDryRun() {
		Log(LogTypeInfo, "Assertion skipped in dry-run", msg)
		return
	}

	Log(LogTypeError, "Assertion FAILED", msg)
	panic(TestError{Message: msg})
}

// Assert checks if the condition is true. If not, it fails the test stage.
func Assert(condition bool, format string, args ...interface{}) {
	if !condition {
		Fail(format, args...)
	}
}

// AssertNoError asserts that the error is nil.
func AssertNoError(err error) {
	if err != nil {
		Fail("Unexpected error: %v", err)
	}
}
