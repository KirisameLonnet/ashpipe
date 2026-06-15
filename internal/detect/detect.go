package detect

import (
	"os"
	"path/filepath"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

type Result struct {
	WorkspaceRoot string
	PortalName    string
	Portal        config.Portal
	Host          config.Host
}

// FromDir detects if dir is inside a portal, returns the portal info.
// Returns nil if dir is not inside any portal.
func FromDir(dir string) (*Result, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	root, err := config.FindRoot(abs)
	if err != nil {
		return nil, nil // not in an ashpipe workspace
	}

	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}

	// Check if abs is inside one of the portal directories.
	for name := range cfg.Portals {
		portalDir := filepath.Join(root, name)
		rel, err := filepath.Rel(portalDir, abs)
		if err != nil {
			continue
		}
		// rel starts with ".." means abs is not inside portalDir
		if len(rel) > 0 && rel[0] != '.' {
			portal, host, err := cfg.ResolvePortal(name)
			if err != nil {
				return nil, err
			}
			return &Result{
				WorkspaceRoot: root,
				PortalName:    name,
				Portal:        portal,
				Host:          host,
			}, nil
		}
		// exact match (rel == ".")
		if rel == "." {
			portal, host, err := cfg.ResolvePortal(name)
			if err != nil {
				return nil, err
			}
			return &Result{
				WorkspaceRoot: root,
				PortalName:    name,
				Portal:        portal,
				Host:          host,
			}, nil
		}
	}
	return nil, nil
}

// FromCwd detects portal from the current working directory.
func FromCwd() (*Result, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FromDir(cwd)
}
