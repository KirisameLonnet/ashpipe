package cmd

import (
	"fmt"
	"os"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/KirisameLonnet/ashpipe/internal/sshfs"
	"github.com/spf13/cobra"
	"path/filepath"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show portal status for the current workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		root, err := config.FindRoot(cwd)
		if err != nil {
			return fmt.Errorf("not in an ashpipe workspace (no .ashpipe/config.yaml found)")
		}

		cfg, err := config.Load(root)
		if err != nil {
			return err
		}
		cfg.WarnInsecure()

		fmt.Printf("Workspace: %s\n\n", root)
		if len(cfg.Portals) == 0 {
			fmt.Println("No portals configured. Use `ashpipe add` to add one.")
			return nil
		}

		for name, portal := range cfg.Portals {
			host := cfg.Hosts[portal.Host]
			localDir := filepath.Join(root, name)
			mounted := sshfs.IsMounted(localDir)
			mountStatus := "unmounted"
			if mounted {
				mountStatus = "mounted"
			}
			fmt.Printf("  %-20s %s@%s:%s  [%s]\n",
				name, host.User, host.Hostname, portal.RemotePath, mountStatus)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
