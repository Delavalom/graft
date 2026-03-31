package agentworkflow

import (
	"fmt"
	"sync"

	"github.com/delavalom/graft"
)

// ToolRegistry maps tool names to graft.Tool implementations.
// Workers register tools at startup; activities look them up by name.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]graft.Tool
}

// NewToolRegistry creates an empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]graft.Tool)}
}

// Register adds one or more tools to the registry.
func (r *ToolRegistry) Register(tools ...graft.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range tools {
		r.tools[t.Name()] = t
	}
}

// Get returns a tool by name.
func (r *ToolRegistry) Get(name string) (graft.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools.
func (r *ToolRegistry) List() []graft.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]graft.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// ModelProvider resolves a model identifier to a graft.LanguageModel.
type ModelProvider interface {
	Model(id string) (graft.LanguageModel, error)
}

// SingleModelProvider wraps a single LanguageModel for the common case
// where only one model is used.
type SingleModelProvider struct {
	model graft.LanguageModel
}

// NewSingleModelProvider creates a provider that always returns the given model.
func NewSingleModelProvider(m graft.LanguageModel) *SingleModelProvider {
	return &SingleModelProvider{model: m}
}

// Model returns the wrapped model regardless of the id.
func (p *SingleModelProvider) Model(_ string) (graft.LanguageModel, error) {
	return p.model, nil
}

// MultiModelProvider maps model IDs to LanguageModel instances.
// Use this when agents in a handoff chain use different models.
type MultiModelProvider struct {
	models map[string]graft.LanguageModel
}

// NewMultiModelProvider creates a provider from a map of model ID to LanguageModel.
func NewMultiModelProvider(models map[string]graft.LanguageModel) *MultiModelProvider {
	return &MultiModelProvider{models: models}
}

// Model returns the LanguageModel for the given id.
func (p *MultiModelProvider) Model(id string) (graft.LanguageModel, error) {
	m, ok := p.models[id]
	if !ok {
		return nil, fmt.Errorf("agentworkflow: unknown model %q", id)
	}
	return m, nil
}
