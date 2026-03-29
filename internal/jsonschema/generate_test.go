package jsonschema

import (
	"encoding/json"
	"testing"
)

type SimpleParams struct {
	Query string `json:"query" description:"The search query"`
	Limit int    `json:"limit,omitempty" description:"Max results"`
}

func TestGenerateFromType(t *testing.T) {
	schema, err := Generate[SimpleParams]()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not a map")
	}
	if _, ok := props["query"]; !ok {
		t.Error("missing property: query")
	}
	if _, ok := props["limit"]; !ok {
		t.Error("missing property: limit")
	}
}

type NestedParams struct {
	Name    string     `json:"name"`
	Options SubOptions `json:"options"`
}

type SubOptions struct {
	Verbose bool `json:"verbose"`
}

func TestGenerateNested(t *testing.T) {
	schema, err := Generate[NestedParams]()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	props := m["properties"].(map[string]any)
	opts, ok := props["options"].(map[string]any)
	if !ok {
		t.Fatal("options property not a map")
	}
	if opts["type"] != "object" {
		t.Errorf("options.type = %v, want object", opts["type"])
	}
}

func TestGenerateEmpty(t *testing.T) {
	type Empty struct{}
	schema, err := Generate[Empty]()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
}
