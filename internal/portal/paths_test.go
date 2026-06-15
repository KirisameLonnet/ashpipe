package portal

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMountDirOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	mountDir := MountDir(root, "prod")
	if strings.HasPrefix(mountDir, root+string(filepath.Separator)) {
		t.Fatalf("MountDir() = %q, must be outside workspace %q", mountDir, root)
	}
}

func TestLinkDirInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	linkDir := LinkDir(root, "prod")
	want := filepath.Join(root, "prod")
	if linkDir != want {
		t.Fatalf("LinkDir() = %q, want %q", linkDir, want)
	}
}

func TestBaseMountDirDarwinUsesVolumes(t *testing.T) {
	got := baseMountDirFor("darwin", "alice")
	want := filepath.Join("/Volumes", "ashpipe")
	if got != want {
		t.Fatalf("baseMountDirFor(darwin) = %q, want %q", got, want)
	}
}

func TestBaseMountDirLinuxUsesMedia(t *testing.T) {
	got := baseMountDirFor("linux", "alice")
	want := filepath.Join("/media", "alice", "ashpipe")
	if got != want {
		t.Fatalf("baseMountDirFor(linux) = %q, want %q", got, want)
	}
}
