package shell

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// DialFunc obtains a fresh SSH client for reconnection.
type DialFunc func() (*ssh.Client, error)

// Shell maintains a persistent bash session over an SSH channel.
// On network failure it automatically reconnects via the provided DialFunc.
type Shell struct {
	mu       sync.Mutex
	dial     DialFunc
	client   *ssh.Client
	session  *ssh.Session
	stdin    io.WriteCloser
	outBuf   *safeBuffer
	cwd      string
	initArgs string // setup command run on each (re)connect
}

type Result struct {
	Stdout   string
	ExitCode int
}

// New creates a persistent shell on the given SSH client.
// dialFn is used to re-establish the connection on failure.
func New(client *ssh.Client, initialCwd string, dialFn DialFunc) (*Shell, error) {
	sh := &Shell{
		dial:   dialFn,
		client: client,
		cwd:    initialCwd,
		// Use $HOME for ~ so the shell expands it correctly (quoting prevents tilde expansion).
		initArgs: fmt.Sprintf("export PS1='' PS2='' HISTFILE=/dev/null && cd %s", shellescape(initialCwd)),
	}
	if err := sh.startSession(); err != nil {
		return nil, err
	}
	return sh, nil
}

// Exec runs a command, auto-reconnecting once on failure.
func (s *Shell) Exec(command string, timeout time.Duration) (*Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.run(command, timeout)
	if err != nil {
		// Attempt reconnect once.
		if reconnErr := s.reconnect(); reconnErr != nil {
			return nil, fmt.Errorf("command failed (%w); reconnect also failed: %v", err, reconnErr)
		}
		result, err = s.run(command, timeout)
		if err != nil {
			return nil, err
		}
	}

	// Sync cwd after every command.
	if pwdResult, e := s.run("pwd", 5*time.Second); e == nil {
		s.cwd = strings.TrimSpace(pwdResult.Stdout)
	}

	return result, nil
}

// Cwd returns the last known remote working directory.
func (s *Shell) Cwd() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cwd
}

// Close terminates the shell session.
func (s *Shell) Close() error {
	if s.stdin != nil {
		_, _ = fmt.Fprintln(s.stdin, "exit")
		s.stdin.Close()
	}
	if s.session != nil {
		return s.session.Close()
	}
	return nil
}

func (s *Shell) startSession() error {
	sess, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}

	buf := &safeBuffer{}
	sess.Stdout = buf
	sess.Stderr = buf

	stdin, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return err
	}

	if err := sess.Shell(); err != nil {
		sess.Close()
		return fmt.Errorf("start shell: %w", err)
	}

	s.session = sess
	s.stdin = stdin
	s.outBuf = buf

	// Initialize shell environment.
	if _, err := s.run(s.initArgs, 10*time.Second); err != nil {
		sess.Close()
		return fmt.Errorf("shell init: %w", err)
	}
	return nil
}

func (s *Shell) reconnect() error {
	// Clean up old session.
	if s.session != nil {
		_ = s.session.Close()
	}

	// Dial fresh connection.
	client, err := s.dial()
	if err != nil {
		return err
	}
	s.client = client

	// Restore to last known cwd.
	s.initArgs = fmt.Sprintf("export PS1='' PS2='' HISTFILE=/dev/null && cd %s", shellescape(s.cwd))
	return s.startSession()
}

func (s *Shell) run(command string, timeout time.Duration) (*Result, error) {
	sentinel := fmt.Sprintf("__ASHPIPE_%d__", time.Now().UnixNano())
	wrapped := fmt.Sprintf("%s\necho %s:$?\n", command, sentinel)

	s.outBuf.reset()
	if _, err := fmt.Fprint(s.stdin, wrapped); err != nil {
		return nil, fmt.Errorf("write command: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data := s.outBuf.string()
		idx := strings.Index(data, sentinel+":")
		if idx >= 0 {
			outputPart := data[:idx]
			rest := data[idx+len(sentinel)+1:]
			exitStr := strings.TrimSpace(strings.SplitN(rest, "\n", 2)[0])

			var code int
			fmt.Sscanf(exitStr, "%d", &code)

			return &Result{
				Stdout:   strings.TrimRight(outputPart, "\n"),
				ExitCode: code,
			}, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("command timed out after %s", timeout)
}

// shellescape quotes a path for use in shell commands.
// ~ and ~/ are left unquoted so the shell expands them.
func shellescape(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		return path
	}
	return fmt.Sprintf("%q", path)
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) string() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *safeBuffer) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}
