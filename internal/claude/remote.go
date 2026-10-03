package claude

import "strings"

// remoteBase is where Claude serves a session reached through Remote
// Control.
const remoteBase = "https://claude.ai/code/"

// RemoteURL is the address that opens a session through Remote Control,
// built from the bridge id in its session file the way Claude builds it: an
// id that starts cse_ is written with session_ in its place, and one that
// starts session_ is used as it is. Any other id, and one holding anything
// but letters, digits, dashes and underscores, gives no address at all, so
// nothing read from a file can turn into a link somewhere else.
func RemoteURL(bridgeID string) string {
	var rest string
	switch {
	case strings.HasPrefix(bridgeID, "session_"):
		rest = strings.TrimPrefix(bridgeID, "session_")
	case strings.HasPrefix(bridgeID, "cse_"):
		rest = strings.TrimPrefix(bridgeID, "cse_")
	default:
		return ""
	}
	if rest == "" || !linkSafe(bridgeID) {
		return ""
	}
	return remoteBase + "session_" + rest
}

// linkSafe reports whether s is made only of ASCII letters, digits, dashes
// and underscores.
func linkSafe(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
