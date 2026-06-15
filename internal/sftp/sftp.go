package sftp

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

type Client struct {
	c *sftp.Client
}

// New opens an SFTP session on the given SSH client.
func New(conn *gossh.Client) (*Client, error) {
	c, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("sftp: %w", err)
	}
	return &Client{c: c}, nil
}

func (c *Client) Close() error { return c.c.Close() }

// ReadFile reads a remote file.
func (c *Client) ReadFile(path string) ([]byte, error) {
	f, err := c.c.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	data := make([]byte, stat.Size())
	_, err = f.Read(data)
	return data, err
}

// WriteFile writes content to a remote file (creates or overwrites).
func (c *Client) WriteFile(path string, content []byte, perm fs.FileMode) error {
	if err := c.c.MkdirAll(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := c.c.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	return err
}

// EditFile applies an old_string → new_string replacement on a remote file.
// Returns error if old_string is not found or is ambiguous (occurs more than once).
func (c *Client) EditFile(path, oldStr, newStr string) error {
	data, err := c.ReadFile(path)
	if err != nil {
		return err
	}
	updated, err := applyEdit(string(data), oldStr, newStr)
	if err != nil {
		return fmt.Errorf("%w in %s", err, path)
	}
	return c.WriteFile(path, []byte(updated), 0o644)
}

// applyEdit applies a single old→new replacement; errors if not found or ambiguous.
func applyEdit(content, oldStr, newStr string) (string, error) {
	count := strings.Count(content, oldStr)
	if count == 0 {
		return "", fmt.Errorf("old_string not found")
	}
	if count > 1 {
		return "", fmt.Errorf("old_string is ambiguous (%d occurrences); provide more context", count)
	}
	return strings.Replace(content, oldStr, newStr, 1), nil
}

type FileInfo struct {
	Name    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
}

// ListDir lists a remote directory.
func (c *Client) ListDir(path string) ([]FileInfo, error) {
	entries, err := c.c.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]FileInfo, len(entries))
	for i, e := range entries {
		out[i] = FileInfo{
			Name:    e.Name(),
			Size:    e.Size(),
			Mode:    e.Mode(),
			ModTime: e.ModTime(),
			IsDir:   e.IsDir(),
		}
	}
	return out, nil
}

// Diff returns a unified diff string between two remote files.
func (c *Client) Diff(pathA, pathB string) (string, error) {
	a, err := c.ReadFile(pathA)
	if err != nil {
		return "", err
	}
	b, err := c.ReadFile(pathB)
	if err != nil {
		return "", err
	}
	return unifiedDiff(pathA, pathB, string(a), string(b)), nil
}

func unifiedDiff(nameA, nameB, a, b string) string {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", nameA, nameB)
	// Simple line-by-line diff (sufficient for agent use).
	i, j := 0, 0
	for i < len(linesA) || j < len(linesB) {
		switch {
		case i >= len(linesA):
			fmt.Fprintf(&sb, "+ %s\n", linesB[j])
			j++
		case j >= len(linesB):
			fmt.Fprintf(&sb, "- %s\n", linesA[i])
			i++
		case linesA[i] == linesB[j]:
			fmt.Fprintf(&sb, "  %s\n", linesA[i])
			i++
			j++
		default:
			fmt.Fprintf(&sb, "- %s\n", linesA[i])
			fmt.Fprintf(&sb, "+ %s\n", linesB[j])
			i++
			j++
		}
	}
	return sb.String()
}
