package provider

import (
	"context"
	"fmt"
)

type RoutingStrategy string

const (
	StrategyFallback   RoutingStrategy = "fallback"
	StrategyRoundRobin RoutingStrategy = "round_robin"
)

type Router struct {
	models   []LanguageModel
	strategy RoutingStrategy
}

func NewRouter(strategy RoutingStrategy, models ...LanguageModel) *Router {
	return &Router{models: models, strategy: strategy}
}

func (r *Router) ModelID() string {
	if len(r.models) > 0 {
		return "router:" + r.models[0].ModelID()
	}
	return "router:empty"
}

func (r *Router) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	var lastErr error
	for _, model := range r.models {
		result, err := model.Generate(ctx, params)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all models failed, last error: %w", lastErr)
}

func (r *Router) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	var lastErr error
	for _, model := range r.models {
		ch, err := model.Stream(ctx, params)
		if err == nil {
			return ch, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all models failed, last error: %w", lastErr)
}
