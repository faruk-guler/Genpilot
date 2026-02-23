package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

// Session represents a saved SSH connection configuration
type Session struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	PrivateKey string `json:"private_key,omitempty"`
	Password   string `json:"-"` // Stored in keyring, not JSON
	Group      string `json:"group,omitempty"`
	LastUsed   string `json:"last_used"`
}

// SessionManager handles saving and loading sessions
type SessionManager struct {
	configPath string
	sessions   []Session
}

// NewSessionManager creates a new session manager
func NewSessionManager() (*SessionManager, error) {
	// Get user's config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".genpilot")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	sm := &SessionManager{
		configPath: filepath.Join(configDir, "sessions.json"),
		sessions:   make([]Session, 0),
	}

	// Load existing sessions
	sm.Load()
	return sm, nil
}

// Save saves all sessions to disk
func (sm *SessionManager) Save() error {
	data, err := json.MarshalIndent(sm.sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.configPath, data, 0600)
}

// Load loads sessions from disk
func (sm *SessionManager) Load() error {
	data, err := os.ReadFile(sm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No sessions file yet, that's ok
		}
		return err
	}

	return json.Unmarshal(data, &sm.sessions)
}

// AddSession adds a new session or updates existing one
func (sm *SessionManager) AddSession(session Session) error {
	session.LastUsed = time.Now().Format(time.RFC3339)

	// Check if session with same name exists
	for i, s := range sm.sessions {
		if s.Name == session.Name {
			sm.sessions[i] = session

			// Save password to keyring if provided
			if session.Password != "" {
				_ = keyring.Set("Genpilot", session.Name, session.Password)
			}

			return sm.Save()
		}
	}

	// Add new session
	sm.sessions = append(sm.sessions, session)

	// Save password to keyring if provided
	if session.Password != "" {
		// Use "Genpilot" as service name, session.Name as user
		// Note: session names must be unique, which is enforced above
		_ = keyring.Set("Genpilot", session.Name, session.Password)
	}

	return sm.Save()
}

// GetSession retrieves a session by name
func (sm *SessionManager) GetSession(name string) *Session {
	for i, s := range sm.sessions {
		if s.Name == name {
			sm.sessions[i].LastUsed = time.Now().Format(time.RFC3339)
			sm.Save()
			return &sm.sessions[i]
		}
	}
	return nil
}

// DeleteSession removes a session by name
func (sm *SessionManager) DeleteSession(name string) error {
	for i, s := range sm.sessions {
		if s.Name == name {
			sm.sessions = append(sm.sessions[:i], sm.sessions[i+1:]...)
			// Delete password from keyring
			_ = keyring.Delete("Genpilot", name)
			return sm.Save()
		}
	}
	return nil
}

// GetAllSessions returns all saved sessions
func (sm *SessionManager) GetAllSessions() []Session {
	return sm.sessions
}

// GetRecentSessions returns sessions sorted by last used
func (sm *SessionManager) GetRecentSessions(limit int) []Session {
	// Sort by last used (most recent first)
	sessions := make([]Session, len(sm.sessions))
	copy(sessions, sm.sessions)

	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			ti, _ := time.Parse(time.RFC3339, sessions[i].LastUsed)
			tj, _ := time.Parse(time.RFC3339, sessions[j].LastUsed)
			if tj.After(ti) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	if limit > 0 && limit < len(sessions) {
		return sessions[:limit]
	}
	return sessions
}
