//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartupScriptInstalledFollowsTheFile(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	if autostartInstalled == nil {
		t.Fatal("the check must be set on this platform")
	}
	if on, err := autostartInstalled(); err != nil || on {
		t.Fatalf("no file yet: %v %v", on, err)
	}
	path := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "CCBabysitter.cmd")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("@echo off\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if on, err := autostartInstalled(); err != nil || !on {
		t.Fatalf("the file is there: %v %v", on, err)
	}
}

func TestStartupScriptInstalledNeedsAppData(t *testing.T) {
	t.Setenv("APPDATA", "")
	if _, err := autostartInstalled(); err == nil {
		t.Fatal("expected an error without APPDATA")
	}
}
