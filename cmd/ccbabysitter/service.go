package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/buildinfo"
	"ccbabysitter.dev/ccbabysitter/internal/procs"
	"ccbabysitter.dev/ccbabysitter/internal/state"
)

// On a Linux server a plain run sets CC Babysitter up as a systemd user
// service and hands the terminal back, instead of serving in the
// foreground for only as long as the ssh session lasts. This file holds
// the decisions and the text, which are the same on every platform; the
// systemd side of it lives in install_linux.go behind serviceControl.

const (
	// pageWaitTimeout is how long a run that started the service waits
	// for its page to answer before saying it did not start.
	pageWaitTimeout = 15 * time.Second
	// pageWaitInterval is how often it looks while it waits.
	pageWaitInterval = 250 * time.Millisecond
)

const (
	noServiceLine  = "Could not set CC Babysitter up as a service here, so it runs only while this terminal stays open."
	notStartedLine = "CC Babysitter did not start. See: journalctl --user -u ccbabysitter"
	otherCopyLine  = "CC Babysitter is already running in another terminal. Stop it there with Ctrl+C, then run ccbabysitter again to set it up as a service."
)

// serviceControl is the systemd user unit, and the user's lingering, as
// far as starting CC Babysitter as a service needs them. The real one runs
// systemctl and loginctl; tests use a fake so they never touch the real
// service manager, lingering or home folder.
type serviceControl interface {
	lingering
	// Usable reports whether the user's service manager answers at all.
	Usable() bool
	// Installed reports whether the unit file is there.
	Installed() bool
	// Install writes the unit, then does what Enable does, telling out
	// about any step that failed. It reports whether the service was
	// enabled and started.
	Install(out io.Writer) bool
	// RefreshUnit writes the installed unit again when it is not the one
	// this program would write now, such as one an older version wrote or
	// one naming a program that has since moved, and has systemd read it
	// again, telling out about any step that failed. It reports whether
	// the unit was written, and whether everything it did worked; a unit
	// that is already the right one is left alone.
	RefreshUnit(out io.Writer) (rewritten, ok bool)
	// Enable enables the installed service so it starts at boot and
	// starts it if it is not running, which leaves a running one alone,
	// telling out about any step that failed. It is safe to repeat. It
	// reports whether the service was enabled and started. Lingering is
	// not its business: runServiceStep turns it on afterwards.
	Enable(out io.Writer) bool
	// Restart stops the running service and starts it again, telling out
	// when that failed, and reports whether it worked.
	Restart(out io.Writer) bool
	// Active reports whether the service is running now.
	Active() bool
}

// servicePage is the page a started service answered on, and the version
// of CC Babysitter that answered, empty when it did not say.
type servicePage struct {
	URL     string
	Version string
}

// waitFunc waits for the service's page to answer, reading its address
// from stateDir, and reports whether it did.
type waitFunc func(stateDir string) (servicePage, bool)

// setsUpService reports whether a run should set CC Babysitter up as a
// service rather than serve in this terminal: only a plain run, with no
// arguments at all, on a Linux machine that is headless, and not one
// systemd itself started, which it marks by setting INVOCATION_ID. Any
// flag at all means the foreground, which keeps units written by older
// versions, whose ExecStart passes --no-open, working as they always have.
func setsUpService(goos string, args []string, invocationID string, headless bool) bool {
	return goos == "linux" && len(args) == 0 && invocationID == "" && headless
}

// serviceStep is what a run that sets up the service does next.
type serviceStep int

const (
	// stepNoService serves in this terminal, since there is no user
	// service manager to hand CC Babysitter to.
	stepNoService serviceStep = iota
	// stepInstall installs the unit and starts it.
	stepInstall
	// stepStart enables the installed unit and starts it if it is not
	// running already.
	stepStart
)

func (s serviceStep) String() string {
	switch s {
	case stepNoService:
		return "no service"
	case stepInstall:
		return "install"
	case stepStart:
		return "start"
	}
	return "unknown"
}

// serviceStepFor picks the step from whether the user's service manager
// answers and whether the unit is installed.
func serviceStepFor(usable, installed bool) serviceStep {
	switch {
	case !usable:
		return stepNoService
	case !installed:
		return stepInstall
	default:
		return stepStart
	}
}

// statusBlock is what a run that set up or found the service prints before
// it gives the terminal back: that it is running, and how to reach its
// page from another computer, with the page's key when there is one.
func statusBlock(version string, port int, user, address, key string) string {
	return fmt.Sprintf("%s %s is running and starts again when this server boots.\n", buildinfo.Name, version) +
		"To open its page, connect from your computer with:\n" +
		"  " + tunnelHint(port, user, address) + "\n" +
		"then open " + state.KeyedURL(fmt.Sprintf("http://127.0.0.1:%d", port), key) + "\n"
}

// startAsService is a plain run on a Linux server. It reports foreground
// true when there is no service manager to use, after saying so, and the
// caller then serves in this terminal as before. Otherwise it installs or
// starts the service, waits for its page and returns the exit code.
func startAsService(out io.Writer, sc serviceControl, stateDir string, wait waitFunc) (rc int, foreground bool) {
	step := serviceStepFor(sc.Usable(), sc.Installed())
	if step == stepNoService {
		fmt.Fprintln(out, noServiceLine)
		return 0, true
	}
	return runServiceStep(out, sc, step, stateDir, wait), false
}

// installService is the install subcommand on Linux: the same as a plain
// run on a server, whether or not this machine has a display, except that
// a service manager that does not answer is reported as the systemctl step
// that failed rather than turned into a run in this terminal.
func installService(out io.Writer, sc serviceControl, stateDir string, wait waitFunc) int {
	return runServiceStep(out, sc, serviceStepFor(true, sc.Installed()), stateDir, wait)
}

// runServiceStep installs, or enables and starts when needed, the
// service, waits for its page to answer and prints the status block. The
// address a person connects to is read first, which also saves the
// address of this ssh session for the service, since it has no ssh
// connection of its own. An installed service is enabled and lingering
// turned on again every time, so the status block's promise to start at
// boot holds even for a unit someone disabled, or one the start at login
// setting wrote without lingering. Lingering that CC Babysitter turns on
// is noted in stateDir, so uninstall turns off only what it turned on.
//
// An installed unit that is not the one this program would write now is
// written again first, and a service that was running from the old one is
// restarted so it runs as the new one says. A service that answers with a
// version other than this program's, such as one still running a program
// replaced since it started, is restarted once too, and the status block
// names the version that answers in the end. Otherwise a running service
// is left running, not restarted.
//
// A copy of CC Babysitter that holds the state folder's lock while the
// service is not running is one started in a terminal: nothing is set up
// while it runs, since the service could not start beside it.
func runServiceStep(out io.Writer, sc serviceControl, step serviceStep, stateDir string, wait waitFunc) int {
	user, address := currentUserAndAddress(stateDir)

	active := sc.Active()
	if !active && anotherCopyRunning(stateDir) {
		fmt.Fprintln(out, otherCopyLine)
		return 1
	}

	restarted := false
	switch step {
	case stepInstall:
		if !sc.Install(out) {
			return 1
		}
	case stepStart:
		rewritten, ok := sc.RefreshUnit(out)
		if !ok {
			return 1
		}
		if !sc.Enable(out) {
			return 1
		}
		if rewritten && active {
			if !sc.Restart(out) {
				return 1
			}
			restarted = true
		}
	}
	turnLingeringOn(out, sc, stateDir, user)

	page, ok := wait(stateDir)
	if ok && !restarted && page.Version != "" && page.Version != buildinfo.Version {
		if !sc.Restart(out) {
			return 1
		}
		page, ok = wait(stateDir)
	}
	port, err := strconv.Atoi(portOf(page.URL))
	if !ok || err != nil {
		fmt.Fprintln(out, notStartedLine)
		return 1
	}
	version := page.Version
	if version == "" {
		version = buildinfo.Version
	}
	fmt.Fprint(out, statusBlock(version, port, user, address, state.ReadPageKey(stateDir)))
	return 0
}

// unitUpToDate reports whether the unit file at path says exactly want.
// A file that cannot be read is not up to date, so it is written again.
func unitUpToDate(path, want string) bool {
	have, err := os.ReadFile(path)
	return err == nil && string(have) == want
}

// anotherCopyRunning reports whether a copy of CC Babysitter is serving
// from stateDir now: its lock is there and names a process that is still
// alive.
func anotherCopyRunning(stateDir string) bool {
	state.SetPIDChecker(procs.NewReal().Exists)
	return state.IsHeld(stateDir)
}

// waitForService waits the usual time for the service's page to answer.
func waitForService(stateDir string) (servicePage, bool) {
	state.SetPIDChecker(procs.NewReal().Exists)
	return waitForPage(stateDir, pageWaitTimeout, pageWaitInterval, keyedPageState(stateDir))
}

// keyedPageState is pageState with the page's key read from stateDir on
// every look: a service started for the first time writes its key only as
// it starts, after the wait for it has begun.
func keyedPageState(stateDir string) func(pageURL string) (string, bool) {
	return func(pageURL string) (string, bool) {
		return pageState(pageURL, state.ReadPageKey(stateDir))
	}
}

// waitForPage looks every interval, for up to timeout, for a page address
// saved in stateDir that answers, and returns the first one that does
// with the version it answered with. A saved address is asked only while
// a live copy holds stateDir's lock: one left behind by a copy that
// crashed may name a port another account's server listens on now, and
// the folder's key must never be sent there.
func waitForPage(stateDir string, timeout, interval time.Duration, answers func(pageURL string) (version string, ok bool)) (servicePage, bool) {
	deadline := time.Now().Add(timeout)
	for {
		if u := state.PageURL(stateDir); u != "" && state.IsHeld(stateDir) {
			if version, ok := answers(u); ok {
				return servicePage{URL: u, Version: version}, true
			}
		}
		if !time.Now().Add(interval).Before(deadline) {
			return servicePage{}, false
		}
		time.Sleep(interval)
	}
}

// pageState asks the page at pageURL for its state, sending key the way
// the command line does, which is a request the page's own guard lets
// through from a program on this machine, and reports whether it answered
// and the version of CC Babysitter it says it is, empty when the answer
// does not say. An empty key is not sent, and the page then refuses.
func pageState(pageURL, key string) (version string, ok bool) {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodGet, pageURL+"/api/state", nil)
	if err != nil {
		return "", false
	}
	if key != "" {
		req.Header.Set("Authorization", "token "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", false
	}
	var view struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&view); err != nil {
		return "", true
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return view.Version, true
}
