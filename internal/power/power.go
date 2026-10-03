// Package power holds a user-mode "keep the system awake" request. It never
// touches display sleep, needs no elevated rights, and ends on its own if
// the process holding it dies, including a hard kill.
package power

// Mode selects when the keep-awake request should be held.
type Mode string

const (
	// ModeAlways holds the request at all times.
	ModeAlways Mode = "always"
	// ModeBabysitting holds the request only while at least one watch is active.
	ModeBabysitting Mode = "babysitting"
	// ModeOff never holds the request.
	ModeOff Mode = "off"
)

// KeepAwake acquires and releases the system-sleep-prevention request for
// the current platform.
type KeepAwake interface {
	// Acquire holds the request and returns a release function. release is
	// safe to call more than once and safe to call from any goroutine.
	// ok is false if the request could not be acquired.
	Acquire() (release func(), ok bool)
	// Supported reports whether this platform can hold the request at all.
	Supported() bool
}

// Wanted decides whether the request should be held right now, given the
// configured mode and the number of currently active watches.
func Wanted(mode Mode, activeWatches int) bool {
	switch mode {
	case ModeAlways:
		return true
	case ModeBabysitting:
		return activeWatches > 0
	default:
		return false
	}
}
