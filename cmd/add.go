package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	portalpath "github.com/KirisameLonnet/ashpipe/internal/portal"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <portal-name> <user@host:remote-path>",
	Short: "Add a portal to the current workspace",
	Example: `  ashpipe add prod ubuntu@server.com:/opt/app
  ashpipe add dev developer@dev.example.com:/home/developer/project`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		portalName := args[0]
		target := args[1]
		if err := portalpath.ValidateName(portalName); err != nil {
			return err
		}

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
		identityFile, _ := cmd.Flags().GetString("identity-file")
		password, _ := cmd.Flags().GetString("password")
		newHost := config.Host{
			Hostname:     hostname,
			User:         user,
			IdentityFile: identityFile,
			Password:     password,
		}

		hostAlias := selectHostAlias(cfg, portalName, newHost)
		if _, ok := cfg.Hosts[hostAlias]; !ok {
			cfg.Hosts[hostAlias] = newHost
		}
		cfg.Portals[portalName] = config.Portal{
			Host:       hostAlias,
			RemotePath: remotePath,
		}

		if err := ensurePortalLink(root, portalName); err != nil {
			return err
		}

		if err := config.Save(root, cfg); err != nil {
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

func ensurePortalLink(root, name string) error {
	if err := portalpath.ValidateName(name); err != nil {
		return err
	}
	mountDir := portalpath.MountDir(root, name)
	linkDir := portalpath.LinkDir(root, name)
	if err := os.MkdirAll(mountDir, 0o755); err != nil {
		return err
	}
	info, err := os.Lstat(linkDir)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(linkDir)
			if err != nil {
				return err
			}
			if target == mountDir {
				return nil
			}
			return fmt.Errorf("%s already exists as a symlink to %s, not %s", linkDir, target, mountDir)
		}
		if info.IsDir() {
			if err := os.Remove(linkDir); err == nil {
				return os.Symlink(mountDir, linkDir)
			}
		}
		return fmt.Errorf("%s already exists and is not an ashpipe symlink", linkDir)
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(mountDir, linkDir)
}

func selectHostAlias(cfg *config.Config, portalName string, newHost config.Host) string {
	type candidate struct {
		alias string
		host  config.Host
	}
	var candidates []candidate
	for alias, h := range cfg.Hosts {
		if sameEndpoint(h, newHost) {
			candidates = append(candidates, candidate{alias: alias, host: h})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].alias < candidates[j].alias
	})
	for _, c := range candidates {
		if canReuseHost(c.host, newHost) {
			return c.alias
		}
	}

	base := portalName + "-host"
	alias := base
	for i := 2; ; i++ {
		if _, exists := cfg.Hosts[alias]; !exists {
			return alias
		}
		alias = fmt.Sprintf("%s-%d", base, i)
	}
}

func sameEndpoint(a, b config.Host) bool {
	return a.Hostname == b.Hostname && a.User == b.User && effectivePort(a.Port) == effectivePort(b.Port)
}

func canReuseHost(existing, requested config.Host) bool {
	if !sameEndpoint(existing, requested) {
		return false
	}
	if requested.IdentityFile == "" && requested.Password == "" {
		return true
	}
	return existing.IdentityFile == requested.IdentityFile && existing.Password == requested.Password
}

func effectivePort(port int) int {
	if port == 0 {
		return 22
	}
	return port
}

func init() {
	addCmd.Flags().StringP("identity-file", "i", "", "SSH private key path")
	addCmd.Flags().StringP("password", "p", "", "SSH password (insecure)")
	rootCmd.AddCommand(addCmd)
}

const agentContextMarker = "<!-- ashpipe:managed -->"

func writeAgentContext(root string, cfg *config.Config) error {
	var sb strings.Builder
	sb.WriteString(agentContextMarker)
	sb.WriteString("\n")
	sb.WriteString("# ashpipe Remote Workspace\n\n")
	sb.WriteString("This workspace contains remote portal directories managed by ashpipe. Public portal paths are symlinks to private SSHFS mount points outside the workspace.\n\n")
	sb.WriteString("## Safety Warning\n\n")
	sb.WriteString("- Do not manually delete ashpipe portal paths or mount directories with `rm -rf`.\n")
	sb.WriteString("- Use `ashpipe unmount` and `ashpipe remove`; ashpipe manages portal symlinks and SSHFS mount points.\n")
	sb.WriteString("- If manual cleanup is unavoidable, unmount first and verify the path is no longer a mount point. Deleting a live SSHFS/FUSE mount can delete remote files.\n\n")
	sb.WriteString("## Portals\n\n")
	names := make([]string, 0, len(cfg.Portals))
	for name := range cfg.Portals {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := cfg.Portals[name]
		h := cfg.Hosts[p.Host]
		sb.WriteString(fmt.Sprintf("- **`%s/`** → `%s@%s:%s`\n", name, h.User, h.Hostname, p.RemotePath))
	}
	sb.WriteString("\n## Agent Instructions\n\n")
	sb.WriteString("- Treat portal directories as ordinary workspace directories. Use the default built-in file tools for read, edit, write, search, and diff.\n")
	sb.WriteString("- Do not switch to ashpipe-specific tools for normal file work. If a portal is not mounted, ask to run `ashpipe mount` from the workspace root.\n")
	sb.WriteString("- Shell commands run by the agent still execute locally unless explicitly run through a remote shell. Prefer native file operations for code changes inside mounted portals.\n")

	content := sb.String()
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		path := filepath.Join(root, name)
		if existing, err := os.ReadFile(path); err == nil && !isManagedAgentContext(string(existing)) {
			fmt.Fprintf(os.Stderr, "[ashpipe] %s already exists and is not ashpipe-managed; leaving it unchanged\n", path)
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func isManagedAgentContext(content string) bool {
	return strings.Contains(content, agentContextMarker) ||
		strings.HasPrefix(content, "# ashpipe Remote Workspace\n\nThis workspace contains remote portal directories managed by ashpipe.")
}
