package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SessionStore struct {
	dir string
}

func NewSessionStore(dir string) *SessionStore {
	return &SessionStore{dir: dir}
}

func (s *SessionStore) CreateSession(session CollaborationSession) (CollaborationSession, error) {
	if s == nil || s.dir == "" {
		return CollaborationSession{}, fmt.Errorf("session store dir is required")
	}
	now := time.Now().UTC()
	if session.ID == "" {
		session.ID = fmt.Sprintf("session_%d", now.UnixNano())
	}
	if session.State == "" {
		session.State = StateCreated
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	if err := s.write(session); err != nil {
		return CollaborationSession{}, err
	}
	return session, nil
}

func (s *SessionStore) UpdateState(id string, state LifecycleState) error {
	if id == "" || state == "" {
		return fmt.Errorf("session id and state are required")
	}
	session, ok, err := s.GetSession(id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	session.State = state
	session.UpdatedAt = time.Now().UTC()
	return s.write(session)
}

func (s *SessionStore) GetSession(id string) (CollaborationSession, bool, error) {
	if s == nil || s.dir == "" {
		return CollaborationSession{}, false, fmt.Errorf("session store dir is required")
	}
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return CollaborationSession{}, false, nil
		}
		return CollaborationSession{}, false, err
	}
	var session CollaborationSession
	if err := json.Unmarshal(data, &session); err != nil {
		return CollaborationSession{}, false, err
	}
	return session, true, nil
}

func (s *SessionStore) ListSessions() ([]CollaborationSession, error) {
	if s == nil || s.dir == "" {
		return nil, fmt.Errorf("session store dir is required")
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sessions []CollaborationSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var session CollaborationSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	return sessions, nil
}

func (s *SessionStore) write(session CollaborationSession) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, session.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
