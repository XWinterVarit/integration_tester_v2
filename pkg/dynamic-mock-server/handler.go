package dynamic_mock_server

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// HandlerExecutor executes the response functions
// XMLNode represents a parsed XML element for path-based queries
type XMLNode struct {
	Name     string
	Attrs    map[string]string
	Text     string
	Children []*XMLNode
}

type HandlerExecutor struct {
	Variables      map[string]interface{}
	Request        *http.Request
	ParsedBody     interface{}
	ParsedXMLBody  *XMLNode
	RawBody        []byte
	ResponseWriter http.ResponseWriter

	// Response State
	StatusCode int
	Body       string
	Headers    map[string]string
	FixedDelay time.Duration
	RandomWait [2]int // min, max
	ActiveCase string
}

func NewHandlerExecutor(w http.ResponseWriter, r *http.Request) *HandlerExecutor {
	return &HandlerExecutor{
		Variables:      make(map[string]interface{}),
		Request:        r,
		ResponseWriter: w,
		StatusCode:     200,
		Headers:        make(map[string]string),
	}
}

func (h *HandlerExecutor) Execute(funcs []ResponseFuncConfig) error {
	// Pre-parse body if needed
	if h.Request.Body != nil {
		bodyBytes, _ := io.ReadAll(h.Request.Body)
		h.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore for reading if needed
		h.RawBody = bodyBytes
		if len(bodyBytes) > 0 {
			json.Unmarshal(bodyBytes, &h.ParsedBody)
			h.ParsedXMLBody = parseXML(bodyBytes)
		}
	}

	for _, f := range funcs {
		if err := h.runFunc(f); err != nil {
			return err
		}
	}

	return nil
}

func (h *HandlerExecutor) Finalize() {
	// Apply delays
	if h.FixedDelay > 0 {
		time.Sleep(h.FixedDelay)
	}
	if h.RandomWait[1] > 0 {
		min := h.RandomWait[0]
		max := h.RandomWait[1]
		if max > min {
			sleepTime := time.Duration(rand.Intn(max-min)+min) * time.Millisecond
			time.Sleep(sleepTime)
		}
	}

	// Apply headers
	for k, v := range h.Headers {
		h.ResponseWriter.Header().Set(k, v)
	}

	// Write status
	h.ResponseWriter.WriteHeader(h.StatusCode)

	// Write body
	// Apply template to body one last time if it contains variables?
	// The requirement says SetJsonBody takes a template string.
	// So h.Body likely already stores the template string.
	// We should execute it now.
	finalBody := h.resolveString(h.Body)
	h.ResponseWriter.Write([]byte(finalBody))
}

func (h *HandlerExecutor) runFunc(f ResponseFuncConfig) error {
	switch f.Group {
	case GroupPrepareData:
		return h.handlePrepareData(f)
	case GroupGenerator:
		return h.handleGenerator(f)
	case GroupDynamicVariable:
		return h.handleDynamicVariable(f)
	case GroupSetupResponse:
		return h.handleSetupResponse(f)
	}
	return nil
}

// Helper to resolve templates in strings
func (h *HandlerExecutor) resolveString(s string) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	t, err := template.New("tmpl").Parse(s)
	if err != nil {
		return s // Return raw if parse fails
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, h.Variables); err != nil {
		return s // Return raw if execute fails
	}
	return buf.String()
}

func (h *HandlerExecutor) resolveArg(arg interface{}) interface{} {
	if s, ok := arg.(string); ok {
		return h.resolveString(s)
	}
	return arg
}

func (h *HandlerExecutor) handlePrepareData(f ResponseFuncConfig) error {
	args := f.Args
	var condition string
	var expectedVal interface{}
	var targetVar string
	var toBeVal interface{}
	var actualVal interface{}

	switch f.Func {
	case FuncIfRequestHeader:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		headerName := fmt.Sprintf("%v", args[0])
		actualVal = h.Request.Header.Get(headerName)
	case FuncIfRequestJsonBody:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		fieldPath := fmt.Sprintf("%v", args[0])
		actualVal = h.getJSONPath(fieldPath)
	case FuncIfRequestXmlBody:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		fieldPath := fmt.Sprintf("%v", args[0])
		actualVal = h.getXMLPath(fieldPath)
	case FuncIfRequestPath:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[0])
		expectedVal = h.resolveArg(args[1])
		targetVar = fmt.Sprintf("%v", args[2])
		toBeVal = h.resolveArg(args[3])
		actualVal = h.Request.URL.Path
	case FuncIfRequestQuery:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		queryField := fmt.Sprintf("%v", args[0])
		actualVal = h.Request.URL.Query().Get(queryField)

	case FuncIfDynamicVariable:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		varName := fmt.Sprintf("%v", args[0])
		if val, ok := h.Variables[varName]; ok {
			actualVal = val
		} else {
			actualVal = nil
		}

	case FuncIfRequestJsonArrayLength:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		fieldPath := fmt.Sprintf("%v", args[0])
		val := h.getJSONPath(fieldPath)
		if arr, ok := val.([]interface{}); ok {
			actualVal = len(arr)
		} else {
			actualVal = -1
		}

	case FuncIfRequestJsonObjectLength:
		if len(args) < 5 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		targetVar = fmt.Sprintf("%v", args[3])
		toBeVal = h.resolveArg(args[4])

		fieldPath := fmt.Sprintf("%v", args[0])
		val := h.getJSONPath(fieldPath)
		if m, ok := val.(map[string]interface{}); ok {
			actualVal = len(m)
		} else {
			actualVal = -1
		}

	case FuncIfRequestJsonType:
		if len(args) < 4 {
			return nil
		}
		// Field, TypeStr, TargetVar, ToBeValue
		// Implicit condition "Equal" for type check
		condition = ConditionEqual
		expectedVal = fmt.Sprintf("%v", args[1])
		targetVar = fmt.Sprintf("%v", args[2])
		toBeVal = h.resolveArg(args[3])

		fieldPath := fmt.Sprintf("%v", args[0])
		val := h.getJSONPath(fieldPath)
		actualVal = getTypeOf(val)

	case FuncIfRequestHeaderSetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		headerName := fmt.Sprintf("%v", args[0])
		actualVal = h.Request.Header.Get(headerName)
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestJsonBodySetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		fieldPath := fmt.Sprintf("%v", args[0])
		actualVal = h.getJSONPath(fieldPath)
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestXmlBodySetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		fieldPath := fmt.Sprintf("%v", args[0])
		actualVal = h.getXMLPath(fieldPath)
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestPathSetCase:
		if len(args) < 3 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[0])
		expectedVal = h.resolveArg(args[1])
		caseStr := fmt.Sprintf("%v", args[2])
		actualVal = h.Request.URL.Path
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestQuerySetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		queryField := fmt.Sprintf("%v", args[0])
		actualVal = h.Request.URL.Query().Get(queryField)
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfDynamicVariableSetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		varName := fmt.Sprintf("%v", args[0])
		if val, ok := h.Variables[varName]; ok {
			actualVal = val
		} else {
			actualVal = nil
		}
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestJsonArrayLengthSetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		fieldPath := fmt.Sprintf("%v", args[0])
		val := h.getJSONPath(fieldPath)
		if arr, ok := val.([]interface{}); ok {
			actualVal = len(arr)
		} else {
			actualVal = -1
		}
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestJsonObjectLengthSetCase:
		if len(args) < 4 {
			return nil
		}
		condition = fmt.Sprintf("%v", args[1])
		expectedVal = h.resolveArg(args[2])
		caseStr := fmt.Sprintf("%v", args[3])

		fieldPath := fmt.Sprintf("%v", args[0])
		val := h.getJSONPath(fieldPath)
		if m, ok := val.(map[string]interface{}); ok {
			actualVal = len(m)
		} else {
			actualVal = -1
		}
		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncIfRequestJsonTypeSetCase:
		if len(args) < 3 {
			return nil
		}
		// Field, TypeStr, CaseStr
		condition = ConditionEqual
		expectedVal = fmt.Sprintf("%v", args[1])
		caseStr := fmt.Sprintf("%v", args[2])

		fieldPath := fmt.Sprintf("%v", args[0])
		val := h.getJSONPath(fieldPath)
		actualVal = getTypeOf(val)

		if h.checkCondition(actualVal, condition, expectedVal) {
			h.ActiveCase = caseStr
		}
		return nil

	case FuncExtractRequestHeader:
		if len(args) < 2 {
			return nil
		}
		headerName := fmt.Sprintf("%v", args[0])
		targetVar := fmt.Sprintf("%v", args[1])
		h.Variables[targetVar] = h.Request.Header.Get(headerName)
		return nil

	case FuncExtractRequestJsonBody:
		if len(args) < 2 {
			return nil
		}
		fieldPath := fmt.Sprintf("%v", args[0])
		targetVar := fmt.Sprintf("%v", args[1])
		val := h.getJSONPath(fieldPath)
		if val != nil {
			h.Variables[targetVar] = val
		}
		return nil

	case FuncExtractRequestXmlBody:
		if len(args) < 2 {
			return nil
		}
		fieldPath := fmt.Sprintf("%v", args[0])
		targetVar := fmt.Sprintf("%v", args[1])
		val := h.getXMLPath(fieldPath)
		if val != nil {
			h.Variables[targetVar] = val
		}
		return nil

	case FuncExtractRequestPath:
		if len(args) < 1 {
			return nil
		}
		targetVar := fmt.Sprintf("%v", args[0])
		h.Variables[targetVar] = h.Request.URL.Path
		return nil

	case FuncExtractRequestQuery:
		if len(args) < 2 {
			return nil
		}
		queryField := fmt.Sprintf("%v", args[0])
		targetVar := fmt.Sprintf("%v", args[1])
		h.Variables[targetVar] = h.Request.URL.Query().Get(queryField)
		return nil
	}

	if h.checkCondition(actualVal, condition, expectedVal) {
		h.Variables[targetVar] = toBeVal
	}

	return nil
}

func (h *HandlerExecutor) checkCondition(actual interface{}, cond string, expected interface{}) bool {
	actStr := fmt.Sprintf("%v", actual)
	expStr := fmt.Sprintf("%v", expected)

	switch cond {
	case ConditionEqual:
		return actStr == expStr
	case ConditionNotEqual:
		return actStr != expStr
	case ConditionContains:
		return strings.Contains(actStr, expStr)
	case ConditionNotContains:
		return !strings.Contains(actStr, expStr)
	case ConditionStartsWith:
		return strings.HasPrefix(actStr, expStr)
	case ConditionEndsWith:
		return strings.HasSuffix(actStr, expStr)
	case ConditionGreaterThan, ConditionLessThan, ConditionGreaterThanOrEqual, ConditionLessThanOrEqual:
		actNum, ok1 := tryToFloat(actual)
		expNum, ok2 := tryToFloat(expected)
		if !ok1 || !ok2 {
			return false
		}
		switch cond {
		case ConditionGreaterThan:
			return actNum > expNum
		case ConditionLessThan:
			return actNum < expNum
		case ConditionGreaterThanOrEqual:
			return actNum >= expNum
		case ConditionLessThanOrEqual:
			return actNum <= expNum
		}
	}
	return false
}

func parseXML(data []byte) *XMLNode {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *XMLNode
	var stack []*XMLNode

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			node := &XMLNode{
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
				// Trim text of the closing element
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

// getXMLPath navigates an XML tree using a dot-separated path.
// Supports element names, array indexing (e.g. "items.item[1]"), and attribute access (e.g. "user.@id").
func (h *HandlerExecutor) getXMLPath(path string) interface{} {
	if h.ParsedXMLBody == nil {
		return nil
	}
	parts := strings.Split(path, ".")
	current := h.ParsedXMLBody

	for i, part := range parts {
		if current == nil {
			return nil
		}

		// Attribute access
		if strings.HasPrefix(part, "@") {
			attrName := part[1:]
			if val, ok := current.Attrs[attrName]; ok {
				return val
			}
			return nil
		}

		// Parse element name and optional index
		key := part
		idx := -1
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			idxStart := strings.Index(part, "[")
			key = part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			idx, _ = strconv.Atoi(idxStr)
		}

		// If first part matches root element name, skip to root
		if i == 0 && current.Name == key {
			if idx >= 0 {
				if idx != 0 {
					return nil
				}
			}
			continue
		}

		// Find matching children
		var matches []*XMLNode
		for _, child := range current.Children {
			if child.Name == key {
				matches = append(matches, child)
			}
		}
		if len(matches) == 0 {
			return nil
		}

		if idx >= 0 {
			if idx >= len(matches) {
				return nil
			}
			current = matches[idx]
		} else {
			current = matches[0]
		}
	}

	// Return text content of the final node
	return current.Text
}

func (h *HandlerExecutor) getJSONPath(path string) interface{} {
	if h.ParsedBody == nil {
		return nil
	}
	parts := strings.Split(path, ".")
	var current interface{} = h.ParsedBody

	for _, part := range parts {
		// handle array index like a[0]
		key := part
		idx := -1

		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			// Extract key and index
			idxStart := strings.Index(part, "[")
			key = part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			idx, _ = strconv.Atoi(idxStr)
		}

		// Access map
		m, ok := current.(map[string]interface{})
		if !ok {
			// Maybe current is just the array? e.g. path "0.field"
			// But JSON root is usually object.
			return nil
		}

		val, exists := m[key]
		if !exists {
			return nil
		}
		current = val

		// Access array if index present
		if idx >= 0 {
			arr, ok := current.([]interface{})
			if !ok || idx >= len(arr) {
				return nil
			}
			current = arr[idx]
		}
	}
	return current
}

func (h *HandlerExecutor) handleGenerator(f ResponseFuncConfig) error {
	args := f.Args
	switch f.Func {
	case FuncGenerateRandomString:
		length := int(toFloat(args[0]))
		targetVar := fmt.Sprintf("%v", args[1])
		h.Variables[targetVar] = randomString(length)
	case FuncGenerateRandomInt:
		min := int(toFloat(args[0]))
		max := int(toFloat(args[1]))
		targetVar := fmt.Sprintf("%v", args[2])
		h.Variables[targetVar] = rand.Intn(max-min+1) + min
	case FuncGenerateRandomIntFixLength:
		length := int(toFloat(args[0]))
		targetVar := fmt.Sprintf("%v", args[1])
		// Not perfect but works for simple case
		min := int(1 * pow10(length-1))
		max := int(1*pow10(length) - 1)
		h.Variables[targetVar] = rand.Intn(max-min+1) + min
	case FuncGenerateRandomDecimal:
		min := toFloat(args[0])
		max := toFloat(args[1])
		// maxDecimal := int(toFloat(args[2])) // unused in simple implementation
		targetVar := fmt.Sprintf("%v", args[3])
		val := min + rand.Float64()*(max-min)
		h.Variables[targetVar] = val
	case FuncHashedString:
		fromVar := fmt.Sprintf("%v", args[0])
		algo := fmt.Sprintf("%v", args[1])
		targetVar := fmt.Sprintf("%v", args[2])

		val := fmt.Sprintf("%v", h.Variables[fromVar])
		var hash string
		if algo == "MD5" {
			sum := md5.Sum([]byte(val))
			hash = hex.EncodeToString(sum[:])
		} else if algo == "SHA256" {
			sum := sha256.Sum256([]byte(val))
			hash = hex.EncodeToString(sum[:])
		}
		h.Variables[targetVar] = hash
	}
	return nil
}

func (h *HandlerExecutor) handleDynamicVariable(f ResponseFuncConfig) error {
	args := f.Args
	targetVar := fmt.Sprintf("%v", args[0])

	switch f.Func {
	case FuncConvertToString:
		if v, ok := h.Variables[targetVar]; ok {
			h.Variables[targetVar] = fmt.Sprintf("%v", v)
		}
	case FuncConvertToInt:
		if v, ok := h.Variables[targetVar]; ok {
			h.Variables[targetVar] = int(toFloat(v))
		}
	case FuncDynamicVarSubstring:
		// Args: sourceVar, start, end, targetVar
		sourceVar := fmt.Sprintf("%v", args[0])
		start := int(toFloat(args[1]))
		end := int(toFloat(args[2]))
		dstVar := fmt.Sprintf("%v", args[3])

		if v, ok := h.Variables[sourceVar]; ok {
			strVal := fmt.Sprintf("%v", v)
			if start < 0 {
				start = 0
			}
			if end > len(strVal) {
				end = len(strVal)
			}
			if start <= end {
				h.Variables[dstVar] = strVal[start:end]
			} else {
				h.Variables[dstVar] = ""
			}
		}
	case FuncDynamicVarJoin:
		// Args: targetVar, separator, part1, part2...
		dstVar := fmt.Sprintf("%v", args[0])
		sep := fmt.Sprintf("%v", args[1])
		var parts []string
		for i := 2; i < len(args); i++ {
			// Resolve each part as a potential template or value
			val := h.resolveArg(args[i])
			parts = append(parts, fmt.Sprintf("%v", val))
		}
		h.Variables[dstVar] = strings.Join(parts, sep)
	case FuncDelete:
		delete(h.Variables, targetVar)
	}
	return nil
}

func (h *HandlerExecutor) handleSetupResponse(f ResponseFuncConfig) error {
	args := f.Args
	if len(args) == 0 {
		return nil
	}
	caseStr := fmt.Sprintf("%v", args[0])
	// If ActiveCase is "", it matches "" (default)
	// If ActiveCase is "CaseA", it matches "CaseA"
	if caseStr != h.ActiveCase {
		return nil
	}

	switch f.Func {
	case FuncSetJsonBody:
		h.Body = fmt.Sprintf("%v", args[1])
	case FuncSetXmlBody:
		h.Body = fmt.Sprintf("%v", args[1])
	case FuncSetStatusCode:
		h.StatusCode = int(toFloat(args[1]))
	case FuncSetWait:
		h.FixedDelay = time.Duration(toFloat(args[1])) * time.Millisecond
	case FuncSetRandomWait:
		h.RandomWait[0] = int(toFloat(args[1]))
		h.RandomWait[1] = int(toFloat(args[2]))
	case FuncSetMethod:
		// Usually response doesn't set method, maybe this is for asserting?
		// Or maybe it's mimicking? The req says "SetMethod".
		// Unclear usage for response, ignoring for now or logging.
	case FuncSetHeader:
		key := fmt.Sprintf("%v", args[1])
		val := h.resolveString(fmt.Sprintf("%v", args[2]))
		h.Headers[key] = val
	case FuncCopyHeaderFromRequest:
		key := fmt.Sprintf("%v", args[1])
		val := h.Request.Header.Get(key)
		if val != "" {
			h.Headers[key] = val
		}
	}
	return nil
}

// Utils

func tryToFloat(i interface{}) (float64, bool) {
	if i == nil {
		return 0, false
	}
	switch v := i.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	// Try parsing string representation
	s := fmt.Sprintf("%v", i)
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func toFloat(i interface{}) float64 {
	switch v := i.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	}
	return 0
}

func pow10(n int) float64 {
	r := 1.0
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getTypeOf(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	}
	return "unknown"
}
