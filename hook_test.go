package graft

import (
	"context"
	"testing"
)

func TestHookRegistryOn(t *testing.T) {
	reg := NewHookRegistry()
	called := false
	reg.On(HookPreToolCall, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		called = true
		return nil, nil
	})
	_, err := reg.Run(context.Background(), &HookPayload{Event: HookPreToolCall})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Error("hook was not called")
	}
}

func TestHookRegistryDeny(t *testing.T) {
	reg := NewHookRegistry()
	reg.On(HookPreToolCall, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		deny := false
		return &HookResult{Allow: &deny}, nil
	})
	secondCalled := false
	reg.On(HookPreToolCall, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		secondCalled = true
		return nil, nil
	})
	result, err := reg.Run(context.Background(), &HookPayload{Event: HookPreToolCall})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == nil || result.Allow == nil || *result.Allow != false {
		t.Error("expected deny result")
	}
	if secondCalled {
		t.Error("second hook should not have been called after deny")
	}
}

func TestHookRegistryPassthrough(t *testing.T) {
	reg := NewHookRegistry()
	reg.On(HookAgentStart, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		return nil, nil
	})
	result, err := reg.Run(context.Background(), &HookPayload{Event: HookAgentStart})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for passthrough")
	}
}

func TestHookRegistryNoHooks(t *testing.T) {
	reg := NewHookRegistry()
	result, err := reg.Run(context.Background(), &HookPayload{Event: HookAgentEnd})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no hooks registered")
	}
}

func TestHookRegistryNilSafe(t *testing.T) {
	var reg *HookRegistry
	result, err := reg.Run(context.Background(), &HookPayload{Event: HookAgentStart})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nil registry")
	}
}

func TestHookRegistryMultipleEvents(t *testing.T) {
	reg := NewHookRegistry()
	startCalled := false
	endCalled := false
	reg.On(HookAgentStart, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		startCalled = true
		return nil, nil
	})
	reg.On(HookAgentEnd, func(ctx context.Context, p *HookPayload) (*HookResult, error) {
		endCalled = true
		return nil, nil
	})
	reg.Run(context.Background(), &HookPayload{Event: HookAgentStart})
	if !startCalled {
		t.Error("start hook not called")
	}
	if endCalled {
		t.Error("end hook should not be called for start event")
	}
}
