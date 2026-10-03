//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Output goes to the journal when JOURNAL_STREAM names the very file
// stdout is, which is how systemd says it; a JOURNAL_STREAM inherited by a
// shell whose output goes to a terminal names some other file.
func TestJournalStream(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	st := fi.Sys().(*syscall.Stat_t)
	same := fmt.Sprintf("%d:%d", uint64(st.Dev), uint64(st.Ino))
	if !toJournal(same, f) {
		t.Fatalf("%s is stdout's own device and inode", same)
	}
	other := fmt.Sprintf("%d:%d", uint64(st.Dev), uint64(st.Ino)+1)
	for _, env := range []string{"", other, "garbage", "1:", ":2", same + ":3"} {
		if toJournal(env, f) {
			t.Errorf("JOURNAL_STREAM %q counted as the journal", env)
		}
	}
}
