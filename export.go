package toolsworkspace

import (
	"github.com/orchestra-mcp/plugin-tools-workspace/internal"
	"github.com/orchestra-mcp/plugin-tools-workspace/internal/store"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// Register adds all 8 workspace management tools to the builder.
func Register(builder *plugin.PluginBuilder) error {
	wsStore, err := store.NewWorkspaceStore()
	if err != nil {
		return err
	}
	wp := &internal.WorkspacePlugin{Store: wsStore}
	wp.RegisterTools(builder)
	return nil
}
