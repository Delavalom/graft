# Bedrock Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an AWS Bedrock provider using the Converse API with zero external SDK dependencies — SigV4 signing and Event Stream binary decoding in pure Go stdlib.

**Architecture:** New `provider/bedrock/` package implementing `graft.LanguageModel`. Uses the Bedrock Converse API (unified format across all Bedrock models). Auth via SigV4 signing or anonymous mode for proxy setups. Streaming via AWS Event Stream binary protocol decoding.

**Tech Stack:** Go stdlib only — `crypto/hmac`, `crypto/sha256`, `encoding/binary`, `hash/crc32`, `net/http`

**Spec:** `docs/superpowers/specs/2026-03-30-bedrock-provider-design.md`

---

## File Structure

```
provider/bedrock/
    types.go          # Converse API request/response structs + message conversion
    sigv4.go          # AWS Signature V4 signing
    eventstream.go    # AWS Event Stream binary frame decoder
    bedrock.go        # Client, New(), Generate(), ModelID(), options, doRequest
    stream.go         # Stream() implementation using eventstream decoder
    bedrock_test.go   # All unit tests
```

---

### Task 1: Converse API Types and Message Conversion

**Files:**
- Create: `provider/bedrock/types.go`
- Create: `provider/bedrock/bedrock_test.go`

- [ ] **Step 1: Write the failing test for message conversion**

Create `provider/bedrock/bedrock_test.go`:

```go
package bedrock

import (
	"encoding/json"
	"testing"

	"github.com/delavalom/graft"
)

func TestConvertMessages_SystemExtracted(t *testing.T) {
	msgs := []graft.Message{
		{Role: graft.RoleSystem, Content: "You are helpful."},
		{Role: graft.RoleUser, Content: "Hello"},
	}
	system, converted := convertMessages(msgs)
	if len(system) != 1 || system[0].Text != "You are helpful." {
		t.Errorf("expected system block, got %+v", system)
	}
	if len(converted) != 1 || converted[0].Role != "user" {
		t.Errorf("expected 1 user message, got %+v", converted)
	}
	if len(converted[0].Content) != 1 || converted[0].Content[0].Text == "" {
		t.Errorf("expected text content block, got %+v", converted[0].Content)
	}
}

func TestConvertMessages_AssistantWithToolCalls(t *testing.T) {
	msgs := []graft.Message{
		{
			Role:    graft.RoleAssistant,
			Content: "Let me check.",
			ToolCalls: []graft.ToolCall{
				{
					ID:        "call_123",
					Name:      "get_weather",
					Arguments: json.RawMessage(`{"city":"NYC"}`),
				},
			},
		},
	}
	_, converted := convertMessages(msgs)
	if len(converted) != 1 {
		t.Fatalf("expected 1 message, got %d", len(converted))
	}
	msg := converted[0]
	if msg.Role != "assistant" {
		t.Errorf("expected assistant role, got %s", msg.Role)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content blocks (text + toolUse), got %d", len(msg.Content))
	}
	if msg.Content[0].Text != "Let me check." {
		t.Errorf("expected text block, got %+v", msg.Content[0])
	}
	if msg.Content[1].ToolUse == nil || msg.Content[1].ToolUse.Name != "get_weather" {
		t.Errorf("expected toolUse block, got %+v", msg.Content[1])
	}
}

func TestConvertMessages_ToolResult(t *testing.T) {
	msgs := []graft.Message{
		{
			Role: graft.RoleTool,
			ToolResult: &graft.ToolResult{
				CallID:  "call_123",
				Content: "72°F, sunny",
				IsError: false,
			},
		},
	}
	_, converted := convertMessages(msgs)
	if len(converted) != 1 {
		t.Fatalf("expected 1 message, got %d", len(converted))
	}
	msg := converted[0]
	if msg.Role != "user" {
		t.Errorf("tool results become user messages in Converse, got %s", msg.Role)
	}
	if len(msg.Content) != 1 || msg.Content[0].ToolResult == nil {
		t.Fatalf("expected toolResult content block, got %+v", msg.Content)
	}
	tr := msg.Content[0].ToolResult
	if tr.ToolUseID != "call_123" {
		t.Errorf("expected call_123, got %s", tr.ToolUseID)
	}
	if tr.Status != "success" {
		t.Errorf("expected success, got %s", tr.Status)
	}
}

func TestConvertMessages_ToolResultError(t *testing.T) {
	msgs := []graft.Message{
		{
			Role: graft.RoleTool,
			ToolResult: &graft.ToolResult{
				CallID:  "call_456",
				Content: "connection refused",
				IsError: true,
			},
		},
	}
	_, converted := convertMessages(msgs)
	tr := converted[0].Content[0].ToolResult
	if tr.Status != "error" {
		t.Errorf("expected error status, got %s", tr.Status)
	}
}

func TestConvertTools(t *testing.T) {
	tools := []graft.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get the weather",
			Schema:      json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
		},
	}
	converted := convertTools(tools)
	if len(converted) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(converted))
	}
	if converted[0].ToolSpec.Name != "get_weather" {
		t.Errorf("expected get_weather, got %s", converted[0].ToolSpec.Name)
	}
	if converted[0].ToolSpec.InputSchema.JSON == nil {
		t.Error("expected schema in InputSchema.JSON")
	}
}

func TestConvertTools_Empty(t *testing.T) {
	converted := convertTools(nil)
	if converted != nil {
		t.Errorf("expected nil for empty tools, got %+v", converted)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./provider/bedrock/ -v -run TestConvert`
Expected: FAIL — `convertMessages` and `convertTools` undefined

- [ ] **Step 3: Write the types and conversion functions**

Create `provider/bedrock/types.go`:

```go
// Package bedrock implements an AWS Bedrock Converse API provider for the graft framework.
package bedrock

import (
	"encoding/json"
	"fmt"

	"github.com/delavalom/graft"
)

// --- Converse API request types ---

type converseRequest struct {
	Messages        []converseMessage `json:"messages"`
	System          []systemBlock     `json:"system,omitempty"`
	InferenceConfig *inferenceConfig  `json:"inferenceConfig,omitempty"`
	ToolConfig      *toolConfig       `json:"toolConfig,omitempty"`
}

type systemBlock struct {
	Text string `json:"text"`
}

type converseMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Text       string           `json:"text,omitempty"`
	ToolUse    *toolUseBlock    `json:"toolUse,omitempty"`
	ToolResult *toolResultBlock `json:"toolResult,omitempty"`
}

type toolUseBlock struct {
	ToolUseID string          `json:"toolUseId"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

type toolResultBlock struct {
	ToolUseID string         `json:"toolUseId"`
	Content   []contentBlock `json:"content"`
	Status    string         `json:"status"`
}

type inferenceConfig struct {
	MaxTokens     int      `json:"maxTokens,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type toolConfig struct {
	Tools      []toolDef   `json:"tools"`
	ToolChoice *toolChoice `json:"toolChoice,omitempty"`
}

type toolDef struct {
	ToolSpec toolSpec `json:"toolSpec"`
}

type toolSpec struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	JSON json.RawMessage `json:"json"`
}

type toolChoice struct {
	Auto *struct{} `json:"auto,omitempty"`
	Any  *struct{} `json:"any,omitempty"`
	Tool *struct {
		Name string `json:"name"`
	} `json:"tool,omitempty"`
}

// --- Converse API response types ---

type converseResponse struct {
	Output     converseOutput `json:"output"`
	StopReason string         `json:"stopReason"`
	Usage      converseUsage  `json:"usage"`
}

type converseOutput struct {
	Message converseMessage `json:"message"`
}

type converseUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

// --- ConverseStream event types ---

type streamMessageStart struct {
	Role string `json:"role"`
}

type streamContentBlockStart struct {
	ContentBlockIndex int                      `json:"contentBlockIndex"`
	Start             *streamContentBlockMeta  `json:"start,omitempty"`
}

type streamContentBlockMeta struct {
	ToolUse *streamToolUseStart `json:"toolUse,omitempty"`
}

type streamToolUseStart struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
}

type streamContentBlockDelta struct {
	ContentBlockIndex int               `json:"contentBlockIndex"`
	Delta             streamDelta       `json:"delta"`
}

type streamDelta struct {
	Text    string          `json:"text,omitempty"`
	ToolUse *streamToolDelta `json:"toolUse,omitempty"`
}

type streamToolDelta struct {
	Input string `json:"input"`
}

type streamContentBlockStop struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
}

type streamMessageStop struct {
	StopReason string `json:"stopReason"`
}

type streamMetadata struct {
	Usage   converseUsage `json:"usage"`
	Metrics *struct {
		LatencyMs int `json:"latencyMs"`
	} `json:"metrics,omitempty"`
}

// --- Conversion functions ---

// convertMessages converts graft messages to Converse format.
// System messages are extracted into a separate slice.
func convertMessages(msgs []graft.Message) ([]systemBlock, []converseMessage) {
	var system []systemBlock
	var out []converseMessage

	for _, m := range msgs {
		switch m.Role {
		case graft.RoleSystem:
			system = append(system, systemBlock{Text: m.Content})

		case graft.RoleUser:
			out = append(out, converseMessage{
				Role:    "user",
				Content: []contentBlock{{Text: m.Content}},
			})

		case graft.RoleAssistant:
			var blocks []contentBlock
			if m.Content != "" {
				blocks = append(blocks, contentBlock{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, contentBlock{
					ToolUse: &toolUseBlock{
						ToolUseID: tc.ID,
						Name:      tc.Name,
						Input:     tc.Arguments,
					},
				})
			}
			out = append(out, converseMessage{
				Role:    "assistant",
				Content: blocks,
			})

		case graft.RoleTool:
			if m.ToolResult != nil {
				content := ""
				switch v := m.ToolResult.Content.(type) {
				case string:
					content = v
				default:
					b, _ := json.Marshal(v)
					content = string(b)
				}
				status := "success"
				if m.ToolResult.IsError {
					status = "error"
				}
				out = append(out, converseMessage{
					Role: "user",
					Content: []contentBlock{
						{
							ToolResult: &toolResultBlock{
								ToolUseID: m.ToolResult.CallID,
								Content:   []contentBlock{{Text: content}},
								Status:    status,
							},
						},
					},
				})
			}
		}
	}
	return system, out
}

// convertTools converts graft tool definitions to Converse format.
func convertTools(tools []graft.ToolDefinition) []toolDef {
	if len(tools) == 0 {
		return nil
	}
	out := make([]toolDef, len(tools))
	for i, t := range tools {
		out[i] = toolDef{
			ToolSpec: toolSpec{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: inputSchema{JSON: t.Schema},
			},
		}
	}
	return out
}

// convertToolChoice maps graft ToolChoice to Converse toolChoice.
func convertToolChoice(tc graft.ToolChoice) *toolChoice {
	switch tc {
	case "", graft.ToolChoiceAuto:
		return &toolChoice{Auto: &struct{}{}}
	case graft.ToolChoiceRequired:
		return &toolChoice{Any: &struct{}{}}
	case graft.ToolChoiceNone:
		return nil
	default:
		// Specific tool name
		return &toolChoice{Tool: &struct{ Name string }{Name: string(tc)}}
	}
}

// parseResponseMessage converts a Converse response message to a graft.Message.
func parseResponseMessage(msg converseMessage) graft.Message {
	result := graft.Message{
		Role: graft.RoleAssistant,
	}
	for _, block := range msg.Content {
		if block.Text != "" {
			if result.Content != "" {
				result.Content += block.Text
			} else {
				result.Content = block.Text
			}
		}
		if block.ToolUse != nil {
			input, _ := json.Marshal(block.ToolUse.Input)
			// If Input is already json.RawMessage, use it directly
			if block.ToolUse.Input != nil {
				input = block.ToolUse.Input
			}
			result.ToolCalls = append(result.ToolCalls, graft.ToolCall{
				ID:        block.ToolUse.ToolUseID,
				Name:      block.ToolUse.Name,
				Arguments: input,
			})
		}
	}
	return result
}

// bedrockEndpoint returns the Bedrock Converse API endpoint URL.
func bedrockEndpoint(baseURL, modelID, method string) string {
	return fmt.Sprintf("%s/model/%s/%s", baseURL, modelID, method)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./provider/bedrock/ -v -run TestConvert`
Expected: All 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add provider/bedrock/types.go provider/bedrock/bedrock_test.go
git commit -m "feat(bedrock): add Converse API types and message conversion"
```

---

### Task 2: AWS Signature V4 Signing

**Files:**
- Create: `provider/bedrock/sigv4.go`
- Modify: `provider/bedrock/bedrock_test.go`

- [ ] **Step 1: Write the failing test for SigV4**

Append to `provider/bedrock/bedrock_test.go`:

```go
func TestSignRequest_SetsRequiredHeaders(t *testing.T) {
	body := []byte(`{"messages":[]}`)
	req, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-3-5-sonnet-20241022-v2:0/converse", nil)
	req.Header.Set("Content-Type", "application/json")

	creds := credentials{
		accessKey: "AKIAIOSFODNN7EXAMPLE",
		secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	signRequest(req, body, creds, "us-east-1")

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("expected Authorization header to be set")
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("expected AWS4-HMAC-SHA256 prefix, got %s", auth[:30])
	}
	if !strings.Contains(auth, "Credential=AKIAIOSFODNN7EXAMPLE/") {
		t.Error("expected access key in credential")
	}
	if !strings.Contains(auth, "/us-east-1/bedrock/aws4_request") {
		t.Error("expected region/service in credential scope")
	}
	if !strings.Contains(auth, "SignedHeaders=") {
		t.Error("expected SignedHeaders in auth header")
	}

	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate == "" {
		t.Fatal("expected X-Amz-Date header")
	}
	if len(amzDate) != 16 || amzDate[8] != 'T' || amzDate[15] != 'Z' {
		t.Errorf("expected ISO 8601 basic format, got %s", amzDate)
	}
}

func TestSignRequest_WithSessionToken(t *testing.T) {
	body := []byte(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	req.Header.Set("Content-Type", "application/json")

	creds := credentials{
		accessKey:    "AKID",
		secretKey:    "SECRET",
		sessionToken: "SESSION_TOKEN_VALUE",
	}
	signRequest(req, body, creds, "us-east-1")

	token := req.Header.Get("X-Amz-Security-Token")
	if token != "SESSION_TOKEN_VALUE" {
		t.Errorf("expected session token header, got %q", token)
	}

	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "x-amz-security-token") {
		t.Error("expected x-amz-security-token in signed headers")
	}
}

func TestSignRequest_DeterministicSignature(t *testing.T) {
	body := []byte(`{"test": true}`)
	creds := credentials{accessKey: "AK", secretKey: "SK"}

	req1, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	req1.Header.Set("Content-Type", "application/json")
	signRequest(req1, body, creds, "us-east-1")

	req2, _ := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	req2.Header.Set("Content-Type", "application/json")
	// Set same timestamp to get deterministic result
	req2.Header.Set("X-Amz-Date", req1.Header.Get("X-Amz-Date"))
	signRequest(req2, body, creds, "us-east-1")

	if req1.Header.Get("Authorization") != req2.Header.Get("Authorization") {
		t.Error("same inputs should produce same signature")
	}
}
```

Add this import to the test file's import block:

```go
"net/http"
"strings"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./provider/bedrock/ -v -run TestSignRequest`
Expected: FAIL — `signRequest` and `credentials` undefined

- [ ] **Step 3: Write the SigV4 implementation**

Create `provider/bedrock/sigv4.go`:

```go
package bedrock

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	sigV4Algorithm = "AWS4-HMAC-SHA256"
	sigV4Service   = "bedrock"
	sigV4Request   = "aws4_request"
	amzDateFormat  = "20060102T150405Z"
	dateFormat     = "20060102"
)

// credentials holds AWS authentication credentials.
type credentials struct {
	accessKey    string
	secretKey    string
	sessionToken string
}

// signRequest signs an HTTP request with AWS Signature V4.
// If the request already has an X-Amz-Date header, it is reused (for testing determinism).
func signRequest(req *http.Request, body []byte, creds credentials, region string) {
	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate == "" {
		amzDate = time.Now().UTC().Format(amzDateFormat)
		req.Header.Set("X-Amz-Date", amzDate)
	}
	dateStamp := amzDate[:8]

	if creds.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.sessionToken)
	}

	// Build sorted list of signed headers.
	signedHeaderKeys := []string{"content-type", "host", "x-amz-date"}
	if creds.sessionToken != "" {
		signedHeaderKeys = append(signedHeaderKeys, "x-amz-security-token")
	}
	sort.Strings(signedHeaderKeys)

	// Build canonical headers string.
	var canonicalHeaders strings.Builder
	for _, k := range signedHeaderKeys {
		var v string
		if k == "host" {
			v = req.Host
			if v == "" {
				v = req.URL.Host
			}
		} else {
			v = req.Header.Get(http.CanonicalHeaderKey(k))
		}
		canonicalHeaders.WriteString(k)
		canonicalHeaders.WriteByte(':')
		canonicalHeaders.WriteString(strings.TrimSpace(v))
		canonicalHeaders.WriteByte('\n')
	}

	signedHeaders := strings.Join(signedHeaderKeys, ";")
	payloadHash := sha256Hex(body)

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/%s", dateStamp, region, sigV4Service, sigV4Request)

	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(creds.secretKey, dateStamp, region)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		sigV4Algorithm, creds.accessKey, credentialScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authHeader)
}

// deriveSigningKey computes the SigV4 signing key.
func deriveSigningKey(secretKey, dateStamp, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(sigV4Service))
	kSigning := hmacSHA256(kService, []byte(sigV4Request))
	return kSigning
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./provider/bedrock/ -v -run TestSignRequest`
Expected: All 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add provider/bedrock/sigv4.go provider/bedrock/bedrock_test.go
git commit -m "feat(bedrock): add AWS Signature V4 signing"
```

---

### Task 3: AWS Event Stream Binary Decoder

**Files:**
- Create: `provider/bedrock/eventstream.go`
- Modify: `provider/bedrock/bedrock_test.go`

- [ ] **Step 1: Write the failing test for event stream decoding**

Append to `provider/bedrock/bedrock_test.go`:

```go
func TestEventStreamDecoder_SingleEvent(t *testing.T) {
	payload := []byte(`{"role":"assistant"}`)
	frame := buildEventStreamFrame(t, "messageStart", payload)

	decoder := newEventStreamDecoder(bytes.NewReader(frame))
	eventType, data, err := decoder.readEvent()
	if err != nil {
		t.Fatalf("readEvent failed: %v", err)
	}
	if eventType != "messageStart" {
		t.Errorf("expected messageStart, got %s", eventType)
	}
	if string(data) != string(payload) {
		t.Errorf("expected %s, got %s", payload, data)
	}
}

func TestEventStreamDecoder_MultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildEventStreamFrame(t, "messageStart", []byte(`{"role":"assistant"}`)))
	buf.Write(buildEventStreamFrame(t, "contentBlockDelta", []byte(`{"delta":{"text":"Hello"}}`)))
	buf.Write(buildEventStreamFrame(t, "messageStop", []byte(`{"stopReason":"end_turn"}`)))

	decoder := newEventStreamDecoder(&buf)

	events := []string{}
	for {
		eventType, _, err := decoder.readEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("readEvent failed: %v", err)
		}
		events = append(events, eventType)
	}

	expected := []string{"messageStart", "contentBlockDelta", "messageStop"}
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d", len(expected), len(events))
	}
	for i, e := range expected {
		if events[i] != e {
			t.Errorf("event %d: expected %s, got %s", i, e, events[i])
		}
	}
}

func TestEventStreamDecoder_EmptyPayload(t *testing.T) {
	frame := buildEventStreamFrame(t, "contentBlockStop", []byte(`{}`))
	decoder := newEventStreamDecoder(bytes.NewReader(frame))
	eventType, data, err := decoder.readEvent()
	if err != nil {
		t.Fatalf("readEvent failed: %v", err)
	}
	if eventType != "contentBlockStop" {
		t.Errorf("expected contentBlockStop, got %s", eventType)
	}
	if string(data) != "{}" {
		t.Errorf("expected {}, got %s", data)
	}
}

// buildEventStreamFrame builds a valid AWS Event Stream binary frame for testing.
func buildEventStreamFrame(t *testing.T, eventType string, payload []byte) []byte {
	t.Helper()
	return encodeEventStreamFrame(eventType, payload)
}
```

Add these imports to the test file:

```go
"bytes"
"io"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./provider/bedrock/ -v -run TestEventStreamDecoder`
Expected: FAIL — `newEventStreamDecoder`, `encodeEventStreamFrame` undefined

- [ ] **Step 3: Write the Event Stream decoder**

Create `provider/bedrock/eventstream.go`:

```go
package bedrock

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

const (
	// Frame layout:
	// [4] total length | [4] headers length | [4] prelude CRC
	// [headers-length] headers
	// [payload-length] payload
	// [4] message CRC
	preludeSize = 12 // 4 + 4 + 4
	crcSize     = 4

	// Header value types
	headerValueTypeString = 7
)

// eventStreamDecoder reads AWS Event Stream binary frames.
type eventStreamDecoder struct {
	reader io.Reader
}

func newEventStreamDecoder(r io.Reader) *eventStreamDecoder {
	return &eventStreamDecoder{reader: r}
}

// readEvent reads the next event from the stream.
// Returns io.EOF when no more events are available.
func (d *eventStreamDecoder) readEvent() (eventType string, payload []byte, err error) {
	// Read prelude: total length (4) + headers length (4)
	var totalLen, headersLen uint32
	if err := binary.Read(d.reader, binary.BigEndian, &totalLen); err != nil {
		return "", nil, err // io.EOF if stream ended
	}
	if err := binary.Read(d.reader, binary.BigEndian, &headersLen); err != nil {
		return "", nil, fmt.Errorf("eventstream: read headers length: %w", err)
	}

	// Compute prelude CRC over the first 8 bytes
	preludeBytes := make([]byte, 8)
	binary.BigEndian.PutUint32(preludeBytes[0:4], totalLen)
	binary.BigEndian.PutUint32(preludeBytes[4:8], headersLen)
	expectedPreludeCRC := crc32.Checksum(preludeBytes, crc32cTable)

	var preludeCRC uint32
	if err := binary.Read(d.reader, binary.BigEndian, &preludeCRC); err != nil {
		return "", nil, fmt.Errorf("eventstream: read prelude CRC: %w", err)
	}
	if preludeCRC != expectedPreludeCRC {
		return "", nil, fmt.Errorf("eventstream: prelude CRC mismatch: expected %08x, got %08x", expectedPreludeCRC, preludeCRC)
	}

	// Read headers
	headersBytes := make([]byte, headersLen)
	if _, err := io.ReadFull(d.reader, headersBytes); err != nil {
		return "", nil, fmt.Errorf("eventstream: read headers: %w", err)
	}

	// Parse headers to find :event-type
	headers := parseHeaders(headersBytes)
	eventType = headers[":event-type"]

	// Read payload
	payloadLen := int(totalLen) - preludeSize - int(headersLen) - crcSize
	if payloadLen < 0 {
		return "", nil, fmt.Errorf("eventstream: invalid payload length %d", payloadLen)
	}
	payload = make([]byte, payloadLen)
	if _, err := io.ReadFull(d.reader, payload); err != nil {
		return "", nil, fmt.Errorf("eventstream: read payload: %w", err)
	}

	// Read and verify message CRC
	var messageCRC uint32
	if err := binary.Read(d.reader, binary.BigEndian, &messageCRC); err != nil {
		return "", nil, fmt.Errorf("eventstream: read message CRC: %w", err)
	}

	// Verify message CRC over entire frame minus last 4 bytes
	msgBytes := make([]byte, 0, int(totalLen)-crcSize)
	msgBytes = append(msgBytes, preludeBytes...)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, preludeCRC)
	msgBytes = append(msgBytes, crcBytes...)
	msgBytes = append(msgBytes, headersBytes...)
	msgBytes = append(msgBytes, payload...)
	expectedMsgCRC := crc32.Checksum(msgBytes, crc32cTable)
	if messageCRC != expectedMsgCRC {
		return "", nil, fmt.Errorf("eventstream: message CRC mismatch: expected %08x, got %08x", expectedMsgCRC, messageCRC)
	}

	return eventType, payload, nil
}

// parseHeaders extracts header name-value pairs from the binary header block.
// Only handles string-type headers (type 7), which is all Bedrock uses.
func parseHeaders(data []byte) map[string]string {
	headers := make(map[string]string)
	offset := 0
	for offset < len(data) {
		if offset >= len(data) {
			break
		}
		// Name length (1 byte)
		nameLen := int(data[offset])
		offset++
		if offset+nameLen > len(data) {
			break
		}
		name := string(data[offset : offset+nameLen])
		offset += nameLen

		// Value type (1 byte)
		if offset >= len(data) {
			break
		}
		valueType := data[offset]
		offset++

		if valueType == headerValueTypeString {
			// String value: length (2 bytes big-endian) + value
			if offset+2 > len(data) {
				break
			}
			valueLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
			if offset+valueLen > len(data) {
				break
			}
			headers[name] = string(data[offset : offset+valueLen])
			offset += valueLen
		} else {
			// Skip unknown types — we only need string headers
			break
		}
	}
	return headers
}

// encodeEventStreamFrame builds a valid AWS Event Stream binary frame.
// Used by tests and potentially for writing event stream data.
func encodeEventStreamFrame(eventType string, payload []byte) []byte {
	// Build headers: :event-type (string) and :content-type (string)
	headers := encodeHeader(":event-type", eventType)
	headers = append(headers, encodeHeader(":content-type", "application/json")...)
	headers = append(headers, encodeHeader(":message-type", "event")...)

	headersLen := uint32(len(headers))
	totalLen := uint32(preludeSize + len(headers) + len(payload) + crcSize)

	// Build frame
	frame := make([]byte, 0, totalLen)

	// Prelude: total length + headers length
	prelude := make([]byte, 8)
	binary.BigEndian.PutUint32(prelude[0:4], totalLen)
	binary.BigEndian.PutUint32(prelude[4:8], headersLen)
	frame = append(frame, prelude...)

	// Prelude CRC
	preludeCRC := crc32.Checksum(prelude, crc32cTable)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, preludeCRC)
	frame = append(frame, crcBuf...)

	// Headers + payload
	frame = append(frame, headers...)
	frame = append(frame, payload...)

	// Message CRC (over everything so far)
	msgCRC := crc32.Checksum(frame, crc32cTable)
	binary.BigEndian.PutUint32(crcBuf, msgCRC)
	frame = append(frame, crcBuf...)

	return frame
}

// encodeHeader encodes a single string-type header.
func encodeHeader(name, value string) []byte {
	buf := make([]byte, 0, 1+len(name)+1+2+len(value))
	buf = append(buf, byte(len(name)))
	buf = append(buf, []byte(name)...)
	buf = append(buf, headerValueTypeString)
	valLen := make([]byte, 2)
	binary.BigEndian.PutUint16(valLen, uint16(len(value)))
	buf = append(buf, valLen...)
	buf = append(buf, []byte(value)...)
	return buf
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./provider/bedrock/ -v -run TestEventStreamDecoder`
Expected: All 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add provider/bedrock/eventstream.go provider/bedrock/bedrock_test.go
git commit -m "feat(bedrock): add AWS Event Stream binary decoder"
```

---

### Task 4: Client, Options, and Generate()

**Files:**
- Create: `provider/bedrock/bedrock.go`
- Modify: `provider/bedrock/bedrock_test.go`

- [ ] **Step 1: Write the failing test for Generate**

Append to `provider/bedrock/bedrock_test.go`:

```go
func TestGenerate_SimpleText(t *testing.T) {
	response := converseResponse{
		Output: converseOutput{
			Message: converseMessage{
				Role: "assistant",
				Content: []contentBlock{
					{Text: "Hello from Bedrock!"},
				},
			},
		},
		StopReason: "end_turn",
		Usage: converseUsage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/converse") {
			t.Errorf("expected /converse in path, got %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "/model/") {
			t.Errorf("expected /model/ in path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("anthropic.claude-3-5-sonnet-20241022-v2:0"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.Message.Content != "Hello from Bedrock!" {
		t.Errorf("expected 'Hello from Bedrock!', got %q", result.Message.Content)
	}
	if result.Message.Role != graft.RoleAssistant {
		t.Errorf("expected assistant role, got %s", result.Message.Role)
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 5 {
		t.Errorf("expected 5 completion tokens, got %d", result.Usage.CompletionTokens)
	}
}

func TestGenerate_WithToolCalls(t *testing.T) {
	response := converseResponse{
		Output: converseOutput{
			Message: converseMessage{
				Role: "assistant",
				Content: []contentBlock{
					{Text: "Let me check."},
					{
						ToolUse: &toolUseBlock{
							ToolUseID: "tooluse_abc123",
							Name:      "get_weather",
							Input:     json.RawMessage(`{"city":"Seattle"}`),
						},
					},
				},
			},
		},
		StopReason: "tool_use",
		Usage:      converseUsage{InputTokens: 20, OutputTokens: 15},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	result, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "What's the weather?"},
		},
		Tools: []graft.ToolDefinition{
			{Name: "get_weather", Description: "Get weather", Schema: json.RawMessage(`{"type":"object"}`)},
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.Message.Content != "Let me check." {
		t.Errorf("expected text content, got %q", result.Message.Content)
	}
	if len(result.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.Message.ToolCalls))
	}
	tc := result.Message.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("expected get_weather, got %s", tc.Name)
	}
	if tc.ID != "tooluse_abc123" {
		t.Errorf("expected tooluse_abc123, got %s", tc.ID)
	}
}

func TestGenerate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Access denied"}`))
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	_, err := client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGenerate_WithSystemPrompt(t *testing.T) {
	var receivedBody converseRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := converseResponse{
			Output: converseOutput{
				Message: converseMessage{
					Role:    "assistant",
					Content: []contentBlock{{Text: "OK"}},
				},
			},
			Usage: converseUsage{InputTokens: 5, OutputTokens: 2},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleSystem, Content: "You are helpful."},
			{Role: graft.RoleUser, Content: "Hi"},
		},
	})

	if len(receivedBody.System) != 1 || receivedBody.System[0].Text != "You are helpful." {
		t.Errorf("expected system block in request, got %+v", receivedBody.System)
	}
	if len(receivedBody.Messages) != 1 || receivedBody.Messages[0].Role != "user" {
		t.Errorf("system should not be in messages array, got %+v", receivedBody.Messages)
	}
}

func TestGenerate_SigV4Auth(t *testing.T) {
	var authHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		resp := converseResponse{
			Output: converseOutput{
				Message: converseMessage{Role: "assistant", Content: []contentBlock{{Text: "OK"}}},
			},
			Usage: converseUsage{InputTokens: 1, OutputTokens: 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithCredentials("AKID_TEST", "SECRET_TEST"),
	)

	client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})

	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") {
		t.Errorf("expected SigV4 auth header, got %q", authHeader)
	}
}

func TestModelID(t *testing.T) {
	client := New(WithModel("anthropic.claude-3-5-sonnet-20241022-v2:0"))
	if client.ModelID() != "anthropic.claude-3-5-sonnet-20241022-v2:0" {
		t.Errorf("expected model ID, got %s", client.ModelID())
	}
}

func TestNew_EnvironmentVariables(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ENV_AK")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SK")
	t.Setenv("AWS_SESSION_TOKEN", "ENV_TOKEN")
	t.Setenv("AWS_REGION", "eu-west-1")

	client := New(WithModel("test-model"))
	if client.region != "eu-west-1" {
		t.Errorf("expected eu-west-1, got %s", client.region)
	}
	if client.creds.accessKey != "ENV_AK" {
		t.Errorf("expected ENV_AK, got %s", client.creds.accessKey)
	}
	if client.creds.sessionToken != "ENV_TOKEN" {
		t.Errorf("expected ENV_TOKEN, got %s", client.creds.sessionToken)
	}
}

func TestNew_WithCustomHeaders(t *testing.T) {
	var receivedHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Project-ID")
		resp := converseResponse{
			Output: converseOutput{
				Message: converseMessage{Role: "assistant", Content: []contentBlock{{Text: "OK"}}},
			},
			Usage: converseUsage{InputTokens: 1, OutputTokens: 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
		WithHeader("X-Project-ID", "my-project"),
	)

	client.Generate(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})

	if receivedHeader != "my-project" {
		t.Errorf("expected my-project header, got %q", receivedHeader)
	}
}
```

Add these imports:

```go
"context"
"net/http/httptest"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./provider/bedrock/ -v -run "TestGenerate|TestModelID|TestNew"`
Expected: FAIL — `New`, `WithRegion`, etc. undefined

- [ ] **Step 3: Write the Client and Generate implementation**

Create `provider/bedrock/bedrock.go`:

```go
package bedrock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/delavalom/graft"
)

const (
	defaultMaxTokens = 4096
)

// Client is an AWS Bedrock Converse API provider.
type Client struct {
	region     string
	model      string
	baseURL    string
	maxTokens  int
	creds      credentials
	anonymous  bool
	httpClient *http.Client
	headers    map[string]string
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithRegion sets the AWS region.
func WithRegion(region string) Option {
	return func(c *Client) { c.region = region }
}

// WithModel sets the Bedrock model ID (e.g., "anthropic.claude-3-5-sonnet-20241022-v2:0").
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithBaseURL overrides the Bedrock endpoint URL (for proxies/service meshes).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithCredentials sets explicit AWS IAM credentials.
func WithCredentials(accessKey, secretKey string) Option {
	return func(c *Client) {
		c.creds.accessKey = accessKey
		c.creds.secretKey = secretKey
	}
}

// WithSessionToken sets the AWS session token for STS/assumed roles.
func WithSessionToken(token string) Option {
	return func(c *Client) { c.creds.sessionToken = token }
}

// WithAnonymousAuth disables SigV4 signing (for proxy setups where auth is handled externally).
func WithAnonymousAuth() Option {
	return func(c *Client) { c.anonymous = true }
}

// WithHTTPClient sets a custom HTTP client (for mTLS, service mesh transports, etc.).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithHeader adds a custom HTTP header to all requests.
func WithHeader(key, value string) Option {
	return func(c *Client) { c.headers[key] = value }
}

// WithMaxTokens sets the default max tokens for responses.
func WithMaxTokens(n int) Option {
	return func(c *Client) { c.maxTokens = n }
}

// New creates a new Bedrock Converse API client.
// Credentials are resolved from options first, then environment variables.
func New(opts ...Option) *Client {
	c := &Client{
		maxTokens:  defaultMaxTokens,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}

	// Resolve credentials from environment if not set explicitly
	if c.creds.accessKey == "" && !c.anonymous {
		c.creds.accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		c.creds.secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		c.creds.sessionToken = os.Getenv("AWS_SESSION_TOKEN")
	}
	if c.region == "" {
		c.region = os.Getenv("AWS_REGION")
		if c.region == "" {
			c.region = os.Getenv("AWS_DEFAULT_REGION")
		}
	}
	if c.baseURL == "" && c.region != "" {
		c.baseURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", c.region)
	}

	return c
}

// ModelID returns the Bedrock model identifier.
func (c *Client) ModelID() string {
	return c.model
}

// Generate sends a non-streaming Converse request.
func (c *Client) Generate(ctx context.Context, params graft.GenerateParams) (*graft.GenerateResult, error) {
	body, err := c.buildRequestBody(params)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, body, "converse")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bedrock: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, graft.NewProviderError(resp.StatusCode, "bedrock", respBytes)
	}

	var apiResp converseResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("bedrock: unmarshal response: %w", err)
	}

	msg := parseResponseMessage(apiResp.Output.Message)

	return &graft.GenerateResult{
		Message: msg,
		Usage: graft.Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
		},
	}, nil
}

// buildRequestBody assembles the Converse API request JSON.
func (c *Client) buildRequestBody(params graft.GenerateParams) ([]byte, error) {
	system, messages := convertMessages(params.Messages)

	req := converseRequest{
		Messages: messages,
		System:   system,
	}

	// Inference config
	maxTokens := c.maxTokens
	if params.MaxTokens != nil {
		maxTokens = *params.MaxTokens
	}
	req.InferenceConfig = &inferenceConfig{
		MaxTokens:   maxTokens,
		Temperature: params.Temperature,
	}
	if len(params.Stop) > 0 {
		req.InferenceConfig.StopSequences = params.Stop
	}

	// Tool config
	tools := convertTools(params.Tools)
	if len(tools) > 0 {
		req.ToolConfig = &toolConfig{
			Tools:      tools,
			ToolChoice: convertToolChoice(params.ToolChoice),
		}
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("bedrock: marshal request: %w", err)
	}
	return b, nil
}

// doRequest builds and executes an HTTP POST to the Bedrock endpoint.
func (c *Client) doRequest(ctx context.Context, body []byte, method string) (*http.Response, error) {
	url := bedrockEndpoint(c.baseURL, c.model, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("bedrock: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	if !c.anonymous && c.creds.accessKey != "" {
		signRequest(req, body, c.creds, c.region)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bedrock: http request: %w", err)
	}
	return resp, nil
}

// Ensure Client satisfies graft.LanguageModel.
var _ graft.LanguageModel = (*Client)(nil)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./provider/bedrock/ -v -run "TestGenerate|TestModelID|TestNew"`
Expected: All 8 tests PASS

- [ ] **Step 5: Commit**

```bash
git add provider/bedrock/bedrock.go provider/bedrock/bedrock_test.go
git commit -m "feat(bedrock): add Client with Generate() and options"
```

---

### Task 5: Stream() Implementation

**Files:**
- Create: `provider/bedrock/stream.go`
- Modify: `provider/bedrock/bedrock_test.go`

- [ ] **Step 1: Write the failing test for Stream**

Append to `provider/bedrock/bedrock_test.go`:

```go
func TestStream_TextDeltas(t *testing.T) {
	// Build a sequence of Event Stream frames simulating a Bedrock response
	var streamBody bytes.Buffer
	streamBody.Write(encodeEventStreamFrame("messageStart", []byte(`{"role":"assistant"}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockStart", []byte(`{"contentBlockIndex":0,"start":{}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockDelta", []byte(`{"contentBlockIndex":0,"delta":{"text":"Hello"}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockDelta", []byte(`{"contentBlockIndex":0,"delta":{"text":" world"}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockStop", []byte(`{"contentBlockIndex":0}`)))
	streamBody.Write(encodeEventStreamFrame("messageStop", []byte(`{"stopReason":"end_turn"}`)))
	streamBody.Write(encodeEventStreamFrame("metadata", []byte(`{"usage":{"inputTokens":10,"outputTokens":5}}`)))

	responseBytes := streamBody.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/converse-stream") {
			t.Errorf("expected /converse-stream in path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		w.Write(responseBytes)
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	ch, err := client.Stream(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{
			{Role: graft.RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	var textParts []string
	var lastUsage *graft.Usage
	for chunk := range ch {
		if chunk.Delta.Type == graft.EventTextDelta {
			if text, ok := chunk.Delta.Data.(string); ok {
				textParts = append(textParts, text)
			}
		}
		if chunk.Usage != nil {
			lastUsage = chunk.Usage
		}
	}

	fullText := strings.Join(textParts, "")
	if fullText != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", fullText)
	}
	if lastUsage == nil {
		t.Fatal("expected usage in stream")
	}
	if lastUsage.PromptTokens != 10 || lastUsage.CompletionTokens != 5 {
		t.Errorf("expected 10/5 tokens, got %d/%d", lastUsage.PromptTokens, lastUsage.CompletionTokens)
	}
}

func TestStream_WithToolCall(t *testing.T) {
	var streamBody bytes.Buffer
	streamBody.Write(encodeEventStreamFrame("messageStart", []byte(`{"role":"assistant"}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockStart", []byte(`{"contentBlockIndex":0,"start":{}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockDelta", []byte(`{"contentBlockIndex":0,"delta":{"text":"Checking..."}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockStop", []byte(`{"contentBlockIndex":0}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockStart", []byte(`{"contentBlockIndex":1,"start":{"toolUse":{"toolUseId":"tu_123","name":"get_weather"}}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockDelta", []byte(`{"contentBlockIndex":1,"delta":{"toolUse":{"input":"{\"city\":"}}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockDelta", []byte(`{"contentBlockIndex":1,"delta":{"toolUse":{"input":"\"NYC\"}"}}}`)))
	streamBody.Write(encodeEventStreamFrame("contentBlockStop", []byte(`{"contentBlockIndex":1}`)))
	streamBody.Write(encodeEventStreamFrame("messageStop", []byte(`{"stopReason":"tool_use"}`)))
	streamBody.Write(encodeEventStreamFrame("metadata", []byte(`{"usage":{"inputTokens":20,"outputTokens":15}}`)))

	responseBytes := streamBody.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		w.Write(responseBytes)
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	ch, err := client.Stream(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Weather?"}},
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	var events []graft.EventType
	for chunk := range ch {
		events = append(events, chunk.Delta.Type)
	}

	// Should see: text_delta, tool_call_start, tool_call_delta(s), tool_call_done, message_done
	hasToolStart := false
	hasToolDelta := false
	hasToolDone := false
	for _, e := range events {
		switch e {
		case graft.EventToolCallStart:
			hasToolStart = true
		case graft.EventToolCallDelta:
			hasToolDelta = true
		case graft.EventToolCallDone:
			hasToolDone = true
		}
	}
	if !hasToolStart {
		t.Error("expected tool_call_start event")
	}
	if !hasToolDelta {
		t.Error("expected tool_call_delta event")
	}
	if !hasToolDone {
		t.Error("expected tool_call_done event")
	}
}

func TestStream_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Access denied"}`))
	}))
	defer srv.Close()

	client := New(
		WithRegion("us-east-1"),
		WithModel("test-model"),
		WithBaseURL(srv.URL),
		WithAnonymousAuth(),
	)

	_, err := client.Stream(context.Background(), graft.GenerateParams{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./provider/bedrock/ -v -run TestStream`
Expected: FAIL — `Stream` method not found on `*Client`

- [ ] **Step 3: Write the Stream implementation**

Create `provider/bedrock/stream.go`:

```go
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/delavalom/graft"
)

// Stream sends a streaming ConverseStream request and returns a channel of StreamChunks.
func (c *Client) Stream(ctx context.Context, params graft.GenerateParams) (<-chan graft.StreamChunk, error) {
	body, err := c.buildRequestBody(params)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, body, "converse-stream")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		respBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, graft.NewProviderError(resp.StatusCode, "bedrock", respBytes)
	}

	ch := make(chan graft.StreamChunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		decoder := newEventStreamDecoder(resp.Body)

		// Track which content blocks are tool calls
		toolBlocks := make(map[int]*streamToolUseStart)

		for {
			eventType, payload, err := decoder.readEvent()
			if err == io.EOF {
				break
			}
			if err != nil {
				select {
				case ch <- graft.StreamChunk{
					Delta: graft.StreamEvent{
						Type:      graft.EventError,
						Data:      fmt.Sprintf("bedrock: event stream decode: %v", err),
						Timestamp: time.Now(),
					},
				}:
				case <-ctx.Done():
				}
				return
			}

			chunks := c.handleStreamEvent(eventType, payload, toolBlocks)
			for _, chunk := range chunks {
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// handleStreamEvent processes a single event stream event and returns zero or more StreamChunks.
func (c *Client) handleStreamEvent(eventType string, payload []byte, toolBlocks map[int]*streamToolUseStart) []graft.StreamChunk {
	switch eventType {
	case "messageStart":
		// No chunk emitted for message start
		return nil

	case "contentBlockStart":
		var evt streamContentBlockStart
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		if evt.Start != nil && evt.Start.ToolUse != nil {
			toolBlocks[evt.ContentBlockIndex] = evt.Start.ToolUse
			return []graft.StreamChunk{{
				Delta: graft.StreamEvent{
					Type: graft.EventToolCallStart,
					Data: map[string]string{
						"id":   evt.Start.ToolUse.ToolUseID,
						"name": evt.Start.ToolUse.Name,
					},
					Timestamp: time.Now(),
				},
			}}
		}
		return nil

	case "contentBlockDelta":
		var evt streamContentBlockDelta
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		// Text delta
		if evt.Delta.Text != "" {
			return []graft.StreamChunk{{
				Delta: graft.StreamEvent{
					Type:      graft.EventTextDelta,
					Data:      evt.Delta.Text,
					Timestamp: time.Now(),
				},
			}}
		}
		// Tool use input delta
		if evt.Delta.ToolUse != nil {
			return []graft.StreamChunk{{
				Delta: graft.StreamEvent{
					Type:      graft.EventToolCallDelta,
					Data:      evt.Delta.ToolUse.Input,
					Timestamp: time.Now(),
				},
			}}
		}
		return nil

	case "contentBlockStop":
		var evt streamContentBlockStop
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		// If this was a tool block, emit tool_call_done
		if _, isToolBlock := toolBlocks[evt.ContentBlockIndex]; isToolBlock {
			delete(toolBlocks, evt.ContentBlockIndex)
			return []graft.StreamChunk{{
				Delta: graft.StreamEvent{
					Type:      graft.EventToolCallDone,
					Timestamp: time.Now(),
				},
			}}
		}
		return nil

	case "messageStop":
		return []graft.StreamChunk{{
			Delta: graft.StreamEvent{
				Type:      graft.EventMessageDone,
				Timestamp: time.Now(),
			},
		}}

	case "metadata":
		var evt streamMetadata
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil
		}
		usage := graft.Usage{
			PromptTokens:     evt.Usage.InputTokens,
			CompletionTokens: evt.Usage.OutputTokens,
		}
		return []graft.StreamChunk{{
			Delta: graft.StreamEvent{
				Type:      graft.EventDone,
				Timestamp: time.Now(),
			},
			Usage: &usage,
		}}

	default:
		return nil
	}
}
```

- [ ] **Step 4: Run all tests to verify they pass**

Run: `go test ./provider/bedrock/ -v`
Expected: All tests PASS (message conversion, SigV4, event stream, Generate, Stream)

- [ ] **Step 5: Commit**

```bash
git add provider/bedrock/stream.go provider/bedrock/bedrock_test.go
git commit -m "feat(bedrock): add Stream() with AWS Event Stream decoding"
```

---

### Task 6: Example and Build Verification

**Files:**
- Create: `examples/bedrock/main.go`

- [ ] **Step 1: Create the Bedrock example**

Create `examples/bedrock/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/bedrock"
)

func main() {
	// Uses AWS credentials from environment variables:
	//   AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
	//
	// For proxy/service-mesh deployments, use:
	//   bedrock.WithBaseURL("http://127.0.0.1:19193"),
	//   bedrock.WithAnonymousAuth(),
	//   bedrock.WithHeader("X-Project", "my-project"),

	model := bedrock.New(
		bedrock.WithRegion("us-east-1"),
		bedrock.WithModel("anthropic.claude-sonnet-4-20250514-v1:0"),
	)

	greetTool := graft.NewTool("greet", "Greet someone by name",
		func(ctx context.Context, p struct {
			Name string `json:"name" description:"The person's name"`
		}) (string, error) {
			return fmt.Sprintf("Hello, %s! Welcome to Graft on Bedrock.", p.Name), nil
		},
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant. Use the greet tool when asked to greet someone."),
		graft.WithTools(greetTool),
	)

	runner := graft.NewDefaultRunner(model)
	result, err := runner.Run(context.Background(), agent, []graft.Message{
		{Role: graft.RoleUser, Content: "Please greet Alice"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.LastAssistantText())
}
```

- [ ] **Step 2: Build the entire project**

Run: `go build ./...`
Expected: Clean build, no errors

- [ ] **Step 3: Build the example specifically**

Run: `go build -o /dev/null ./examples/bedrock/`
Expected: Clean build

- [ ] **Step 4: Run go vet**

Run: `go vet ./provider/bedrock/`
Expected: No issues

- [ ] **Step 5: Run all provider tests to ensure nothing is broken**

Run: `go test ./provider/bedrock/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add examples/bedrock/main.go
git commit -m "feat(bedrock): add example"
```

---

### Task 7: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add Bedrock to the packages table**

In the packages table in `README.md`, add after the `provider/google` row:

```markdown
| `provider/bedrock` | AWS Bedrock (Converse API) — Claude, Titan, Llama, Mistral |
```

- [ ] **Step 2: Add Bedrock to the examples table**

In the examples table, add:

```markdown
| [bedrock](examples/bedrock/) | AWS Bedrock with Converse API |
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add Bedrock provider to README"
```
