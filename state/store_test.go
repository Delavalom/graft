package state_test

import (
	"context"
	"errors"
	"testing"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/state"
)

// storeFactory creates a fresh Store for each sub-test.
type storeFactory func(t *testing.T) state.Store

func testSaveAndLoad(t *testing.T, factory storeFactory) {
	t.Helper()
	store := factory(t)
	ctx := context.Background()

	sess := state.NewSession("myagent")
	sess.Messages = []graft.Message{
		{Role: graft.RoleUser, Content: "hello"},
		{Role: graft.RoleAssistant, Content: "hi there"},
	}

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.ID != sess.ID {
		t.Errorf("ID: got %q, want %q", got.ID, sess.ID)
	}
	if got.AgentName != sess.AgentName {
		t.Errorf("AgentName: got %q, want %q", got.AgentName, sess.AgentName)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(Messages): got %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Content != "hello" {
		t.Errorf("Messages[0].Content: got %q, want %q", got.Messages[0].Content, "hello")
	}
	if got.Messages[1].Content != "hi there" {
		t.Errorf("Messages[1].Content: got %q, want %q", got.Messages[1].Content, "hi there")
	}
}

func testListByAgentName(t *testing.T, factory storeFactory) {
	t.Helper()
	store := factory(t)
	ctx := context.Background()

	s1 := state.NewSession("agent-a")
	s2 := state.NewSession("agent-a")
	s3 := state.NewSession("agent-b")

	for _, s := range []*state.Session{s1, s2, s3} {
		if err := store.Save(ctx, s); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	list, err := store.List(ctx, "agent-a")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List(agent-a): got %d sessions, want 2", len(list))
	}

	listB, err := store.List(ctx, "agent-b")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("List(agent-b): got %d sessions, want 1", len(listB))
	}
}

func testDelete(t *testing.T, factory storeFactory) {
	t.Helper()
	store := factory(t)
	ctx := context.Background()

	sess := state.NewSession("agent")
	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load(ctx, sess.ID)
	if err == nil {
		t.Fatal("Load after Delete: expected error, got nil")
	}
	if !errors.Is(err, state.ErrNotFound) {
		t.Errorf("Load after Delete: got %v, want ErrNotFound", err)
	}
}

func testLoadNonexistent(t *testing.T, factory storeFactory) {
	t.Helper()
	store := factory(t)
	ctx := context.Background()

	_, err := store.Load(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("Load nonexistent: expected error, got nil")
	}
	if !errors.Is(err, state.ErrNotFound) {
		t.Errorf("Load nonexistent: got %v, want ErrNotFound", err)
	}
}

func testMultipleSessionsSameAgent(t *testing.T, factory storeFactory) {
	t.Helper()
	store := factory(t)
	ctx := context.Background()

	const agent = "multi-agent"
	ids := make([]string, 5)
	for i := range ids {
		s := state.NewSession(agent)
		ids[i] = s.ID
		if err := store.Save(ctx, s); err != nil {
			t.Fatalf("Save[%d]: %v", i, err)
		}
	}

	list, err := store.List(ctx, agent)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("List: got %d sessions, want 5", len(list))
	}
}

func testUpdateExistingSession(t *testing.T, factory storeFactory) {
	t.Helper()
	store := factory(t)
	ctx := context.Background()

	sess := state.NewSession("upd-agent")
	sess.Messages = []graft.Message{{Role: graft.RoleUser, Content: "first"}}

	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save (1): %v", err)
	}

	// Mutate and save again
	sess.Messages = append(sess.Messages, graft.Message{Role: graft.RoleAssistant, Content: "second"})
	if err := store.Save(ctx, sess); err != nil {
		t.Fatalf("Save (2): %v", err)
	}

	got, err := store.Load(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("Messages after update: got %d, want 2", len(got.Messages))
	}
	if got.Messages[1].Content != "second" {
		t.Errorf("Messages[1].Content: got %q, want %q", got.Messages[1].Content, "second")
	}
}

// runSuite runs all store tests against the provided factory.
func runSuite(t *testing.T, name string, factory storeFactory) {
	t.Helper()
	t.Run(name+"/SaveAndLoad", func(t *testing.T) { testSaveAndLoad(t, factory) })
	t.Run(name+"/ListByAgentName", func(t *testing.T) { testListByAgentName(t, factory) })
	t.Run(name+"/Delete", func(t *testing.T) { testDelete(t, factory) })
	t.Run(name+"/LoadNonexistent", func(t *testing.T) { testLoadNonexistent(t, factory) })
	t.Run(name+"/MultipleSessionsSameAgent", func(t *testing.T) { testMultipleSessionsSameAgent(t, factory) })
	t.Run(name+"/UpdateExistingSession", func(t *testing.T) { testUpdateExistingSession(t, factory) })
}

func TestMemoryStore(t *testing.T) {
	runSuite(t, "MemoryStore", func(t *testing.T) state.Store {
		return state.NewMemoryStore()
	})
}

func TestFileStore(t *testing.T) {
	runSuite(t, "FileStore", func(t *testing.T) state.Store {
		dir := t.TempDir()
		store, err := state.NewFileStore(dir)
		if err != nil {
			t.Fatalf("NewFileStore: %v", err)
		}
		return store
	})
}
