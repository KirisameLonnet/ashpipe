package sshfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

// Mount mounts host:remotePath at localDir via sshfs.
func Mount(h config.Host, remotePath, localDir string) error {
	if err := checkDeps(); err != nil {
		return err
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return err
	}

	sshfsArgs := buildArgs(h, remotePath, localDir)
	var cmd *exec.Cmd
	if h.Password != "" {
		if _, err := exec.LookPath("sshpass"); err != nil {
			return fmt.Errorf(
				"password auth requires sshpass (not found).\n"+
					"Install with: brew install sshpass  (macOS) / apt install sshpass  (Linux)\n"+
					"Or switch to SSH key auth (recommended).",
			)
		}
		cmd = exec.Command("sshpass", append([]string{"-p", h.Password, "sshfs"}, sshfsArgs...)...)
	} else {
		cmd = exec.Command("sshfs", sshfsArgs...)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sshfs mount failed: %w", err)
	}
	// Brief wait for mount to settle.
	time.Sleep(300 * time.Millisecond)
	return nil
}

// Unmount unmounts the given local directory.
func Unmount(localDir string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// macFUSE volumes must be unmounted with diskutil, not plain umount.
		cmd = exec.Command("diskutil", "unmount", localDir)
	default:
		// -z: lazy unmount — detaches immediately even if still busy.
		cmd = exec.Command("fusermount", "-uz", localDir)
	}
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsMounted returns true if localDir currently has an sshfs mount.
func IsMounted(localDir string) bool {
	out, err := exec.Command("mount").Output()
	if err != nil {
		return false
	}
	target, err := filepath.Abs(localDir)
	if err != nil {
		target = localDir
	}
	for _, line := range strings.Split(string(out), "\n") {
		if mountTarget(line) == target {
			return true
		}
	}
	return false
}

// MustNotBeMounted returns an error if localDir is currently a mount point.
// Call this immediately before any recursive delete operation.
func MustNotBeMounted(localDir string) error {
	if IsMounted(localDir) {
		return fmt.Errorf("%s is still mounted; refusing to delete recursively", localDir)
	}
	return nil
}

func mountTarget(line string) string {
	if line == "" {
		return ""
	}
	// macOS:
	//   user@host:/path on /mount/path (macfuse, ...)
	// Linux:
	//   user@host:/path on /mount/path type fuse.sshfs (...)
	_, rest, ok := strings.Cut(line, " on ")
	if !ok {
		return ""
	}
	for _, sep := range []string{" type ", " ("} {
		if target, _, ok := strings.Cut(rest, sep); ok {
			return target
		}
	}
	return ""
}

func buildArgs(h config.Host, remotePath, localDir string) []string {
	remote := fmt.Sprintf("%s@%s:%s", h.User, h.Hostname, remotePath)
	args := []string{
		remote, localDir,
		"-o", fmt.Sprintf("port=%d", h.Port),
		"-o", "reconnect",
		"-o", "ServerAliveInterval=15",
		"-o", "ServerAliveCountMax=3",
	}
	if h.IdentityFile != "" {
		args = append(args, "-o", "IdentityFile="+expandHome(h.IdentityFile))
	}
	return args
}

func checkDeps() error {
	if _, err := exec.LookPath("sshfs"); err != nil {
		switch runtime.GOOS {
		case "darwin":
			return fmt.Errorf(
				"sshfs not found. Install macFUSE from https://osxfuse.github.io and then:\n  brew install sshfs",
			)
		default:
			return fmt.Errorf("sshfs not found. Install with: sudo apt install sshfs  (or equivalent)")
		}
	}
	return nil
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
