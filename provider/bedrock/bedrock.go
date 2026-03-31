// Package bedrock implements the AWS Bedrock Converse API provider for the graft framework.
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
	return func(c *Client) {
		c.region = region
	}
}

// WithModel sets the Bedrock model ID.
func WithModel(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

// WithBaseURL overrides the endpoint URL (useful for proxies or local testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithCredentials sets IAM access key and secret key.
func WithCredentials(accessKey, secretKey string) Option {
	return func(c *Client) {
		c.creds.accessKey = accessKey
		c.creds.secretKey = secretKey
	}
}

// WithSessionToken sets an STS session token.
func WithSessionToken(token string) Option {
	return func(c *Client) {
		c.creds.sessionToken = token
	}
}

// WithAnonymousAuth disables SigV4 signing.
func WithAnonymousAuth() Option {
	return func(c *Client) {
		c.anonymous = true
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithHeader adds a custom header to all requests.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithMaxTokens sets the default maximum number of tokens to generate.
func WithMaxTokens(n int) Option {
	return func(c *Client) {
		c.maxTokens = n
	}
}

// New creates a new Bedrock client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		maxTokens:  defaultMaxTokens,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}

	// Resolve credentials from environment if not set and not anonymous.
	if !c.anonymous && c.creds.accessKey == "" {
		c.creds.accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		c.creds.secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		c.creds.sessionToken = os.Getenv("AWS_SESSION_TOKEN")
	}

	// Resolve region from environment if not set.
	if c.region == "" {
		c.region = os.Getenv("AWS_REGION")
		if c.region == "" {
			c.region = os.Getenv("AWS_DEFAULT_REGION")
		}
	}

	// Build default base URL from region if not explicitly set.
	if c.baseURL == "" && c.region != "" {
		c.baseURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", c.region)
	}

	return c
}

// Ensure Client implements graft.LanguageModel.
var _ graft.LanguageModel = (*Client)(nil)

// ModelID returns the configured model identifier.
func (c *Client) ModelID() string {
	return c.model
}

// Generate sends a non-streaming generation request to the Bedrock Converse API.
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

// Stream is implemented in stream.go.

// buildRequestBody assembles the JSON request body for the Converse API.
func (c *Client) buildRequestBody(params graft.GenerateParams) ([]byte, error) {
	system, messages := convertMessages(params.Messages)

	req := converseRequest{
		Messages: messages,
		System:   system,
	}

	// Inference config.
	maxTokens := c.maxTokens
	if params.MaxTokens != nil {
		maxTokens = *params.MaxTokens
	}
	req.InferenceConfig = &inferenceConfig{
		MaxTokens:   &maxTokens,
		Temperature: params.Temperature,
	}
	if len(params.Stop) > 0 {
		req.InferenceConfig.StopSequences = params.Stop
	}

	// Tool config.
	if len(params.Tools) > 0 {
		tc := convertToolChoice(params.ToolChoice)
		// If ToolChoiceNone, omit tool config entirely.
		if tc != nil {
			req.ToolConfig = &toolConfig{
				Tools:      convertTools(params.Tools),
				ToolChoice: tc,
			}
		}
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("bedrock: marshal request: %w", err)
	}
	return b, nil
}

// doRequest builds and executes an HTTP POST request to the Bedrock API.
func (c *Client) doRequest(ctx context.Context, body []byte, method string) (*http.Response, error) {
	url := fmt.Sprintf("%s/model/%s/%s", c.baseURL, c.model, method)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("bedrock: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers.
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Sign request with SigV4 if not anonymous and credentials are present.
	if !c.anonymous && c.creds.accessKey != "" {
		signRequest(req, body, c.creds, c.region)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bedrock: http request: %w", err)
	}
	return resp, nil
}
