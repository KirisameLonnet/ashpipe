package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configDir = ".ashpipe"
const configFile = "config.yaml"

type Host struct {
	Hostname     string `yaml:"hostname"`
	User         string `yaml:"user"`
	IdentityFile string `yaml:"identity_file,omitempty"`
	Password     string `yaml:"password,omitempty"`
	Port         int    `yaml:"port,omitempty"`
}

type Portal struct {
	Host       string `yaml:"host"`
	RemotePath string `yaml:"remote_path"`
}

type Config struct {
	Hosts   map[string]Host   `yaml:"hosts"`
	Portals map[string]Portal `yaml:"portals"`
}

// FindRoot walks up from dir looking for .ashpipe/config.yaml, returns the root dir.
func FindRoot(dir string) (string, error) {
	for {
		candidate := filepath.Join(dir, configDir, configFile)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .ashpipe/config.yaml found in %s or any parent directory", dir)
		}
		dir = parent
	}
}

// Load reads config from the given workspace root.
func Load(root string) (*Config, error) {
	path := filepath.Join(root, configDir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = map[string]Host{}
	}
	if cfg.Portals == nil {
		cfg.Portals = map[string]Portal{}
	}
	return &cfg, nil
}

// Save writes config to the given workspace root.
func Save(root string, cfg *Config) error {
	dir := filepath.Join(root, configDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0o600)
}

// Init creates an empty config at root if one doesn't exist.
func Init(root string) error {
	path := filepath.Join(root, configDir, configFile)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return Save(root, &Config{
		Hosts:   map[string]Host{},
		Portals: map[string]Portal{},
	})
}

// ResolvePortal returns the host and portal config for the given portal name.
func (c *Config) ResolvePortal(name string) (Portal, Host, error) {
	portal, ok := c.Portals[name]
	if !ok {
		return Portal{}, Host{}, fmt.Errorf("portal %q not found", name)
	}
	host, ok := c.Hosts[portal.Host]
	if !ok {
		return Portal{}, Host{}, fmt.Errorf("host %q (referenced by portal %q) not found", portal.Host, name)
	}
	if host.Port == 0 {
		host.Port = 22
	}
	return portal, host, nil
}

// WarnInsecure prints a warning if any host uses password auth.
func (c *Config) WarnInsecure() {
	for name, host := range c.Hosts {
		if host.Password != "" {
			fmt.Fprintf(os.Stderr,
				"[ashpipe] WARNING: password auth for host %q is insecure. Consider using SSH key or agent instead.\n",
				name,
			)
		}
	}
}
