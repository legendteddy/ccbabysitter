//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitEnabledFollowsTheWantsLink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if autostartInstalled == nil {
		t.Fatal("the check must be set on this platform")
	}
	unit, err := writeUnit("/home/dev/.local/bin/ccbabysitter")
	if err != nil {
		t.Fatal(err)
	}
	if on, err := autostartInstalled(); err != nil || on {
		t.Fatalf("a unit that is only written is not enabled: %v %v", on, err)
	}
	link := filepath.Join(home, ".config", "systemd", "user", "default.target.wants", "ccbabysitter.service")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(unit, link); err != nil {
		t.Fatal(err)
	}
	if on, err := autostartInstalled(); err != nil || !on {
		t.Fatalf("the link is there: %v %v", on, err)
	}
}

func TestUnitEnabledReportsOtherErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// A plain file where the wants folder should be makes the look fail
	// with something other than not there.
	dir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "default.target.wants"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := autostartInstalled(); err == nil {
		t.Fatal("expected an error")
	}
}
