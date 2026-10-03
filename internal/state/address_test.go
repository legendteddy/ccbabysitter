package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestServerAddressRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	if got := ServerAddress(dir); got != "" {
		t.Fatalf("nothing saved yet, got %q", got)
	}
	if err := SaveServerAddress(dir, "203.0.113.7"); err != nil {
		t.Fatal(err)
	}
	if got := ServerAddress(dir); got != "203.0.113.7" {
		t.Fatalf("got %q", got)
	}
	if err := SaveServerAddress(dir, "2001:db8::7"); err != nil {
		t.Fatal(err)
	}
	if got := ServerAddress(dir); got != "2001:db8::7" {
		t.Fatalf("a newer address replaces the older one, got %q", got)
	}
	info, err := os.Stat(filepath.Join(dir, addressFile))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("the file is private to the account: %v", info.Mode())
	}
}

// Anything that could not stand as one word in a command line is neither
// saved nor read back.
func TestServerAddressRefusesOddValues(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{"", "203.0.113.7; rm -rf x", "two words", "$(x)", "[2001:db8::7]"} {
		if err := SaveServerAddress(dir, bad); err != nil {
			t.Fatal(err)
		}
		if got := ServerAddress(dir); got != "" {
			t.Fatalf("%q was saved as %q", bad, got)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, addressFile), []byte("a b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ServerAddress(dir); got != "" {
		t.Fatalf("a file holding no address reads back as none, got %q", got)
	}
}
