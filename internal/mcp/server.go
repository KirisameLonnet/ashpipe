package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/KirisameLonnet/ashpipe/internal/config"
	"github.com/KirisameLonnet/ashpipe/internal/pool"
	"github.com/KirisameLonnet/ashpipe/internal/sftp"
	"github.com/KirisameLonnet/ashpipe/internal/shell"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gossh "golang.org/x/crypto/ssh"
)

type sessionKey struct{ host string }

type session struct {
	sh       *shell.Shell
	sftp     *sftp.Client
	homeDir  string // resolved $HOME on the remote host
}

type Server struct {
	cfg      *config.Config
	root     string
	sessions map[sessionKey]*session
}

func NewServer(root string, cfg *config.Config) *Server {
	return &Server{
		cfg:      cfg,
		root:     root,
		sessions: map[sessionKey]*session{},
	}
}

func (s *Server) Run() error {
	srv := server.NewMCPServer("ashpipe", "0.1.0",
		server.WithToolCapabilities(true),
	)

	s.registerTools(srv)
	return server.ServeStdio(srv)
}

func (s *Server) registerTools(srv *server.MCPServer) {
	srv.AddTool(mcp.NewTool("bash",
		mcp.WithDescription("Run a shell command on the remote host"),
		mcp.WithString("command", mcp.Required(), mcp.Description("Shell command to execute")),
		mcp.WithString("host", mcp.Description("Portal or host alias (default: first configured portal)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Timeout in seconds (default: 60)")),
	), s.handleBash)

	srv.AddTool(mcp.NewTool("read",
		mcp.WithDescription("Read a file from the remote host"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path (local portal path or absolute remote path)")),
		mcp.WithString("host", mcp.Description("Portal or host alias")),
	), s.handleRead)

	srv.AddTool(mcp.NewTool("write",
		mcp.WithDescription("Write content to a file on the remote host"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path")),
		mcp.WithString("content", mcp.Required(), mcp.Description("File content")),
		mcp.WithString("host", mcp.Description("Portal or host alias")),
	), s.handleWrite)

	srv.AddTool(mcp.NewTool("edit",
		mcp.WithDescription("Edit a remote file by replacing old_string with new_string"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path")),
		mcp.WithString("old_string", mcp.Required(), mcp.Description("Exact string to replace (must be unique in file)")),
		mcp.WithString("new_string", mcp.Required(), mcp.Description("Replacement string")),
		mcp.WithString("host", mcp.Description("Portal or host alias")),
	), s.handleEdit)

	srv.AddTool(mcp.NewTool("diff",
		mcp.WithDescription("Show unified diff between two remote files"),
		mcp.WithString("path_a", mcp.Required(), mcp.Description("First file path")),
		mcp.WithString("path_b", mcp.Required(), mcp.Description("Second file path")),
		mcp.WithString("host", mcp.Description("Portal or host alias")),
	), s.handleDiff)

	srv.AddTool(mcp.NewTool("ls",
		mcp.WithDescription("List files in a remote directory"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path")),
		mcp.WithString("host", mcp.Description("Portal or host alias")),
	), s.handleLs)

	srv.AddTool(mcp.NewTool("glob",
		mcp.WithDescription("Find files matching a pattern on the remote host"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Shell glob pattern (e.g. '**/*.go')")),
		mcp.WithString("dir", mcp.Description("Base directory (default: remote root)")),
		mcp.WithString("host", mcp.Description("Portal or host alias")),
	), s.handleGlob)
}

func (s *Server) getSession(hostHint string) (*session, config.Portal, config.Host, error) {
	portal, host, err := s.resolveHost(hostHint)
	if err != nil {
		return nil, config.Portal{}, config.Host{}, err
	}

	key := sessionKey{host: host.Hostname}
	if sess, ok := s.sessions[key]; ok {
		return sess, portal, host, nil
	}

	conn, err := pool.Default.Get(host)
	if err != nil {
		return nil, config.Portal{}, config.Host{}, fmt.Errorf("SSH connect: %w", err)
	}

	sh, err := shell.New(conn, portal.RemotePath, func() (*gossh.Client, error) {
		return pool.Default.Get(host)
	})
	if err != nil {
		return nil, config.Portal{}, config.Host{}, fmt.Errorf("shell: %w", err)
	}

	// SFTP is optional — some servers (e.g. dropbear on routers) don't support it.
	// bash/ls/glob tools still work without it; file tools return a clear error.
	var sc *sftp.Client
	if s, err := sftp.New(conn); err == nil {
		sc = s
	}

	// Resolve $HOME once so we can expand ~ in file paths for SFTP.
	homeDir := "~"
	if r, err := sh.Exec("echo $HOME", 5*time.Second); err == nil {
		homeDir = strings.TrimSpace(r.Stdout)
	}

	sess := &session{sh: sh, sftp: sc, homeDir: homeDir}
	s.sessions[key] = sess
	return sess, portal, host, nil
}

func (s *Server) resolveHost(hint string) (config.Portal, config.Host, error) {
	if hint != "" {
		if portal, host, err := s.cfg.ResolvePortal(hint); err == nil {
			return portal, host, nil
		}
		// Maybe hint is a host alias directly.
		if h, ok := s.cfg.Hosts[hint]; ok {
			return config.Portal{RemotePath: "/"}, h, nil
		}
		return config.Portal{}, config.Host{}, fmt.Errorf("unknown host or portal %q", hint)
	}
	// Use first portal as default.
	for name := range s.cfg.Portals {
		return s.cfg.ResolvePortal(name)
	}
	return config.Portal{}, config.Host{}, fmt.Errorf("no portals configured")
}

// translatePath converts a local portal path to a remote path, expanding ~ using
// the session's resolved $HOME.
func (s *Server) translatePath(localPath string, portal config.Portal) string {
	// Expand tilde using the session's resolved home directory.
	if sess, ok := s.sessionForPortal(portal); ok && sess.homeDir != "" && sess.homeDir != "~" {
		if localPath == "~" {
			return sess.homeDir
		}
		if strings.HasPrefix(localPath, "~/") {
			return sess.homeDir + localPath[1:]
		}
	}

	// Translate local portal path → remote path.
	for name := range s.cfg.Portals {
		portalDir := filepath.Join(s.root, name)
		if rel, err := filepath.Rel(portalDir, localPath); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join(portal.RemotePath, rel)
		}
	}
	return localPath
}

// sessionForPortal finds the active session for a given portal.
func (s *Server) sessionForPortal(portal config.Portal) (*session, bool) {
	host, ok := s.cfg.Hosts[portal.Host]
	if !ok {
		return nil, false
	}
	sess, ok := s.sessions[sessionKey{host: host.Hostname}]
	return sess, ok
}

func (s *Server) handleBash(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command := req.GetString("command", "")
	hostHint := req.GetString("host", "")
	timeoutSec := req.GetFloat("timeout_seconds", 60.0)

	sess, _, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := sess.sh.Exec(command, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	out := result.Stdout
	if result.ExitCode != 0 {
		out += fmt.Sprintf("\n[exit code: %d]", result.ExitCode)
	}
	return mcp.NewToolResultText(out), nil
}

func (s *Server) handleRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("path", "")
	hostHint := req.GetString("host", "")

	sess, portal, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if sess.sftp == nil {
		return mcp.NewToolResultError("SFTP not available on this host; use bash with cat instead"), nil
	}

	remotePath := s.translatePath(path, portal)
	data, err := sess.sftp.ReadFile(remotePath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleWrite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("path", "")
	content := req.GetString("content", "")
	hostHint := req.GetString("host", "")

	sess, portal, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if sess.sftp == nil {
		return mcp.NewToolResultError("SFTP not available on this host; use bash with tee/cat instead"), nil
	}

	remotePath := s.translatePath(path, portal)
	if err := sess.sftp.WriteFile(remotePath, []byte(content), 0o644); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Written: %s", remotePath)), nil
}

func (s *Server) handleEdit(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("path", "")
	oldStr := req.GetString("old_string", "")
	newStr := req.GetString("new_string", "")
	hostHint := req.GetString("host", "")

	sess, portal, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if sess.sftp == nil {
		return mcp.NewToolResultError("SFTP not available on this host"), nil
	}

	remotePath := s.translatePath(path, portal)
	if err := sess.sftp.EditFile(remotePath, oldStr, newStr); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Edited: %s", remotePath)), nil
}

func (s *Server) handleDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pathA := req.GetString("path_a", "")
	pathB := req.GetString("path_b", "")
	hostHint := req.GetString("host", "")

	sess, portal, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if sess.sftp == nil {
		return mcp.NewToolResultError("SFTP not available on this host; use bash with diff instead"), nil
	}

	diff, err := sess.sftp.Diff(
		s.translatePath(pathA, portal),
		s.translatePath(pathB, portal),
	)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(diff), nil
}

func (s *Server) handleLs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("path", "")
	hostHint := req.GetString("host", "")

	sess, portal, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Fall back to bash ls when SFTP is unavailable.
	if sess.sftp == nil {
		result, err := sess.sh.Exec(fmt.Sprintf("ls -la %s", path), 10*time.Second)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result.Stdout), nil
	}

	entries, err := sess.sftp.ListDir(s.translatePath(path, portal))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	for _, e := range entries {
		indicator := " "
		if e.IsDir {
			indicator = "/"
		}
		fmt.Fprintf(&sb, "%s%s  %d  %s\n", e.Name, indicator, e.Size, e.ModTime.Format("2006-01-02 15:04"))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGlob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern := req.GetString("pattern", "")
	dir := req.GetString("dir", "")
	hostHint := req.GetString("host", "")

	sess, portal, _, err := s.getSession(hostHint)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if dir == "" {
		dir = portal.RemotePath
	} else {
		dir = s.translatePath(dir, portal)
	}

	cmd := fmt.Sprintf("find %q -name %q 2>/dev/null | head -200", dir, pattern)
	result, err := sess.sh.Exec(cmd, 30*time.Second)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(result.Stdout), nil
}
