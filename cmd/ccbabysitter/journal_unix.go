//go:build !windows

package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// stdoutToJournal reports whether this process's output goes to the
// systemd journal.
func stdoutToJournal() bool {
	return toJournal(os.Getenv("JOURNAL_STREAM"), os.Stdout)
}

// toJournal reports whether f is the journal stream journalStream names.
// systemd sets JOURNAL_STREAM to "DEVICE:INODE" of the stream it connects
// a unit's output to, and says to compare it with the output's own device
// and inode: the variable alone is inherited by everything the unit
// starts, a shell in a terminal included, whose output goes elsewhere.
func toJournal(journalStream string, f *os.File) bool {
	devS, inoS, ok := strings.Cut(journalStream, ":")
	if !ok {
		return false
	}
	dev, err1 := strconv.ParseUint(devS, 10, 64)
	ino, err2 := strconv.ParseUint(inoS, 10, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && uint64(st.Dev) == dev && uint64(st.Ino) == ino
}
