package cmd

import (
	"strings"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

func TestBuildSSHArgvQuotesRemotePath(t *testing.T) {
	argv := buildSSHArgv(config.Host{
		Hostname: "example.com",
		User:     "alice",
		Port:     22,
	}, `/tmp/$(touch pwned)/it's here`)

	remoteCmd := argv[len(argv)-1]
	if strings.Contains(remoteCmd, `cd "/tmp/$(touch pwned)`) {
		t.Fatalf("remote path used double-quoted shell syntax: %q", remoteCmd)
	}
	if !strings.Contains(remoteCmd, `cd '/tmp/$(touch pwned)/it'\''s here'`) {
		t.Fatalf("remote command did not shell-quote path safely: %q", remoteCmd)
	}
	if !containsArgPair(argv, "-o", "ConnectTimeout=10") {
		t.Fatalf("ssh argv missing ConnectTimeout: %#v", argv)
	}
}

func TestParseRemoteHomeOutputIgnoresBanner(t *testing.T) {
	out := "Welcome to Ubuntu\nLast login: today\n/home/alice\n"
	if got := parseRemoteHomeOutput(out); got != "/home/alice" {
		t.Fatalf("parseRemoteHomeOutput() = %q, want /home/alice", got)
	}
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
