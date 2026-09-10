package main

import (
	"fmt"

	dms "github.com/XWinterVarit/integrate_tester_v2/pkg/dynamic-mock-server"
)

func runGeneratorExamples(client *dms.Client) {
	// Example 1: Random String and Int
	fmt.Println("1. Generators: Random String & Int")
	err := client.RegisterRoute(MockPort, "GET", "/random", []dms.ResponseFuncConfig{
		dms.GenerateRandomString(8, "RAND_CODE"),
		dms.GenerateRandomInt(100, 999, "RAND_ID"),

		// Note: "RAND_CODE" is string, so we quote it in JSON. "RAND_ID" is int, so we don't quote.
		dms.SetJsonBody("", `{"code": "{{.RAND_CODE}}", "id": {{.RAND_ID}}}`),
		dms.SetStatusCode("", 200),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	printRequest("GET", fmt.Sprintf("http://localhost:%d/random", MockPort), nil)
	printRequest("GET", fmt.Sprintf("http://localhost:%d/random", MockPort), nil) // Should be different

	// Example 2: Hashing and Conversion
	fmt.Println("2. Generators: Hashing & Conversion")
	err = client.RegisterRoute(MockPort, "GET", "/hash", []dms.ResponseFuncConfig{
		dms.GenerateRandomString(5, "SALT"),
		dms.HashedString("SALT", "MD5", "HASHED_SALT"),

		// Convert int to string to demonstrate usage (though template handles int fine usually)
		dms.GenerateRandomInt(1, 10, "NUM"),
		dms.ConvertToString("NUM"),

		dms.SetJsonBody("", `{"salt": "{{.SALT}}", "hash": "{{.HASHED_SALT}}", "num_as_string": "{{.NUM}}"}`),
		dms.SetStatusCode("", 200),
	})
	if err != nil {
		fmt.Printf("Error registering: %v\n", err)
		return
	}

	printRequest("GET", fmt.Sprintf("http://localhost:%d/hash", MockPort), nil)
}
