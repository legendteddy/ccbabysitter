package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKeyIsCreatedOrRepaired(t *testing.T) {
	dir := t.TempDir()
	k1, err := PageKey(dir)
	if err != nil || len(k1) != 64 {
		t.Fatalf("first key %q, %v", k1, err)
	}
	k2, _ := PageKey(dir)
	if k2 != k1 {
		t.Fatal("the key changed between starts")
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(filepath.Join(dir, "page-key"))
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v, want 0600", fi.Mode().Perm())
		}
	}
	for _, bad := range []string{"", "short", strings.Repeat("z", 64)} {
		os.WriteFile(filepath.Join(dir, "page-key"), []byte(bad), 0o600)
		if ReadPageKey(dir) != "" {
			t.Fatalf("ReadPageKey accepted %q", bad)
		}
		k, err := PageKey(dir)
		if err != nil || len(k) != 64 || k == bad {
			t.Fatalf("not repaired from %q: %q %v", bad, k, err)
		}
	}
}

func TestKeyedURL(t *testing.T) {
	if got := KeyedURL("http://127.0.0.1:47391", "ab"); got != "http://127.0.0.1:47391/?token=ab" {
		t.Fatal(got)
	}
}

// A key file an earlier build or a person left readable by others is
// tightened to owner-only when it is read for use.
func TestKeyFileIsTightened(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no such permissions")
	}
	dir := t.TempDir()
	key := strings.Repeat("ab", 32)
	path := filepath.Join(dir, "page-key")
	if err := os.WriteFile(path, []byte(key+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PageKey(dir)
	if err != nil || got != key {
		t.Fatalf("PageKey = %q, %v", got, err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, want 0600", fi.Mode().Perm())
	}
}
