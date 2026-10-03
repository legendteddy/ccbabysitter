package claude

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func notOnPath(string) (string, error) { return "", errors.New("not found") }

func envOf(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// hostAbs turns a slash-separated absolute path into one this OS reads as
// absolute, on drive C: on Windows. The Locator takes paths apart with the
// host's own rules, so the tests that run it as the host OS use the host's
// paths.
func hostAbs(p string) string {
	if runtime.GOOS == "windows" {
		return `C:` + filepath.FromSlash(p)
	}
	return p
}

// cliName is the CLI's file name on the host OS.
func cliName() string {
	if runtime.GOOS == "windows" {
		return "claude.exe"
	}
	return "claude"
}

// PATH wins whenever it has the CLI, and what it finds comes back as an
// absolute path.
func TestLocatorPrefersPath(t *testing.T) {
	onPath := hostAbs("/usr/bin/" + cliName())
	l := Locator{
		OS:           runtime.GOOS,
		LookPath:     func(string) (string, error) { return onPath, nil },
		Getenv:       envOf(map[string]string{"HOME": hostAbs("/home/dev")}),
		IsExecutable: func(string) bool { return true },
	}
	if got, ok := l.Find(); !ok || got != onPath {
		t.Fatalf("got %q %v", got, ok)
	}
}

// A service manager's PATH leaves out the folder the installer uses, so
// that folder is tried next, then the older local install.
func TestLocatorFallsBackToInstallFolders(t *testing.T) {
	home := hostAbs("/home/dev")
	local := filepath.Join(home, ".local", "bin", cliName())
	older := filepath.Join(home, ".claude", "local", cliName())
	for _, c := range []struct {
		present map[string]bool
		want    string
	}{
		{map[string]bool{local: true, older: true}, local},
		{map[string]bool{older: true}, older},
	} {
		l := Locator{
			OS:           runtime.GOOS,
			LookPath:     notOnPath,
			Getenv:       envOf(map[string]string{"HOME": home}),
			IsExecutable: func(p string) bool { return c.present[p] },
		}
		if got, ok := l.Find(); !ok || got != c.want {
			t.Fatalf("%v: got %q %v, want %q", c.present, got, ok, c.want)
		}
	}
}

func TestLocatorFindsNothing(t *testing.T) {
	l := Locator{
		OS:           "linux",
		LookPath:     notOnPath,
		Getenv:       envOf(map[string]string{"HOME": "/home/dev"}),
		IsExecutable: func(string) bool { return false },
	}
	if got, ok := l.Find(); ok || got != "" {
		t.Fatalf("got %q %v", got, ok)
	}
	// With no way to look at files, only PATH is asked.
	l = Locator{OS: "linux", LookPath: notOnPath}
	if got, ok := l.Find(); ok || got != "" {
		t.Fatalf("got %q %v", got, ok)
	}
	// A home folder that is not absolute is never searched.
	l = Locator{
		OS:           "linux",
		LookPath:     notOnPath,
		Getenv:       envOf(map[string]string{"HOME": "relative"}),
		IsExecutable: func(string) bool { return true },
	}
	if got, ok := l.Find(); ok || got != "" {
		t.Fatalf("got %q %v", got, ok)
	}
}

// On Windows the file is claude.exe, and the profile folder is tried after
// a HOME that is set.
func TestLocatorWindowsNames(t *testing.T) {
	var tried []string
	l := Locator{
		OS:       "windows",
		LookPath: notOnPath,
		Getenv: envOf(map[string]string{
			"HOME":        `D:\home\dev`,
			"USERPROFILE": `C:\Users\dev`,
		}),
		IsExecutable: func(p string) bool { tried = append(tried, p); return false },
	}
	l.Find()
	want := []string{
		filepath.Join(`D:\home\dev`, ".local", "bin", "claude.exe"),
		filepath.Join(`D:\home\dev`, ".claude", "local", "claude.exe"),
		filepath.Join(`C:\Users\dev`, ".local", "bin", "claude.exe"),
	}
	if len(tried) != len(want) {
		t.Fatalf("tried %q, want %q", tried, want)
	}
	for i := range want {
		if tried[i] != want[i] {
			t.Fatalf("tried %q, want %q", tried, want)
		}
	}
}

func TestIsExecutable(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain")
	run := filepath.Join(dir, "run")
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(run, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsExecutable(run) || IsExecutable(dir) || IsExecutable(filepath.Join(dir, "missing")) {
		t.Fatal("wrong answer for a runnable file, a folder or a missing file")
	}
	if runtime.GOOS != "windows" && IsExecutable(plain) {
		t.Fatal("a file with no execute bit cannot be run")
	}
}
