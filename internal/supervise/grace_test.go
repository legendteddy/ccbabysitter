package supervise

import (
	"strings"
	"testing"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/claude"
	"ccbabysitter.dev/ccbabysitter/internal/hosts"
	"ccbabysitter.dev/ccbabysitter/internal/observe"
	"ccbabysitter.dev/ccbabysitter/internal/state"
)

// graceFixture is a machine whose clock the test moves by hand, with one
// babysat desktop session that is running nowhere, a transcript to resume
// it from, and a CLI whose resume brings up a background copy with Remote
// Control on.
func graceFixture(t *testing.T, id string) (*fixture, *Supervisor, *time.Time) {
	t.Helper()
	f := newFixture(t, nil)
	f.r.SetRespond(func(args []string) (string, error) {
		a := strings.Join(args, " ")
		if a == "agents --json" {
			return "[]", nil
		}
		if strings.HasPrefix(a, "--bg --resume") {
			f.p.SetAlive(9, true)
			f.setFiles(bgSession(9, id, "b"))
			return "backgrounded \u00b7 " + id[:8] + " \u00b7 demo-a1", nil
		}
		return "", nil
	})
	f.writeTranscript(t, "/home/dev/ws", id)
	clock := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	f.d.Now = func() time.Time { return clock }
	f.d.StartupGrace = 90 * time.Second
	// The loop is deliberately not started: every pass here is one this
	// test drove, at a time this test chose.
	s := New(*f.d)
	s.addWatchForTest(state.Watch{SessionID: id, ShortID: id[:8], Cwd: "/home/dev/ws",
		OriginHost: claude.HostDesktop, PromiseState: "inplace", HasSavedOptions: true})
	return f, s, &clock
}

// For 90 seconds after start, on a machine with a display, a session
// missing from its app is not started in the background: the app may be
// about to restore it. Absence is still counted, so a session that is
// really gone is started the moment the grace is over.
func TestStartUpGraceHoldsTheFallbackBack(t *testing.T) {
	id := "11111111-2222-4333-8444-5555555555d0"
	f, s, clock := graceFixture(t, id)
	start := *clock

	for i := 0; i < 8; i++ {
		*clock = clock.Add(10 * time.Second)
		s.reconcileForTest(observe.Snapshot{At: *clock})
	}
	if n := countCalls(f, "--bg --resume "); n != 0 {
		t.Fatalf("nothing is started during the grace: %v", f.r.CallList())
	}
	if n := s.absentForTest(id); n != 8 {
		t.Fatalf("absence is still counted during the grace, got %d", n)
	}
	if got, want := s.View().GraceUntil, start.Add(90*time.Second).Format(time.RFC3339); got != want {
		t.Fatalf("graceUntil %q, want %q", got, want)
	}

	*clock = start.Add(91 * time.Second)
	s.reconcileForTest(observe.Snapshot{At: *clock})
	if n := countCalls(f, "--bg --resume "); n != 1 {
		t.Fatalf("the fallback runs the moment the grace is over: %v", f.r.CallList())
	}
	if got := s.View().GraceUntil; got != "" {
		t.Fatalf("no grace is running any more: %q", got)
	}
}

// An app that restores its session during the grace needs nothing from
// anybody: the watch is simply Watching, and nothing is started after the
// grace either.
func TestASessionRestoredDuringTheGraceIsWatching(t *testing.T) {
	id := "22222222-2222-4333-8444-5555555555d1"
	f, s, clock := graceFixture(t, id)
	start := *clock

	for i := 0; i < 3; i++ {
		*clock = clock.Add(10 * time.Second)
		s.reconcileForTest(observe.Snapshot{At: *clock})
	}
	f.p.SetAlive(10, true)
	f.setFiles(desktopSession(10, id))
	f.d.Obs.RefreshNow()
	app := f.d.Obs.Current().Sessions
	*clock = clock.Add(10 * time.Second)
	s.reconcileForTest(observe.Snapshot{At: *clock, Sessions: app})
	*clock = start.Add(2 * time.Minute)
	s.reconcileForTest(observe.Snapshot{At: *clock, Sessions: app})

	w, _ := s.watchForTest(id)
	if got := StateOf(w, app); got != StateWatching {
		t.Fatalf("state %s, want watching", got)
	}
	if n := countCalls(f, "--bg --resume "); n != 0 {
		t.Fatalf("nothing is started for a session its app restored: %v", f.r.CallList())
	}
}

// Nothing on a server restores a session by itself, so a server has no
// grace: two absent passes start the background copy as they always did.
func TestAServerHasNoGrace(t *testing.T) {
	id := "33333333-2222-4333-8444-5555555555d2"
	f, _, clock := graceFixture(t, id)
	f.d.Env = func() hosts.Env {
		return hosts.Env{Platform: "linux", Headless: true, CLIFound: true, CLIPresent: true}
	}
	// A supervisor built on the same saved state, now on a server: it reads
	// the watch the fixture saved.
	s := New(*f.d)

	for i := 0; i < 2; i++ {
		*clock = clock.Add(10 * time.Second)
		s.reconcileForTest(observe.Snapshot{At: *clock})
	}
	if n := countCalls(f, "--bg --resume "); n != 1 {
		t.Fatalf("a server starts the copy without waiting: %v", f.r.CallList())
	}
	if got := s.View().GraceUntil; got != "" {
		t.Fatalf("a server has no grace: %q", got)
	}
}
