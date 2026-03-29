package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/delavalom/graft"
)

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	serverName string
	timeout    time.Duration
}

// WithServerName sets the server name used in tool naming (mcp__<serverName>__<toolName>).
func WithServerName(name string) ClientOption {
	return func(c *clientConfig) { c.serverName = name }
}

// WithTimeout sets the default timeout for RPC calls.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) { c.timeout = d }
}

// Client connects to an MCP server and discovers/invokes tools.
type Client struct {
	transport Transport
	cfg       clientConfig
	nextID    atomic.Int64
	pending   map[int64]chan *Response
	mu        sync.Mutex
	done      chan struct{}
}

// NewClient creates a new MCP client over the given transport.
func NewClient(transport Transport, opts ...ClientOption) *Client {
	cfg := clientConfig{
		serverName: "default",
		timeout:    30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	c := &Client{
		transport: transport,
		cfg:       cfg,
		pending:   make(map[int64]chan *Response),
		done:      make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func (c *Client) readLoop() {
	for {
		select {
		case data, ok := <-c.transport.Receive():
			if !ok {
				return
			}
			var resp Response
			if err := json.Unmarshal(data, &resp); err != nil {
				continue
			}
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.mu.Unlock()
			if ok {
				ch <- &resp
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) call(ctx context.Context, method string, params any) (*Response, error) {
	id := c.nextID.Add(1)
	req, err := newRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Response, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := sendJSON(ctx, c.transport, req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Initialize performs the MCP initialization handshake.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    &ClientCapabilities{},
		ClientInfo:      Implementation{Name: "graft", Version: "1.0.0"},
	}
	resp, err := c.call(ctx, MethodInitialize, params)
	if err != nil {
		return nil, fmt.Errorf("mcp: initialize: %w", err)
	}
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: initialize unmarshal: %w", err)
	}

	// Send initialized notification
	notif := Notification{JSONRPC: jsonRPCVersion, Method: MethodInitialized}
	_ = sendJSON(ctx, c.transport, notif)

	return &result, nil
}

// ListTools discovers tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	resp, err := c.call(ctx, MethodToolsList, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: list tools: %w", err)
	}
	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: list tools unmarshal: %w", err)
	}
	return result.Tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (*ToolCallResult, error) {
	params := ToolCallParams{Name: name, Arguments: args}
	resp, err := c.call(ctx, MethodToolsCall, params)
	if err != nil {
		return nil, fmt.Errorf("mcp: call tool %s: %w", name, err)
	}
	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: call tool unmarshal: %w", err)
	}
	return &result, nil
}

// ListResources discovers resources from the MCP server.
func (c *Client) ListResources(ctx context.Context) ([]ResourceInfo, error) {
	resp, err := c.call(ctx, MethodResourcesList, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: list resources: %w", err)
	}
	var result ListResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: list resources unmarshal: %w", err)
	}
	return result.Resources, nil
}

// ReadResource reads the contents of a resource from the MCP server.
func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	params := ReadResourceParams{URI: uri}
	resp, err := c.call(ctx, MethodResourcesRead, params)
	if err != nil {
		return nil, fmt.Errorf("mcp: read resource: %w", err)
	}
	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: read resource unmarshal: %w", err)
	}
	return &result, nil
}

// AsTools converts MCP server tools into graft.Tool instances.
// Tool naming convention: mcp__<serverName>__<toolName>.
func (c *Client) AsTools(ctx context.Context) ([]graft.Tool, error) {
	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]graft.Tool, len(tools))
	for i, t := range tools {
		result[i] = &mcpTool{
			client:     c,
			serverName: c.cfg.serverName,
			info:       t,
		}
	}
	return result, nil
}

// Close shuts down the client and its transport.
func (c *Client) Close() error {
	close(c.done)
	return c.transport.Close()
}

// mcpTool wraps an MCP tool as a graft.Tool.
type mcpTool struct {
	client     *Client
	serverName string
	info       ToolInfo
}

func (t *mcpTool) Name() string {
	return "mcp__" + t.serverName + "__" + t.info.Name
}

func (t *mcpTool) Description() string { return t.info.Description }
func (t *mcpTool) Schema() json.RawMessage { return t.info.InputSchema }

func (t *mcpTool) Execute(ctx context.Context, params json.RawMessage) (any, error) {
	result, err := t.client.CallTool(ctx, t.info.Name, params)
	if err != nil {
		return nil, graft.NewAgentError(graft.ErrToolExecution, fmt.Sprintf("mcp tool %s failed", t.info.Name), err)
	}
	if result.IsError {
		text := ""
		for _, c := range result.Content {
			if c.Type == "text" {
				text += c.Text
			}
		}
		return nil, graft.NewAgentError(graft.ErrToolExecution, text, nil)
	}
	// Return text content
	var texts []string
	for _, c := range result.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}
	if len(texts) == 1 {
		return texts[0], nil
	}
	return texts, nil
}
