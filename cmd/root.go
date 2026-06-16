package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ashpipe",
	Short: "cd into a directory, get an SSH session — for humans and AI agents",
	Long: `ashpipe makes remote SSH hosts feel local.

  cd workspace/remote-space-prod/  →  terminal becomes a remote SSH session
  ashpipe mount                    →  mount portals for native agent file tools

Quick start:
  mkdir -p ~/workspaces/prod-servers
  cd ~/workspaces/prod-servers
  ashpipe init                        # create the stable workspace relationship
  ashpipe add prod ubuntu@host:/path
  ashpipe hook zsh                    # add output to your shell config once
  ashpipe mount                       # optional: mount portals for agents
  cd prod/                            # you're now on the remote host`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		if timeout > 0 {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			_ = cancel // leaked intentionally; context expires on its own
			cmd.SetContext(ctx)
		}
	},
}

func init() {
	var def time.Duration
	if v := os.Getenv("ASHPIPE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			def = d
		} else {
			fmt.Fprintf(os.Stderr, "[ashpipe] WARNING: invalid ASHPIPE_TIMEOUT=%q, using default %s\n", v, def)
		}
	}
	rootCmd.PersistentFlags().Duration("timeout", def, "maximum execution time (0 = no limit)")
}

func SetVersion(version string) {
	rootCmd.Version = version
}

func Execute() error {
	return rootCmd.Execute()
}
