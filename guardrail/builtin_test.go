package guardrail

import (
	"context"
	"testing"

	"github.com/delavalom/graft"
)

func TestMaxTokensPass(t *testing.T) {
	g := MaxTokens(100)
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "short"}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Message)
	}
}

func TestMaxTokensFail(t *testing.T) {
	g := MaxTokens(5)
	longMsg := "This is a message that definitely exceeds five tokens worth of content and should fail the guardrail check"
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: longMsg}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Pass {
		t.Error("expected fail for long message, got pass")
	}
}

func TestContentFilterPass(t *testing.T) {
	g := ContentFilter([]string{`(?i)password`, `(?i)secret`})
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "hello world"}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Message)
	}
}

func TestContentFilterFail(t *testing.T) {
	g := ContentFilter([]string{`(?i)password`, `(?i)secret`})
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleUser, Content: "my Password is 1234"}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Pass {
		t.Error("expected fail for message containing 'password'")
	}
}

func TestContentFilterName(t *testing.T) {
	g := ContentFilter([]string{`test`})
	if g.Name() != "content_filter" {
		t.Errorf("Name() = %q, want %q", g.Name(), "content_filter")
	}
	if g.Type() != graft.GuardrailInput {
		t.Errorf("Type() = %v, want %v", g.Type(), graft.GuardrailInput)
	}
}

func TestSchemaValidatorPass(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	g := SchemaValidator(schema)
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: `{"name":"Alice"}`}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Message)
	}
}

func TestSchemaValidatorFail(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	g := SchemaValidator(schema)
	result, err := g.Validate(context.Background(), &graft.ValidationData{
		Messages: []graft.Message{{Role: graft.RoleAssistant, Content: `{"age":30}`}},
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Pass {
		t.Error("expected fail for missing required field")
	}
}
