// Package tools implements the 8 MCP tool handlers for the tools.workspace plugin.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-tools-workspace/internal/store"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToolHandler is the standard tool handler function signature.
type ToolHandler = func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error)

// ---------- Schemas ----------

func ListWorkspacesSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	})
	return s
}

func CreateWorkspaceSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Workspace display name",
			},
			"folders": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of folder paths",
			},
		},
		"required": []any{"name", "folders"},
	})
	return s
}

func GetWorkspaceSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workspace_id": map[string]any{
				"type":        "string",
				"description": "Workspace ID (e.g. WS-XXXX)",
			},
		},
		"required": []any{"workspace_id"},
	})
	return s
}

func UpdateWorkspaceSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workspace_id": map[string]any{
				"type":        "string",
				"description": "Workspace ID",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "New workspace name (optional)",
			},
			"metadata": map[string]any{
				"type":        "string",
				"description": "JSON string of metadata key-values to merge (optional)",
			},
		},
		"required": []any{"workspace_id"},
	})
	return s
}

func DeleteWorkspaceSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workspace_id": map[string]any{
				"type":        "string",
				"description": "Workspace ID to delete",
			},
		},
		"required": []any{"workspace_id"},
	})
	return s
}

func SwitchWorkspaceSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workspace_id": map[string]any{
				"type":        "string",
				"description": "Workspace ID to activate",
			},
		},
		"required": []any{"workspace_id"},
	})
	return s
}

func AddFolderSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workspace_id": map[string]any{
				"type":        "string",
				"description": "Workspace ID",
			},
			"folder": map[string]any{
				"type":        "string",
				"description": "Folder path to add",
			},
		},
		"required": []any{"workspace_id", "folder"},
	})
	return s
}

func RemoveFolderSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workspace_id": map[string]any{
				"type":        "string",
				"description": "Workspace ID",
			},
			"folder": map[string]any{
				"type":        "string",
				"description": "Folder path to remove",
			},
		},
		"required": []any{"workspace_id", "folder"},
	})
	return s
}

// ---------- Handlers ----------

func ListWorkspaces(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		reg, err := s.List()
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		if len(reg.Workspaces) == 0 {
			return helpers.TextResult("## Workspaces\n\nNo workspaces configured.\n"), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Workspaces (%d)\n\n", len(reg.Workspaces))
		fmt.Fprintf(&b, "| ID | Name | Folders | Primary | Last Used | Active |\n")
		fmt.Fprintf(&b, "|----|------|---------|---------|-----------|--------|\n")
		for _, ws := range reg.Workspaces {
			active := ""
			if ws.ID == reg.ActiveWorkspaceID {
				active = "**>>**"
			}
			folders := strings.Join(ws.Folders, ", ")
			if len(folders) > 60 {
				folders = folders[:57] + "..."
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
				ws.ID, ws.Name, folders, ws.PrimaryFolder, ws.LastUsed, active)
		}
		return helpers.TextResult(b.String()), nil
	}
}

func CreateWorkspace(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "name", "folders"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		name := helpers.GetString(req.Arguments, "name")
		foldersStr := helpers.GetString(req.Arguments, "folders")

		// Parse comma-separated folders.
		var folders []string
		for _, f := range strings.Split(foldersStr, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				folders = append(folders, f)
			}
		}
		if len(folders) == 0 {
			return helpers.ErrorResult("validation_error", "at least one folder is required"), nil
		}

		ws, err := s.Create(name, folders)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := formatWorkspaceMD(ws, "Created workspace")
		return helpers.TextResult(md), nil
	}
}

func GetWorkspace(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "workspace_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "workspace_id")
		ws, err := s.Get(id)
		if err != nil {
			return helpers.ErrorResult("not_found", err.Error()), nil
		}

		md := formatWorkspaceMD(ws, "Workspace details")
		return helpers.TextResult(md), nil
	}
}

func UpdateWorkspace(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "workspace_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "workspace_id")
		name := helpers.GetString(req.Arguments, "name")
		metadataStr := helpers.GetString(req.Arguments, "metadata")

		err := s.Update(id, func(ws *store.Workspace) {
			if name != "" {
				ws.Name = name
			}
			if metadataStr != "" {
				var meta map[string]string
				if json_err := parseJSON(metadataStr, &meta); json_err == nil {
					if ws.Metadata == nil {
						ws.Metadata = make(map[string]string)
					}
					for k, v := range meta {
						ws.Metadata[k] = v
					}
				}
			}
		})
		if err != nil {
			return helpers.ErrorResult("not_found", err.Error()), nil
		}

		ws, _ := s.Get(id)
		md := formatWorkspaceMD(ws, "Updated workspace")
		return helpers.TextResult(md), nil
	}
}

func DeleteWorkspace(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "workspace_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "workspace_id")
		if err := s.Delete(id); err != nil {
			return helpers.ErrorResult("not_found", err.Error()), nil
		}

		return helpers.TextResult(fmt.Sprintf("Deleted workspace **%s**", id)), nil
	}
}

func SwitchWorkspace(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "workspace_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "workspace_id")
		ws, err := s.Switch(id)
		if err != nil {
			return helpers.ErrorResult("not_found", err.Error()), nil
		}

		md := formatWorkspaceMD(ws, "Switched to workspace")
		return helpers.TextResult(md), nil
	}
}

func AddFolder(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "workspace_id", "folder"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "workspace_id")
		folder := helpers.GetString(req.Arguments, "folder")

		if err := s.AddFolder(id, folder); err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		return helpers.TextResult(fmt.Sprintf("Added folder `%s` to workspace **%s**", folder, id)), nil
	}
}

func RemoveFolder(s *store.WorkspaceStore) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "workspace_id", "folder"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "workspace_id")
		folder := helpers.GetString(req.Arguments, "folder")

		if err := s.RemoveFolder(id, folder); err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		return helpers.TextResult(fmt.Sprintf("Removed folder `%s` from workspace **%s**", folder, id)), nil
	}
}

// ---------- Helpers ----------

func formatWorkspaceMD(ws *store.Workspace, header string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s: %s (%s)\n\n", header, ws.Name, ws.ID)
	fmt.Fprintf(&b, "- **Primary Folder:** `%s`\n", ws.PrimaryFolder)
	fmt.Fprintf(&b, "- **Created:** %s\n", ws.CreatedAt)
	fmt.Fprintf(&b, "- **Last Used:** %s\n", ws.LastUsed)

	if len(ws.Folders) > 0 {
		fmt.Fprintf(&b, "\n**Folders (%d):**\n", len(ws.Folders))
		for _, f := range ws.Folders {
			fmt.Fprintf(&b, "- `%s`\n", f)
		}
	}

	if len(ws.Metadata) > 0 {
		fmt.Fprintf(&b, "\n**Metadata:**\n")
		for k, v := range ws.Metadata {
			fmt.Fprintf(&b, "- `%s`: `%s`\n", k, v)
		}
	}

	return b.String()
}

func parseJSON(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
