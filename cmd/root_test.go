package cmd

import "testing"

func TestSetVersion(t *testing.T) {
	old := rootCmd.Version
	t.Cleanup(func() {
		rootCmd.Version = old
	})

	SetVersion("1.2.3")
	if rootCmd.Version != "1.2.3" {
		t.Fatalf("rootCmd.Version = %q, want 1.2.3", rootCmd.Version)
	}
}
