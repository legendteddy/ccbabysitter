// Package supervise keeps the promise made about a babysat session: the
// computer stays awake, and if the session's app goes away the session
// carries on as a background copy with Remote Control. This file is the
// pure reconcile policy; the rest of the package acts on it.
package supervise

import (
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/claude"
	"ccbabysitter.dev/ccbabysitter/internal/observe"
	"ccbabysitter.dev/ccbabysitter/internal/state"
)

// Decision is what a reconcile pass should do about one watch.
type Decision int

const (
	None Decision = iota
	Fallback
	Dedupe
	Rejoin
)

const (
	// AbsentSweepsRequired is how many consecutive sweeps must find a watch
	// with no live session before a fallback resume is attempted.
	AbsentSweepsRequired = 2
	// MaxFailures is how many resume failures within FailureWindow pause a
	// watch rather than trying again.
	MaxFailures = 3
	// FailureWindow is the sliding window failures are counted over.
	FailureWindow = 5 * time.Minute
)

// Decide is the whole reconcile policy for one watch against one snapshot.
// A paused watch is never acted on. A watch with no live session at all
// falls back to a background copy once it has been absent for
// AbsentSweepsRequired sweeps in a row, and stays put otherwise. A watch
// promised in place is left alone regardless of how many apps currently
// show it: the user's own apps are the user's business.
//
// A watch served by our own background copy is left alone too, even when
// an app shows the session beside it: a copy is never stopped to make room.
// Two things are done about such a watch. When the copy is gone and an app
// shows the session, that is Rejoin: the person reopened it, or the app
// restored it, and the watch goes back to watching it there. When there is
// more than one background copy, that is Dedupe: the watch's own short id
// is kept and the rest are stopped and removed.
func Decide(w state.Watch, snap observe.Snapshot, absentCount int) Decision {
	if w.Paused {
		return None
	}
	present := snap.All(w.SessionID)
	if len(present) == 0 {
		if absentCount >= AbsentSweepsRequired {
			return Fallback
		}
		return None
	}
	if w.PromiseState != "fallback" {
		return None
	}
	background := 0
	for _, s := range present {
		if s.Host == claude.HostBackground {
			background++
		}
	}
	if background == 0 {
		return Rejoin
	}
	if background > 1 {
		return Dedupe
	}
	return None
}

// ShouldPauseForFailures reports whether MaxFailures or more of the given
// failure times fall within FailureWindow of now.
func ShouldPauseForFailures(failures []time.Time, now time.Time) bool {
	n := 0
	for _, f := range failures {
		if now.Sub(f) <= FailureWindow {
			n++
		}
	}
	return n >= MaxFailures
}
