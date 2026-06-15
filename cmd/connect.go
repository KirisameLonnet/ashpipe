package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/KirisameLonnet/ashpipe/internal/detect"
	"github.com/KirisameLonnet/ashpipe/internal/sshfs"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect <portal-name>",
	Short: "Connect to a portal (mount SSHFS + hand terminal to remote SSH session)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		portalName := args[0]

		cwd, _ := os.Getwd()
		root, err := func() (string, error) {
			if r, err := detect.FromCwd(); err == nil && r != nil {
				return r.WorkspaceRoot, nil
			}
			return config.FindRoot(cwd)
		}()
		if err != nil {
			return fmt.Errorf("not in an ashpipe workspace")
		}

		cfg, err := config.Load(root)
		if err != nil {
			return err
		}
		cfg.WarnInsecure()

		portal, host, err := cfg.ResolvePortal(portalName)
		if err != nil {
			return err
		}

		localDir := filepath.Join(root, portalName)
		return connectPortal(host, portal, localDir)
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

func connectPortal(host config.Host, portal config.Portal, localDir string) error {
	// Mount SSHFS for file transparency. If it fails (e.g. host runs dropbear
	// without sftp-server), warn and continue — shell access still works.
	sshfsMounted := sshfs.IsMounted(localDir)
	if !sshfsMounted {
		fmt.Fprintf(os.Stderr, "[ashpipe] Mounting %s@%s:%s → %s\n",
			host.User, host.Hostname, portal.RemotePath, localDir)
		if err := sshfs.Mount(host, portal.RemotePath, localDir); err != nil {
			fmt.Fprintf(os.Stderr,
				"[ashpipe] WARNING: SSHFS mount failed (%v)\n"+
					"          File transparency unavailable — portal directory will be empty.\n"+
					"          Shell session will still work normally.\n",
				err,
			)
		} else {
			sshfsMounted = true
		}
	}

	sshPath, err := findSSH()
	if err != nil {
		return err
	}

	argv := buildSSHArgv(host, portal.RemotePath)
	fmt.Fprintf(os.Stderr, "[ashpipe] Connecting to %s@%s ...\n", host.User, host.Hostname)

	// Run SSH as a child process so we can unmount SSHFS after the session ends.
	// Attach the terminal directly so the user experience is identical to plain ssh.
	sshCmd := exec.Command(sshPath, argv...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Start(); err != nil {
		return fmt.Errorf("starting ssh: %w", err)
	}

	// Forward SIGWINCH (terminal resize) to the SSH child process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for sig := range sigCh {
			if sshCmd.Process != nil {
				_ = sshCmd.Process.Signal(sig)
			}
		}
	}()

	runErr := sshCmd.Wait()

	signal.Stop(sigCh)
	close(sigCh)

	// Unmount SSHFS only if we mounted it successfully.
	if sshfsMounted && sshfs.IsMounted(localDir) {
		fmt.Fprintf(os.Stderr, "\n[ashpipe] Unmounting %s ...\n", localDir)
		if err := sshfs.Unmount(localDir); err != nil {
			fmt.Fprintf(os.Stderr, "[ashpipe] Warning: unmount failed: %v\n", err)
		}
	}

	// A non-zero SSH exit is normal (e.g. user typed `exit 1`) — don't surface as error.
	_ = runErr
	return nil
}

func buildSSHArgv(host config.Host, remotePath string) []string {
	args := []string{
		"-t",
		"-p", fmt.Sprintf("%d", host.Port),
		"-o", "StrictHostKeyChecking=yes",
	}
	if host.IdentityFile != "" {
		args = append(args, "-i", expandSSHHome(host.IdentityFile))
	}
	args = append(args,
		fmt.Sprintf("%s@%s", host.User, host.Hostname),
		fmt.Sprintf("cd %q && exec $SHELL -l", remotePath),
	)
	return args
}

func findSSH() (string, error) {
	// Prefer known absolute paths to avoid shell aliases (e.g. kitten ssh, custom wrappers).
	for _, p := range []string{
		"/usr/bin/ssh",
		"/usr/local/bin/ssh",
		"/opt/homebrew/bin/ssh",
		"/nix/var/nix/profiles/default/bin/ssh",
	} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	// Last resort: resolve via PATH, but only accept real binaries (not scripts).
	if p, err := exec.LookPath("ssh"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("ssh binary not found; ensure openssh-client is installed")
}

func expandSSHHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
