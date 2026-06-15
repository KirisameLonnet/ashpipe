package detect

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	portalpath "github.com/KirisameLonnet/ashpipe/internal/portal"
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

	realAbs, err := filepath.EvalSymlinks(abs)
	if err != nil {
		realAbs = abs
	}

	// Check if abs is inside one of the user-facing portal directories or their
	// private mount directories.
	for name := range cfg.Portals {
		if pathInside(portalpath.LinkDir(root, name), abs) || pathInside(portalpath.MountDir(root, name), realAbs) {
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

func pathInside(base, path string) bool {
	base = canonicalPath(base)
	path = canonicalPath(path)
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func canonicalPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return real
}

// FromCwd detects portal from the current working directory.
func FromCwd() (*Result, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FromDir(cwd)
}
