package graft

import (
	"context"
	"encoding/json"
	"testing"
)

type SearchParams struct {
	Query string `json:"query" description:"Search query"`
	Limit int    `json:"limit,omitempty" description:"Max results"`
}

type SearchResult struct {
	Results []string `json:"results"`
}

func TestNewToolName(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{Results: []string{"r1"}}, nil
		},
	)
	if got := tool.Name(); got != "search" {
		t.Errorf("Name() = %q, want %q", got, "search")
	}
}

func TestNewToolDescription(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{}, nil
		},
	)
	if got := tool.Description(); got != "Search the web" {
		t.Errorf("Description() = %q, want %q", got, "Search the web")
	}
}

func TestNewToolSchema(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{}, nil
		},
	)
	schema := tool.Schema()
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
	props := m["properties"].(map[string]any)
	if _, ok := props["query"]; !ok {
		t.Error("schema missing 'query' property")
	}
}

func TestNewToolExecute(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{Results: []string{"result for: " + p.Query}}, nil
		},
	)
	input := json.RawMessage(`{"query":"golang","limit":5}`)
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	sr, ok := result.(SearchResult)
	if !ok {
		t.Fatalf("result type = %T, want SearchResult", result)
	}
	if len(sr.Results) != 1 || sr.Results[0] != "result for: golang" {
		t.Errorf("Results = %v, want [result for: golang]", sr.Results)
	}
}

func TestNewToolExecuteInvalidJSON(t *testing.T) {
	tool := NewTool("search", "Search the web",
		func(ctx context.Context, p SearchParams) (SearchResult, error) {
			return SearchResult{}, nil
		},
	)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestToolDefinitionFromTool(t *testing.T) {
	tool := NewTool("greet", "Greet someone",
		func(ctx context.Context, p struct{ Name string }) (string, error) {
			return "hi " + p.Name, nil
		},
	)
	def := ToolDefFromTool(tool)
	if def.Name != "greet" {
		t.Errorf("Name = %q, want %q", def.Name, "greet")
	}
	if def.Description != "Greet someone" {
		t.Errorf("Description = %q, want %q", def.Description, "Greet someone")
	}
	if len(def.Schema) == 0 {
		t.Error("Schema is empty")
	}
}
