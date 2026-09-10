package main

import (
	"fmt"
	"strings"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runExtendedConditionsExamples(client *dms.Client) {
	fmt.Println("8. Extended Conditions: NotEqual, Contains, Numeric Comparisons")

	// Example 1: Age Verification (Numeric)
	// - Age >= 18 -> Allowed
	// - Age < 18 -> Denied
	err := client.RegisterRoute(MockPort, "POST", "/check-age", []dms.ResponseFuncConfig{
		dms.ExtractRequestJsonBody("age", "USER_AGE"),

		// Default Denied
		dms.SetStatusCode("", 403),
		dms.SetJsonBody("", `{"status": "denied", "reason": "Underage"}`),

		// Check if Age >= 18
		dms.IfDynamicVariableSetCase("USER_AGE", dms.ConditionGreaterThanOrEqual, 18, "Allowed"),

		dms.SetStatusCode("Allowed", 200),
		dms.SetJsonBody("Allowed", `{"status": "allowed", "age": {{.USER_AGE}}}`),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	fmt.Println("-> Checking Age 20 (Expect Allowed)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/check-age", MockPort),
		strings.NewReader(`{"age": 20}`))

	fmt.Println("-> Checking Age 15 (Expect Denied)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/check-age", MockPort),
		strings.NewReader(`{"age": 15}`))

	fmt.Println("-> Checking Age 'abc' (Expect Denied - Invalid Type)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/check-age", MockPort),
		strings.NewReader(`{"age": "abc"}`))

	// Example 2: Email Domain Check (String Contains/EndsWith)
	// - Email ends with "@company.com" -> Internal
	// - Otherwise -> External
	err = client.RegisterRoute(MockPort, "POST", "/check-email", []dms.ResponseFuncConfig{
		dms.ExtractRequestJsonBody("email", "USER_EMAIL"),

		// Default External
		dms.SetStatusCode("", 200),
		dms.SetJsonBody("", `{"type": "external"}`),

		// Check Domain
		dms.IfDynamicVariableSetCase("USER_EMAIL", dms.ConditionEndsWith, "@company.com", "Internal"),

		dms.SetJsonBody("Internal", `{"type": "internal"}`),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	fmt.Println("-> Checking 'user@company.com' (Expect Internal)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/check-email", MockPort),
		strings.NewReader(`{"email": "user@company.com"}`))

	fmt.Println("-> Checking 'user@gmail.com' (Expect External)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/check-email", MockPort),
		strings.NewReader(`{"email": "user@gmail.com"}`))
}
