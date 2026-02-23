package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Bookmark represents a saved directory location
type Bookmark struct {
	Name       string `json:"name"`
	RemotePath string `json:"remote_path"`
	LocalPath  string `json:"local_path,omitempty"`
}

// BookmarkManager handles saving and loading bookmarks
type BookmarkManager struct {
	configPath string
	bookmarks  []Bookmark
}

// NewBookmarkManager creates a new bookmark manager
func NewBookmarkManager() (*BookmarkManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".genpilot")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	bm := &BookmarkManager{
		configPath: filepath.Join(configDir, "bookmarks.json"),
		bookmarks:  make([]Bookmark, 0),
	}

	bm.Load()
	return bm, nil
}

// Save saves bookmarks to disk
func (bm *BookmarkManager) Save() error {
	data, err := json.MarshalIndent(bm.bookmarks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(bm.configPath, data, 0600)
}

// Load loads bookmarks from disk
func (bm *BookmarkManager) Load() error {
	data, err := os.ReadFile(bm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &bm.bookmarks)
}

// Add adds a new bookmark
func (bm *BookmarkManager) Add(b Bookmark) error {
	for i, existing := range bm.bookmarks {
		if existing.Name == b.Name {
			bm.bookmarks[i] = b
			return bm.Save()
		}
	}
	bm.bookmarks = append(bm.bookmarks, b)
	return bm.Save()
}

// Delete removes a bookmark
func (bm *BookmarkManager) Delete(name string) error {
	for i, b := range bm.bookmarks {
		if b.Name == name {
			bm.bookmarks = append(bm.bookmarks[:i], bm.bookmarks[i+1:]...)
			return bm.Save()
		}
	}
	return nil
}

// GetAll returns all bookmarks
func (bm *BookmarkManager) GetAll() []Bookmark {
	return bm.bookmarks
}

// Get returns a bookmark by name
func (bm *BookmarkManager) Get(name string) *Bookmark {
	for _, b := range bm.bookmarks {
		if b.Name == name {
			return &b
		}
	}
	return nil
}
