package supervise

import (
	"path/filepath"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/claude"
)

// trustInterval is how often the stats worker is asked to look at the CLI's
// settings file again. The worker reads it only when it has changed.
const trustInterval = 5 * time.Second

// The two reasons a background copy of a session would not start where the
// session is, said before anything is tried. Babysitting is still allowed;
// the card keeps saying it.
const (
	FallbackUntrusted = "Background won't start here: run `claude` once in this folder."
	FallbackHome      = "Background can't start in the home folder."
)

// fallbackWarning says why a background copy of a session living in app,
// started in cwd, would fail to start, or nothing when it would start or
// the answer is not known. A session that already lives in the background
// has shown the folder works.
func (s *Supervisor) fallbackWarning(cwd string, app claude.Host) string {
	switch app {
	case claude.HostTerminal, claude.HostDesktop, claude.HostVSCode:
	default:
		return ""
	}
	if cwd == "" {
		return ""
	}
	if s.deps.Home != "" && filepath.Clean(cwd) == filepath.Clean(s.deps.Home) {
		return FallbackHome
	}
	if trusted, known := s.trust.Trusted(cwd); known && !trusted {
		return FallbackUntrusted
	}
	return ""
}

// askForTrust asks the stats worker to look at the CLI's settings file
// again when the reading held is old. The file can be large and the CLI
// rewrites it often, so it is never read on the loop: until the first
// answer is back nothing is known, and no warning is claimed.
func (s *Supervisor) askForTrust() {
	if s.trustInFlight {
		return
	}
	now := s.deps.Now()
	if !s.trustAt.IsZero() && now.Sub(s.trustAt) < trustInterval {
		return
	}
	if s.postStats(statsRequest{trust: true, at: now}) {
		s.trustAt, s.trustInFlight = now, true
	}
}
