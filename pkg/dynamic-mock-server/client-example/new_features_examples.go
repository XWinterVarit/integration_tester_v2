package main

import (
	"fmt"
	"strings"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runNewFeaturesExamples(client *dms.Client) {
	fmt.Println("7. New Features: Variable Logic, JSON Validation, Manipulation & SetCase")

	// Route: POST /validate-order
	// Logic:
	// 1. Validate "items" array length == 2 -> LEN_OK="yes"
	// 2. Validate "metadata" is object -> META_OK="yes"
	// 3. Extract order_id, substring prefix, check if "ORD" -> PREFIX_OK="yes"
	// 4. Join validation results -> VALIDATION_KEY
	// 5. If VALIDATION_KEY == "yes-yes-yes" -> SetCase "Success"
	err := client.RegisterRoute(MockPort, "POST", "/validate-order", []dms.ResponseFuncConfig{
		// 1. Validations
		dms.IfRequestJsonType("items", "array", "ITEMS_IS_ARRAY", "yes"), // Just for info
		dms.IfRequestJsonArrayLength("items", dms.ConditionEqual, 2, "LEN_OK", "yes"),
		dms.IfRequestJsonType("metadata", "object", "META_OK", "yes"),

		dms.ExtractRequestJsonBody("order_id", "ORDER_ID"),      // e.g. "ORD-123"
		dms.DynamicVarSubstring("ORDER_ID", 0, 3, "ORD_PREFIX"), // "ORD"
		dms.IfDynamicVariable("ORD_PREFIX", dms.ConditionEqual, "ORD", "PREFIX_OK", "yes"),

		// 2. Aggregate Logic
		// We join the success flags. If any is missing (nil/empty), the join will look different (e.g. "-yes-yes" or "yes--yes")
		dms.DynamicVarJoin("VALIDATION_KEY", "-", "{{.LEN_OK}}", "{{.META_OK}}", "{{.PREFIX_OK}}"),

		// 3. Case Switching
		dms.IfDynamicVariableSetCase("VALIDATION_KEY", dms.ConditionEqual, "yes-yes-yes", "Success"),

		// 4. Response - Default (Failure)
		dms.SetStatusCode("", 400),
		dms.SetJsonBody("", `{
			"status": "error",
			"message": "Validation Failed",
			"debug_key": "{{.VALIDATION_KEY}}"
		}`),

		// 5. Response - Success
		dms.SetStatusCode("Success", 200),
		dms.SetJsonBody("Success", `{
			"status": "success",
			"order_id": "{{.ORDER_ID}}",
			"message": "Order validated and processed"
		}`),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	// Test 1: Valid Request
	fmt.Println("-> Sending Valid Request (Expect 200 Success)")
	validBody := `{
		"order_id": "ORD-999",
		"items": [{"id":1}, {"id":2}],
		"metadata": {"source": "web"}
	}`
	printRequest("POST", fmt.Sprintf("http://localhost:%d/validate-order", MockPort), strings.NewReader(validBody))

	// Test 2: Invalid Request (Wrong Length)
	fmt.Println("-> Sending Invalid Request (Wrong Array Length) (Expect 400 Error)")
	invalidBody1 := `{
		"order_id": "ORD-888",
		"items": [{"id":1}],
		"metadata": {"source": "web"}
	}`
	printRequest("POST", fmt.Sprintf("http://localhost:%d/validate-order", MockPort), strings.NewReader(invalidBody1))

	// Test 3: Invalid Request (Wrong Prefix)
	fmt.Println("-> Sending Invalid Request (Wrong Prefix) (Expect 400 Error)")
	invalidBody2 := `{
		"order_id": "XXX-777",
		"items": [{"id":1}, {"id":2}],
		"metadata": {"source": "web"}
	}`
	printRequest("POST", fmt.Sprintf("http://localhost:%d/validate-order", MockPort), strings.NewReader(invalidBody2))
}
