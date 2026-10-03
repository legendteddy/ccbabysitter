package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ccbabysitter.dev/ccbabysitter/internal/buildinfo"
	"ccbabysitter.dev/ccbabysitter/internal/claude"
)

func TestRoundTripStampsVersion(t *testing.T) {
	dir := t.TempDir()
	st := &Store{Dir: dir}
	s := State{Settings: DefaultSettings()}
	s.Watches = append(s.Watches, Watch{SessionID: "11111111-2222-4333-8444-555555555501", ShortID: "11111111", Cwd: "/home/dev/ws", Name: "demo-a1",
		OriginHost: claude.HostDesktop, OriginRemoteControl: true, HasSavedOptions: true,
		WatchedSince: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC), Paused: true, PauseReason: "transcript missing",
		Failures: []time.Time{time.Date(2026, 9, 21, 10, 1, 0, 0, time.UTC)}, PromiseState: "paused"})
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if !strings.Contains(string(raw), `"version": "`+buildinfo.Version+`"`) || !strings.Contains(string(raw), `"originHost": "desktop"`) {
		t.Fatalf("file:\n%s", raw)
	}
	back, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	w := back.Watches[0]
	if w.OriginHost != claude.HostDesktop || !w.Paused || w.PauseReason != "transcript missing" || len(w.Failures) != 1 || !w.HasSavedOptions {
		t.Fatalf("%+v", w)
	}
}

// A state file that a power cut caught mid-save would come back as
// nonsense, so the bytes have to be on the disk before the rename makes
// them the state file.
func TestSaveFlushesBeforeTheRename(t *testing.T) {
	dir := t.TempDir()
	st := &Store{Dir: dir}
	if err := st.Save(State{Settings: DefaultSettings()}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("state.json is empty after a save")
	}
	if runtime.GOOS != "windows" {
		if mode := fi.Mode().Perm(); mode != 0o600 {
			t.Fatalf("state.json mode is %v, want 0600", mode)
		}
		di, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if mode := di.Mode().Perm(); mode != 0o700 {
			t.Fatalf("state folder mode is %v, want 0700", mode)
		}
	}
}

// A folder left behind by an earlier build can be readable by every
// account on the machine, and it holds working directories and session
// names.
func TestEnsureDirTightensAnExistingFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows has no such permission bits")
	}
	dir := filepath.Join(t.TempDir(), "loose")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if mode := fi.Mode().Perm(); mode != 0o700 {
		t.Fatalf("mode is %v, want 0700", mode)
	}
}

// On a fresh account the path to our folder runs through folders nothing
// has created yet, such as the account's own data directory. Those belong
// to the account and other programs keep their files in them, so locking
// them down would be this program deciding something that is none of its
// business.
func TestEnsureDirLeavesParentsAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows has no such permission bits")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "share", "local", "ccbabysitter")
	if err := EnsureDir(dir); err != nil {
		t.Fatal(err)
	}

	// The reference is a folder created the ordinary way, so the comparison
	// says "the same as any other folder" whatever the account's umask is.
	reference := filepath.Join(root, "reference")
	if err := os.MkdirAll(reference, 0o755); err != nil {
		t.Fatal(err)
	}
	want := modeOf(t, reference)
	for _, parent := range []string{filepath.Join(root, "share"), filepath.Join(root, "share", "local")} {
		if got := modeOf(t, parent); got != want {
			t.Errorf("%s has mode %v, want %v like any other folder", parent, got, want)
		}
		if got := modeOf(t, parent); got == 0o700 {
			t.Errorf("%s was locked down to this program's own permissions", parent)
		}
	}
	if got := modeOf(t, dir); got != 0o700 {
		t.Fatalf("our own folder has mode %v, want 0700", got)
	}
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// A file the program itself would refuse to read must not be quietly
// summarised by anything that only wants to look, or `reset` would list
// watches out of a file that is about to be set aside unread.
func TestPeekRefusesAnIncompatibleMajorVersion(t *testing.T) {
	dir := t.TempDir()
	saved := `{"version":"9.4.1","watches":[{"sessionId":"11111111-2222-4333-8444-555555555501"}]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(saved), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := (&Store{Dir: dir}).Peek(); ok {
		t.Fatal("a file from another major version must not be reported as readable")
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json.bad")); err == nil {
		t.Fatal("Peek must still never rename anything")
	}

	same := t.TempDir()
	sameSaved := `{"version":"` + buildinfo.Version + `","watches":[{"sessionId":"11111111-2222-4333-8444-555555555501"}]}`
	if err := os.WriteFile(filepath.Join(same, "state.json"), []byte(sameSaved), 0o600); err != nil {
		t.Fatal(err)
	}
	st, ok := (&Store{Dir: same}).Peek()
	if !ok || len(st.Watches) != 1 {
		t.Fatalf("a file this build could read must still be readable: %+v %v", st, ok)
	}
}

// Starting from nothing looks exactly like every babysat session having
// been forgotten, so a caller has to be able to tell the two apart.
func TestLoadReportSaysWhenAFileWasSetAside(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, report, err := (&Store{Dir: dir}).LoadReport()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Healed || report.Reason == "" {
		t.Fatalf("a corrupt file must be reported: %+v", report)
	}
	if len(st.Watches) != 0 {
		t.Fatalf("%+v", st)
	}

	clean := t.TempDir()
	if _, report, err := (&Store{Dir: clean}).LoadReport(); err != nil || report.Healed {
		t.Fatalf("a missing file is not a healed one: %+v %v", report, err)
	}
	if err := (&Store{Dir: clean}).Save(State{Settings: DefaultSettings()}); err != nil {
		t.Fatal(err)
	}
	if _, report, err := (&Store{Dir: clean}).LoadReport(); err != nil || report.Healed {
		t.Fatalf("a file this build wrote is not a healed one: %+v %v", report, err)
	}
}

// The version stamped into the file is there to be read: a major number
// this build does not know means the shape of the file changed, and
// reading it anyway would quietly misinterpret every watch in it.
func TestLoadRefusesAnIncompatibleMajorVersion(t *testing.T) {
	dir := t.TempDir()
	saved := `{"version":"9.4.1","settings":{"theme":"system"},"watches":[{"sessionId":"11111111-2222-4333-8444-555555555501"}]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(saved), 0o600); err != nil {
		t.Fatal(err)
	}
	st, report, err := (&Store{Dir: dir}).LoadReport()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Healed {
		t.Fatal("a file from another major version must be set aside")
	}
	if !strings.Contains(report.Reason, "9.4.1") || !strings.Contains(report.Reason, buildinfo.Version) {
		t.Fatalf("the reason must name both versions: %q", report.Reason)
	}
	// The sentence leads with what happened, so the first words say which
	// of the two reasons it was rather than what the consequence is.
	if !strings.HasPrefix(report.Reason, "saved state was written by version") {
		t.Fatalf("the reason must lead with the version mismatch: %q", report.Reason)
	}
	if !strings.Contains(report.Reason, "state.json.bad") || !strings.Contains(report.Reason, "no sessions are being babysat") {
		t.Fatalf("the reason must say what happened to the file and what it means: %q", report.Reason)
	}
	if len(st.Watches) != 0 || st.Settings.Theme != "dark" {
		t.Fatalf("a fresh state must be used instead: %+v", st)
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json.bad")); err != nil {
		t.Fatal(".bad missing")
	}

	// A file written by another build of this same major version is read
	// exactly as it is.
	same := t.TempDir()
	sameSaved := `{"version":"` + buildinfo.Version + `","settings":{"theme":"light"}}`
	if err := os.WriteFile(filepath.Join(same, "state.json"), []byte(sameSaved), 0o600); err != nil {
		t.Fatal(err)
	}
	st2, report2, err := (&Store{Dir: same}).LoadReport()
	if err != nil || report2.Healed || st2.Settings.Theme != "light" {
		t.Fatalf("%+v %+v %v", st2, report2, err)
	}
}

func TestLoadMissingIsFreshWithDefaults(t *testing.T) {
	s, err := (&Store{Dir: t.TempDir()}).Load()
	if err != nil || len(s.Watches) != 0 || s.Settings.KeepAwakeMode != "babysitting" || !s.Settings.AutoOpenBrowser {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestLoadHealsMissingSettingsFieldsOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "state.json"), []byte("{}"), 0o644)
	s, err := (&Store{Dir: dir}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Settings.KeepAwakeMode != "babysitting" || s.Settings.Theme != "dark" {
		t.Fatalf("%+v", s.Settings)
	}

	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "state.json"), []byte(`{"settings":{"theme":"system"}}`), 0o644)
	s2, err := (&Store{Dir: dir2}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if s2.Settings.KeepAwakeMode != "babysitting" {
		t.Fatalf("KeepAwakeMode is fixed at babysitting, got %+v", s2.Settings)
	}
	if s2.Settings.Theme != "auto" {
		t.Fatalf("a saved Follow system theme reads back as auto, got %+v", s2.Settings)
	}
	if s2.Settings.AutoOpenBrowser {
		t.Fatalf("AutoOpenBrowser must not be guessed, got %+v", s2.Settings)
	}
}

func TestLoadCorruptRenamesToBad(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "state.json"), []byte("{ not json"), 0o644)
	s, err := (&Store{Dir: dir}).Load()
	if err != nil || len(s.Watches) != 0 {
		t.Fatal(s, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json.bad")); err != nil {
		t.Fatal(".bad missing")
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json")); !os.IsNotExist(err) {
		t.Fatal("corrupt file should have moved")
	}
}

func TestPeekNeverMutates(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "state.json"), []byte("{ not json"), 0o644)
	if _, ok := (&Store{Dir: dir}).Peek(); ok {
		t.Fatal("unreadable must be !ok")
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json.bad")); err == nil {
		t.Fatal("Peek must not rename")
	}
}

func TestSaveIsAtomicAndConcurrent(t *testing.T) {
	dir := t.TempDir()
	st := &Store{Dir: dir}
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func(i int) {
			for j := 0; j < 25; j++ {
				_ = st.Save(State{Settings: Settings{KeepAwakeMode: "off"}})
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected only state.json, got %d files", len(entries))
	}
	if _, err := st.Load(); err != nil {
		t.Fatal(err)
	}
}

// A file sitting where our folder should be is not something to carry on
// past: every later write would fail with something far less obvious than
// saying so here.
func TestEnsureDirRefusesAPathThatIsNotAFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ccbabysitter")
	if err := os.WriteFile(path, []byte("not a folder"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := EnsureDir(path)
	if err == nil {
		t.Fatal("a file where the folder should be must be refused")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "is not a folder") {
		t.Fatalf("the error must name the path and say what is wrong: %v", err)
	}
	// Nothing may have been written through it.
	body, readErr := os.ReadFile(path)
	if readErr != nil || string(body) != "not a folder" {
		t.Fatalf("the file must be left alone: %q %v", body, readErr)
	}
}

// A file written by an earlier build can carry keys this one no longer
// has any use for. They are ignored as the file is read, the watch they sat
// on is kept, and nothing of them is written back.
func TestAStateFileWithDroppedKeysStillLoads(t *testing.T) {
	dir := t.TempDir()
	old := `{"version":"0.1.0","explainerSeen":true,"settings":{"theme":"dark"},"watches":[` +
		`{"sessionId":"11111111-2222-4333-8444-555555555501","shortId":"11111111","cwd":"/home/dev/ws",` +
		`"originHost":"desktop","promiseState":"fallback","movedByUser":true,` +
		`"desktopEntryId":"local_11111111-2222-4333-8444-555555555501"}]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	st := &Store{Dir: dir}
	s, report, err := st.LoadReport()
	if err != nil || report.Healed {
		t.Fatalf("an older file must load as it is: %v %+v", err, report)
	}
	if len(s.Watches) != 1 || s.Watches[0].PromiseState != "fallback" || s.Watches[0].OriginHost != claude.HostDesktop {
		t.Fatalf("%+v", s.Watches)
	}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"explainerSeen", "movedByUser", "desktopEntryId"} {
		if strings.Contains(string(raw), gone) {
			t.Fatalf("%s must not be written back: %s", gone, raw)
		}
	}
}

// The computer is kept awake while a session is babysat, and that is not a
// choice any more. A file that still carries an older answer loads fine and
// reads back as the rule that is actually in force.
func TestASavedKeepAwakeModeIsIgnored(t *testing.T) {
	for _, saved := range []string{"always", "off", "sometimes"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "state.json"),
			[]byte(`{"settings":{"keepAwakeMode":"`+saved+`","theme":"dark"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		s, err := (&Store{Dir: dir}).Load()
		if err != nil {
			t.Fatal(err)
		}
		if s.Settings.KeepAwakeMode != "babysitting" {
			t.Fatalf("saved %q must read back as babysitting, got %q", saved, s.Settings.KeepAwakeMode)
		}
	}
}

// A session this program started used to be saved as having no app of its
// own. It lives in the background, and it reads back that way.
func TestAnOriginOfNoneReadsAsTheBackground(t *testing.T) {
	dir := t.TempDir()
	saved := `{"watches":[{"sessionId":"11111111-2222-4333-8444-555555555501","shortId":"11111111","originHost":"none","promiseState":"inplace"}]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(saved), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := (&Store{Dir: dir}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Watches) != 1 || s.Watches[0].OriginHost != claude.HostBackground {
		t.Fatalf("%+v", s.Watches)
	}
}

// A session this program started in the background has no other app to
// fall back from, so a watch saved as fallback for it reads as Watching in
// the background again rather than as In background for good.
func TestABackgroundOriginSavedAsFallbackReadsAsWatching(t *testing.T) {
	dir := t.TempDir()
	saved := `{"watches":[{"sessionId":"11111111-2222-4333-8444-555555555501","shortId":"11111111","originHost":"none","promiseState":"fallback"}]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(saved), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := (&Store{Dir: dir}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Watches) != 1 || s.Watches[0].OriginHost != claude.HostBackground || s.Watches[0].PromiseState != "inplace" {
		t.Fatalf("%+v", s.Watches)
	}
}

// Settings once carried a price table. A file that still has one loads as
// it is, with nothing set aside, and the table is not written back.
func TestAStateFileWithAPriceTableStillLoads(t *testing.T) {
	dir := t.TempDir()
	old := `{"version":"0.2.0","settings":{"theme":"system","prices":{` +
		`"model-a":{"inputPerM":3,"outputPerM":15,"cacheWritePerM":3.75,"cacheReadPerM":0.3},` +
		`"model-b":{"inputPerM":-4,"outputPerM":1e308}}},"watches":[` +
		`{"sessionId":"11111111-2222-4333-8444-555555555501","shortId":"11111111","cwd":"/home/dev/ws",` +
		`"originHost":"terminal","promiseState":"inplace"}]}`
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	st := &Store{Dir: dir}
	s, report, err := st.LoadReport()
	if err != nil || report.Healed {
		t.Fatalf("a file with a price table must load as it is: %v %+v", err, report)
	}
	if _, err := os.Stat(path + ".bad"); !os.IsNotExist(err) {
		t.Fatalf("nothing may be set aside: %v", err)
	}
	if len(s.Watches) != 1 || s.Watches[0].SessionID != "11111111-2222-4333-8444-555555555501" || s.Settings.Theme != "auto" {
		t.Fatalf("%+v", s)
	}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"prices", "inputPerM", "model-a"} {
		if strings.Contains(string(raw), gone) {
			t.Fatalf("%s must not be written back: %s", gone, raw)
		}
	}
}

// When a watch went to the background is kept across a restart of this
// program, and a file written before it was recorded loads with none.
func TestBackgroundSinceIsKeptAndMayBeMissing(t *testing.T) {
	dir := t.TempDir()
	st := &Store{Dir: dir}
	since := time.Date(2026, 9, 25, 8, 46, 0, 0, time.UTC)
	s := State{Settings: DefaultSettings(), Watches: []Watch{{SessionID: "11111111-2222-4333-8444-555555555501",
		ShortID: "4d4d4d4d", OriginHost: claude.HostDesktop, PromiseState: "fallback", BackgroundSince: since}}}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if !strings.Contains(string(raw), `"backgroundSince": "2026-09-25T08:46:00Z"`) {
		t.Fatalf("file:\n%s", raw)
	}
	back, err := st.Load()
	if err != nil || !back.Watches[0].BackgroundSince.Equal(since) {
		t.Fatalf("%v %+v", err, back.Watches)
	}

	s.Watches[0].BackgroundSince = time.Time{}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(dir, "state.json"))
	if strings.Contains(string(raw), "backgroundSince") {
		t.Fatalf("a watch that is not in the background has no time for it:\n%s", raw)
	}

	old := `{"version":"0.2.0","settings":{"theme":"dark"},"watches":[` +
		`{"sessionId":"11111111-2222-4333-8444-555555555501","shortId":"11111111","cwd":"/home/dev/ws",` +
		`"originHost":"desktop","promiseState":"fallback"}]}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, report, err := st.LoadReport()
	if err != nil || report.Healed || len(loaded.Watches) != 1 || !loaded.Watches[0].BackgroundSince.IsZero() {
		t.Fatalf("%v %+v %+v", err, report, loaded.Watches)
	}
}

// Babysitting every new background session on a server is on unless the
// user switched it off. A fresh start and a saved file that never mentions
// it, including one from a build that kept the choice under another name
// and started it off, both load with it on; an off that was saved stays
// off.
func TestAutoBabysitIsOnUnlessSwitchedOff(t *testing.T) {
	fresh, err := (&Store{Dir: t.TempDir()}).Load()
	if err != nil || !fresh.Settings.AutoBabysit {
		t.Fatalf("fresh: %+v %v", fresh.Settings, err)
	}

	for _, saved := range []string{
		`{"settings":{"theme":"dark"}}`,
		`{"version":"` + buildinfo.Version + `","settings":{"autostart":true,"serverAutoBabysit":false,"autoOpenBrowser":true,"theme":"dark"}}`,
		`{"version":"` + buildinfo.Version + `","settings":{"serverAutoBabysit":true,"theme":"system"}}`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(saved), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := (&Store{Dir: dir}).Load()
		if err != nil || !st.Settings.AutoBabysit {
			t.Fatalf("%s: %+v %v", saved, st.Settings, err)
		}
	}

	dir := t.TempDir()
	store := &Store{Dir: dir}
	st, _ := store.Load()
	st.Settings.AutoBabysit = false
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	back, err := store.Load()
	if err != nil || back.Settings.AutoBabysit {
		t.Fatalf("a saved off must stay off: %+v %v", back.Settings, err)
	}
	peeked, ok := store.Peek()
	if !ok || peeked.Settings.AutoBabysit {
		t.Fatalf("peek must read the saved off too: %+v", peeked.Settings)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if !strings.Contains(string(data), `"autoBabysit": false`) || strings.Contains(string(data), "serverAutoBabysit") {
		t.Fatalf("the choice is saved under its own name only:\n%s", data)
	}
}

// The saved themes read back as the three the page offers: dark stays
// dark, Follow system becomes auto, and anything else is dark.
func TestSavedThemesReadBackAsThePageOffersThem(t *testing.T) {
	for saved, want := range map[string]string{
		"dark": "dark", "light": "light", "auto": "auto", "system": "auto", "": "dark", "sepia": "dark",
	} {
		dir := t.TempDir()
		body := `{"settings":{"theme":"` + saved + `"}}`
		if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := (&Store{Dir: dir}).Load()
		if err != nil || st.Settings.Theme != want {
			t.Errorf("saved %q: got %q, want %q (%v)", saved, st.Settings.Theme, want, err)
		}
	}
}
