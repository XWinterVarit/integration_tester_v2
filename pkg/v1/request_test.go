package v1

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetValueByPath(t *testing.T) {
	// Helper to build a JSON-decoded structure from a JSON string
	makeData := func(raw string) interface{} {
		var d interface{}
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			t.Fatalf("makeData: %v", err)
		}
		return d
	}

	t.Run("set top-level string field", func(t *testing.T) {
		data := makeData(`{"name":"alice"}`)
		if err := setValueByPath(data, "name", "bob"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["name"]
		if got != "bob" {
			t.Fatalf("expected bob, got %v", got)
		}
	})

	t.Run("set top-level numeric field", func(t *testing.T) {
		data := makeData(`{"score":10}`)
		if err := setValueByPath(data, "score", float64(99)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["score"]
		if got != float64(99) {
			t.Fatalf("expected 99, got %v", got)
		}
	})

	t.Run("set top-level boolean field", func(t *testing.T) {
		data := makeData(`{"active":false}`)
		if err := setValueByPath(data, "active", true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["active"]
		if got != true {
			t.Fatalf("expected true, got %v", got)
		}
	})

	t.Run("set top-level null field to value", func(t *testing.T) {
		data := makeData(`{"val":null}`)
		if err := setValueByPath(data, "val", "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["val"]
		if got != "hello" {
			t.Fatalf("expected hello, got %v", got)
		}
	})

	t.Run("set nested two-level field", func(t *testing.T) {
		data := makeData(`{"a":{"b":"old"}}`)
		if err := setValueByPath(data, "a.b", "new"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["a"].(map[string]interface{})["b"]
		if got != "new" {
			t.Fatalf("expected new, got %v", got)
		}
	})

	t.Run("set nested three-level field", func(t *testing.T) {
		data := makeData(`{"x":{"y":{"z":"before"}}}`)
		if err := setValueByPath(data, "x.y.z", "after"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["x"].(map[string]interface{})["y"].(map[string]interface{})["z"]
		if got != "after" {
			t.Fatalf("expected after, got %v", got)
		}
	})

	t.Run("set array element by index", func(t *testing.T) {
		data := makeData(`{"items":["a","b","c"]}`)
		if err := setValueByPath(data, "items[1]", "B"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr := data.(map[string]interface{})["items"].([]interface{})
		if arr[1] != "B" {
			t.Fatalf("expected B, got %v", arr[1])
		}
	})

	t.Run("set field inside array element (dot+bracket)", func(t *testing.T) {
		data := makeData(`{"a":{"b":[{"c":"old"}]}}`)
		if err := setValueByPath(data, "a.b[0].c", "something"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["a"].(map[string]interface{})["b"].([]interface{})[0].(map[string]interface{})["c"]
		if got != "something" {
			t.Fatalf("expected something, got %v", got)
		}
	})

	t.Run("set field inside second array element", func(t *testing.T) {
		data := makeData(`{"users":[{"name":"alice"},{"name":"bob"}]}`)
		if err := setValueByPath(data, "users[1].name", "charlie"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["users"].([]interface{})[1].(map[string]interface{})["name"]
		if got != "charlie" {
			t.Fatalf("expected charlie, got %v", got)
		}
	})

	t.Run("set deeply nested field inside array", func(t *testing.T) {
		data := makeData(`{"level1":{"level2":[{"level3":{"level4":"deep"}}]}}`)
		if err := setValueByPath(data, "level1.level2[0].level3.level4", "changed"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["level1"].(map[string]interface{})["level2"].([]interface{})[0].(map[string]interface{})["level3"].(map[string]interface{})["level4"]
		if got != "changed" {
			t.Fatalf("expected changed, got %v", got)
		}
	})

	t.Run("set field to map value", func(t *testing.T) {
		data := makeData(`{"meta":{}}`)
		newMap := map[string]interface{}{"key": "value"}
		if err := setValueByPath(data, "meta", newMap); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["meta"]
		m, ok := got.(map[string]interface{})
		if !ok || m["key"] != "value" {
			t.Fatalf("expected map with key=value, got %v", got)
		}
	})

	t.Run("set field to slice value", func(t *testing.T) {
		data := makeData(`{"tags":[]}`)
		if err := setValueByPath(data, "tags", []interface{}{"go", "test"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["tags"].([]interface{})
		if len(got) != 2 || got[0] != "go" || got[1] != "test" {
			t.Fatalf("expected [go test], got %v", got)
		}
	})

	t.Run("set new top-level key (upsert)", func(t *testing.T) {
		data := makeData(`{"a":1}`)
		if err := setValueByPath(data, "b", "x"); err != nil {
			t.Fatalf("unexpected error setting new key: %v", err)
		}
		got := data.(map[string]interface{})["b"]
		if got != "x" {
			t.Fatalf("expected x, got %v", got)
		}
	})

	t.Run("error: missing nested key", func(t *testing.T) {
		data := makeData(`{"a":{"b":1}}`)
		err := setValueByPath(data, "a.c.d", "x")
		if err == nil {
			t.Fatal("expected error for missing key 'c'")
		}
	})

	t.Run("error: index out of bounds", func(t *testing.T) {
		data := makeData(`{"arr":[1,2,3]}`)
		err := setValueByPath(data, "arr[5]", "x")
		if err == nil {
			t.Fatal("expected error for out-of-bounds index")
		}
	})

	t.Run("error: index on non-array", func(t *testing.T) {
		data := makeData(`{"arr":"notanarray"}`)
		err := setValueByPath(data, "arr[0]", "x")
		if err == nil {
			t.Fatal("expected error when indexing non-array")
		}
	})

	t.Run("error: map key on non-map", func(t *testing.T) {
		data := makeData(`{"a":42}`)
		err := setValueByPath(data, "a.b", "x")
		if err == nil {
			t.Fatal("expected error when traversing into non-map")
		}
	})

	t.Run("error: invalid array index (non-numeric)", func(t *testing.T) {
		data := makeData(`{"arr":[1,2,3]}`)
		err := setValueByPath(data, "arr[abc]", "x")
		if err == nil {
			t.Fatal("expected error for non-numeric array index")
		}
	})

	t.Run("set last array element", func(t *testing.T) {
		data := makeData(`{"arr":[10,20,30]}`)
		if err := setValueByPath(data, "arr[2]", 99); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr := data.(map[string]interface{})["arr"].([]interface{})
		if arr[2] != 99 {
			t.Fatalf("expected 99, got %v", arr[2])
		}
	})

	t.Run("set field to nil (null)", func(t *testing.T) {
		data := makeData(`{"a":"something"}`)
		if err := setValueByPath(data, "a", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := data.(map[string]interface{})["a"]
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("multiple sequential sets on same data", func(t *testing.T) {
		data := makeData(`{"a":"x","b":"y","c":"z"}`)
		if err := setValueByPath(data, "a", "A"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := setValueByPath(data, "b", "B"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := setValueByPath(data, "c", "C"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := data.(map[string]interface{})
		if m["a"] != "A" || m["b"] != "B" || m["c"] != "C" {
			t.Fatalf("unexpected values: %v", m)
		}
	})
}

func TestSendRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "Value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"key": "value"}`)
	}))
	defer server.Close()

	resp := SendRequest(server.URL)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"key": "value"}` {
		t.Errorf("Expected body '{\"key\": \"value\"}', got '%s'", resp.Body)
	}
	if resp.Header["X-Test"] != "Value" {
		t.Errorf("Expected header X-Test=Value, got %s", resp.Header["X-Test"])
	}
}

func TestExpectFunctions(t *testing.T) {
	resp := Response{
		StatusCode: 200,
		Body:       `{"a": 1, "b": {"c": 2}, "d": [3, 4]}`,
		Header:     map[string]string{"Content-Type": "application/json"},
	}

	// Success cases (should not panic)
	ExpectStatusCode(resp, 200)
	ExpectHeader(resp, "Content-Type", "application/json")
	ExpectJsonBody(resp, `{"a": 1, "b": {"c": 2}, "d": [3, 4]}`)
	ExpectJsonBodyField(resp, "a", 1)
	ExpectJsonBodyField(resp, "b.c", 2)
	ExpectJsonBodyField(resp, "d[0]", 3)

	// Failure cases (should panic with TestError)
	assertPanic := func(name string, f func()) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s expected to panic", name)
			}
			if _, ok := r.(TestError); !ok {
				t.Errorf("%s panicked with unexpected type: %T", name, r)
			}
		}()
		f()
	}

	assertPanic("ExpectStatusCode", func() { ExpectStatusCode(resp, 404) })
	assertPanic("ExpectHeader", func() { ExpectHeader(resp, "Content-Type", "xml") })
	assertPanic("ExpectJsonBody", func() { ExpectJsonBody(resp, `{"a": 2}`) })
	assertPanic("ExpectJsonBodyField", func() { ExpectJsonBodyField(resp, "a", 999) })
	assertPanic("ExpectJsonBodyField path", func() { ExpectJsonBodyField(resp, "x.y", 1) })
}

func TestExpectJsonBodyFieldCond(t *testing.T) {
	resp := Response{
		Body: `{"num": 5, "text": "hello world", "nullField": null, "nested": {"arr": [1, 2]}}`,
	}

	// Success cases
	ExpectJsonBodyFieldCond(resp, "num", ConditionGreaterThan, 3)
	ExpectJsonBodyFieldCond(resp, "num", ConditionLessThanOrEqual, 5)
	ExpectJsonBodyFieldCond(resp, "text", ConditionContains, "hello")
	ExpectJsonBodyFieldCond(resp, "text", ConditionStartsWith, "hello")
	ExpectJsonBodyFieldCond(resp, "text", ConditionEndsWith, "world")
	ExpectJsonBodyFieldCond(resp, "nested.arr[1]", ConditionEqual, 2)
	ExpectJsonBodyFieldCond(resp, "nullField", ConditionEqual, nil)
	ExpectJsonBodyFieldCond(resp, "nullField", ConditionNotEqual, "not-nil")

	// Failure cases (should panic)
	assertPanic := func(name string, f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("%s expected to panic", name)
			}
		}()
		f()
	}

	assertPanic("invalid path", func() { ExpectJsonBodyFieldCond(resp, "missing", ConditionEqual, 1) })
	assertPanic("condition mismatch", func() { ExpectJsonBodyFieldCond(resp, "num", ConditionLessThan, 1) })
}

func TestSendRESTRequestWithMethodHeadersAndJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.Header.Get("X-Test") != "Value" {
			t.Errorf("expected header X-Test=Value, got %s", r.Header.Get("X-Test"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"a":1}` {
			t.Errorf("expected body {\"a\":1}, got %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	resp := SendRESTRequest(server.URL,
		WithMethod(http.MethodPost),
		WithHeader("X-Test", "Value"),
		WithJSONBody(map[string]int{"a": 1}),
	)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	if resp.Body != `{"ok":true}` {
		t.Fatalf("expected body '{\"ok\":true}', got %s", resp.Body)
	}
	if resp.Header["Content-Type"] != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", resp.Header["Content-Type"])
	}
}

func TestWithXMLBody(t *testing.T) {
	type Req struct {
		XMLName xml.Name `xml:"request"`
		ID      string   `xml:"id"`
		Status  string   `xml:"status"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/xml" {
			t.Errorf("expected Content-Type application/xml, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		var got Req
		if err := xml.Unmarshal(body, &got); err != nil {
			t.Fatalf("failed to unmarshal request XML: %v", err)
		}
		if got.ID != "1" || got.Status != "active" {
			t.Errorf("unexpected body: %+v", got)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<response><id>1</id><status>active</status></response>`)
	}))
	defer server.Close()

	resp := SendRESTRequest(server.URL,
		WithMethod(http.MethodPost),
		WithXMLBody(Req{ID: "1", Status: "active"}),
	)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestExpectXmlFunctions(t *testing.T) {
	resp := Response{
		StatusCode: 200,
		Body:       `<response><user id="42"><name>Alice</name><role>admin</role></user><items><item>one</item><item>two</item></items></response>`,
	}

	// Success cases
	ExpectXmlBody(resp, `<response><user id="42"><name>Alice</name><role>admin</role></user><items><item>one</item><item>two</item></items></response>`)
	ExpectXmlBodyField(resp, "response.user.name", "Alice")
	ExpectXmlBodyField(resp, "response.user.role", "admin")
	ExpectXmlBodyField(resp, "response.user.@id", "42")
	ExpectXmlBodyField(resp, "response.items.item[0]", "one")
	ExpectXmlBodyField(resp, "response.items.item[1]", "two")

	// Condition checks
	ExpectXmlBodyFieldCond(resp, "response.user.name", ConditionEqual, "Alice")
	ExpectXmlBodyFieldCond(resp, "response.user.name", ConditionContains, "lic")
	ExpectXmlBodyFieldCond(resp, "response.user.name", ConditionStartsWith, "Ali")
	ExpectXmlBodyFieldCond(resp, "response.user.name", ConditionEndsWith, "ice")
	ExpectXmlBodyFieldCond(resp, "response.user.name", ConditionNotEqual, "Bob")

	// Failure cases
	assertPanic := func(name string, f func()) {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s expected to panic", name)
			}
			if _, ok := r.(TestError); !ok {
				t.Errorf("%s panicked with unexpected type: %T", name, r)
			}
		}()
		f()
	}

	assertPanic("ExpectXmlBody mismatch", func() { ExpectXmlBody(resp, `<other/>`) })
	assertPanic("ExpectXmlBodyField missing", func() { ExpectXmlBodyField(resp, "response.missing", "x") })
	assertPanic("ExpectXmlBodyField wrong value", func() { ExpectXmlBodyField(resp, "response.user.name", "Bob") })
	assertPanic("ExpectXmlBodyFieldCond fail", func() {
		ExpectXmlBodyFieldCond(resp, "response.user.name", ConditionEqual, "Bob")
	})
	assertPanic("ExpectXmlBody invalid xml", func() { ExpectXmlBody(Response{Body: "not xml {"}, `<a/>`) })
}

func TestPrettyXml(t *testing.T) {
	input := `<root><child>text</child></root>`
	result := PrettyXml(input)
	if !strings.Contains(result, "\n") {
		t.Errorf("Expected pretty XML with newlines, got: %s", result)
	}
	if !strings.Contains(result, "  <child>") {
		t.Errorf("Expected indented child element, got: %s", result)
	}

	// Invalid XML returns original
	invalid := "not xml at all {"
	if PrettyXml(invalid) != invalid {
		t.Error("Expected invalid XML to be returned unchanged")
	}
}

func TestSendRESTRequestIgnoreSSL(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "secure")
	}))
	defer server.Close()

	resp := SendRESTRequest(server.URL, WithIgnoreServerSSL(true))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != "secure" {
		t.Fatalf("expected body 'secure', got %s", resp.Body)
	}
}
