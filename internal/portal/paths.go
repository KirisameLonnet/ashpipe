package portal

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
)

// LinkDir is the user-facing portal path in the workspace.
func LinkDir(root, name string) string {
	return filepath.Join(root, name)
}

// MountDir is the private SSHFS mount point. It deliberately lives outside the
// workspace, in a system location normally associated with mounted volumes, so
// accidental `rm -rf <portal>` or `rm -rf .ashpipe` cannot recurse into a
// mounted remote tree.
func MountDir(root, name string) string {
	return filepath.Join(MountsDir(root), name)
}

func MountsDir(root string) string {
	return filepath.Join(baseMountDir(), workspaceID(root))
}

func baseMountDir() string {
	if dir := os.Getenv("ASHPIPE_MOUNT_DIR"); dir != "" {
		return dir
	}
	return baseMountDirFor(runtime.GOOS, currentUserName())
}

func baseMountDirFor(goos, user string) string {
	switch goos {
	case "darwin":
		return filepath.Join("/Volumes", "ashpipe")
	case "linux":
		if user != "" {
			return filepath.Join("/media", user, "ashpipe")
		}
		return filepath.Join("/mnt", "ashpipe")
	default:
		return filepath.Join(os.TempDir(), "ashpipe", "mounts")
	}
}

func currentUserName() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("LOGNAME"); user != "" {
		return user
	}
	return ""
}

func workspaceID(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:])[:16]
}
