package supervise

import (
	"context"
	"errors"

	"ccbabysitter.dev/ccbabysitter/internal/hosts"
)

// OpenTerminalRefused is the answer to Open in a terminal on a machine
// with no display, where there is no window to open. The page offers the
// command to run over ssh there instead.
const OpenTerminalRefused = "There is no display here to open a terminal on."

// OpenTerminal opens the system's own terminal attached to a session's
// background copy. The copy keeps running as it is: closing that window
// only detaches from it.
func (s *Supervisor) OpenTerminal(id string) Result {
	return s.ask(func(context.Context) Result { return s.openTerminal(id) })
}

func (s *Supervisor) openTerminal(id string) Result {
	if s.env.Headless {
		return Result{Message: OpenTerminalRefused}
	}
	s.refreshSnap()
	live := s.snap.All(id)
	if _, running := backgroundCopy(live); !running {
		return Result{Message: "That session has no background copy running."}
	}
	known, name := "", ""
	if w := s.find(id); w != nil {
		known, name = w.ShortID, w.Name
	}
	bg, ok := copyToStop(live, known)
	if !ok {
		return Result{Message: "That session has more than one background copy running. Check `claude agents` and attach to the one you mean by hand."}
	}
	if !validShortID(bg.ShortID) {
		return Result{Message: "That background copy has an id this program cannot use safely. Check `claude agents` and attach to it by hand."}
	}
	if name == "" {
		name = bg.Name
	}
	label := sessionLabel(name, id)
	opened, err := "", errors.New("there is no way to open one here")
	if s.deps.OpenTerminal != nil {
		opened, err = s.deps.OpenTerminal(bg.ShortID)
	}
	if err != nil {
		return Result{Message: "Could not open a terminal: " + err.Error() + ". Run " + hosts.AttachCommand(bg.ShortID) + " in any terminal."}
	}
	s.logInfo(label, "opened "+opened+" attached to background copy "+bg.ShortID)
	return Result{OK: true, Message: "Opened " + opened + " on " + label + ".", ShortID: bg.ShortID}
}
