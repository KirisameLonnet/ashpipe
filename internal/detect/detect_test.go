package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/KirisameLonnet/ashpipe/internal/detect"
	portalpath "github.com/KirisameLonnet/ashpipe/internal/portal"
)

func setupWorkspace(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("ASHPIPE_MOUNT_DIR", filepath.Join(t.TempDir(), "mounts"))

	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"prod-host": {Hostname: "server.example.com", User: "ubuntu"},
		},
		Portals: map[string]config.Portal{
			"remote-space-prod": {Host: "prod-host", RemotePath: "/opt/app"},
		},
	}
	if err := config.Save(root, cfg); err != nil {
		t.Fatalf("setup: %v", err)
	}
	mountDir := portalpath.MountDir(root, "remote-space-prod")
	if err := os.MkdirAll(filepath.Join(mountDir, "src"), 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	if err := os.Symlink(mountDir, portalpath.LinkDir(root, "remote-space-prod")); err != nil {
		t.Fatalf("setup symlink: %v", err)
	}
	return root
}

func TestDetectAtPortalRoot(t *testing.T) {
	root := setupWorkspace(t)
	portalDir := filepath.Join(root, "remote-space-prod")

	result, err := detect.FromDir(portalDir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if result == nil {
		t.Fatal("expected portal detection, got nil")
	}
	if result.PortalName != "remote-space-prod" {
		t.Errorf("PortalName = %q, want remote-space-prod", result.PortalName)
	}
	if result.Portal.RemotePath != "/opt/app" {
		t.Errorf("RemotePath = %q, want /opt/app", result.Portal.RemotePath)
	}
}

func TestDetectInsidePortalSubdir(t *testing.T) {
	root := setupWorkspace(t)
	subDir := filepath.Join(root, "remote-space-prod", "src")

	result, err := detect.FromDir(subDir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if result == nil {
		t.Fatal("expected portal detection inside subdir, got nil")
	}
	if result.PortalName != "remote-space-prod" {
		t.Errorf("PortalName = %q, want remote-space-prod", result.PortalName)
	}
}

func TestDetectInsideSymlinkedPortalSubdir(t *testing.T) {
	root := setupWorkspace(t)
	subDir := filepath.Join(root, "remote-space-prod", "src")

	result, err := detect.FromDir(subDir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if result == nil {
		t.Fatal("expected portal detection inside symlinked portal subdir, got nil")
	}
	if result.PortalName != "remote-space-prod" {
		t.Errorf("PortalName = %q, want remote-space-prod", result.PortalName)
	}
}

func TestDetectOutsidePortal(t *testing.T) {
	root := setupWorkspace(t)

	// At workspace root (not inside a portal dir).
	result, err := detect.FromDir(root)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil at workspace root, got portal %q", result.PortalName)
	}
}

func TestDetectNoWorkspace(t *testing.T) {
	dir := t.TempDir()
	result, err := detect.FromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil in non-workspace dir, got %+v", result)
	}
}
