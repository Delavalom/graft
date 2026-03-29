package provider

import (
	"context"
	"log/slog"
	"time"
)

type Middleware func(LanguageModel) LanguageModel

func Chain(model LanguageModel, mw ...Middleware) LanguageModel {
	for i := len(mw) - 1; i >= 0; i-- {
		model = mw[i](model)
	}
	return model
}

type loggingModel struct {
	inner  LanguageModel
	logger *slog.Logger
}

func WithLogging(logger *slog.Logger) Middleware {
	return func(m LanguageModel) LanguageModel {
		return &loggingModel{inner: m, logger: logger}
	}
}

func (l *loggingModel) ModelID() string { return l.inner.ModelID() }

func (l *loggingModel) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	start := time.Now()
	l.logger.Info("llm.generate.start", "model", l.inner.ModelID(), "messages", len(params.Messages))
	result, err := l.inner.Generate(ctx, params)
	duration := time.Since(start)
	if err != nil {
		l.logger.Error("llm.generate.error", "model", l.inner.ModelID(), "duration", duration, "error", err)
		return nil, err
	}
	l.logger.Info("llm.generate.done",
		"model", l.inner.ModelID(),
		"duration", duration,
		"prompt_tokens", result.Usage.PromptTokens,
		"completion_tokens", result.Usage.CompletionTokens,
	)
	return result, nil
}

func (l *loggingModel) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	l.logger.Info("llm.stream.start", "model", l.inner.ModelID(), "messages", len(params.Messages))
	return l.inner.Stream(ctx, params)
}
