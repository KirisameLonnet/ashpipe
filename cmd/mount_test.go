package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/KirisameLonnet/ashpipe/internal/config"
)

func TestMountPortalsReturnsAllFailures(t *testing.T) {
	wantFirst := errors.New("first failure")
	wantSecond := errors.New("second failure")
	called := make([]string, 0, 3)
	mount := func(_ context.Context, _ string, _ *config.Config, name string) error {
		called = append(called, name)
		switch name {
		case "first":
			return wantFirst
		case "second":
			return wantSecond
		default:
			return nil
		}
	}

	err := mountPortals(
		context.Background(),
		"/workspace",
		&config.Config{},
		[]string{"first", "ok", "second"},
		false,
		mount,
	)
	if !errors.Is(err, wantFirst) || !errors.Is(err, wantSecond) {
		t.Fatalf("mountPortals() error = %v, want both mount failures", err)
	}
	if got := strings.Join(called, ","); got != "first,ok,second" {
		t.Fatalf("mount order = %q, want %q", got, "first,ok,second")
	}
}

func TestMountPortalsBestEffortSuppressesFailure(t *testing.T) {
	want := errors.New("mount failure")
	called := 0
	mount := func(_ context.Context, _ string, _ *config.Config, _ string) error {
		called++
		return want
	}

	err := mountPortals(
		context.Background(),
		"/workspace",
		&config.Config{},
		[]string{"first", "second"},
		true,
		mount,
	)
	if err != nil {
		t.Fatalf("mountPortals() error = %v, want nil", err)
	}
	if called != 2 {
		t.Fatalf("mount called %d times, want 2", called)
	}
}
