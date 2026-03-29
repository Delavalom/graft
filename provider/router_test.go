package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/delavalom/graft"
)

type mockModel struct {
	id        string
	err       error
	result    *GenerateResult
	callCount int
}

func (m *mockModel) ModelID() string { return m.id }

func (m *mockModel) Generate(ctx context.Context, params GenerateParams) (*GenerateResult, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockModel) Stream(ctx context.Context, params GenerateParams) (<-chan StreamChunk, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}

func TestRouterFallback(t *testing.T) {
	failing := &mockModel{id: "fail", err: fmt.Errorf("503 service unavailable")}
	working := &mockModel{
		id: "work",
		result: &GenerateResult{
			Message: graft.Message{Role: graft.RoleAssistant, Content: "ok"},
		},
	}

	router := NewRouter(StrategyFallback, failing, working)
	result, err := router.Generate(context.Background(), GenerateParams{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "ok" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "ok")
	}
	if failing.callCount != 1 {
		t.Errorf("failing.callCount = %d, want 1", failing.callCount)
	}
	if working.callCount != 1 {
		t.Errorf("working.callCount = %d, want 1", working.callCount)
	}
}

func TestRouterAllFail(t *testing.T) {
	m1 := &mockModel{id: "m1", err: fmt.Errorf("error 1")}
	m2 := &mockModel{id: "m2", err: fmt.Errorf("error 2")}

	router := NewRouter(StrategyFallback, m1, m2)
	_, err := router.Generate(context.Background(), GenerateParams{})
	if err == nil {
		t.Fatal("expected error when all models fail")
	}
}

func TestRouterSingleModel(t *testing.T) {
	m := &mockModel{
		id: "m1",
		result: &GenerateResult{
			Message: graft.Message{Role: graft.RoleAssistant, Content: "hello"},
		},
	}

	router := NewRouter(StrategyFallback, m)
	result, err := router.Generate(context.Background(), GenerateParams{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Message.Content != "hello" {
		t.Errorf("Content = %q, want %q", result.Message.Content, "hello")
	}
}
