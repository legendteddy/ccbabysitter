package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccbabysitter.dev/ccbabysitter/internal/state"
)

func TestResetFlow(t *testing.T) {
	dir := t.TempDir()
	st := &state.Store{Dir: dir}
	s, _ := st.Load()
	s.Watches = append(s.Watches, state.Watch{SessionID: "abcdefgh-1", ShortID: "abcdefgh", Name: "work", Cwd: "/w"})
	st.Save(s)

	var out strings.Builder
	if rc := resetFlow(dir, strings.NewReader("n\n"), &out, func(string) bool { return false }); rc != 0 || !strings.Contains(out.String(), "Nothing changed") {
		t.Fatal(rc, out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json")); err != nil {
		t.Fatal("declining must keep the folder")
	}

	out.Reset()
	if rc := resetFlow(dir, strings.NewReader("y\n"), &out, func(string) bool { return true }); rc != 1 || !strings.Contains(out.String(), "running") {
		t.Fatal("locked refuses", rc, out.String())
	}

	out.Reset()
	if rc := resetFlow(dir, strings.NewReader("y\n"), &out, func(string) bool { return false }); rc != 0 || !strings.Contains(out.String(), "work") {
		t.Fatal(rc, out.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("folder must be deleted")
	}
}
