package dynamic_mock_server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClientHelpers(t *testing.T) {
	tests := []struct {
		name     string
		got      ResponseFuncConfig
		expected ResponseFuncConfig
	}{
		{
			name: "IfRequestHeader",
			got:  IfRequestHeader("Auth", "Equal", "Bearer", "VAR", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestHeader,
				Args:  []interface{}{"Auth", "Equal", "Bearer", "VAR", "true"},
			},
		},
		{
			name: "IfRequestJsonBody",
			got:  IfRequestJsonBody("field", "Equal", "val", "VAR", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestJsonBody,
				Args:  []interface{}{"field", "Equal", "val", "VAR", "true"},
			},
		},
		{
			name: "IfRequestPath",
			got:  IfRequestPath("Equal", "/abc", "VAR", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestPath,
				Args:  []interface{}{"Equal", "/abc", "VAR", "true"},
			},
		},
		{
			name: "IfRequestQuery",
			got:  IfRequestQuery("q", "Equal", "1", "VAR", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestQuery,
				Args:  []interface{}{"q", "Equal", "1", "VAR", "true"},
			},
		},
		{
			name: "GenerateRandomString",
			got:  GenerateRandomString(10, "R_STR"),
			expected: ResponseFuncConfig{
				Group: GroupGenerator,
				Func:  FuncGenerateRandomString,
				Args:  []interface{}{10, "R_STR"},
			},
		},
		{
			name: "GenerateRandomInt",
			got:  GenerateRandomInt(1, 10, "R_INT"),
			expected: ResponseFuncConfig{
				Group: GroupGenerator,
				Func:  FuncGenerateRandomInt,
				Args:  []interface{}{1, 10, "R_INT"},
			},
		},
		{
			name: "GenerateRandomIntFixLength",
			got:  GenerateRandomIntFixLength(5, "R_INT_FIX"),
			expected: ResponseFuncConfig{
				Group: GroupGenerator,
				Func:  FuncGenerateRandomIntFixLength,
				Args:  []interface{}{5, "R_INT_FIX"},
			},
		},
		{
			name: "GenerateRandomDecimal",
			got:  GenerateRandomDecimal(1.0, 2.0, 2, "R_DEC"),
			expected: ResponseFuncConfig{
				Group: GroupGenerator,
				Func:  FuncGenerateRandomDecimal,
				Args:  []interface{}{1.0, 2.0, 2, "R_DEC"},
			},
		},
		{
			name: "HashedString",
			got:  HashedString("VAR", "MD5", "HASH"),
			expected: ResponseFuncConfig{
				Group: GroupGenerator,
				Func:  FuncHashedString,
				Args:  []interface{}{"VAR", "MD5", "HASH"},
			},
		},
		{
			name: "ConvertToString",
			got:  ConvertToString("VAR"),
			expected: ResponseFuncConfig{
				Group: GroupDynamicVariable,
				Func:  FuncConvertToString,
				Args:  []interface{}{"VAR"},
			},
		},
		{
			name: "ConvertToInt",
			got:  ConvertToInt("VAR"),
			expected: ResponseFuncConfig{
				Group: GroupDynamicVariable,
				Func:  FuncConvertToInt,
				Args:  []interface{}{"VAR"},
			},
		},
		{
			name: "Delete",
			got:  Delete("VAR"),
			expected: ResponseFuncConfig{
				Group: GroupDynamicVariable,
				Func:  FuncDelete,
				Args:  []interface{}{"VAR"},
			},
		},
		{
			name: "IfRequestHeaderSetCase",
			got:  IfRequestHeaderSetCase("Auth", "Equal", "val", "CaseA"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestHeaderSetCase,
				Args:  []interface{}{"Auth", "Equal", "val", "CaseA"},
			},
		},
		{
			name: "IfDynamicVariable",
			got:  IfDynamicVariable("V1", "Equal", "val", "V2", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfDynamicVariable,
				Args:  []interface{}{"V1", "Equal", "val", "V2", "true"},
			},
		},
		{
			name: "IfRequestJsonArrayLength",
			got:  IfRequestJsonArrayLength("arr", "Equal", 5, "VAR", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestJsonArrayLength,
				Args:  []interface{}{"arr", "Equal", 5, "VAR", "true"},
			},
		},
		{
			name: "IfRequestJsonType",
			got:  IfRequestJsonType("field", "string", "VAR", "true"),
			expected: ResponseFuncConfig{
				Group: GroupPrepareData,
				Func:  FuncIfRequestJsonType,
				Args:  []interface{}{"field", "string", "VAR", "true"},
			},
		},
		{
			name: "DynamicVarSubstring",
			got:  DynamicVarSubstring("SRC", 0, 2, "DST"),
			expected: ResponseFuncConfig{
				Group: GroupDynamicVariable,
				Func:  FuncDynamicVarSubstring,
				Args:  []interface{}{"SRC", 0, 2, "DST"},
			},
		},
		{
			name: "DynamicVarJoin",
			got:  DynamicVarJoin("DST", "-", "A", "B"),
			expected: ResponseFuncConfig{
				Group: GroupDynamicVariable,
				Func:  FuncDynamicVarJoin,
				Args:  []interface{}{"DST", "-", "A", "B"},
			},
		},
		{
			name: "SetJsonBody",
			got:  SetJsonBody("C1", `{"a":1}`),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncSetJsonBody,
				Args:  []interface{}{"C1", `{"a":1}`},
			},
		},
		{
			name: "SetStatusCode",
			got:  SetStatusCode("", 201),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncSetStatusCode,
				Args:  []interface{}{"", 201},
			},
		},
		{
			name: "SetWait",
			got:  SetWait("", 100),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncSetWait,
				Args:  []interface{}{"", 100},
			},
		},
		{
			name: "SetRandomWait",
			got:  SetRandomWait("", 10, 20),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncSetRandomWait,
				Args:  []interface{}{"", 10, 20},
			},
		},
		{
			name: "SetMethod",
			got:  SetMethod("", "POST"),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncSetMethod,
				Args:  []interface{}{"", "POST"},
			},
		},
		{
			name: "SetHeader",
			got:  SetHeader("", "Content-Type", "application/json"),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncSetHeader,
				Args:  []interface{}{"", "Content-Type", "application/json"},
			},
		},
		{
			name: "CopyHeaderFromRequest",
			got:  CopyHeaderFromRequest("", "X-Trace-ID"),
			expected: ResponseFuncConfig{
				Group: GroupSetupResponse,
				Func:  FuncCopyHeaderFromRequest,
				Args:  []interface{}{"", "X-Trace-ID"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.expected) {
				t.Errorf("got %v, want %v", tt.got, tt.expected)
			}
		})
	}
}

func TestClient_Methods(t *testing.T) {
	// Mock Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registerRoute":
			var req RegisterRouteRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Port == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/resetPort":
			var req map[string]int
			json.NewDecoder(r.Body).Decode(&req)
			if req["port"] == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/resetAll":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL)

	t.Run("RegisterRoute", func(t *testing.T) {
		err := client.RegisterRoute(8080, "GET", "/test", []ResponseFuncConfig{})
		if err != nil {
			t.Errorf("RegisterRoute failed: %v", err)
		}

		// Test error case
		err = client.RegisterRoute(0, "GET", "/test", nil)
		if err == nil {
			t.Errorf("Expected error for bad request")
		}
	})

	t.Run("ResetPort", func(t *testing.T) {
		err := client.ResetPort(8080)
		if err != nil {
			t.Errorf("ResetPort failed: %v", err)
		}

		err = client.ResetPort(0)
		if err == nil {
			t.Errorf("Expected error for bad port")
		}
	})

	t.Run("ResetAll", func(t *testing.T) {
		err := client.ResetAll()
		if err != nil {
			t.Errorf("ResetAll failed: %v", err)
		}
	})
}

func TestClient_HTTPSInsecureSkipVerify(t *testing.T) {
	// HTTPS server with self-signed certificate
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/resetAll" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	client := NewClient(mockServer.URL)

	if err := client.ResetAll(); err != nil {
		t.Fatalf("ResetAll over HTTPS failed: %v", err)
	}
}
