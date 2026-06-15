package sshfs

import (
	"strings"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

func TestMountTargetDarwin(t *testing.T) {
	line := "root@192.168.1.1:/root/ on /Users/me/project/test/wrt (macfuse, nodev, nosuid, synchronous, mounted by me)"
	got := mountTarget(line)
	want := "/Users/me/project/test/wrt"
	if got != want {
		t.Fatalf("mountTarget() = %q, want %q", got, want)
	}
}

func TestMountTargetLinux(t *testing.T) {
	line := "root@192.168.1.1:/root/ on /home/me/project/test/wrt type fuse.sshfs (rw,nosuid,nodev,relatime,user_id=1000,group_id=1000)"
	got := mountTarget(line)
	want := "/home/me/project/test/wrt"
	if got != want {
		t.Fatalf("mountTarget() = %q, want %q", got, want)
	}
}

func TestMountTargetIgnoresPartialPath(t *testing.T) {
	line := "root@192.168.1.1:/root/ on /Users/me/project/test/wrt-other (macfuse, nodev)"
	got := mountTarget(line)
	if got == "/Users/me/project/test/wrt" {
		t.Fatalf("mountTarget() returned partial match %q", got)
	}
}

func TestMountTargetUnescapesLinuxPath(t *testing.T) {
	line := `root@192.168.1.1:/root/ on /home/me/project/remote\040space type fuse.sshfs (rw,nosuid,nodev)`
	got := mountTarget(line)
	want := "/home/me/project/remote space"
	if got != want {
		t.Fatalf("mountTarget() = %q, want %q", got, want)
	}
}

func TestBuildArgsIncludesSSHHardeningOptions(t *testing.T) {
	args := buildArgs(config.Host{
		Hostname: "example.com",
		User:     "alice",
		Port:     22,
	}, "/srv/app", "/mnt/app")

	joined := strings.Join(args, "\x00")
	for _, want := range []string{
		"StrictHostKeyChecking=yes",
		"ConnectTimeout=10",
		"ServerAliveInterval=15",
		"ServerAliveCountMax=3",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("buildArgs() missing %q: %#v", want, args)
		}
	}
}
