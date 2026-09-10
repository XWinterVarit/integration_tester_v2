package dynamic_mock_server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerExecutor_ExtendedConditions(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	h := NewHandlerExecutor(httptest.NewRecorder(), req)

	// Setup Variables
	h.Variables["STR_VAL"] = "Hello World"
	h.Variables["NUM_VAL"] = 100
	h.Variables["NUM_STR"] = "200"
	h.Variables["BAD_NUM"] = "abc"

	tests := []struct {
		name      string
		condition string
		expected  interface{}
		target    string
		toBe      interface{}
		varName   string
		shouldSet bool
	}{
		// String Comparisons
		{"NotEqual_True", ConditionNotEqual, "Goodbye", "RES_NE", "yes", "STR_VAL", true},
		{"NotEqual_False", ConditionNotEqual, "Hello World", "RES_NE_F", "yes", "STR_VAL", false},
		{"Contains_True", ConditionContains, "World", "RES_C", "yes", "STR_VAL", true},
		{"Contains_False", ConditionContains, "Universe", "RES_C_F", "yes", "STR_VAL", false},
		{"NotContains_True", ConditionNotContains, "Universe", "RES_NC", "yes", "STR_VAL", true},
		{"StartsWith_True", ConditionStartsWith, "Hello", "RES_SW", "yes", "STR_VAL", true},
		{"EndsWith_True", ConditionEndsWith, "World", "RES_EW", "yes", "STR_VAL", true},

		// Numeric Comparisons (Int Variable)
		{"GT_True", ConditionGreaterThan, 50, "RES_GT", "yes", "NUM_VAL", true},
		{"GT_False", ConditionGreaterThan, 150, "RES_GT_F", "yes", "NUM_VAL", false},
		{"LT_True", ConditionLessThan, 150, "RES_LT", "yes", "NUM_VAL", true},
		{"GTE_True", ConditionGreaterThanOrEqual, 100, "RES_GTE", "yes", "NUM_VAL", true},
		{"LTE_True", ConditionLessThanOrEqual, 100, "RES_LTE", "yes", "NUM_VAL", true},

		// Numeric Comparisons (String Variable "200")
		{"GT_Str_True", ConditionGreaterThan, 150, "RES_GT_S", "yes", "NUM_STR", true},

		// Type Mismatch / Invalid Number
		{"GT_Invalid_True", ConditionGreaterThan, 50, "RES_INV", "yes", "BAD_NUM", false},     // "abc" > 50 -> false
		{"GT_Invalid_Exp", ConditionGreaterThan, "xyz", "RES_INV_2", "yes", "NUM_VAL", false}, // 100 > "xyz" -> false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using IfDynamicVariable for easier testing of logic
			steps := []ResponseFuncConfig{
				IfDynamicVariable(tt.varName, tt.condition, fmt.Sprintf("%v", tt.expected), tt.target, tt.toBe),
			}
			h.Execute(steps)

			val, set := h.Variables[tt.target]
			if tt.shouldSet {
				if !set || val != tt.toBe {
					t.Errorf("Expected variable %s to be set to %v, got %v (set=%v)", tt.target, tt.toBe, val, set)
				}
			} else {
				if set {
					t.Errorf("Expected variable %s NOT to be set, but it was set to %v", tt.target, val)
				}
			}
		})
	}
}

func TestHandlerExecutor_PrepareData(t *testing.T) {
	t.Run("IfRequestHeader", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Auth", "Bearer 123")
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestHeader("Auth", "Equal", "Bearer 123", "AUTH_OK", "true"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if val, ok := h.Variables["AUTH_OK"]; !ok || val != "true" {
			t.Errorf("Expected AUTH_OK=true, got %v", val)
		}
	})

	t.Run("IfRequestJsonBody", func(t *testing.T) {
		body := `{"user": {"id": 1, "roles": ["admin", "editor"]}}`
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestJsonBody("user.id", "Equal", 1, "ID_MATCH", "yes"),
			IfRequestJsonBody("user.roles[1]", "Equal", "editor", "ROLE_MATCH", "yes"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if h.Variables["ID_MATCH"] != "yes" {
			t.Error("ID_MATCH not set")
		}
		if h.Variables["ROLE_MATCH"] != "yes" {
			t.Error("ROLE_MATCH not set")
		}
	})

	t.Run("IfRequestPath", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestPath("Equal", "/api/v1/test", "PATH_OK", "true"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if h.Variables["PATH_OK"] != "true" {
			t.Error("PATH_OK not set")
		}
	})

	t.Run("IfRequestQuery", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test?q=hello", nil)
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestQuery("q", "Equal", "hello", "QUERY_OK", "true"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if h.Variables["QUERY_OK"] != "true" {
			t.Error("QUERY_OK not set")
		}
	})
}

func TestHandlerExecutor_Generator(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h := NewHandlerExecutor(w, req)

	steps := []ResponseFuncConfig{
		GenerateRandomString(10, "R_STR"),
		GenerateRandomInt(10, 20, "R_INT"),
		GenerateRandomIntFixLength(4, "R_FIX"),
		GenerateRandomDecimal(1.0, 5.0, 2, "R_DEC"),
	}
	h.Execute(steps)

	if len(h.Variables["R_STR"].(string)) != 10 {
		t.Error("R_STR length mismatch")
	}
	rInt := h.Variables["R_INT"].(int)
	if rInt < 10 || rInt > 20 {
		t.Error("R_INT out of range")
	}
	rFix := h.Variables["R_FIX"].(int)
	if rFix < 1000 || rFix > 9999 {
		t.Error("R_FIX length mismatch")
	}
	rDec := h.Variables["R_DEC"].(float64)
	if rDec < 1.0 || rDec > 5.0 {
		t.Error("R_DEC out of range")
	}

	// Test HashedString
	h.Variables["SRC"] = "test"
	stepsHash := []ResponseFuncConfig{
		HashedString("SRC", "MD5", "HASH_MD5"),
		HashedString("SRC", "SHA256", "HASH_SHA"),
	}
	h.Execute(stepsHash)

	if len(h.Variables["HASH_MD5"].(string)) != 32 { // MD5 hex length
		t.Error("MD5 hash length mismatch")
	}
	if len(h.Variables["HASH_SHA"].(string)) != 64 { // SHA256 hex length
		t.Error("SHA256 hash length mismatch")
	}
}

func TestHandlerExecutor_DynamicVariable(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h := NewHandlerExecutor(w, req)

	h.Variables["VAL_INT"] = 123
	h.Variables["VAL_STR"] = "456"
	h.Variables["VAL_DEL"] = "trash"

	steps := []ResponseFuncConfig{
		ConvertToString("VAL_INT"),
		ConvertToInt("VAL_STR"),
		Delete("VAL_DEL"),
	}
	h.Execute(steps)

	if _, ok := h.Variables["VAL_INT"].(string); !ok {
		t.Error("VAL_INT not converted to string")
	}
	if _, ok := h.Variables["VAL_STR"].(int); !ok {
		t.Error("VAL_STR not converted to int")
	}
	if _, ok := h.Variables["VAL_DEL"]; ok {
		t.Error("VAL_DEL not deleted")
	}
}

func TestHandlerExecutor_SetupResponse(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Req", "ReqVal")
	w := httptest.NewRecorder()
	h := NewHandlerExecutor(w, req)

	h.Variables["V"] = "World"

	steps := []ResponseFuncConfig{
		SetStatusCode("", 201),
		SetHeader("", "X-Resp", "RespVal"),
		CopyHeaderFromRequest("", "X-Req"),
		SetJsonBody("", `{"hello": "{{.V}}"}`),
	}
	h.Execute(steps)
	h.Finalize()

	resp := w.Result()
	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Resp") != "RespVal" {
		t.Error("X-Resp header missing")
	}
	if resp.Header.Get("X-Req") != "ReqVal" {
		t.Error("X-Req header not copied")
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if !strings.Contains(buf.String(), `"hello": "World"`) {
		t.Errorf("Body template not resolved: %s", buf.String())
	}
}

func TestHandlerExecutor_SetCase(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Type", "B")
	w := httptest.NewRecorder()
	h := NewHandlerExecutor(w, req)

	steps := []ResponseFuncConfig{
		// Default response (Case "")
		SetStatusCode("", 200),
		SetJsonBody("", "Default"),

		// Switch Case
		IfRequestHeaderSetCase("Type", "Equal", "B", "CaseB"),

		// Case B response
		SetStatusCode("CaseB", 201),
		SetJsonBody("CaseB", "ResponseB"),
	}
	h.Execute(steps)
	h.Finalize()

	if h.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", h.StatusCode)
	}
	if h.Body != "ResponseB" {
		t.Errorf("Expected Body ResponseB, got %s", h.Body)
	}
}

func TestResolveString(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h := NewHandlerExecutor(w, req)
	h.Variables["A"] = "AAA"

	// Direct testing of private method if needed, or via side effects
	res := h.resolveString("Val-{{.A}}")
	if res != "Val-AAA" {
		t.Errorf("Expected Val-AAA, got %s", res)
	}
}

func TestHandlerExecutor_ExtractRequestData(t *testing.T) {
	body := `{"user": {"id": 99, "name": "Alice"}, "items": [{"price": 10.5}, {"price": 20.0}]}`
	req, _ := http.NewRequest("GET", "/api/data?q=search", bytes.NewBufferString(body))
	req.Header.Set("X-Token", "secret-token")
	w := httptest.NewRecorder()
	h := NewHandlerExecutor(w, req)

	steps := []ResponseFuncConfig{
		ExtractRequestHeader("X-Token", "TOKEN"),
		ExtractRequestJsonBody("user.name", "USER_NAME"),
		ExtractRequestJsonBody("items[0].price", "ITEM_PRICE"),
		ExtractRequestPath("REQ_PATH"),
		ExtractRequestQuery("q", "QUERY_Q"),
	}

	if err := h.Execute(steps); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if h.Variables["TOKEN"] != "secret-token" {
		t.Errorf("TOKEN mismatch, got %v", h.Variables["TOKEN"])
	}
	if h.Variables["USER_NAME"] != "Alice" {
		t.Errorf("USER_NAME mismatch, got %v", h.Variables["USER_NAME"])
	}
	if h.Variables["ITEM_PRICE"] != 10.5 {
		t.Errorf("ITEM_PRICE mismatch, got %v", h.Variables["ITEM_PRICE"])
	}
	if h.Variables["REQ_PATH"] != "/api/data" {
		t.Errorf("REQ_PATH mismatch, got %v", h.Variables["REQ_PATH"])
	}
	if h.Variables["QUERY_Q"] != "search" {
		t.Errorf("QUERY_Q mismatch, got %v", h.Variables["QUERY_Q"])
	}
}

func TestHandlerExecutor_NewFeatures(t *testing.T) {
	t.Run("IfDynamicVariable", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		h := NewHandlerExecutor(httptest.NewRecorder(), req)
		h.Variables["EXISTING"] = "some_value"

		steps := []ResponseFuncConfig{
			IfDynamicVariable("EXISTING", "Equal", "some_value", "CHECK_OK", "yes"),
			IfDynamicVariable("EXISTING", "Equal", "other", "CHECK_FAIL", "yes"),
		}
		h.Execute(steps)

		if h.Variables["CHECK_OK"] != "yes" {
			t.Error("CHECK_OK not set")
		}
		if _, ok := h.Variables["CHECK_FAIL"]; ok {
			t.Error("CHECK_FAIL should not be set")
		}
	})

	t.Run("JSON_Checks", func(t *testing.T) {
		body := `{"list": [1, 2, 3], "obj": {"a": 1, "b": 2}, "str": "hello", "num": 123}`
		req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
		h := NewHandlerExecutor(httptest.NewRecorder(), req)

		steps := []ResponseFuncConfig{
			IfRequestJsonArrayLength("list", "Equal", 3, "LEN_ARR_OK", "yes"),
			IfRequestJsonObjectLength("obj", "Equal", 2, "LEN_OBJ_OK", "yes"),
			IfRequestJsonType("str", "string", "TYPE_STR_OK", "yes"),
			IfRequestJsonType("num", "number", "TYPE_NUM_OK", "yes"),
			IfRequestJsonType("list", "array", "TYPE_ARR_OK", "yes"),
		}
		h.Execute(steps)

		if h.Variables["LEN_ARR_OK"] != "yes" {
			t.Error("LEN_ARR_OK not set")
		}
		if h.Variables["LEN_OBJ_OK"] != "yes" {
			t.Error("LEN_OBJ_OK not set")
		}
		if h.Variables["TYPE_STR_OK"] != "yes" {
			t.Error("TYPE_STR_OK not set")
		}
		if h.Variables["TYPE_NUM_OK"] != "yes" {
			t.Error("TYPE_NUM_OK not set")
		}
		if h.Variables["TYPE_ARR_OK"] != "yes" {
			t.Error("TYPE_ARR_OK not set")
		}
	})

	t.Run("DynamicVar_Manipulate", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		h := NewHandlerExecutor(httptest.NewRecorder(), req)
		h.Variables["SRC"] = "Hello World"
		h.Variables["PART1"] = "A"
		h.Variables["PART2"] = "B"

		steps := []ResponseFuncConfig{
			DynamicVarSubstring("SRC", 0, 5, "SUB"),                        // "Hello"
			DynamicVarJoin("JOINED", "-", "{{.PART1}}", "{{.PART2}}", "C"), // "A-B-C"
		}
		h.Execute(steps)

		if h.Variables["SUB"] != "Hello" {
			t.Errorf("SUB mismatch, got '%v'", h.Variables["SUB"])
		}
		if h.Variables["JOINED"] != "A-B-C" {
			t.Errorf("JOINED mismatch, got '%v'", h.Variables["JOINED"])
		}
	})
}

func TestHandlerExecutor_XmlBody(t *testing.T) {
	xmlBody := `<request><user id="42"><name>Alice</name><role>admin</role></user><items><item>one</item><item>two</item></items></request>`

	t.Run("IfRequestXmlBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(xmlBody))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestXmlBody("request.user.name", "Equal", "Alice", "NAME_OK", "yes"),
			IfRequestXmlBody("request.user.role", "Equal", "admin", "ROLE_OK", "yes"),
			IfRequestXmlBody("request.user.role", "Equal", "guest", "ROLE_FAIL", "yes"),
			IfRequestXmlBody("request.user.@id", "Equal", "42", "ATTR_OK", "yes"),
			IfRequestXmlBody("request.items.item[1]", "Equal", "two", "ITEM_OK", "yes"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if h.Variables["NAME_OK"] != "yes" {
			t.Error("NAME_OK not set")
		}
		if h.Variables["ROLE_OK"] != "yes" {
			t.Error("ROLE_OK not set")
		}
		if _, ok := h.Variables["ROLE_FAIL"]; ok {
			t.Error("ROLE_FAIL should not be set")
		}
		if h.Variables["ATTR_OK"] != "yes" {
			t.Error("ATTR_OK not set")
		}
		if h.Variables["ITEM_OK"] != "yes" {
			t.Error("ITEM_OK not set")
		}
	})

	t.Run("IfRequestXmlBodySetCase", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(xmlBody))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestXmlBodySetCase("request.user.name", "Equal", "Alice", "CaseAlice"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if h.ActiveCase != "CaseAlice" {
			t.Errorf("Expected ActiveCase=CaseAlice, got %s", h.ActiveCase)
		}
	})

	t.Run("ExtractRequestXmlBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(xmlBody))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			ExtractRequestXmlBody("request.user.name", "USER_NAME"),
			ExtractRequestXmlBody("request.user.@id", "USER_ID"),
			ExtractRequestXmlBody("request.items.item[0]", "FIRST_ITEM"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if h.Variables["USER_NAME"] != "Alice" {
			t.Errorf("USER_NAME mismatch, got %v", h.Variables["USER_NAME"])
		}
		if h.Variables["USER_ID"] != "42" {
			t.Errorf("USER_ID mismatch, got %v", h.Variables["USER_ID"])
		}
		if h.Variables["FIRST_ITEM"] != "one" {
			t.Errorf("FIRST_ITEM mismatch, got %v", h.Variables["FIRST_ITEM"])
		}
	})

	t.Run("SetXmlBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(xmlBody))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		respXml := `<response><status>ok</status><name>{{.USER_NAME}}</name></response>`
		steps := []ResponseFuncConfig{
			ExtractRequestXmlBody("request.user.name", "USER_NAME"),
			SetXmlBody("", respXml),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		h.Finalize()

		result := w.Body.String()
		expected := `<response><status>ok</status><name>Alice</name></response>`
		if result != expected {
			t.Errorf("Body mismatch.\nExpected: %s\nGot:      %s", expected, result)
		}
	})

	t.Run("XmlBody_EndToEnd_WithCase", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(xmlBody))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestXmlBodySetCase("request.user.role", "Equal", "admin", "AdminCase"),
			SetXmlBody("AdminCase", `<response><granted>true</granted></response>`),
			SetXmlBody("", `<response><granted>false</granted></response>`),
			SetStatusCode("AdminCase", 200),
			SetStatusCode("", 403),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		h.Finalize()

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		expected := `<response><granted>true</granted></response>`
		if w.Body.String() != expected {
			t.Errorf("Body mismatch.\nExpected: %s\nGot:      %s", expected, w.Body.String())
		}
	})

	t.Run("XmlBody_NonExistentPath", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(xmlBody))
		w := httptest.NewRecorder()
		h := NewHandlerExecutor(w, req)

		steps := []ResponseFuncConfig{
			IfRequestXmlBody("request.nonexistent.field", "Equal", "anything", "SHOULD_NOT_SET", "yes"),
			ExtractRequestXmlBody("request.nonexistent.field", "SHOULD_NOT_EXIST"),
		}
		if err := h.Execute(steps); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if _, ok := h.Variables["SHOULD_NOT_SET"]; ok {
			t.Error("SHOULD_NOT_SET should not be set for non-existent path")
		}
		if _, ok := h.Variables["SHOULD_NOT_EXIST"]; ok {
			t.Error("SHOULD_NOT_EXIST should not be set for non-existent path")
		}
	})
}
