package cli

import (
	"context"
	"os/signal"
	"syscall"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/user/gitlab-cli/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server on stdio",
	Long:  "Start an MCP (Model Context Protocol) server that exposes gitlab-cli tools for AI assistants.",
	RunE:  runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	srv, err := mcp.NewServer()
	if err != nil {
		return err
	}

	sdkServer := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "gitlab-cli-mcp",
		Version: "1.0.0",
	}, nil)

	srv.RegisterTools(sdkServer)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return sdkServer.Run(ctx, &sdkmcp.StdioTransport{})
}
