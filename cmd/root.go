package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ashpipe",
	Short: "cd into a directory, get an SSH session — for humans and AI agents",
	Long: `ashpipe makes remote SSH hosts feel local.

  cd workspace/remote-space-prod/  →  terminal becomes a remote SSH session
  ashpipe mcp                      →  MCP server for Claude Code / Codex

Quick start:
  ashpipe hook zsh                    # add output to your shell config
  ashpipe add prod ubuntu@host:/path remote-space-prod
  cd remote-space-prod/               # you're now on the remote host`,
}

func Execute() error {
	return rootCmd.Execute()
}
