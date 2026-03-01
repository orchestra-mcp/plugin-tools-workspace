# Orchestra Plugin: tools-workspace

A tools plugin for the [Orchestra MCP](https://github.com/orchestra-mcp/framework) framework. Provides multi-folder workspace management with switching support.

## Install

```bash
go install github.com/orchestra-mcp/plugin-tools-workspace/cmd@latest
```

## Usage

Add to your `plugins.yaml`:

```yaml
- id: tools.workspace
  binary: ./bin/tools-workspace
  enabled: true
```

## Tools

| Tool | Description |
|------|-------------|
| `list_workspaces` | List all workspaces with active marker |
| `create_workspace` | Create a new workspace with name and folder paths |
| `get_workspace` | Get a single workspace by ID |
| `update_workspace` | Update a workspace's name or metadata |
| `delete_workspace` | Remove a workspace from the registry |
| `switch_workspace` | Set a workspace as active and update its last-used timestamp |
| `add_folder` | Add a folder to an existing workspace |
| `remove_folder` | Remove a folder from a workspace (error if last folder) |

## Related Packages

- [sdk-go](https://github.com/orchestra-mcp/sdk-go) — Plugin SDK
- [gen-go](https://github.com/orchestra-mcp/gen-go) — Generated Protobuf types
