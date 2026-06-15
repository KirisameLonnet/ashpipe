package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

func TestInitAndLoad(t *testing.T) {
	dir := t.TempDir()

	if err := config.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Second call should be a no-op (file already exists).
	if err := config.Init(dir); err != nil {
		t.Fatalf("Init (second): %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Hosts == nil || cfg.Portals == nil {
		t.Fatal("expected non-nil maps")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"prod-host": {
				Hostname:     "server.example.com",
				User:         "ubuntu",
				IdentityFile: "~/.ssh/id_ed25519",
				Port:         22,
			},
		},
		Portals: map[string]config.Portal{
			"remote-space-prod": {
				Host:       "prod-host",
				RemotePath: "/opt/app",
			},
		},
	}

	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	h, ok := loaded.Hosts["prod-host"]
	if !ok {
		t.Fatal("host prod-host not found")
	}
	if h.Hostname != "server.example.com" {
		t.Errorf("hostname = %q, want %q", h.Hostname, "server.example.com")
	}

	p, ok := loaded.Portals["remote-space-prod"]
	if !ok {
		t.Fatal("portal remote-space-prod not found")
	}
	if p.RemotePath != "/opt/app" {
		t.Errorf("remote_path = %q, want %q", p.RemotePath, "/opt/app")
	}
}

func TestFindRoot(t *testing.T) {
	root := t.TempDir()
	if err := config.Init(root); err != nil {
		t.Fatal(err)
	}

	// Create a nested subdirectory.
	sub := filepath.Join(root, "portal", "src", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := config.FindRoot(sub)
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	if found != root {
		t.Errorf("FindRoot = %q, want %q", found, root)
	}
}

func TestFindRootNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := config.FindRoot(dir)
	if err == nil {
		t.Error("expected error when no .ashpipe found")
	}
}

func TestResolvePortal(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"h": {Hostname: "host.example.com", User: "user"},
		},
		Portals: map[string]config.Portal{
			"myportal": {Host: "h", RemotePath: "/srv"},
		},
	}

	portal, host, err := cfg.ResolvePortal("myportal")
	if err != nil {
		t.Fatalf("ResolvePortal: %v", err)
	}
	if portal.RemotePath != "/srv" {
		t.Errorf("remote_path = %q, want /srv", portal.RemotePath)
	}
	if host.Port != 22 {
		t.Errorf("port = %d, want 22 (default)", host.Port)
	}
}

func TestResolvePortalMissingHost(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{},
		Portals: map[string]config.Portal{
			"p": {Host: "nonexistent", RemotePath: "/"},
		},
	}
	_, _, err := cfg.ResolvePortal("p")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestPasswordWarning(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"legacy": {Hostname: "old.example.com", User: "admin", Password: "secret"},
		},
		Portals: map[string]config.Portal{},
	}
	// WarnInsecure writes to stderr — just verify it doesn't panic.
	cfg.WarnInsecure()
}
