package v1

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// SendRESTRequest sends an HTTP request with flexible options.
// Common usage:
//
//	SendRESTRequest(url,
//	    WithMethod("POST"),
//	    WithHeader("Authorization", "Bearer ..."),
//	    WithJSONBody(map[string]interface{}{...}),
//	    WithIgnoreServerSSL(true),
//	)
func SendRESTRequest(url string, opts ...RESTRequestOption) Response {
	cfg := restRequestConfig{
		method:  http.MethodGet,
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	RecordAction(fmt.Sprintf("Request: %s %s", cfg.method, url), func() {
		SendRESTRequest(url, opts...)
	})
	if IsDryRun() {
		return Response{}
	}

	var bodyReader io.Reader
	if len(cfg.body) > 0 {
		bodyReader = bytes.NewReader(cfg.body)
	}

	req, err := http.NewRequest(cfg.method, url, bodyReader)
	if err != nil {
		Fail("Request build failed: %v", err)
	}

	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	ignoreSSL := false
	if cfg.ignoreServerSSL != nil {
		ignoreSSL = *cfg.ignoreServerSSL
	} else if strings.HasPrefix(strings.ToLower(url), "https://") {
		ignoreSSL = true
	}

	if ignoreSSL {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	requestBody := string(cfg.body)
	requestPrettyBody := requestBody
	if len(cfg.body) > 0 {
		var jsonObj interface{}
		if json.Unmarshal(cfg.body, &jsonObj) == nil {
			if pretty, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
				requestPrettyBody = string(pretty)
			}
		} else if p := PrettyXml(string(cfg.body)); p != string(cfg.body) {
			requestPrettyBody = p
		}
	}

	reqHeaderLines := make([]string, 0, len(cfg.headers))
	for k, v := range cfg.headers {
		if v == "" {
			reqHeaderLines = append(reqHeaderLines, fmt.Sprintf("%s:", k))
			continue
		}
		reqHeaderLines = append(reqHeaderLines, fmt.Sprintf("%s: %s", k, v))
	}

	Log(LogTypeRequest, fmt.Sprintf("Sending %s request to: %s", cfg.method, url), fmt.Sprintf("Body:\n%s\nHeaders:\n%s", requestPrettyBody, strings.Join(reqHeaderLines, "\n")))
	resp, err := client.Do(req)
	if err != nil {
		Fail("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	prettyBody := string(respBody)
	if len(respBody) > 0 {
		var jsonObj interface{}
		if json.Unmarshal(respBody, &jsonObj) == nil {
			if pretty, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
				prettyBody = string(pretty)
			}
		} else if p := PrettyXml(string(respBody)); p != string(respBody) {
			prettyBody = p
		}
	}

	header := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			header[k] = v[0]
		}
	}

	headerLines := make([]string, 0, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) == 0 {
			headerLines = append(headerLines, fmt.Sprintf("%s:", k))
			continue
		}
		for _, vv := range v {
			headerLines = append(headerLines, fmt.Sprintf("%s: %s", k, vv))
		}
	}

	Log(LogTypeRequest, fmt.Sprintf("Received status %d from %s", resp.StatusCode, url), fmt.Sprintf("Body:\n%s\nHeaders:\n%s", prettyBody, strings.Join(headerLines, "\n")))
	return Response{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Header:     header,
	}
}

// SendRequest keeps backward compatibility; it is equivalent to GET via SendRESTRequest.
func SendRequest(url string) Response {
	return SendRESTRequest(url)
}

// RESTRequestOption configures SendRESTRequest.
type RESTRequestOption func(*restRequestConfig)

type restRequestConfig struct {
	method          string
	headers         map[string]string
	body            []byte
	ignoreServerSSL *bool
}

// WithMethod sets HTTP method (GET by default).
func WithMethod(method string) RESTRequestOption {
	return func(c *restRequestConfig) {
		if method != "" {
			c.method = method
		}
	}
}

// WithHeader adds a single header.
func WithHeader(key, value string) RESTRequestOption {
	return func(c *restRequestConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

// WithHeaders merges multiple headers.
func WithHeaders(headers map[string]string) RESTRequestOption {
	return func(c *restRequestConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// WithJSONBody marshals the given value as JSON and sets it as body.
// It also sets Content-Type to application/json if not already provided.
func WithJSONBody(v interface{}) RESTRequestOption {
	return func(c *restRequestConfig) {
		if v == nil {
			return
		}
		data, err := json.Marshal(v)
		if err != nil {
			Fail("Failed to marshal JSON body: %v", err)
		}
		c.body = data
		if _, ok := c.headers["Content-Type"]; !ok {
			c.headers["Content-Type"] = "application/json"
		}
	}
}

// WithXMLBody marshals the given value as XML and sets it as body.
// It also sets Content-Type to application/xml if not already provided.
func WithXMLBody(v any) RESTRequestOption {
	return func(c *restRequestConfig) {
		if v == nil {
			return
		}
		data, err := xml.Marshal(v)
		if err != nil {
			Fail("Failed to marshal XML body: %v", err)
		}
		c.body = data
		if _, ok := c.headers["Content-Type"]; !ok {
			c.headers["Content-Type"] = "application/xml"
		}
	}
}

// WithBody sets raw bytes as body (caller can set headers accordingly).
func WithBody(body []byte) RESTRequestOption {
	return func(c *restRequestConfig) {
		c.body = body
	}
}

// WithBodyString sets body from string.
func WithBodyString(body string) RESTRequestOption {
	return func(c *restRequestConfig) {
		c.body = []byte(body)
	}
}

// WithIgnoreServerSSL skips server certificate verification (useful for tests/self-signed certs).
func WithIgnoreServerSSL(ignore bool) RESTRequestOption {
	return func(c *restRequestConfig) {
		c.ignoreServerSSL = &ignore
	}
}

// ExpectStatusCode asserts that the response status code matches the expected code.
func ExpectStatusCode(resp Response, expected int) {
	if IsDryRun() {
		return
	}
	if resp.StatusCode != expected {
		// Include body in failure message for debugging
		Fail("Expected Status Code %d, got %d. Body: %s", expected, resp.StatusCode, resp.Body)
	}
	Logf(LogTypeExpect, "Status Code %d == %d - PASSED", resp.StatusCode, expected)
}

// ExpectHeader asserts that the response has the expected header.
func ExpectHeader(resp Response, key, value string) {
	if IsDryRun() {
		return
	}
	if got, ok := resp.Header[key]; !ok || got != value {
		Fail("ExpectHeader failed: expected %s=%s, got %s", key, value, got)
	}
	Logf(LogTypeExpect, "Header '%s' == '%s' - PASSED", key, value)
}

// ExpectJsonBody asserts that the response body matches the expected JSON.
// This is a simple implementation that compares unmarshaled objects.
func ExpectJsonBody(resp Response, expectedJson interface{}) {
	if IsDryRun() {
		return
	}
	var got interface{}
	if err := json.Unmarshal([]byte(resp.Body), &got); err != nil {
		Fail("ExpectJsonBody failed: response body is not valid JSON: %v. Body: %s", err, resp.Body)
	}

	// If expectedJson is string, unmarshal it too
	var expected interface{}
	if s, ok := expectedJson.(string); ok {
		if err := json.Unmarshal([]byte(s), &expected); err != nil {
			Fail("ExpectJsonBody failed: expectedJson string is not valid JSON: %v", err)
		}
	} else {
		expected = expectedJson
	}

	if !reflect.DeepEqual(got, expected) {
		Fail("ExpectJsonBody failed:\nExpected: %v\nGot:      %v", expected, got)
	}
	Log(LogTypeExpect, "JSON body matches expected value - PASSED", "")
}

// ExpectJsonBodyField asserts that a specific field in the JSON response body matches the expected value.
// field supports dot notation and array index (e.g. "data.users[0].name")
func ExpectJsonBodyField(resp Response, field string, expectedValue interface{}) {
	if IsDryRun() {
		return
	}

	var body interface{}
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		Fail("ExpectJsonBodyField failed: response body is not valid JSON: %v. Body: %s", err, resp.Body)
	}

	gotValue, err := getValueByPath(body, field)
	if err != nil {
		Fail("ExpectJsonBodyField failed to get field '%s': %v. Body: %s", field, err, resp.Body)
	}

	match := false
	if isNumber(gotValue) && isNumber(expectedValue) {
		if toFloat64(gotValue) == toFloat64(expectedValue) {
			match = true
		}
	} else {
		if reflect.DeepEqual(gotValue, expectedValue) {
			match = true
		}
	}

	if !match {
		Fail("ExpectJsonBodyField failed for field '%s':\nExpected: %v (%T)\nGot:      %v (%T)", field, expectedValue, expectedValue, gotValue, gotValue)
	}
	Logf(LogTypeExpect, "JSON Field '%s' == %v - PASSED", field, expectedValue)
}

// ExpectJsonBodyFieldCond asserts that a specific field in the JSON response body
// satisfies the provided condition against the expected value.
// Supported conditions are the same as dynamic mock server conditions (e.g., Equal, NotEqual, GreaterThan).
// It also supports checking for JSON null by passing expectedValue as nil with ConditionEqual/ConditionNotEqual.
func ExpectJsonBodyFieldCond(resp Response, field string, condition string, expectedValue interface{}) {
	if IsDryRun() {
		return
	}

	var body interface{}
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		Fail("ExpectJsonBodyFieldCond failed: response body is not valid JSON: %v. Body: %s", err, resp.Body)
	}

	gotValue, err := getValueByPath(body, field)
	if err != nil {
		Fail("ExpectJsonBodyFieldCond failed to get field '%s': %v. Body: %s", field, err, resp.Body)
	}

	if !evaluateCondition(gotValue, condition, expectedValue) {
		Fail("ExpectJsonBodyFieldCond failed for field '%s' with condition '%s':\nExpected: %v (%T)\nGot:      %v (%T)", field, condition, expectedValue, expectedValue, gotValue, gotValue)
	}

	Logf(LogTypeExpect, "JSON Field '%s' %s %v - PASSED", field, condition, expectedValue)
}

func getValueByPath(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		key := part
		index := -1

		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			open := strings.Index(part, "[")
			close := strings.LastIndex(part, "]")
			if open > 0 {
				key = part[:open]
			} else {
				key = "" // e.g. [0] at start or after dot
			}
			idxStr := part[open+1 : close]
			var err error
			index, err = strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index in path segment '%s': %v", part, err)
			}
		}

		if key != "" {
			m, ok := current.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map at '%s' but got %T (value: %v)", key, current, current)
			}
			val, exists := m[key]
			if !exists {
				return nil, fmt.Errorf("key '%s' not found", key)
			}
			current = val
		}

		if index >= 0 {
			arr, ok := current.([]interface{})
			if !ok {
				return nil, fmt.Errorf("expected array for index [%d] at '%s' but got %T (value: %v)", index, part, current, current)
			}
			if index >= len(arr) {
				return nil, fmt.Errorf("index %d out of bounds (len: %d)", index, len(arr))
			}
			current = arr[index]
		}
	}
	return current, nil
}

// setValueByPath sets a value in a nested map/slice structure at the given dot+bracket path.
// It supports dot notation and array indices (e.g. "a.b[0].c").
func setValueByPath(data interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")

	// parsePart splits a segment like "key[0]" into (key, index).
	// If no array index, index is -1.
	parsePart := func(part string) (string, int, error) {
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			open := strings.Index(part, "[")
			close := strings.LastIndex(part, "]")
			key := ""
			if open > 0 {
				key = part[:open]
			}
			idx, err := strconv.Atoi(part[open+1 : close])
			if err != nil {
				return "", -1, fmt.Errorf("invalid array index in path segment '%s': %v", part, err)
			}
			return key, idx, nil
		}
		return part, -1, nil
	}

	var walk func(current interface{}, parts []string) error
	walk = func(current interface{}, parts []string) error {
		if len(parts) == 0 {
			return nil
		}

		part := parts[0]
		key, index, err := parsePart(part)
		if err != nil {
			return err
		}

		isLast := len(parts) == 1

		// helper: resolve the "next" container after applying key then index
		resolveNext := func() (interface{}, error) {
			var afterKey interface{} = current
			if key != "" {
				m, ok := current.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("expected map at '%s' but got %T", key, current)
				}
				v, exists := m[key]
				if !exists {
					return nil, fmt.Errorf("key '%s' not found", key)
				}
				afterKey = v
			}
			if index >= 0 {
				arr, ok := afterKey.([]interface{})
				if !ok {
					return nil, fmt.Errorf("expected array for index [%d] but got %T", index, afterKey)
				}
				if index >= len(arr) {
					return nil, fmt.Errorf("index %d out of bounds (len: %d)", index, len(arr))
				}
				return arr[index], nil
			}
			return afterKey, nil
		}

		if isLast {
			// Set the value here
			if key != "" {
				m, ok := current.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map at '%s' but got %T", key, current)
				}
				if index >= 0 {
					// e.g. "arr[0]" as last segment
					v, exists := m[key]
					if !exists {
						return fmt.Errorf("key '%s' not found", key)
					}
					arr, ok := v.([]interface{})
					if !ok {
						return fmt.Errorf("expected array for index [%d] but got %T", index, v)
					}
					if index >= len(arr) {
						return fmt.Errorf("index %d out of bounds (len: %d)", index, len(arr))
					}
					arr[index] = value
				} else {
					m[key] = value
				}
			} else if index >= 0 {
				arr, ok := current.([]interface{})
				if !ok {
					return fmt.Errorf("expected array for index [%d] but got %T", index, current)
				}
				if index >= len(arr) {
					return fmt.Errorf("index %d out of bounds (len: %d)", index, len(arr))
				}
				arr[index] = value
			}
			return nil
		}

		// Not last: recurse into next container
		next, err := resolveNext()
		if err != nil {
			return err
		}
		return walk(next, parts[1:])
	}

	return walk(data, parts)
}

func isNumber(i interface{}) bool {
	switch i.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}
	return false
}

func toFloat64(i interface{}) float64 {
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	}
	return 0
}

// xmlNode represents a parsed XML element for path-based queries.
type xmlNode struct {
	Name     string
	Attrs    map[string]string
	Text     string
	Children []*xmlNode
}

func parseXMLToNode(data []byte) *xmlNode {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *xmlNode
	var stack []*xmlNode

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			node := &xmlNode{
				Name:  t.Name.Local,
				Attrs: make(map[string]string),
			}
			for _, attr := range t.Attr {
				node.Attrs[attr.Name.Local] = attr.Value
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 0 {
				stack[len(stack)-1].Text = strings.TrimSpace(stack[len(stack)-1].Text)
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(t)
			}
		}
	}
	return root
}

// getXMLPathValue navigates an XML tree using a dot-separated path.
// Supports element names, array indexing (e.g. "items.item[1]"), and attribute access (e.g. "user.@id").
func getXMLPathValue(root *xmlNode, path string) (any, error) {
	if root == nil {
		return nil, fmt.Errorf("XML body is nil or not valid XML")
	}
	parts := strings.Split(path, ".")
	current := root

	for i, part := range parts {
		if current == nil {
			return nil, fmt.Errorf("node is nil at path segment '%s'", part)
		}

		// Attribute access
		if strings.HasPrefix(part, "@") {
			attrName := part[1:]
			if val, ok := current.Attrs[attrName]; ok {
				return val, nil
			}
			return nil, fmt.Errorf("attribute '%s' not found", attrName)
		}

		// Parse element name and optional index
		key := part
		idx := -1
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			idxStart := strings.Index(part, "[")
			key = part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			var err error
			idx, err = strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index in '%s': %v", part, err)
			}
		}

		// If first part matches root element name, skip
		if i == 0 && current.Name == key {
			if idx >= 0 && idx != 0 {
				return nil, fmt.Errorf("root index %d out of bounds", idx)
			}
			continue
		}

		// Find matching children
		var matches []*xmlNode
		for _, child := range current.Children {
			if child.Name == key {
				matches = append(matches, child)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("element '%s' not found", key)
		}

		if idx >= 0 {
			if idx >= len(matches) {
				return nil, fmt.Errorf("index %d out of bounds for element '%s' (count: %d)", idx, key, len(matches))
			}
			current = matches[idx]
		} else {
			current = matches[0]
		}
	}

	return current.Text, nil
}

// ExpectXmlBody asserts that the response body matches the expected XML.
// Comparison is done by re-parsing both into canonical form.
func ExpectXmlBody(resp Response, expectedXml string) {
	if IsDryRun() {
		return
	}
	gotNode := parseXMLToNode([]byte(resp.Body))
	if gotNode == nil {
		Fail("ExpectXmlBody failed: response body is not valid XML. Body: %s", resp.Body)
	}
	expNode := parseXMLToNode([]byte(expectedXml))
	if expNode == nil {
		Fail("ExpectXmlBody failed: expected string is not valid XML: %s", expectedXml)
	}
	if !xmlNodesEqual(gotNode, expNode) {
		Fail("ExpectXmlBody failed:\nExpected: %s\nGot:      %s", expectedXml, resp.Body)
	}
	Log(LogTypeExpect, "XML body matches expected value - PASSED", "")
}

func xmlNodesEqual(a, b *xmlNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name || a.Text != b.Text {
		return false
	}
	if len(a.Attrs) != len(b.Attrs) {
		return false
	}
	for k, v := range a.Attrs {
		if b.Attrs[k] != v {
			return false
		}
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !xmlNodesEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}
	return true
}

// ExpectXmlBodyField asserts that a specific element in the XML response body matches the expected value.
// field supports dot notation, array index (e.g. "root.items.item[0]"), and attribute access (e.g. "root.user.@id").
func ExpectXmlBodyField(resp Response, field string, expectedValue string) {
	if IsDryRun() {
		return
	}
	root := parseXMLToNode([]byte(resp.Body))
	if root == nil {
		Fail("ExpectXmlBodyField failed: response body is not valid XML. Body: %s", resp.Body)
	}
	gotValue, err := getXMLPathValue(root, field)
	if err != nil {
		Fail("ExpectXmlBodyField failed to get field '%s': %v. Body: %s", field, err, resp.Body)
	}
	gotStr := fmt.Sprintf("%v", gotValue)
	if gotStr != expectedValue {
		Fail("ExpectXmlBodyField failed for field '%s':\nExpected: %s\nGot:      %s", field, expectedValue, gotStr)
	}
	Logf(LogTypeExpect, "XML Field '%s' == %s - PASSED", field, expectedValue)
}

// ExpectXmlBodyFieldCond asserts that a specific element in the XML response body
// satisfies the provided condition against the expected value.
func ExpectXmlBodyFieldCond(resp Response, field string, condition string, expectedValue string) {
	if IsDryRun() {
		return
	}
	root := parseXMLToNode([]byte(resp.Body))
	if root == nil {
		Fail("ExpectXmlBodyFieldCond failed: response body is not valid XML. Body: %s", resp.Body)
	}
	gotValue, err := getXMLPathValue(root, field)
	if err != nil {
		Fail("ExpectXmlBodyFieldCond failed to get field '%s': %v. Body: %s", field, err, resp.Body)
	}
	gotStr := fmt.Sprintf("%v", gotValue)
	if !evaluateCondition(gotStr, condition, expectedValue) {
		Fail("ExpectXmlBodyFieldCond failed for field '%s' with condition '%s':\nExpected: %s\nGot:      %s", field, condition, expectedValue, gotStr)
	}
	Logf(LogTypeExpect, "XML Field '%s' %s %s - PASSED", field, condition, expectedValue)
}

// PrettyXml formats an XML string with indentation for readability.
// If the input is not valid XML, it returns the original string unchanged.
func PrettyXml(xmlStr string) string {
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if err := encoder.EncodeToken(tok); err != nil {
			return xmlStr
		}
	}
	if err := encoder.Flush(); err != nil {
		return xmlStr
	}
	result := buf.String()
	if result == "" {
		return xmlStr
	}
	return result
}
