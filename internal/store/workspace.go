// Package store provides the data layer for the tools.workspace plugin.
// WorkspaceStore handles CRUD operations on workspaces, persisted to
// ~/.orchestra/workspaces.json.
package store

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Workspace represents a named collection of project folders.
type Workspace struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Folders       []string          `json:"folders"`
	PrimaryFolder string            `json:"primary_folder"`
	CreatedAt     string            `json:"created_at"`
	LastUsed      string            `json:"last_used"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Registry is the top-level structure persisted to disk.
type Registry struct {
	Workspaces        []*Workspace `json:"workspaces"`
	ActiveWorkspaceID string       `json:"active_workspace_id"`
}

// WorkspaceStore provides thread-safe CRUD operations on the workspace registry.
type WorkspaceStore struct {
	mu   sync.Mutex
	path string
}

// NewWorkspaceStore creates a new store. Ensures the directory exists.
func NewWorkspaceStore() (*WorkspaceStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	dir := filepath.Join(home, ".orchestra")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create orchestra dir: %w", err)
	}

	return &WorkspaceStore{
		path: filepath.Join(dir, "workspaces.json"),
	}, nil
}

// NewWorkspaceID generates an ID in the format "WS-XXXX".
func NewWorkspaceID() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "WS-" + string(b)
}

// load reads the registry from disk.
func (s *WorkspaceStore) load() (*Registry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, fmt.Errorf("read workspaces file: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("unmarshal workspaces: %w", err)
	}
	return &reg, nil
}

// save writes the registry to disk.
func (s *WorkspaceStore) save(reg *Registry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspaces: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write workspaces file: %w", err)
	}
	return nil
}

// findByID returns the workspace and its index, or -1 if not found.
func findByID(reg *Registry, id string) (*Workspace, int) {
	for i, ws := range reg.Workspaces {
		if ws.ID == id {
			return ws, i
		}
	}
	return nil, -1
}

// List returns all workspaces and the active workspace ID.
func (s *WorkspaceStore) List() (*Registry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Get returns a single workspace by ID.
func (s *WorkspaceStore) Get(id string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return nil, err
	}

	ws, _ := findByID(reg, id)
	if ws == nil {
		return nil, fmt.Errorf("workspace %q not found", id)
	}
	return ws, nil
}

// Create adds a new workspace and returns it.
func (s *WorkspaceStore) Create(name string, folders []string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	primary := ""
	if len(folders) > 0 {
		primary = folders[0]
	}

	ws := &Workspace{
		ID:            NewWorkspaceID(),
		Name:          name,
		Folders:       folders,
		PrimaryFolder: primary,
		CreatedAt:     now,
		LastUsed:      now,
		Metadata:      make(map[string]string),
	}

	reg.Workspaces = append(reg.Workspaces, ws)

	// If this is the first workspace, auto-activate it.
	if reg.ActiveWorkspaceID == "" {
		reg.ActiveWorkspaceID = ws.ID
	}

	if err := s.save(reg); err != nil {
		return nil, err
	}
	return ws, nil
}

// Update modifies an existing workspace via a mutation function.
func (s *WorkspaceStore) Update(id string, fn func(ws *Workspace)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return err
	}

	ws, _ := findByID(reg, id)
	if ws == nil {
		return fmt.Errorf("workspace %q not found", id)
	}

	fn(ws)
	return s.save(reg)
}

// Delete removes a workspace by ID.
func (s *WorkspaceStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return err
	}

	_, idx := findByID(reg, id)
	if idx < 0 {
		return fmt.Errorf("workspace %q not found", id)
	}

	reg.Workspaces = append(reg.Workspaces[:idx], reg.Workspaces[idx+1:]...)
	if reg.ActiveWorkspaceID == id {
		reg.ActiveWorkspaceID = ""
	}

	return s.save(reg)
}

// Switch sets the active workspace and updates its LastUsed timestamp.
func (s *WorkspaceStore) Switch(id string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return nil, err
	}

	ws, _ := findByID(reg, id)
	if ws == nil {
		return nil, fmt.Errorf("workspace %q not found", id)
	}

	reg.ActiveWorkspaceID = id
	ws.LastUsed = time.Now().UTC().Format(time.RFC3339)

	if err := s.save(reg); err != nil {
		return nil, err
	}
	return ws, nil
}

// AddFolder adds a folder to a workspace.
func (s *WorkspaceStore) AddFolder(id, folder string) error {
	return s.Update(id, func(ws *Workspace) {
		for _, f := range ws.Folders {
			if f == folder {
				return // already present
			}
		}
		ws.Folders = append(ws.Folders, folder)
	})
}

// RemoveFolder removes a folder from a workspace. Errors if it would leave zero folders.
func (s *WorkspaceStore) RemoveFolder(id, folder string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return err
	}

	ws, _ := findByID(reg, id)
	if ws == nil {
		return fmt.Errorf("workspace %q not found", id)
	}

	idx := -1
	for i, f := range ws.Folders {
		if f == folder {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("folder %q not in workspace", folder)
	}
	if len(ws.Folders) <= 1 {
		return fmt.Errorf("cannot remove last folder from workspace")
	}

	ws.Folders = append(ws.Folders[:idx], ws.Folders[idx+1:]...)

	// If we removed the primary folder, update it.
	if ws.PrimaryFolder == folder {
		ws.PrimaryFolder = ws.Folders[0]
	}

	return s.save(reg)
}
