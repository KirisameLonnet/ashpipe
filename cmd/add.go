package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <portal-name> <user@host:remote-path>",
	Short: "Add a portal to the current workspace",
	Example: `  ashpipe add prod ubuntu@server.com:/opt/app remote-space-prod
  ashpipe add dev developer@dev.example.com:/home/developer/project`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		portalName := args[0]
		target := args[1]

		userHost, remotePath, ok := strings.Cut(target, ":")
		if !ok {
			return fmt.Errorf("expected <user@host:remote-path>, got %q", target)
		}
		user, hostname, ok := strings.Cut(userHost, "@")
		if !ok {
			return fmt.Errorf("expected <user@host>, got %q", userHost)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Try to find existing workspace root; if not found, use cwd.
		root, err := config.FindRoot(cwd)
		if err != nil {
			root = cwd
		}

		cfg, loadErr := config.Load(root)
		if loadErr != nil {
			cfg = &config.Config{
				Hosts:   map[string]config.Host{},
				Portals: map[string]config.Portal{},
			}
		}

		// Derive a host alias from hostname.
		hostAlias := portalName + "-host"
		// Reuse existing host entry if hostname matches.
		for alias, h := range cfg.Hosts {
			if h.Hostname == hostname && h.User == user {
				hostAlias = alias
				break
			}
		}

		identityFile, _ := cmd.Flags().GetString("identity-file")
		password, _ := cmd.Flags().GetString("password")

		cfg.Hosts[hostAlias] = config.Host{
			Hostname:     hostname,
			User:         user,
			IdentityFile: identityFile,
			Password:     password,
		}
		cfg.Portals[portalName] = config.Portal{
			Host:       hostAlias,
			RemotePath: remotePath,
		}

		if err := config.Save(root, cfg); err != nil {
			return err
		}

		// Create the portal directory.
		portalDir := filepath.Join(root, portalName)
		if err := os.MkdirAll(portalDir, 0o755); err != nil {
			return err
		}

		// Generate CLAUDE.md and AGENTS.md at workspace root.
		if err := writeAgentContext(root, cfg); err != nil {
			return err
		}

		fmt.Printf("Portal %q added → %s@%s:%s\n", portalName, user, hostname, remotePath)
		if password != "" {
			fmt.Fprintf(os.Stderr,
				"[ashpipe] WARNING: password auth for host %q is insecure. Consider using SSH key or agent instead.\n",
				hostAlias,
			)
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringP("identity-file", "i", "", "SSH private key path")
	addCmd.Flags().StringP("password", "p", "", "SSH password (insecure)")
	rootCmd.AddCommand(addCmd)
}

func writeAgentContext(root string, cfg *config.Config) error {
	var sb strings.Builder
	sb.WriteString("# ashpipe Remote Workspace\n\n")
	sb.WriteString("This workspace contains remote portal directories managed by ashpipe.\n\n")
	sb.WriteString("## Portals\n\n")
	for name, p := range cfg.Portals {
		h := cfg.Hosts[p.Host]
		sb.WriteString(fmt.Sprintf("- **`%s/`** → `%s@%s:%s`\n", name, h.User, h.Hostname, p.RemotePath))
	}
	sb.WriteString("\n## Agent Instructions\n\n")
	sb.WriteString("When working inside a portal directory:\n")
	sb.WriteString("- Use `ssh:bash` for running commands (they execute on the remote host)\n")
	sb.WriteString("- Use `ssh:read`, `ssh:write`, `ssh:edit` for file operations\n")
	sb.WriteString("- File paths are automatically translated from local portal path to remote path\n")
	sb.WriteString("- The portal directory is also SSHFS-mounted, so native Read/Write tools work transparently\n")

	content := sb.String()
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
