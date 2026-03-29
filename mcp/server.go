package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/delavalom/graft"
)

// ServerOption configures a Server.
type ServerOption func(*serverConfig)

type serverConfig struct{}

// Server exposes graft tools as an MCP server.
type Server struct {
	name    string
	version string
	tools   map[string]graft.Tool
	resources map[string]ResourceInfo
	resourceData map[string]string // uri → text content
	mu      sync.RWMutex
}

// NewServer creates a new MCP server.
func NewServer(name, version string, opts ...ServerOption) *Server {
	return &Server{
		name:         name,
		version:      version,
		tools:        make(map[string]graft.Tool),
		resources:    make(map[string]ResourceInfo),
		resourceData: make(map[string]string),
	}
}

// AddTool registers a graft tool for exposure via MCP.
func (s *Server) AddTool(tool graft.Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name()] = tool
}

// AddTools registers multiple graft tools.
func (s *Server) AddTools(tools []graft.Tool) {
	for _, t := range tools {
		s.AddTool(t)
	}
}

// AddResource registers a text resource.
func (s *Server) AddResource(info ResourceInfo, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[info.URI] = info
	s.resourceData[info.URI] = content
}

// handleRequest processes a single JSON-RPC request and returns a response.
func (s *Server) handleRequest(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodToolsList:
		return s.handleToolsList(req)
	case MethodToolsCall:
		return s.handleToolsCall(ctx, req)
	case MethodResourcesList:
		return s.handleResourcesList(req)
	case MethodResourcesRead:
		return s.handleResourcesRead(req)
	case MethodPing:
		resp, _ := newResponse(req.ID, map[string]any{})
		return resp
	default:
		return newErrorResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools:     &ToolCapability{},
			Resources: &ResourceCapability{},
		},
		ServerInfo: Implementation{Name: s.name, Version: s.version},
	}
	resp, _ := newResponse(req.ID, result)
	return resp
}

func (s *Server) handleToolsList(req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]ToolInfo, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}
	resp, _ := newResponse(req.ID, ListToolsResult{Tools: tools})
	return resp
}

func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newErrorResponse(req.ID, CodeInvalidParams, "invalid params")
	}

	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()
	if !ok {
		return newErrorResponse(req.ID, CodeInvalidParams, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	result, err := tool.Execute(ctx, params.Arguments)
	if err != nil {
		tcr := ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			IsError: true,
		}
		resp, _ := newResponse(req.ID, tcr)
		return resp
	}

	var text string
	switch v := result.(type) {
	case string:
		text = v
	default:
		b, _ := json.Marshal(v)
		text = string(b)
	}
	tcr := ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
	resp, _ := newResponse(req.ID, tcr)
	return resp
}

func (s *Server) handleResourcesList(req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]ResourceInfo, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}
	resp, _ := newResponse(req.ID, ListResourcesResult{Resources: resources})
	return resp
}

func (s *Server) handleResourcesRead(req *Request) *Response {
	var params ReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newErrorResponse(req.ID, CodeInvalidParams, "invalid params")
	}

	s.mu.RLock()
	content, ok := s.resourceData[params.URI]
	s.mu.RUnlock()
	if !ok {
		return newErrorResponse(req.ID, CodeInvalidParams, fmt.Sprintf("unknown resource: %s", params.URI))
	}

	result := ReadResourceResult{
		Contents: []ResourceContents{{URI: params.URI, Text: content, MimeType: "text/plain"}},
	}
	resp, _ := newResponse(req.ID, result)
	return resp
}

// ServeTransport runs the server over a generic transport.
func (s *Server) ServeTransport(ctx context.Context, transport Transport) error {
	for {
		select {
		case data, ok := <-transport.Receive():
			if !ok {
				return nil
			}
			// Try to parse as request
			var req Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			// Skip notifications (no ID)
			if req.Method == MethodInitialized {
				continue
			}
			resp := s.handleRequest(ctx, &req)
			if resp != nil {
				_ = sendJSON(ctx, transport, resp)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ServeHTTP returns an http.Handler that serves MCP over HTTP POST.
func (s *Server) ServeHTTP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		resp := s.handleRequest(r.Context(), &req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}
