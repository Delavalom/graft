package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileStore persists sessions as JSON files in a directory.
// Each session is stored as {dir}/{session_id}.json.
type FileStore struct {
	dir string
}

// NewFileStore creates a FileStore that reads and writes files under dir.
// The directory is created if it does not already exist.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("state/file: create directory: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

// Save writes the session to disk atomically (temp file + rename).
func (f *FileStore) Save(_ context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("state/file: session must not be nil")
	}
	session.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("state/file: marshal session: %w", err)
	}

	target := f.path(session.ID)
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("state/file: write temp file: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("state/file: rename temp file: %w", err)
	}
	return nil
}

// Load reads a session from disk by ID.
func (f *FileStore) Load(_ context.Context, sessionID string) (*Session, error) {
	data, err := os.ReadFile(f.path(sessionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, sessionID)
		}
		return nil, fmt.Errorf("state/file: read session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("state/file: unmarshal session: %w", err)
	}
	return &s, nil
}

// List returns all sessions stored for the given agent name.
func (f *FileStore) List(_ context.Context, agentName string) ([]*Session, error) {
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return nil, fmt.Errorf("state/file: read directory: %w", err)
	}

	var out []*Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(f.dir, e.Name()))
		if err != nil {
			continue // skip unreadable files
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue // skip corrupt files
		}
		if s.AgentName == agentName {
			out = append(out, &s)
		}
	}
	return out, nil
}

// Delete removes the session file from disk.
func (f *FileStore) Delete(_ context.Context, sessionID string) error {
	err := os.Remove(f.path(sessionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, sessionID)
		}
		return fmt.Errorf("state/file: delete session: %w", err)
	}
	return nil
}

func (f *FileStore) path(sessionID string) string {
	return filepath.Join(f.dir, sessionID+".json")
}
