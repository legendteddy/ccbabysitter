package claude

import (
	"regexp"
	"strings"
)

var (
	reBackgrounded = regexp.MustCompile(`(?i)backgrounded\s+\S\s+([0-9a-f]{8})\b`)
	reCopy         = regexp.MustCompile(`(?i)started a copy as ([0-9a-f]{8})\b`)
)

// ParseBackgrounded extracts the short id from a line like
// "backgrounded . <short> . <name> (...)".
func ParseBackgrounded(out string) (string, bool) {
	m := reBackgrounded.FindStringSubmatch(out)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// ParseCopy detects the flags trap, where re-running a background session
// with flags starts a separate copy instead of resuming it: "... started a
// copy as <short>." It returns the copy's short id.
func ParseCopy(out string) (string, bool) {
	m := reCopy.FindStringSubmatch(out)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// IsWoke reports whether out contains the "woke session <short> with its
// saved options" note, meaning the same session resumed rather than forked.
func IsWoke(out string) bool { return strings.Contains(strings.ToLower(out), "woke session") }
