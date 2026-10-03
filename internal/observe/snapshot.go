// Package observe turns Claude Code's on-disk session files into immutable
// snapshots and raises events when the set of live sessions changes.
package observe

import (
	"sort"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/claude"
)

// Snapshot is a point-in-time view of every live session found on disk.
type Snapshot struct {
	At       time.Time
	Sessions []claude.Session
}

// All returns every session in the snapshot with the given session id. More
// than one can exist at once, for example while one host is taking over a
// session another host still has open.
func (s Snapshot) All(id string) []claude.Session {
	var out []claude.Session
	for _, x := range s.Sessions {
		if x.ID == id {
			out = append(out, x)
		}
	}
	return out
}

// Find returns the first session in the snapshot with the given session id.
func (s Snapshot) Find(id string) (claude.Session, bool) {
	for _, x := range s.Sessions {
		if x.ID == id {
			return x, true
		}
	}
	return claude.Session{}, false
}

// Build parses every file, keeping only sessions whose process is still
// alive. A file that fails to parse is skipped without logging: a session
// file can be read while Claude Code is still writing it, so a parse
// failure is usually transient rather than a real problem. Duplicate
// session ids are kept on purpose, since a session can be open in two apps
// at once and the page says so.
func Build(files [][]byte, alive func(pid int, procStart string) bool, now time.Time) Snapshot {
	var list []claude.Session
	for _, f := range files {
		s, err := claude.ParseSessionFile(f)
		if err != nil || !alive(s.PID, s.ProcStart) {
			continue
		}
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		ki, kj := list[i].Name, list[j].Name
		if ki == "" {
			ki = list[i].ShortID
		}
		if kj == "" {
			kj = list[j].ShortID
		}
		if ki != kj {
			return ki < kj
		}
		return list[i].PID < list[j].PID
	})
	return Snapshot{At: now, Sessions: list}
}

// same reports whether two snapshots hold the same sessions, ignoring the
// time each was taken.
func same(a, b Snapshot) bool {
	if len(a.Sessions) != len(b.Sessions) {
		return false
	}
	for i := range a.Sessions {
		if a.Sessions[i] != b.Sessions[i] {
			return false
		}
	}
	return true
}
