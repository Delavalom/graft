# Bedrock Provider Design

**Date:** 2026-03-30
**Status:** Proposed
**Package:** `provider/bedrock`

## Summary

Add an AWS Bedrock provider to graft using the Converse API. Zero external SDK dependencies — SigV4 signing and AWS Event Stream binary decoding implemented with Go stdlib only (`crypto/hmac`, `crypto/sha256`, `encoding/binary`, `hash/crc32`). Supports both direct AWS auth and proxy/service-mesh deployments.

## API Surface

### Constructor and Options

```go
package bedrock

// Client is an AWS Bedrock Converse API provider.
type Client struct { ... }

func New(opts ...Option) *Client

// Required
func WithRegion(region string) Option          // AWS region (e.g., "us-east-1")
func WithModel(model string) Option            // Bedrock model ID

// Auth — pick one approach
func WithCredentials(accessKey, secretKey string) Option          // Static IAM credentials
func WithSessionToken(token string) Option                        // For STS/assumed roles
func WithAnonymousAuth() Option                                   // Skip SigV4 (proxy handles auth)

// Network
func WithBaseURL(url string) Option            // Override endpoint (for proxies)
func WithHTTPClient(c *http.Client) Option     // Custom transport (mTLS, service mesh)
func WithHeader(key, value string) Option      // Custom headers (routing, project IDs)

// Defaults
func WithMaxTokens(n int) Option               // Default max tokens (default: 4096)
```

### Credential Resolution Order

1. Explicit `WithCredentials()` / `WithSessionToken()`
2. Environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_REGION`
3. If neither is set and `WithAnonymousAuth()` is not set, return error on first request

### Usage Examples

**Direct AWS auth:**
```go
model := bedrock.New(
    bedrock.WithRegion("us-east-1"),
    bedrock.WithModel("anthropic.claude-sonnet-4-20250514-v1:0"),
    bedrock.WithCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY")),
)
```

**Behind a proxy/service mesh (like the Pinterest pattern):**
```go
model := bedrock.New(
    bedrock.WithRegion("us-east-1"),
    bedrock.WithModel("anthropic.claude-sonnet-4-20250514-v1:0"),
    bedrock.WithBaseURL("http://127.0.0.1:19193"),
    bedrock.WithAnonymousAuth(),
    bedrock.WithHeader("X-Forwarded-Project", "my-project"),
    bedrock.WithHTTPClient(&http.Client{
        Transport: &serviceMeshTransport{base: http.DefaultTransport},
    }),
)
```

**Environment variables (zero-config):**
```go
// Reads AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN, AWS_REGION
model := bedrock.New(
    bedrock.WithModel("anthropic.claude-sonnet-4-20250514-v1:0"),
)
```

## Architecture

### File Layout

```
provider/bedrock/
    bedrock.go       # Client, New(), Generate(), ModelID(), options
    stream.go        # Stream() implementation
    sigv4.go         # AWS Signature V4 signing (stdlib only)
    eventstream.go   # AWS Event Stream binary decoder
    types.go         # Converse API request/response types
    bedrock_test.go  # Tests
```

### LanguageModel Interface Implementation

The `Client` implements `graft.LanguageModel`:

- **`Generate()`** → `POST /model/{modelId}/converse`
- **`Stream()`** → `POST /model/{modelId}/converse-stream` with AWS Event Stream binary decoding
- **`ModelID()`** → returns the configured Bedrock model ID string

### Request Flow

```
graft.GenerateParams
  → convertMessages()     # graft.Message → Converse message format
  → convertTools()        # graft.ToolDefinition → toolSpec format
  → buildRequest()        # Assemble Converse API JSON body
  → signRequest()         # SigV4 signing (unless anonymous)
  → HTTP POST
  → parseResponse()       # Converse response → graft.GenerateResult
```

## Type Mappings

### Converse API Request Structure

```go
type converseRequest struct {
    Messages       []converseMessage      `json:"messages"`
    System         []systemBlock          `json:"system,omitempty"`
    InferenceConfig *inferenceConfig      `json:"inferenceConfig,omitempty"`
    ToolConfig     *toolConfig            `json:"toolConfig,omitempty"`
}

type systemBlock struct {
    Text string `json:"text"`
}

type converseMessage struct {
    Role    string         `json:"role"`    // "user" or "assistant"
    Content []contentBlock `json:"content"`
}

// contentBlock is a union — only one field is set per block
type contentBlock struct {
    Text       string          `json:"text,omitempty"`
    ToolUse    *toolUseBlock   `json:"toolUse,omitempty"`
    ToolResult *toolResultBlock `json:"toolResult,omitempty"`
}

type toolUseBlock struct {
    ToolUseId string         `json:"toolUseId"`
    Name      string         `json:"name"`
    Input     map[string]any `json:"input"`
}

type toolResultBlock struct {
    ToolUseId string         `json:"toolUseId"`
    Content   []contentBlock `json:"content"`
    Status    string         `json:"status"` // "success" or "error"
}

type inferenceConfig struct {
    MaxTokens     int      `json:"maxTokens,omitempty"`
    Temperature   *float64 `json:"temperature,omitempty"`
    TopP          *float64 `json:"topP,omitempty"`
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
```

### Converse API Response Structure

```go
type converseResponse struct {
    Output   converseOutput `json:"output"`
    StopReason string      `json:"stopReason"`
    Usage    converseUsage  `json:"usage"`
}

type converseOutput struct {
    Message converseMessage `json:"message"`
}

type converseUsage struct {
    InputTokens  int `json:"inputTokens"`
    OutputTokens int `json:"outputTokens"`
}
```

### Message Conversion: graft → Converse

| graft.Message | Converse |
|--------------|----------|
| `Role: "system"` | Extracted to top-level `system: [{"text": "..."}]` |
| `Role: "user"` | `{"role": "user", "content": [{"text": "..."}]}` |
| `Role: "assistant"` (text only) | `{"role": "assistant", "content": [{"text": "..."}]}` |
| `Role: "assistant"` (with ToolCalls) | `{"role": "assistant", "content": [{"text": "..."}, {"toolUse": {...}}, ...]}` |
| `Role: "tool"` (ToolResult) | `{"role": "user", "content": [{"toolResult": {...}}]}` |

### Message Conversion: Converse → graft

| Converse Response Content | graft.Message |
|--------------------------|---------------|
| `{"text": "..."}` block | Appended to `Message.Content` |
| `{"toolUse": {...}}` block | Appended to `Message.ToolCalls` as `graft.ToolCall{ID: toolUseId, Name: name, Arguments: json.Marshal(input)}` |

### Tool Definition Conversion: graft → Converse

```
graft.ToolDefinition{Name, Description, Schema}
  → toolDef{ToolSpec{Name, Description, InputSchema{JSON: Schema}}}
```

The `Schema` field (`json.RawMessage`) maps directly into `inputSchema.json` — just an extra wrapper level.

### ToolChoice Mapping

| graft.ToolChoice | Converse toolChoice |
|-----------------|---------------------|
| `""` (empty/auto) | `{"auto": {}}` |
| `"any"` | `{"any": {}}` |
| `"tool:get_weather"` | `{"tool": {"name": "get_weather"}}` |

## SigV4 Signing (sigv4.go)

Pure Go implementation using `crypto/hmac`, `crypto/sha256`, `encoding/hex`.

```go
func signRequest(req *http.Request, body []byte, creds credentials, region string)
```

**Algorithm:**
1. Create canonical request (method, path, query, headers, payload hash)
2. Create string to sign (algorithm, timestamp, scope, hash of canonical request)
3. Derive signing key: `HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), "bedrock"), "aws4_request")`
4. Compute signature: `HMAC(signingKey, stringToSign)`
5. Set `Authorization` header with credential scope, signed headers, signature
6. Set `X-Amz-Date` header
7. Set `X-Amz-Security-Token` header if session token is present

**Service name:** `bedrock` (for Bedrock Runtime)

**Signed headers:** `content-type`, `host`, `x-amz-date`, and optionally `x-amz-security-token`

## AWS Event Stream Decoding (eventstream.go)

Binary frame format for ConverseStream responses.

### Frame Structure

```
[total-length:  4 bytes, big-endian uint32]
[headers-length: 4 bytes, big-endian uint32]
[prelude-crc:    4 bytes, CRC-32C of first 8 bytes]
[headers:        variable, headers-length bytes]
[payload:        variable, total-length - headers-length - 16 bytes]
[message-crc:    4 bytes, CRC-32C of entire frame minus last 4 bytes]
```

### Header Format

Each header:
```
[name-length:  1 byte]
[name:         name-length bytes, UTF-8]
[value-type:   1 byte]  // 7 = string
[value-length: 2 bytes, big-endian uint16]
[value:        value-length bytes]
```

Key headers to extract:
- `:event-type` — event name string (e.g., "contentBlockDelta")
- `:message-type` — "event" or "exception"

### Stream Event Types → graft.StreamChunk Mapping

| Bedrock Event | Payload | graft Mapping |
|--------------|---------|---------------|
| `messageStart` | `{"role": "assistant"}` | No chunk emitted |
| `contentBlockStart` | `{"contentBlockIndex": N, "start": {...}}` | `EventToolCallStart` if toolUse present |
| `contentBlockDelta` (text) | `{"delta": {"text": "..."}}` | `EventTextDelta` with text data |
| `contentBlockDelta` (toolUse) | `{"delta": {"toolUse": {"input": "..."}}}` | `EventToolCallDelta` with input fragment |
| `contentBlockStop` | `{"contentBlockIndex": N}` | `EventToolCallDone` if was tool block |
| `messageStop` | `{"stopReason": "..."}` | `EventMessageDone` |
| `metadata` | `{"usage": {...}}` | Final `StreamChunk` with `Usage` |

### CRC Validation

Uses CRC-32C (Castagnoli polynomial): `crc32.MakeTable(crc32.Castagnoli)`.

Both prelude CRC and message CRC are validated. If either fails, return a decode error.

### Decoder Interface

```go
type eventStreamDecoder struct {
    reader io.Reader
    crcTab *crc32.Table
}

func newEventStreamDecoder(r io.Reader) *eventStreamDecoder

// readEvent returns the next event type and JSON payload.
// Returns io.EOF when the stream ends.
func (d *eventStreamDecoder) readEvent() (eventType string, payload []byte, err error)
```

## Error Handling

Uses the same pattern as other providers:

```go
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, graft.NewProviderError(resp.StatusCode, "bedrock", body)
}
```

Bedrock error responses are JSON:
```json
{"message": "The model is not accessible. ..."}
```

The existing `NewProviderError` handles status code mapping (401/403 → auth error, 429 → rate limit, etc.).

## Testing Strategy

- **Unit tests** with `httptest.Server` mocking Converse responses — same pattern as existing providers
- **SigV4 tests** with known test vectors (AWS publishes reference inputs/outputs)
- **Event Stream decoder tests** with pre-built binary frames
- **Integration test** (skipped without `AWS_ACCESS_KEY_ID`) hitting real Bedrock
- **No mocked provider tests for integration** — consistent with project conventions

## What's NOT in Scope

- AWS credentials file (`~/.aws/credentials`) parsing — can be added later
- AWS SSO / Identity Center — requires browser-based OAuth flow
- EC2 instance metadata (IMDS) credential fetching — can be added later
- Cross-region inference profiles — just use the cross-region model ID string
- Bedrock Guardrails (AWS-managed guardrails) — graft has its own guardrail system
- InvokeModel API — Converse API covers all use cases with a unified format
