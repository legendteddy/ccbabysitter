package claude

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxTitleRunes is the longest title shown for a session.
const maxTitleRunes = 120

// maxSidecarBytes bounds the sidecar title file that is read. It holds one
// short title, so anything larger is not what it claims to be.
const maxSidecarBytes = 4 << 10

// CleanTitle makes a title fit for one line on the page: whatever does not
// print is dropped, whitespace is trimmed and collapsed, and the result is
// cut to maxTitleRunes characters.
func CleanTitle(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == utf8.RuneError:
			return -1
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsPrint(r):
			return r
		default:
			return -1
		}
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(capRunes(s, maxTitleRunes))
}

// titleTracker follows the title records of one transcript, in order, the
// way the Claude Code panel in VS Code picks the name it shows.
//
// A custom-title is a name a person gave, except the one Claude gives by
// itself when Remote Control connects, which is followed at once by an
// agent-name record with the same value. So a custom-title is only taken
// once the record after it has been seen and is not that agent-name.
type titleTracker struct {
	custom string
	ai     string
	// pending is a custom-title whose next record has not been read yet.
	pending    string
	hasPending bool
}

// titleRecord is the part of a transcript line the tracker reads.
type titleRecord struct {
	CustomTitle string `json:"customTitle"`
	AITitle     string `json:"aiTitle"`
	AgentName   string `json:"agentName"`
}

// record takes in the next record of the transcript.
func (t *titleTracker) record(typ string, rec *titleRecord) {
	if t.hasPending {
		if typ != "agent-name" || CleanTitle(rec.AgentName) != t.pending {
			t.custom = t.pending
		}
		t.pending, t.hasPending = "", false
	}
	switch typ {
	case "custom-title":
		if v := CleanTitle(rec.CustomTitle); v != "" {
			t.pending, t.hasPending = v, true
		}
	case "ai-title":
		if v := CleanTitle(rec.AITitle); v != "" {
			t.ai = v
		}
	}
}

// settle takes a pending custom-title as given, for when no record will
// follow it or it has been left alone long enough.
func (t *titleTracker) settle() {
	if t.hasPending {
		t.custom = t.pending
		t.pending, t.hasPending = "", false
	}
}

// title is the best name found so far in the transcript itself.
func (t *titleTracker) title() string {
	if t.custom != "" {
		return t.custom
	}
	return t.ai
}

// sidecarTitle reads the title VS Code keeps in a small file beside the
// transcript, in a folder named after the session, and returns "" when
// there is none or it is not a regular file of the expected shape and
// size. Anything else is never opened, since a pipe there would hold up
// the stats worker for good. It only ever reads.
func sidecarTitle(transcript string) string {
	if !strings.HasSuffix(transcript, ".jsonl") {
		return ""
	}
	path := strings.TrimSuffix(transcript, ".jsonl") + string(os.PathSeparator) + "custom-title.json"
	if fi, err := os.Lstat(path); err != nil || !fi.Mode().IsRegular() {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSidecarBytes+1))
	if err != nil || len(data) > maxSidecarBytes {
		return ""
	}
	var side struct {
		CustomTitle string `json:"customTitle"`
	}
	if json.Unmarshal(data, &side) != nil {
		return ""
	}
	return CleanTitle(side.CustomTitle)
}
