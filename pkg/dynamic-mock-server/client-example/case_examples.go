package main

import (
	"fmt"
	"net/http"
	"strings"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runCaseExamples(client *dms.Client) {
	fmt.Println("6. Case Logic Example: Success vs Failure scenarios")

	// Scenario: Create User
	// - Default: 400 Bad Request (Missing or invalid data)
	// - Case "Success": username = "new_user" -> 201 Created
	// - Case "Conflict": username = "existing_user" -> 409 Conflict
	// - Case "Maintenance": header X-Mode = "maintenance" -> 503 Service Unavailable

	err := client.RegisterRoute(MockPort, "POST", "/create-user", []dms.ResponseFuncConfig{
		// 1. Determine Case
		// Check for maintenance mode first via Header
		dms.IfRequestHeaderSetCase("X-Mode", dms.ConditionEqual, "maintenance", "Maintenance"),

		// Check body content for specific usernames (only if not maintenance, though simple logic evaluates all)
		// Note: The last matching SetCase will overwrite previous ones if logic allows,
		// but usually we expect distinct conditions.
		dms.IfRequestJsonBodySetCase("username", dms.ConditionEqual, "new_user", "Success"),
		dms.IfRequestJsonBodySetCase("username", dms.ConditionEqual, "existing_user", "Conflict"),

		// 2. Define Responses for each Case

		// Default Case (Validation Error)
		dms.SetStatusCode("", 400),
		dms.SetJsonBody("", `{"error": "Validation failed or invalid username"}`),

		// Case: Success
		dms.SetStatusCode("Success", 201),
		dms.SetJsonBody("Success", `{"id": 101, "status": "created", "username": "new_user"}`),

		// Case: Conflict
		dms.SetStatusCode("Conflict", 409),
		dms.SetJsonBody("Conflict", `{"error": "Username already exists"}`),

		// Case: Maintenance
		dms.SetStatusCode("Maintenance", 503),
		dms.SetJsonBody("Maintenance", `{"error": "System is under maintenance, please try again later"}`),
	})

	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	// Test 1: Success
	fmt.Println("-> Sending Request: Create 'new_user' (Expect 201)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/create-user", MockPort),
		strings.NewReader(`{"username": "new_user"}`))

	// Test 2: Conflict
	fmt.Println("-> Sending Request: Create 'existing_user' (Expect 409)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/create-user", MockPort),
		strings.NewReader(`{"username": "existing_user"}`))

	// Test 3: Maintenance (Header based)
	fmt.Println("-> Sending Request: With Maintenance Header (Expect 503)")
	// Helper to send with header
	requestWithHeader(fmt.Sprintf("http://localhost:%d/create-user", MockPort), "X-Mode", "maintenance", `{"username": "any"}`)

	// Test 4: Default (Invalid/Unknown)
	fmt.Println("-> Sending Request: Create 'invalid_user' (Expect 400)")
	printRequest("POST", fmt.Sprintf("http://localhost:%d/create-user", MockPort),
		strings.NewReader(`{"username": "invalid_user"}`))
}

func requestWithHeader(url, key, value, bodyStr string) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(bodyStr))
	req.Header.Set(key, value)
	printRequestWithReq(req)
}
