// Package store provides the data layer for the tools.workspace plugin.
// WorkspaceStore handles CRUD operations on workspaces, persisted to
// globaldb (~/.orchestra/db/global.db). No markdown export — workspaces are
// local machine config and must NOT be synced via git.
package store

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/orchestra-mcp/sdk-go/globaldb"
)

// Workspace represents a named collection of project folders.
// This is a thin wrapper around globaldb.Workspace for backward compatibility.
type Workspace = globaldb.Workspace

// Registry is the top-level structure for listing workspaces.
type Registry struct {
	Workspaces        []*Workspace `json:"workspaces"`
	ActiveWorkspaceID string       `json:"active_workspace_id"`
}

// WorkspaceStore provides thread-safe CRUD operations on the workspace registry
// via globaldb. All mutations go through the global SQLite database.
type WorkspaceStore struct{}

// NewWorkspaceStore creates a new store. The global database is lazily
// initialized on first use.
func NewWorkspaceStore() (*WorkspaceStore, error) {
	// Migrate existing JSON workspaces into globaldb on first use.
	globaldb.MigrateWorkspacesJSON()
	return &WorkspaceStore{}, nil
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

// List returns all workspaces and the active workspace ID.
func (s *WorkspaceStore) List() (*Registry, error) {
	ws, err := globaldb.ListWorkspaces()
	if err != nil {
		return nil, err
	}
	return &Registry{
		Workspaces:        ws,
		ActiveWorkspaceID: globaldb.GetActiveWorkspaceID(),
	}, nil
}

// Get returns a single workspace by ID.
func (s *WorkspaceStore) Get(id string) (*Workspace, error) {
	return globaldb.GetWorkspace(id)
}

// Create adds a new workspace and returns it.
func (s *WorkspaceStore) Create(name string, folders []string) (*Workspace, error) {
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

	if err := globaldb.CreateWorkspace(ws); err != nil {
		return nil, err
	}

	// If this is the first workspace, auto-activate it.
	if globaldb.GetActiveWorkspaceID() == "" {
		globaldb.SetActiveWorkspaceID(ws.ID)
	}

	return ws, nil
}

// Update modifies an existing workspace via a mutation function.
func (s *WorkspaceStore) Update(id string, fn func(ws *Workspace)) error {
	ws, err := globaldb.GetWorkspace(id)
	if err != nil {
		return err
	}
	fn(ws)
	return globaldb.SaveWorkspace(ws)
}

// Delete removes a workspace by ID.
func (s *WorkspaceStore) Delete(id string) error {
	if err := globaldb.DeleteWorkspace(id); err != nil {
		return err
	}
	// Clear active workspace if we deleted it.
	if globaldb.GetActiveWorkspaceID() == id {
		globaldb.SetActiveWorkspaceID("")
	}
	return nil
}

// Switch sets the active workspace and updates its LastUsed timestamp.
func (s *WorkspaceStore) Switch(id string) (*Workspace, error) {
	ws, err := globaldb.GetWorkspace(id)
	if err != nil {
		return nil, err
	}

	ws.LastUsed = time.Now().UTC().Format(time.RFC3339)
	if err := globaldb.SaveWorkspace(ws); err != nil {
		return nil, err
	}

	if err := globaldb.SetActiveWorkspaceID(id); err != nil {
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
	ws, err := globaldb.GetWorkspace(id)
	if err != nil {
		return err
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

	return globaldb.SaveWorkspace(ws)
}
