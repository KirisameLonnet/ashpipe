package pool

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Pool struct {
	mu    sync.Mutex
	conns map[string]*ssh.Client
}

var Default = &Pool{conns: map[string]*ssh.Client{}}

func (p *Pool) Get(h config.Host) (*ssh.Client, error) {
	key := fmt.Sprintf("%s@%s:%d", h.User, h.Hostname, h.Port)
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.conns[key]; ok {
		// Probe connection with a keepalive.
		if _, _, err := c.SendRequest("keepalive@ashpipe", true, nil); err == nil {
			return c, nil
		}
		c.Close()
		delete(p.conns, key)
	}

	c, err := dial(h)
	if err != nil {
		return nil, err
	}
	p.conns[key] = c
	return c, nil
}

func (p *Pool) Close(h config.Host) {
	key := fmt.Sprintf("%s@%s:%d", h.User, h.Hostname, h.Port)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.conns[key]; ok {
		c.Close()
		delete(p.conns, key)
	}
}

func dial(h config.Host) (*ssh.Client, error) {
	authMethods, err := buildAuthMethods(h)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := knownHostsCallback()
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            h.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		// Explicit algorithm order ensures we negotiate the same key type
		// that is stored in known_hosts, avoiding "key mismatch" errors
		// when the server (e.g. dropbear) offers multiple key types.
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoECDSA256,
			ssh.KeyAlgoECDSA384,
			ssh.KeyAlgoECDSA521,
			ssh.KeyAlgoRSA,
		},
		Timeout: 15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", h.Hostname, h.Port)
	return ssh.Dial("tcp", addr, cfg)
}

// buildAuthMethods returns auth methods in priority order:
// 1. Explicit identity_file
// 2. SSH agent
// 3. Default key paths
// 4. No auth (empty)
// 5. Password (with warning already printed by config.WarnInsecure)
func buildAuthMethods(h config.Host) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// 1. Explicit key file
	if h.IdentityFile != "" {
		expanded := expandHome(h.IdentityFile)
		m, err := publicKeyFile(expanded)
		if err != nil {
			return nil, fmt.Errorf("loading identity_file %q: %w", h.IdentityFile, err)
		}
		methods = append(methods, m)
	}

	// 2. SSH agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	// 3. Default key paths (only if no explicit key given)
	if h.IdentityFile == "" {
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
			path := expandHome(filepath.Join("~/.ssh", name))
			if m, err := publicKeyFile(path); err == nil {
				methods = append(methods, m)
			}
		}
	}

	// 4. Password (insecure, warning printed separately)
	if h.Password != "" {
		methods = append(methods, ssh.Password(h.Password))
	}

	// Allow empty methods (passwordless hosts)
	return methods, nil
}

func publicKeyFile(path string) (ssh.AuthMethod, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// knownHostsCallback returns a host key callback backed by ~/.ssh/known_hosts.
// If the file doesn't exist, returns an error prompting the user to do an
// initial `ssh user@host` to populate it.
func knownHostsCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	khPath := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(khPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"~/.ssh/known_hosts not found.\n"+
				"Connect to the host manually once to add its key:\n"+
				"  ssh user@hostname",
		)
	}
	cb, err := knownhosts.New(khPath)
	if err != nil {
		return nil, fmt.Errorf("loading known_hosts: %w", err)
	}
	return cb, nil
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
