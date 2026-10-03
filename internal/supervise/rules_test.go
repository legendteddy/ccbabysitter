package supervise

import (
	"fmt"
	"testing"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/claude"
	"ccbabysitter.dev/ccbabysitter/internal/observe"
	"ccbabysitter.dev/ccbabysitter/internal/state"
)

func sess(pid int, id, kind, entry string, jobID string) []byte {
	j := ""
	if jobID != "" {
		j = fmt.Sprintf(`,"jobId":"%s"`, jobID)
	}
	return []byte(fmt.Sprintf(`{"pid":%d,"sessionId":"%s","cwd":"/home/dev/ws","procStart":"7","kind":"%s","entrypoint":"%s","bridgeSessionId":"b"%s}`, pid, id, kind, entry, j))
}

func snapOf(files ...[]byte) observe.Snapshot {
	return observe.Build(files, func(int, string) bool { return true }, time.Now())
}

func watch(id, short, promise string) state.Watch {
	return state.Watch{SessionID: id, ShortID: short, PromiseState: promise}
}

func TestDecideInPlace(t *testing.T) {
	id := "11111111-2222-4333-8444-555555555501"
	w := watch(id, "11111111", "inplace")
	if Decide(w, snapOf(sess(1, id, "interactive", "claude-vscode", "")), 0) != None {
		t.Fatal("alive in its host")
	}
	if Decide(w, snapOf(sess(1, id, "interactive", "claude-vscode", ""), sess(2, id, "interactive", "cli", "")), 0) != None {
		t.Fatal("two user hosts: never act")
	}
	if Decide(w, snapOf(), 1) != None {
		t.Fatal("first absent sweep")
	}
	if Decide(w, snapOf(), 2) != Fallback {
		t.Fatal("second absent sweep falls back")
	}
	w.Paused = true
	if Decide(w, snapOf(), 5) != None {
		t.Fatal("paused never acts")
	}
}

func TestDecideFallback(t *testing.T) {
	id := "22222222-2222-4333-8444-555555555502"
	w := watch(id, "22222222", "fallback")
	ours := sess(1, id, "bg", "cli", "22222222")
	if Decide(w, snapOf(ours), 0) != None {
		t.Fatal("our copy alone")
	}
	if Decide(w, snapOf(ours, sess(2, id, "interactive", "claude-desktop", "")), 0) != None {
		t.Fatal("an app showing the session beside our copy: the copy is never stopped to make room")
	}
	if Decide(w, snapOf(ours, sess(3, id, "bg", "cli", "aabbccdd")), 0) != Dedupe {
		t.Fatal("two of ours")
	}
	if Decide(w, snapOf(), 2) != Fallback {
		t.Fatal("our copy died: resume again")
	}
}

func TestShouldPauseForFailures(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 10, 0, 0, time.UTC)
	if ShouldPauseForFailures([]time.Time{now.Add(-4 * time.Minute), now.Add(-2 * time.Minute)}, now) {
		t.Fatal("two")
	}
	if !ShouldPauseForFailures([]time.Time{now.Add(-4 * time.Minute), now.Add(-2 * time.Minute), now}, now) {
		t.Fatal("three")
	}
	if ShouldPauseForFailures([]time.Time{now.Add(-9 * time.Minute), now.Add(-2 * time.Minute), now}, now) {
		t.Fatal("one outside window")
	}
}

func TestDecideRejoin(t *testing.T) {
	id := "44444444-2222-4333-8444-555555555504"
	w := watch(id, "44444444", "fallback")
	app := sess(2, id, "interactive", "claude-desktop", "")
	if Decide(w, snapOf(app), 0) != Rejoin {
		t.Fatal("our copy is gone and an app has the session: rejoin")
	}
	if Decide(watch(id, "44444444", "inplace"), snapOf(app), 0) != None {
		t.Fatal("a watch in place is already where it should be")
	}
}

func TestStateOf(t *testing.T) {
	id := "55555555-2222-4333-8444-555555555505"
	ours := claude.Session{ID: id, ShortID: "55555555", Host: claude.HostBackground}
	app := claude.Session{ID: id, ShortID: "55555555", Host: claude.HostVSCode}
	stuck := watch(id, "55555555", "paused")
	stuck.Paused = true
	cases := []struct {
		name string
		w    state.Watch
		live []claude.Session
		want WatchState
	}{
		{"live in its app", watch(id, "55555555", "inplace"), []claude.Session{app}, StateWatching},
		{"in place but running nowhere", watch(id, "55555555", "inplace"), nil, StateStarting},
		{"our copy carries it", watch(id, "55555555", "fallback"), []claude.Session{ours}, StateInBackground},
		{"our copy beside an app", watch(id, "55555555", "fallback"), []claude.Session{ours, app}, StateInBackground},
		{"our copy died", watch(id, "55555555", "fallback"), nil, StateStarting},
		{"an app has it again", watch(id, "55555555", "fallback"), []claude.Session{app}, StateWatching},
		{"paused", stuck, []claude.Session{app}, StateStuck},
	}
	for _, c := range cases {
		if got := StateOf(c.w, c.live); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}
