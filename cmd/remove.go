package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/KirisameLonnet/ashpipe/internal/sshfs"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <portal-name>",
	Aliases: []string{"rm"},
	Short:   "Remove a portal from the current workspace",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		cwd, _ := os.Getwd()
		root, err := config.FindRoot(cwd)
		if err != nil {
			return fmt.Errorf("not in an ashpipe workspace")
		}

		cfg, err := config.Load(root)
		if err != nil {
			return err
		}

		portal, ok := cfg.Portals[name]
		if !ok {
			return fmt.Errorf("portal %q not found", name)
		}

		localDir := filepath.Join(root, name)

		// Unmount SSHFS if currently mounted.
		if sshfs.IsMounted(localDir) {
			fmt.Fprintf(os.Stderr, "[ashpipe] Unmounting %s ...\n", localDir)
			if err := sshfs.Unmount(localDir); err != nil {
				return fmt.Errorf("unmount failed: %w", err)
			}
		}

		// Remove portal directory if empty (or --force).
		if force {
			if err := os.RemoveAll(localDir); err != nil {
				return fmt.Errorf("removing portal directory: %w", err)
			}
		} else {
			if err := os.Remove(localDir); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr,
					"[ashpipe] Portal directory %s is not empty, keeping it (use --force to delete)\n",
					localDir,
				)
			}
		}

		// Remove portal config entry.
		hostAlias := portal.Host
		delete(cfg.Portals, name)

		// Remove orphaned host entry (not referenced by any remaining portal).
		hostStillUsed := false
		for _, p := range cfg.Portals {
			if p.Host == hostAlias {
				hostStillUsed = true
				break
			}
		}
		if !hostStillUsed {
			delete(cfg.Hosts, hostAlias)
		}

		if err := config.Save(root, cfg); err != nil {
			return err
		}

		// Regenerate CLAUDE.md / AGENTS.md.
		if err := writeAgentContext(root, cfg); err != nil {
			return err
		}

		fmt.Printf("Portal %q removed.\n", name)
		return nil
	},
}

func init() {
	removeCmd.Flags().Bool("force", false, "Delete portal directory even if not empty")
	rootCmd.AddCommand(removeCmd)
}
