package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	portalpath "github.com/KirisameLonnet/ashpipe/internal/portal"
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

		ctx := cmd.Context()

		if len(args) == 1 {
			return mountOne(ctx, root, cfg, args[0])
		}

		if len(cfg.Portals) == 0 {
			fmt.Println("No portals configured. Use `ashpipe add` to add one.")
			return nil
		}
		var errs []error
		for name := range cfg.Portals {
			if ctx.Err() != nil {
				errs = append(errs, ctx.Err())
				break
			}
			if err := mountOne(ctx, root, cfg, name); err != nil {
				fmt.Fprintf(os.Stderr, "[ashpipe] %s: %v\n", name, err)
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
		return errors.Join(errs...)
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
			if _, ok := cfg.Portals[args[0]]; !ok {
				return fmt.Errorf("portal %q not found", args[0])
			}
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

func mountOne(ctx context.Context, root string, cfg *config.Config, name string) error {
	portal, host, err := cfg.ResolvePortal(name)
	if err != nil {
		return err
	}

	if err := ensurePortalLink(root, name); err != nil {
		return err
	}

	mountDir := portalpath.MountDir(root, name)

	switch sshfs.Probe(ctx, mountDir) {
	case sshfs.Healthy:
		fmt.Printf("[ashpipe] %s already mounted at %s\n", name, mountDir)
		return nil
	case sshfs.Stale:
		fmt.Fprintf(os.Stderr, "[ashpipe] %s is stale, remounting ...\n", name)
		_ = sshfs.Unmount(mountDir)
	}

	remotePath := resolveRemotePath(host, portal.RemotePath)
	fmt.Fprintf(os.Stderr, "[ashpipe] Mounting %s@%s:%s -> %s\n",
		host.User, host.Hostname, remotePath, mountDir)
	if err := sshfs.Mount(ctx, host, remotePath, mountDir); err != nil {
		return fmt.Errorf("sshfs: %w", err)
	}

	if sshfs.Probe(ctx, mountDir) != sshfs.Healthy {
		return fmt.Errorf("mounted but not responsive (remote path may not exist)")
	}
	return nil
}

func unmountOne(root string, name string) error {
	mountDir := portalpath.MountDir(root, name)
	if !sshfs.IsMounted(mountDir) {
		fmt.Printf("[ashpipe] %s is not mounted\n", name)
		return nil
	}
	fmt.Fprintf(os.Stderr, "[ashpipe] Unmounting %s ...\n", mountDir)
	if err := sshfs.Unmount(mountDir); err != nil {
		return fmt.Errorf("unmounting %s: %w", name, err)
	}
	return nil
}
