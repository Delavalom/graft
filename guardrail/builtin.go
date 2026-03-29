package guardrail

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/delavalom/graft"
)

func MaxTokens(limit int) graft.Guardrail {
	return &maxTokensGuardrail{limit: limit}
}

type maxTokensGuardrail struct{ limit int }

func (g *maxTokensGuardrail) Name() string              { return "max_tokens" }
func (g *maxTokensGuardrail) Type() graft.GuardrailType { return graft.GuardrailInput }

func (g *maxTokensGuardrail) Validate(ctx context.Context, data *graft.ValidationData) (*graft.ValidationResult, error) {
	totalChars := 0
	for _, msg := range data.Messages {
		totalChars += len(msg.Content)
	}
	estimatedTokens := totalChars / 4
	if estimatedTokens > g.limit {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("estimated %d tokens exceeds limit of %d", estimatedTokens, g.limit),
		}, nil
	}
	return &graft.ValidationResult{Pass: true}, nil
}

func ContentFilter(patterns []string) graft.Guardrail {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return &contentFilterGuardrail{patterns: compiled}
}

type contentFilterGuardrail struct{ patterns []*regexp.Regexp }

func (g *contentFilterGuardrail) Name() string              { return "content_filter" }
func (g *contentFilterGuardrail) Type() graft.GuardrailType { return graft.GuardrailInput }

func (g *contentFilterGuardrail) Validate(ctx context.Context, data *graft.ValidationData) (*graft.ValidationResult, error) {
	for _, msg := range data.Messages {
		for _, p := range g.patterns {
			if p.MatchString(msg.Content) {
				return &graft.ValidationResult{
					Pass:    false,
					Message: fmt.Sprintf("content matches blocked pattern: %s", p.String()),
				}, nil
			}
		}
	}
	return &graft.ValidationResult{Pass: true}, nil
}

func SchemaValidator(schema json.RawMessage) graft.Guardrail {
	return &schemaValidatorGuardrail{schema: schema}
}

type schemaValidatorGuardrail struct{ schema json.RawMessage }

func (g *schemaValidatorGuardrail) Name() string              { return "schema_validator" }
func (g *schemaValidatorGuardrail) Type() graft.GuardrailType { return graft.GuardrailOutput }

func (g *schemaValidatorGuardrail) Validate(ctx context.Context, data *graft.ValidationData) (*graft.ValidationResult, error) {
	var content string
	for i := len(data.Messages) - 1; i >= 0; i-- {
		if data.Messages[i].Role == graft.RoleAssistant {
			content = data.Messages[i].Content
			break
		}
	}
	if content == "" {
		return &graft.ValidationResult{Pass: true}, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("content is not valid JSON: %v", err),
		}, nil
	}

	var schemaDef struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(g.schema, &schemaDef); err != nil {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("invalid schema: %v", err),
		}, nil
	}

	var missing []string
	for _, field := range schemaDef.Required {
		if _, ok := parsed[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return &graft.ValidationResult{
			Pass:    false,
			Message: fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", ")),
		}, nil
	}

	return &graft.ValidationResult{Pass: true}, nil
}
