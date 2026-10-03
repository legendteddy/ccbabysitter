package claude

import "testing"

// A Remote Control address is built only from a bridge id of the two
// shapes Claude hands out, and nothing else ever becomes a link.
func TestRemoteURL(t *testing.T) {
	cases := []struct {
		name, bridge, want string
	}{
		{"session id is used as it is", "session_01TESTBRIDGE02", "https://claude.ai/code/session_01TESTBRIDGE02"},
		{"cse id is rewritten", "cse_01TESTBRIDGE02", "https://claude.ai/code/session_01TESTBRIDGE02"},
		{"dashes and underscores are fine", "session_a-b_C9", "https://claude.ai/code/session_a-b_C9"},
		{"empty", "", ""},
		{"prefix only", "session_", ""},
		{"cse prefix only", "cse_", ""},
		{"another prefix", "bridge_01TEST", ""},
		{"a slash", "session_01/../x", ""},
		{"a query", "session_01?x=1", ""},
		{"a space", "session_01 TEST", ""},
		{"not ascii", "session_01\u00e9", ""},
	}
	for _, c := range cases {
		if got := RemoteURL(c.bridge); got != c.want {
			t.Errorf("%s: RemoteURL(%q) = %q, want %q", c.name, c.bridge, got, c.want)
		}
	}
}

// The bridge id is kept from the session file, so the address can be
// built from the copy that is running.
func TestParseSessionFileKeepsTheBridgeID(t *testing.T) {
	s, err := ParseSessionFile([]byte(bgFile))
	if err != nil || s.BridgeSessionID != "session_TESTBRIDGE02" {
		t.Fatalf("%v %+v", err, s)
	}
}
