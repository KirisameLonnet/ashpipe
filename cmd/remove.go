package cmd

import (
	"fmt"
	"os"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	portalpath "github.com/KirisameLonnet/ashpipe/internal/portal"
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

		p, ok := cfg.Portals[name]
		if !ok {
			return fmt.Errorf("portal %q not found", name)
		}

		linkDir := portalpath.LinkDir(root, name)
		mountDir := portalpath.MountDir(root, name)

		if sshfs.IsMounted(mountDir) {
			fmt.Fprintf(os.Stderr, "[ashpipe] Unmounting %s ...\n", mountDir)
			if err := sshfs.Unmount(mountDir); err != nil {
				return fmt.Errorf("unmount failed: %w", err)
			}
		}
		if err := sshfs.MustNotBeMounted(mountDir); err != nil {
			return err
		}

		if err := removePortalLink(linkDir, force); err != nil {
			return err
		}
		if err := removePrivateMountDir(mountDir, force); err != nil {
			return err
		}

		// Legacy defense: older ashpipe versions used the public portal path as
		// the mount point. Never recurse into it while mounted.
		if err := sshfs.MustNotBeMounted(linkDir); err != nil {
			return err
		}

		hostAlias := p.Host
		delete(cfg.Portals, name)

		hostStillUsed := false
		for _, remaining := range cfg.Portals {
			if remaining.Host == hostAlias {
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
		if err := writeAgentContext(root, cfg); err != nil {
			return err
		}

		_ = os.Remove(portalpath.MountsDir(root)) // ok if non-empty
		fmt.Printf("Portal %q removed.\n", name)
		return nil
	},
}

func init() {
	removeCmd.Flags().Bool("force", false, "Delete portal symlink and private mount directory even if not empty")
	rootCmd.AddCommand(removeCmd)
}

func removePortalLink(path string, force bool) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}
	if !force {
		fmt.Fprintf(os.Stderr,
			"[ashpipe] Portal path %s is not an ashpipe symlink, keeping it (use --force to delete)\n",
			path,
		)
		return nil
	}
	if err := sshfs.MustNotBeMounted(path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func removePrivateMountDir(path string, force bool) error {
	if err := sshfs.MustNotBeMounted(path); err != nil {
		return err
	}
	if force {
		return os.RemoveAll(path)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr,
			"[ashpipe] Private mount directory %s is not empty, keeping it (use --force to delete)\n",
			path,
		)
	}
	return nil
}
