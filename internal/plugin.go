// Package internal contains the core registration logic for the tools.workspace
// plugin. The WorkspacePlugin struct wires all 8 tool handlers to the plugin
// builder with their schemas and descriptions.
package internal

import (
	"github.com/orchestra-mcp/plugin-tools-workspace/internal/store"
	"github.com/orchestra-mcp/plugin-tools-workspace/internal/tools"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// WorkspacePlugin holds the shared dependencies for all tool handlers.
type WorkspacePlugin struct {
	Store *store.WorkspaceStore
}

// RegisterTools registers all 8 workspace tools on the given plugin builder.
func (p *WorkspacePlugin) RegisterTools(builder *plugin.PluginBuilder) {
	s := p.Store

	builder.RegisterTool("list_workspaces",
		"List all workspaces with active marker",
		tools.ListWorkspacesSchema(), tools.ListWorkspaces(s))

	builder.RegisterTool("create_workspace",
		"Create a new workspace with name and folder paths",
		tools.CreateWorkspaceSchema(), tools.CreateWorkspace(s))

	builder.RegisterTool("get_workspace",
		"Get a single workspace by ID",
		tools.GetWorkspaceSchema(), tools.GetWorkspace(s))

	builder.RegisterTool("update_workspace",
		"Update a workspace's name or metadata",
		tools.UpdateWorkspaceSchema(), tools.UpdateWorkspace(s))

	builder.RegisterTool("delete_workspace",
		"Remove a workspace from the registry",
		tools.DeleteWorkspaceSchema(), tools.DeleteWorkspace(s))

	builder.RegisterTool("switch_workspace",
		"Set a workspace as active and update its last-used timestamp",
		tools.SwitchWorkspaceSchema(), tools.SwitchWorkspace(s))

	builder.RegisterTool("add_folder",
		"Add a folder to an existing workspace",
		tools.AddFolderSchema(), tools.AddFolder(s))

	builder.RegisterTool("remove_folder",
		"Remove a folder from a workspace (error if last folder)",
		tools.RemoveFolderSchema(), tools.RemoveFolder(s))
}
