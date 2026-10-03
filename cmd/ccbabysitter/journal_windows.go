package main

// stdoutToJournal reports whether this process's output goes to the
// systemd journal, which there is none of on Windows.
func stdoutToJournal() bool { return false }
