package cmd

import (
	"fmt"
	"os"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	mcpserver "github.com/KirisameLonnet/ashpipe/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio) for Claude Code / Codex",
	Long: `Start the ashpipe MCP server over stdio.

Configure in Claude Code (~/.claude/settings.json):
  {
    "mcpServers": {
      "ssh": {
        "command": "ashpipe",
        "args": ["mcp"],
        "type": "stdio"
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		root, err := config.FindRoot(cwd)
		if err != nil {
			return fmt.Errorf("not in an ashpipe workspace: %w", err)
		}
		cfg, err := config.Load(root)
		if err != nil {
			return err
		}
		cfg.WarnInsecure()

		srv := mcpserver.NewServer(root, cfg)
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
