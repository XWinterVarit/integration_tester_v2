package main

import (
	"fmt"
	"strings"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runComplexExamples(client *dms.Client) {
	fmt.Println("5. Complex Example: Payment Transaction (Extraction & Logic)")

	// Register route
	err := client.RegisterRoute(MockPort, "POST", "/payment", []dms.ResponseFuncConfig{
		// 1. Extract Data from Request
		dms.ExtractRequestJsonBody("transaction_id", "TX_ID"),
		dms.ExtractRequestJsonBody("amount", "AMOUNT"),
		dms.ExtractRequestJsonBody("user.tier", "USER_TIER"),
		dms.ExtractRequestJsonBody("items[0].product_id", "FIRST_ITEM_ID"),

		// 2. Logic (Set Discount based on Tier)
		// Initialize DISCOUNT_DESC default
		dms.IfRequestJsonBody("user.tier", dms.ConditionEqual, "standard", "DISCOUNT_DESC", "No Discount"),
		dms.IfRequestJsonBody("user.tier", dms.ConditionEqual, "gold", "DISCOUNT_DESC", "10% Gold Discount Applied"),
		dms.IfRequestJsonBody("user.tier", dms.ConditionEqual, "platinum", "DISCOUNT_DESC", "20% Platinum Discount Applied"),

		// 3. Generate Data
		dms.GenerateRandomString(12, "REF_CODE"),
		dms.GenerateRandomInt(100, 500, "PROC_TIME_MS"),

		// 4. Setup Response
		dms.SetStatusCode("", 200),
		dms.SetHeader("", "X-Ref-Code", "{{.REF_CODE}}"),
		dms.SetJsonBody("", `{
			"status": "success",
			"transaction_id": "{{.TX_ID}}",
			"confirmation_code": "{{.REF_CODE}}",
			"details": {
				"amount_charged": {{.AMOUNT}},
				"discount_info": "{{.DISCOUNT_DESC}}",
				"first_item": "{{.FIRST_ITEM_ID}}"
			},
			"meta": {
				"tier": "{{.USER_TIER}}",
				"processed_in_ms": {{.PROC_TIME_MS}}
			}
		}`),
	})

	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	// Make Request
	reqBody := `{
		"transaction_id": "TX-999-888",
		"amount": 150.75,
		"user": {
			"id": 101,
			"tier": "gold"
		},
		"items": [
			{"product_id": "PROD-001", "qty": 1},
			{"product_id": "PROD-002", "qty": 2}
		]
	}`

	fmt.Printf("Sending Request: %s\n", reqBody)
	printRequest("POST", fmt.Sprintf("http://localhost:%d/payment", MockPort), strings.NewReader(reqBody))
}
