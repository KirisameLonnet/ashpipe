package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/KirisameLonnet/ashpipe/internal/sshfs"
	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:   "mount [portal-name]",
	Short: "Mount portal directories via SSHFS without starting an SSH shell",
	Long: `Mount portal directories as real local directories via SSHFS.

This is the agent-native path: once mounted, Claude Code and Codex can use
their built-in Read, Edit, Write, diff, and file search tools directly against
the portal directory without switching to ashpipe-specific tools.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, cfg, err := loadWorkspace()
		if err != nil {
			return err
		}
		cfg.WarnInsecure()

		if len(args) == 1 {
			return mountOne(root, cfg, args[0])
		}

		if len(cfg.Portals) == 0 {
			fmt.Println("No portals configured. Use `ashpipe add` to add one.")
			return nil
		}
		for name := range cfg.Portals {
			if err := mountOne(root, cfg, name); err != nil {
				return err
			}
		}
		return nil
	},
}

var unmountCmd = &cobra.Command{
	Use:   "unmount [portal-name]",
	Short: "Unmount portal directories",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, cfg, err := loadWorkspace()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			return unmountOne(root, args[0])
		}

		if len(cfg.Portals) == 0 {
			fmt.Println("No portals configured.")
			return nil
		}
		for name := range cfg.Portals {
			if err := unmountOne(root, name); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mountCmd, unmountCmd)
}

func loadWorkspace() (string, *config.Config, error) {
	cwd, _ := os.Getwd()
	root, err := config.FindRoot(cwd)
	if err != nil {
		return "", nil, fmt.Errorf("not in an ashpipe workspace (no .ashpipe/config.yaml found)")
	}
	cfg, err := config.Load(root)
	if err != nil {
		return "", nil, err
	}
	return root, cfg, nil
}

func mountOne(root string, cfg *config.Config, name string) error {
	portal, host, err := cfg.ResolvePortal(name)
	if err != nil {
		return err
	}

	localDir := filepath.Join(root, name)
	if sshfs.IsMounted(localDir) {
		fmt.Printf("[ashpipe] %s already mounted at %s\n", name, localDir)
		return nil
	}

	remotePath := resolveRemotePath(host, portal.RemotePath)
	fmt.Fprintf(os.Stderr, "[ashpipe] Mounting %s@%s:%s -> %s\n",
		host.User, host.Hostname, remotePath, localDir)
	if err := sshfs.Mount(host, remotePath, localDir); err != nil {
		return fmt.Errorf("mounting %s: %w", name, err)
	}
	return nil
}

func unmountOne(root string, name string) error {
	localDir := filepath.Join(root, name)
	if !sshfs.IsMounted(localDir) {
		fmt.Printf("[ashpipe] %s is not mounted\n", name)
		return nil
	}
	fmt.Fprintf(os.Stderr, "[ashpipe] Unmounting %s ...\n", localDir)
	if err := sshfs.Unmount(localDir); err != nil {
		return fmt.Errorf("unmounting %s: %w", name, err)
	}
	return nil
}
