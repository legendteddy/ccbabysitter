package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPageURLRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	if got := PageURL(dir); got != "" {
		t.Fatalf("nothing saved yet, got %q", got)
	}
	if err := SavePageURL(dir, "http://127.0.0.1:47391"); err != nil {
		t.Fatal(err)
	}
	if got := PageURL(dir); got != "http://127.0.0.1:47391" {
		t.Fatalf("got %q", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, pageURLFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "http://127.0.0.1:47391\n" {
		t.Fatalf("the file is the URL on one line, got %q", data)
	}

	if err := RemovePageURL(dir, "http://127.0.0.1:50000"); err != nil {
		t.Fatal(err)
	}
	if got := PageURL(dir); got != "http://127.0.0.1:47391" {
		t.Fatalf("removing someone else's URL leaves the file alone, got %q", got)
	}
	if err := RemovePageURL(dir, "http://127.0.0.1:47391"); err != nil {
		t.Fatal(err)
	}
	if got := PageURL(dir); got != "" {
		t.Fatalf("the file is gone, got %q", got)
	}
	if err := RemovePageURL(dir, "http://127.0.0.1:47391"); err != nil {
		t.Fatalf("removing it twice is fine: %v", err)
	}
}

// Only a loopback page address is saved or read back, since whoever reads
// it goes on to connect to it.
func TestPageURLRefusesAnythingButLoopback(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{
		"",
		"http://203.0.113.7:47391",
		"https://127.0.0.1:47391",
		"http://127.0.0.1",
		"http://127.0.0.1:0",
		"http://127.0.0.1:99999",
		"http://127.0.0.1:47391/x",
		"http://127.0.0.1:47391?x=1",
		"not a url",
	} {
		if err := SavePageURL(dir, bad); err == nil {
			t.Fatalf("%q was saved", bad)
		}
		if err := os.WriteFile(filepath.Join(dir, pageURLFile), []byte(bad+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := PageURL(dir); got != "" {
			t.Fatalf("%q was read back as %q", bad, got)
		}
	}
}
