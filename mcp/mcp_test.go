package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/delavalom/graft"
)

func TestProtocolSerialization(t *testing.T) {
	req, err := newRequest(1, "tools/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Request
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", parsed.JSONRPC)
	}
	if parsed.Method != "tools/list" {
		t.Errorf("expected method tools/list, got %s", parsed.Method)
	}
	if parsed.ID != 1 {
		t.Errorf("expected ID 1, got %d", parsed.ID)
	}
}

func TestResponseSerialization(t *testing.T) {
	resp, err := newResponse(42, map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != 42 {
		t.Errorf("expected ID 42, got %d", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}
}

func TestErrorResponse(t *testing.T) {
	resp := newErrorResponse(1, CodeMethodNotFound, "not found")
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("expected code %d, got %d", CodeMethodNotFound, resp.Error.Code)
	}
}

// testTool is a simple tool for testing.
type testTool struct{}

func (t *testTool) Name() string             { return "echo" }
func (t *testTool) Description() string       { return "Echoes input back" }
func (t *testTool) Schema() json.RawMessage   { return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`) }
func (t *testTool) Execute(_ context.Context, params json.RawMessage) (any, error) {
	var p struct{ Text string `json:"text"` }
	json.Unmarshal(params, &p)
	return "echo: " + p.Text, nil
}

func startTestServer(t *testing.T) (*Client, *Server) {
	t.Helper()
	clientTransport, serverTransport := NewInMemoryTransport()

	server := NewServer("test-server", "1.0.0")
	server.AddTool(&testTool{})
	server.AddResource(ResourceInfo{
		URI:  "file:///test.txt",
		Name: "test.txt",
	}, "hello world")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go server.ServeTransport(ctx, serverTransport)

	client := NewClient(clientTransport, WithServerName("test"))
	t.Cleanup(func() { client.Close() })

	return client, server
}

func TestClientInitialize(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	result, err := client.Initialize(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.ServerInfo.Name != "test-server" {
		t.Errorf("expected server name test-server, got %s", result.ServerInfo.Name)
	}
}

func TestClientListTools(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Errorf("expected tool name echo, got %s", tools[0].Name)
	}
}

func TestClientCallTool(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	args, _ := json.Marshal(map[string]string{"text": "hello"})
	result, err := client.CallTool(ctx, "echo", args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected no error in result")
	}
	if len(result.Content) != 1 || result.Content[0].Text != "echo: hello" {
		t.Errorf("unexpected result: %+v", result.Content)
	}
}

func TestClientAsTools(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	tools, err := client.AsTools(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 graft tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name() != "mcp__test__echo" {
		t.Errorf("expected name mcp__test__echo, got %s", tool.Name())
	}
	if tool.Description() != "Echoes input back" {
		t.Errorf("expected description 'Echoes input back', got %s", tool.Description())
	}

	args, _ := json.Marshal(map[string]string{"text": "world"})
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	if result != "echo: world" {
		t.Errorf("expected 'echo: world', got %v", result)
	}
}

func TestClientListResources(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	resources, err := client.ListResources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].URI != "file:///test.txt" {
		t.Errorf("expected URI file:///test.txt, got %s", resources[0].URI)
	}
}

func TestClientReadResource(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	result, err := client.ReadResource(ctx, "file:///test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Contents) != 1 || result.Contents[0].Text != "hello world" {
		t.Errorf("unexpected contents: %+v", result.Contents)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	_, err := client.call(ctx, "unknown/method", nil)
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestServerUnknownTool(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	args := json.RawMessage(`{}`)
	_, err := client.CallTool(ctx, "nonexistent", args)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestInMemoryTransport(t *testing.T) {
	a, b := NewInMemoryTransport()
	defer a.Close()

	ctx := context.Background()
	msg := []byte(`{"test":"data"}`)

	if err := a.Send(ctx, msg); err != nil {
		t.Fatal(err)
	}

	received := <-b.Receive()
	if string(received) != string(msg) {
		t.Errorf("expected %s, got %s", msg, received)
	}
}

// Ensure mcpTool satisfies graft.Tool interface.
var _ graft.Tool = (*mcpTool)(nil)
