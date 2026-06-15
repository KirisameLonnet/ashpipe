package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

func TestEnsurePortalLinkRejectsUnsafeName(t *testing.T) {
	if err := ensurePortalLink(t.TempDir(), "../escape"); err == nil {
		t.Fatal("expected unsafe portal name to be rejected")
	}
}

func TestSelectHostAliasDoesNotOverwriteDistinctCredentials(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"prod-host": {
				Hostname:     "example.com",
				User:         "alice",
				IdentityFile: "~/.ssh/id_ed25519",
			},
		},
		Portals: map[string]config.Portal{},
	}

	alias := selectHostAlias(cfg, "prod2", config.Host{
		Hostname:     "example.com",
		User:         "alice",
		IdentityFile: "~/.ssh/other",
	})
	if alias == "prod-host" {
		t.Fatal("expected a distinct host alias for distinct credentials")
	}
	if cfg.Hosts["prod-host"].IdentityFile != "~/.ssh/id_ed25519" {
		t.Fatal("existing host was mutated")
	}
}

func TestWriteAgentContextDoesNotOverwriteUserFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# user notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hosts: map[string]config.Host{}, Portals: map[string]config.Portal{}}
	if err := writeAgentContext(root, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# user notes\n" {
		t.Fatalf("user file was overwritten: %q", string(data))
	}
}

func TestWriteAgentContextOverwritesManagedFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte(agentContextMarker+"\nold\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"h": {Hostname: "example.com", User: "alice"},
		},
		Portals: map[string]config.Portal{
			"prod": {Host: "h", RemotePath: "/srv/app"},
		},
	}
	if err := writeAgentContext(root, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "**`prod/`**") {
		t.Fatalf("managed file was not regenerated: %q", string(data))
	}
}
