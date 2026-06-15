package sshfs

import "testing"

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
