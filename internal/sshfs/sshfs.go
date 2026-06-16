package sshfs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

const defaultProbeTimeout = 5 * time.Second

type MountHealth int

const (
	Unmounted MountHealth = iota
	Healthy
	Stale
)

func (h MountHealth) String() string {
	switch h {
	case Healthy:
		return "mounted"
	case Stale:
		return "stale"
	default:
		return "unmounted"
	}
}

// Probe checks whether a mounted SSHFS path is actually responsive.
// A stale FUSE mount blocks os.Stat indefinitely, so we run it in a
// goroutine and race against a timeout. The goroutine may leak on
// timeout — this is unavoidable with blocked kernel VFS calls and is
// accepted practice (see rclone, nfs-mountpoint-check).
func Probe(ctx context.Context, localDir string) MountHealth {
	if !IsMounted(localDir) {
		return Unmounted
	}
	deadline, ok := ctx.Deadline()
	timeout := defaultProbeTimeout
	if ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		return Stale
	}
	ch := make(chan error, 1)
	go func() {
		_, err := os.ReadDir(localDir)
		ch <- err
	}()
	select {
	case err := <-ch:
		if err != nil {
			return Stale
		}
		return Healthy
	case <-time.After(timeout):
		return Stale
	}
}

// Mount mounts host:remotePath at localDir via sshfs.
func Mount(ctx context.Context, h config.Host, remotePath, localDir string) error {
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
				"password auth requires sshpass (not found).\n" +
					"Install with: brew install sshpass  (macOS) / apt install sshpass  (Linux)\n" +
					"Or switch to SSH key auth (recommended).",
			)
		}
		cmd = exec.CommandContext(ctx, "sshpass", append([]string{"-e", "sshfs"}, sshfsArgs...)...)
		cmd.Env = append(os.Environ(), "SSHPASS="+h.Password)
	} else {
		cmd = exec.CommandContext(ctx, "sshfs", sshfsArgs...)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("sshfs mount timed out: %w", ctx.Err())
		}
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
	if isMountPointByStat(localDir) {
		return true
	}
	out, err := exec.Command("mount").Output()
	if err != nil {
		return false
	}
	target, err := filepath.Abs(localDir)
	if err != nil {
		target = localDir
	}
	for _, line := range strings.Split(string(out), "\n") {
		mountPath := mountTarget(line)
		if mountPath == target {
			return true
		}
		if abs, err := filepath.Abs(mountPath); err == nil && abs == target {
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
			return unescapeMountPath(target)
		}
	}
	return ""
}

func unescapeMountPath(path string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(path)
}

func isMountPointByStat(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	parent := filepath.Dir(path)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	parentStat, ok := parentInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return stat.Dev != parentStat.Dev || stat.Ino == parentStat.Ino
}

func buildArgs(h config.Host, remotePath, localDir string) []string {
	remote := fmt.Sprintf("%s@%s:%s", h.User, h.Hostname, remotePath)
	args := []string{
		remote, localDir,
		"-o", fmt.Sprintf("port=%d", h.Port),
		"-o", "reconnect",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "ConnectTimeout=10",
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
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return path
		}
		return home + path[1:]
	}
	return path
}
