// Command tools-workspace is the entry point for the tools.workspace plugin
// binary. It manages multi-folder workspaces with switching support.
// Data is stored locally at ~/.orchestra/workspaces.json.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/orchestra-mcp/plugin-tools-workspace/internal"
	"github.com/orchestra-mcp/plugin-tools-workspace/internal/store"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

func main() {
	builder := plugin.New("tools.workspace").
		Version("0.1.0").
		Description("Multi-folder workspace management with switching support").
		Author("Orchestra").
		Binary("tools-workspace")

	wsStore, err := store.NewWorkspaceStore()
	if err != nil {
		log.Fatalf("tools.workspace: init workspace store: %v", err)
	}

	wp := &internal.WorkspacePlugin{
		Store: wsStore,
	}
	wp.RegisterTools(builder)

	p := builder.BuildWithTools()
	p.ParseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := p.Run(ctx); err != nil {
		log.Fatalf("tools.workspace: %v", err)
	}
}
