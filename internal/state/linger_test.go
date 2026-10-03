package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLingeringTurnedOnRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	if LingeringTurnedOn(dir) {
		t.Fatal("nothing has been remembered yet")
	}
	if err := ForgetLingeringTurnedOn(dir); err != nil {
		t.Fatalf("forgetting nothing is not an error: %v", err)
	}
	if err := MarkLingeringTurnedOn(dir); err != nil {
		t.Fatal(err)
	}
	if !LingeringTurnedOn(dir) {
		t.Fatal("it was just remembered")
	}
	info, err := os.Stat(filepath.Join(dir, lingerFile))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("the file is private to the account: %v", info.Mode())
	}
	if err := MarkLingeringTurnedOn(dir); err != nil {
		t.Fatalf("remembering it again: %v", err)
	}
	if err := ForgetLingeringTurnedOn(dir); err != nil {
		t.Fatal(err)
	}
	if LingeringTurnedOn(dir) {
		t.Fatal("it was forgotten")
	}
}
