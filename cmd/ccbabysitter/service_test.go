package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/buildinfo"
	"ccbabysitter.dev/ccbabysitter/internal/procs"
	"ccbabysitter.dev/ccbabysitter/internal/state"
)

func TestSetsUpService(t *testing.T) {
	cases := []struct {
		name         string
		goos         string
		args         []string
		invocationID string
		headless     bool
		want         bool
	}{
		{"plain run on a linux server", "linux", nil, "", true, true},
		{"linux with a display", "linux", nil, "", false, false},
		{"macOS over ssh", "darwin", nil, "", true, false},
		{"windows over ssh", "windows", nil, "", true, false},
		{"started by systemd", "linux", nil, "4d1c0a5b", true, false},
		{"--service", "linux", []string{"--service"}, "", true, false},
		{"--no-open", "linux", []string{"--no-open"}, "", true, false},
		{"--port", "linux", []string{"--port", "5000"}, "", true, false},
		{"--demo", "linux", []string{"--demo"}, "", true, false},
	}
	for _, c := range cases {
		if got := setsUpService(c.goos, c.args, c.invocationID, c.headless); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestServiceStepFor(t *testing.T) {
	if got := serviceStepFor(false, false); got != stepNoService {
		t.Errorf("no user manager, not installed: got %v", got)
	}
	if got := serviceStepFor(false, true); got != stepNoService {
		t.Errorf("no user manager, installed: got %v", got)
	}
	if got := serviceStepFor(true, false); got != stepInstall {
		t.Errorf("not installed: got %v", got)
	}
	if got := serviceStepFor(true, true); got != stepStart {
		t.Errorf("installed: got %v", got)
	}
}

func TestStatusBlock(t *testing.T) {
	key := strings.Repeat("ab", 32)
	want := "CC Babysitter 1.2.3 is running and starts again when this server boots.\n" +
		"To open its page, connect from your computer with:\n" +
		"  ssh -L 47391:127.0.0.1:47391 alice@203.0.113.7\n" +
		"then open http://127.0.0.1:47391/?token=" + key + "\n"
	if got := statusBlock("1.2.3", 47391, "alice", "203.0.113.7", key); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	// Without a key to read there is only the plain address to give.
	if got := statusBlock("1.2.3", 47391, "alice", "203.0.113.7", ""); !strings.HasSuffix(got, "then open http://127.0.0.1:47391\n") {
		t.Fatalf("no key: %q", got)
	}
}

// fakeService records what was asked of it and answers from its fields.
type fakeService struct {
	usable, installed, active bool
	installOK, enableOK       bool
	onInstall                 func()
	// staleUnit is whether the installed unit differs from the one this
	// program would write, which RefreshUnit then writes again.
	staleUnit bool
	refreshOK bool
	restartOK bool
	// lingerOn is whether lingering is on; lingerErr makes asking about
	// it fail, and setLingerErr makes changing it fail.
	lingerOn     bool
	lingerErr    error
	setLingerErr error

	calls []string
}

func (f *fakeService) Lingering(user string) (bool, error) {
	f.calls = append(f.calls, "lingering")
	return f.lingerOn, f.lingerErr
}

func (f *fakeService) SetLingering(user string, on bool) error {
	if on {
		f.calls = append(f.calls, "linger on")
	} else {
		f.calls = append(f.calls, "linger off")
	}
	if f.setLingerErr != nil {
		return f.setLingerErr
	}
	f.lingerOn = on
	return nil
}

func (f *fakeService) RefreshUnit(out io.Writer) (bool, bool) {
	f.calls = append(f.calls, "refresh")
	if !f.refreshOK {
		fmt.Fprintln(out, "could not write the service file: permission denied")
		return false, false
	}
	rewritten := f.staleUnit
	f.staleUnit = false
	return rewritten, true
}

func (f *fakeService) Restart(out io.Writer) bool {
	f.calls = append(f.calls, "restart")
	if !f.restartOK {
		fmt.Fprintln(out, "systemctl --user restart ccbabysitter failed: exit status 1")
	}
	return f.restartOK
}

// count is how many times call was made.
func (f *fakeService) count(call string) int {
	n := 0
	for _, c := range f.calls {
		if c == call {
			n++
		}
	}
	return n
}

func (f *fakeService) Usable() bool    { f.calls = append(f.calls, "usable"); return f.usable }
func (f *fakeService) Installed() bool { f.calls = append(f.calls, "installed"); return f.installed }
func (f *fakeService) Active() bool    { f.calls = append(f.calls, "active"); return f.active }
func (f *fakeService) Enable(out io.Writer) bool {
	f.calls = append(f.calls, "enable")
	if !f.enableOK {
		fmt.Fprintln(out, "systemctl --user enable --now ccbabysitter failed: exit status 1")
	}
	return f.enableOK
}
func (f *fakeService) Install(out io.Writer) bool {
	f.calls = append(f.calls, "install")
	if f.onInstall != nil {
		f.onInstall()
	}
	if !f.installOK {
		fmt.Fprintln(out, "systemctl --user daemon-reload failed: exit status 1")
	}
	return f.installOK
}

func (f *fakeService) did(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

// answeringAt is a service whose page answers at url with this program's
// own version.
func answeringAt(url string) waitFunc {
	return func(string) (servicePage, bool) { return servicePage{URL: url, Version: buildinfo.Version}, true }
}

// answeringWith is a service whose page answers at url with each version
// in turn, one per wait, and with the last one from then on.
func answeringWith(url string, versions ...string) (waitFunc, *int) {
	waits := 0
	return func(string) (servicePage, bool) {
		v := versions[min(waits, len(versions)-1)]
		waits++
		return servicePage{URL: url, Version: v}, true
	}, &waits
}

func neverAnswering(string) (servicePage, bool) { return servicePage{}, false }

func TestStartAsServiceFallsBackWithoutAUserManager(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: false}
	rc, foreground := startAsService(&out, f, t.TempDir(), neverAnswering)
	if !foreground || rc != 0 {
		t.Fatalf("rc %d, foreground %v", rc, foreground)
	}
	if out.String() != "Could not set CC Babysitter up as a service here, so it runs only while this terminal stays open.\n" {
		t.Fatalf("got %q", out.String())
	}
	if f.did("install") || f.did("enable") {
		t.Fatalf("nothing is set up: %v", f.calls)
	}
}

func TestStartAsServiceInstallsThenPrintsTheStatus(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "198.51.100.4 50000 203.0.113.7 22")
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	var out strings.Builder
	f := &fakeService{usable: true, installOK: true}
	// The service has no ssh connection of its own, so the address has to
	// be saved before it is started.
	savedFirst := false
	f.onInstall = func() { savedFirst = state.ServerAddress(dir) == "203.0.113.7" }

	rc, foreground := startAsService(&out, f, dir, answeringAt("http://127.0.0.1:47500"))
	if foreground || rc != 0 {
		t.Fatalf("rc %d, foreground %v, out %q", rc, foreground, out.String())
	}
	if !f.did("install") || f.did("enable") {
		t.Fatalf("calls %v", f.calls)
	}
	if !savedFirst {
		t.Fatal("the ssh address was not saved before the service was started")
	}
	if want := statusBlock(buildinfo.Version, 47500, currentUser(), "203.0.113.7", ""); out.String() != want {
		t.Fatalf("got\n%s\nwant\n%s", out.String(), want)
	}
}

func TestStartAsServiceStopsWhenInstallFails(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installOK: false}
	rc, foreground := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47500"))
	if foreground || rc != 1 {
		t.Fatalf("rc %d, foreground %v", rc, foreground)
	}
	if strings.Contains(out.String(), "is running") {
		t.Fatalf("no status after a failed install:\n%s", out.String())
	}
}

// An installed service is enabled and started every time, not only
// started, so the promise to start at boot holds even for a unit someone
// disabled or one written without lingering.
func TestStartAsServiceEnablesAnInstalledServiceThatIsNotActive(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: false, enableOK: true, refreshOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 0 || !f.did("enable") || f.did("install") {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
	if !strings.Contains(out.String(), "is running and starts again when this server boots.") {
		t.Fatalf("got %q", out.String())
	}
}

// A service that is already running is enabled again, which leaves it
// running rather than restarting it, and its status is printed.
func TestStartAsServiceEnablesAnActiveServiceToo(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, enableOK: true, refreshOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 0 || !f.did("enable") || f.did("install") {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
}

func TestStartAsServiceReportsAnEnableFailure(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, enableOK: false, refreshOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 1 || out.String() != "systemctl --user enable --now ccbabysitter failed: exit status 1\n" {
		t.Fatalf("rc %d, out %q", rc, out.String())
	}
}

// holdLock takes the state folder's lock the way a running copy does,
// naming this test process, which is alive.
func holdLock(t *testing.T, dir string) {
	t.Helper()
	createMs, ok := procs.NewReal().CreateTime(os.Getpid())
	if !ok {
		t.Skip("cannot read this process's start time here")
	}
	lock, err := state.Acquire(dir, createMs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lock.Release)
}

// A copy started in a terminal holds the lock while the service is not
// running. Nothing is installed or started beside it, since the service
// could not start while it runs.
func TestStartAsServiceRefusesBesideACopyInATerminal(t *testing.T) {
	for _, installed := range []bool{false, true} {
		dir := filepath.Join(t.TempDir(), "ccbabysitter")
		holdLock(t, dir)
		var out strings.Builder
		f := &fakeService{usable: true, installed: installed, active: false, installOK: true, enableOK: true, refreshOK: true}
		rc, foreground := startAsService(&out, f, dir, answeringAt("http://127.0.0.1:47391"))
		if rc != 1 || foreground {
			t.Fatalf("installed %v: rc %d, foreground %v", installed, rc, foreground)
		}
		if want := "CC Babysitter is already running in another terminal. Stop it there with Ctrl+C, then run ccbabysitter again to set it up as a service.\n"; out.String() != want {
			t.Fatalf("installed %v: got %q", installed, out.String())
		}
		if f.did("install") || f.did("enable") {
			t.Fatalf("installed %v: calls %v", installed, f.calls)
		}
	}
}

// The same held lock with the service running is the service itself: the
// normal case, which prints the status.
func TestStartAsServiceWithTheServiceHoldingTheLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	holdLock(t, dir)
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, enableOK: true, refreshOK: true}
	rc, _ := startAsService(&out, f, dir, answeringAt("http://127.0.0.1:47391"))
	if rc != 0 || !strings.Contains(out.String(), "is running and starts again when this server boots.") {
		t.Fatalf("rc %d, out %q", rc, out.String())
	}
}

// install refuses beside a copy in a terminal too.
func TestInstallServiceRefusesBesideACopyInATerminal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	holdLock(t, dir)
	var out strings.Builder
	f := &fakeService{installOK: true}
	if rc := installService(&out, f, dir, answeringAt("http://127.0.0.1:47391")); rc != 1 || f.did("install") {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
}

func TestStartAsServiceSaysWhenTheServiceNeverAnswers(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, enableOK: true, refreshOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), neverAnswering)
	if rc != 1 || out.String() != "CC Babysitter did not start. See: journalctl --user -u ccbabysitter\n" {
		t.Fatalf("rc %d, out %q", rc, out.String())
	}
}

// install skips the look at the user manager: a failure there shows up as
// the systemctl step that failed, as it always has, and never as a run in
// this terminal.
func TestInstallServiceDoesNotFallBack(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: false, installOK: true}
	if rc := installService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391")); rc != 0 {
		t.Fatalf("rc %d, out %q", rc, out.String())
	}
	if f.did("usable") || !f.did("install") {
		t.Fatalf("calls %v", f.calls)
	}
}

func TestWaitForPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/state" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"version":"1.2.3","sessions":[]}`)
	}))
	defer ts.Close()

	dir := t.TempDir()
	if _, ok := waitForPage(dir, 100*time.Millisecond, 10*time.Millisecond, keyedPageState(dir)); ok {
		t.Fatal("no page address saved, so nothing answers")
	}

	if err := state.SavePageURL(dir, ts.URL); err != nil {
		t.Fatal(err)
	}
	if _, ok := waitForPage(dir, 100*time.Millisecond, 10*time.Millisecond, keyedPageState(dir)); ok {
		t.Fatal("an address no live copy holds the lock for is not asked")
	}
	holdLock(t, dir)
	got, ok := waitForPage(dir, time.Second, 10*time.Millisecond, keyedPageState(dir))
	if !ok || got.URL != ts.URL || got.Version != "1.2.3" {
		t.Fatalf("got %+v, %v", got, ok)
	}

	ts.Close()
	if _, ok := waitForPage(dir, 100*time.Millisecond, 10*time.Millisecond, keyedPageState(dir)); ok {
		t.Fatal("a page address nobody serves any more does not answer")
	}
}

// The address may be written only after the wait has begun, as it is
// when the service is still starting up.
func TestWaitForPageKeepsLooking(t *testing.T) {
	dir := t.TempDir()
	looks := 0
	answers := func(url string) (string, bool) { looks++; return "1.2.3", true }
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = state.SavePageURL(dir, "http://127.0.0.1:47391")
	}()
	holdLock(t, dir)
	got, ok := waitForPage(dir, 2*time.Second, 10*time.Millisecond, answers)
	if !ok || got.URL != "http://127.0.0.1:47391" || got.Version != "1.2.3" || looks != 1 {
		t.Fatalf("got %+v, %v, looks %d", got, ok, looks)
	}
}

// A page that answers without saying its version still answers.
func TestPageStateWithoutAVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not json")
	}))
	defer ts.Close()
	if version, ok := pageState(ts.URL, ""); !ok || version != "" {
		t.Fatalf("got %q, %v", version, ok)
	}
}

func TestUnitUpToDate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ccbabysitter.service")
	want := mustUnitFile(t, "/home/dev/.local/bin/ccbabysitter", "/home/dev/.local/bin/claude")
	if unitUpToDate(path, want) {
		t.Fatal("a missing unit is not up to date")
	}
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	if !unitUpToDate(path, want) {
		t.Fatal("the same unit is up to date")
	}
	// One an older version wrote: another ExecStart and no PATH.
	old := strings.Replace(mustUnitFile(t, "/home/dev/.local/bin/ccbabysitter", ""), " --service", " --no-open", 1)
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if unitUpToDate(path, want) {
		t.Fatal("an older unit is not up to date")
	}
	// One written before KillMode=process was added, which would stop
	// Claude's background sessions along with the service.
	if err := os.WriteFile(path, []byte(strings.Replace(want, "KillMode=process\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if unitUpToDate(path, want) {
		t.Fatal("a unit without KillMode is not up to date")
	}
	// One naming the program where it used to be.
	if err := os.WriteFile(path, []byte(mustUnitFile(t, "/opt/old/ccbabysitter", "/home/dev/.local/bin/claude")), 0o644); err != nil {
		t.Fatal(err)
	}
	if unitUpToDate(path, want) {
		t.Fatal("a unit naming a moved program is not up to date")
	}
}

// A running service whose unit is not the one this program would write
// now gets the unit written again and is restarted, so it runs as the new
// unit says.
func TestStartAsServiceRewritesAStaleUnitAndRestarts(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, staleUnit: true, refreshOK: true, enableOK: true, restartOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 0 {
		t.Fatalf("rc %d, out %q", rc, out.String())
	}
	if got := strings.Join(f.calls, " "); got != "usable installed active refresh enable restart lingering linger on" {
		t.Fatalf("calls %q", got)
	}
	if !strings.Contains(out.String(), "is running and starts again when this server boots.") {
		t.Fatalf("got %q", out.String())
	}
}

// A stale unit on a service that is not running is written again and then
// started by enabling it, with no restart on top.
func TestStartAsServiceRewritesAStaleUnitOfAStoppedService(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: false, staleUnit: true, refreshOK: true, enableOK: true, restartOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 0 || !f.did("refresh") || !f.did("enable") || f.did("restart") {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
}

// A unit that is already the right one is left alone, and a running
// service answering with this program's version is not restarted.
func TestStartAsServiceLeavesARightUnitAndItsServiceAlone(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, refreshOK: true, enableOK: true, restartOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 0 || f.did("restart") || f.did("install") {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
}

func TestStartAsServiceReportsAUnitThatCouldNotBeWritten(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, staleUnit: true, refreshOK: false, enableOK: true, restartOK: true}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 1 || f.did("enable") || f.did("restart") || out.String() != "could not write the service file: permission denied\n" {
		t.Fatalf("rc %d, calls %v, out %q", rc, f.calls, out.String())
	}
}

func TestStartAsServiceReportsARestartFailure(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, staleUnit: true, refreshOK: true, enableOK: true, restartOK: false}
	rc, _ := startAsService(&out, f, t.TempDir(), answeringAt("http://127.0.0.1:47391"))
	if rc != 1 || out.String() != "systemctl --user restart ccbabysitter failed: exit status 1\n" {
		t.Fatalf("rc %d, out %q", rc, out.String())
	}
}

// A service still running another version, such as one whose program was
// replaced after it started, is restarted once, waited for again, and the
// status names the version that answers then.
func TestStartAsServiceRestartsAnotherVersionOnce(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "198.51.100.4 50000 203.0.113.7 22")
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, refreshOK: true, enableOK: true, restartOK: true}
	wait, waits := answeringWith("http://127.0.0.1:47391", "0.0.1-older", buildinfo.Version)
	rc, _ := startAsService(&out, f, t.TempDir(), wait)
	if rc != 0 || f.count("restart") != 1 || *waits != 2 {
		t.Fatalf("rc %d, calls %v, waits %d", rc, f.calls, *waits)
	}
	if want := statusBlock(buildinfo.Version, 47391, currentUser(), "203.0.113.7", ""); out.String() != want {
		t.Fatalf("got\n%s\nwant\n%s", out.String(), want)
	}
}

// Once is all: a service that still answers with another version after the
// restart is not restarted again, and the status names the version it
// says it is.
func TestStartAsServiceRestartsAnotherVersionOnlyOnce(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, refreshOK: true, enableOK: true, restartOK: true}
	wait, waits := answeringWith("http://127.0.0.1:47391", "0.0.1-older")
	rc, _ := startAsService(&out, f, t.TempDir(), wait)
	if rc != 0 || f.count("restart") != 1 || *waits != 2 {
		t.Fatalf("rc %d, calls %v, waits %d", rc, f.calls, *waits)
	}
	if !strings.HasPrefix(out.String(), "CC Babysitter 0.0.1-older is running") {
		t.Fatalf("got %q", out.String())
	}
}

// A service just restarted for a rewritten unit is not restarted a second
// time for its version.
func TestStartAsServiceRestartsAtMostOnce(t *testing.T) {
	var out strings.Builder
	f := &fakeService{usable: true, installed: true, active: true, staleUnit: true, refreshOK: true, enableOK: true, restartOK: true}
	wait, _ := answeringWith("http://127.0.0.1:47391", "0.0.1-older")
	if rc, _ := startAsService(&out, f, t.TempDir(), wait); rc != 0 || f.count("restart") != 1 {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
}

// install restarts a service answering with another version too.
func TestInstallServiceRestartsAnotherVersion(t *testing.T) {
	var out strings.Builder
	f := &fakeService{installed: true, active: true, refreshOK: true, enableOK: true, restartOK: true}
	wait, _ := answeringWith("http://127.0.0.1:47391", "0.0.1-older", buildinfo.Version)
	if rc := installService(&out, f, t.TempDir(), wait); rc != 0 || f.count("restart") != 1 {
		t.Fatalf("rc %d, calls %v", rc, f.calls)
	}
}

// keyedPage serves /api/state with this program's version only to a
// request that carries key, the way CC Babysitter's guard does.
func keyedPage(t *testing.T, key string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/state" || r.Header.Get("Authorization") != "token "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"version":"`+buildinfo.Version+`"}`)
	}))
	t.Cleanup(ts.Close)
	return ts
}

// The service's page needs its key like any other copy's, and the line
// printed once it answers gives the address with that key. The key is
// read again on every look, since a service started for the first time
// writes it only as it starts.
func TestServicePageCarriesTheKey(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "198.51.100.4 50000 203.0.113.7 22")
	dir := filepath.Join(t.TempDir(), "ccbabysitter")
	key := strings.Repeat("cd", 32)
	ts := keyedPage(t, key)
	if err := state.SavePageURL(dir, ts.URL); err != nil {
		t.Fatal(err)
	}
	if _, ok := waitForPage(dir, 50*time.Millisecond, 10*time.Millisecond, keyedPageState(dir)); ok {
		t.Fatal("the page answered before there was a key to send")
	}
	f := &fakeService{usable: true, installOK: true}
	// The service, as it starts, takes the lock and writes its key.
	f.onInstall = func() {
		holdLock(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "page-key"), []byte(key+"\n"), 0o600); err != nil {
			t.Error(err)
		}
	}
	wait := func(stateDir string) (servicePage, bool) {
		return waitForPage(stateDir, time.Second, 10*time.Millisecond, keyedPageState(stateDir))
	}
	var out strings.Builder
	rc, foreground := startAsService(&out, f, dir, wait)
	if foreground || rc != 0 {
		t.Fatalf("rc %d, foreground %v, out %q", rc, foreground, out.String())
	}
	if want := "then open " + ts.URL + "/?token=" + key + "\n"; !strings.Contains(out.String(), want) {
		t.Fatalf("missing %q in\n%s", want, out.String())
	}
}

// A page address left behind by a service that crashed may name a port
// another account's server listens on now: until a live copy holds the
// lock again, the wait asks nothing there and keeps waiting.
func TestWaitForPageSendsNoKeyToAStaleAddress(t *testing.T) {
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"version":"1.2.3"}`)
	}))
	defer ts.Close()
	dir := t.TempDir()
	if _, err := state.PageKey(dir); err != nil {
		t.Fatal(err)
	}
	if err := state.SavePageURL(dir, ts.URL); err != nil {
		t.Fatal(err)
	}
	if _, ok := waitForPage(dir, 100*time.Millisecond, 10*time.Millisecond, keyedPageState(dir)); ok || len(seen) != 0 {
		t.Fatalf("ok %v, seen %q", ok, seen)
	}
}
