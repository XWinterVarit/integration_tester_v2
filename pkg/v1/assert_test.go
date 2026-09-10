package v1

import (
	"fmt"
	"testing"
)

func TestFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("Fail did not panic")
		}
		te, ok := r.(TestError)
		if !ok {
			t.Errorf("Fail did not panic with TestError, got %T", r)
		}
		if te.Message != "Fail message: 123" {
			t.Errorf("Unexpected message: %s", te.Message)
		}
	}()

	Fail("Fail message: %d", 123)
}

func TestAssert(t *testing.T) {
	// Case 1: Success
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Assert(true) panicked: %v", r)
			}
		}()
		Assert(true, "Should not panic")
	}()

	// Case 2: Failure
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("Assert(false) did not panic")
			}
			te, ok := r.(TestError)
			if !ok {
				t.Errorf("Panic was not TestError")
			}
			if te.Message != "Assertion failed" {
				t.Errorf("Unexpected message: %s", te.Message)
			}
		}()
		Assert(false, "Assertion failed")
	}()
}

func TestAssertNoError(t *testing.T) {
	// Case 1: No Error
	AssertNoError(nil)

	// Case 2: Error
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("AssertNoError(err) did not panic")
		}
		te, ok := r.(TestError)
		if !ok || te.Message != "Unexpected error: some error" {
			t.Errorf("Unexpected panic value: %v", r)
		}
	}()
	AssertNoError(fmt.Errorf("some error"))
}
