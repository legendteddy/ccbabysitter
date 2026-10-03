package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Locator finds the claude CLI. PATH comes first; when the program is not
// there, the folders the official installers put it in are tried, since a
// program started by a service manager or over a non-interactive ssh
// command often runs with a PATH that leaves those folders out. Every way
// it reaches the system is a field, so a test can script each answer.
type Locator struct {
	// OS decides the file name, claude.exe on Windows and claude elsewhere,
	// and which environment variables name a home folder.
	OS string
	// LookPath finds a program on PATH.
	LookPath func(string) (string, error)
	// Getenv reads an environment variable. A nil Getenv means no install
	// folder is tried, only PATH.
	Getenv func(string) string
	// IsExecutable reports whether path is a file that can be run. A nil
	// IsExecutable means no install folder is tried, only PATH.
	IsExecutable func(path string) bool
}

// Find returns the absolute path to the claude CLI and whether one was
// found at all.
func (l Locator) Find() (string, bool) {
	if l.LookPath != nil {
		if p, err := l.LookPath("claude"); err == nil && p != "" {
			if filepath.IsAbs(p) || isWindowsAbs(l.OS, p) {
				return p, true
			}
			if abs, err := filepath.Abs(p); err == nil {
				return abs, true
			}
		}
	}
	if l.Getenv == nil || l.IsExecutable == nil {
		return "", false
	}
	for _, c := range l.candidates() {
		if l.IsExecutable(c) {
			return c, true
		}
	}
	return "", false
}

// candidates lists the install locations tried after PATH, in order:
// ~/.local/bin, where the official installer puts the CLI, then
// ~/.claude/local, where an older local install kept it. On Windows the
// same two under %HOME% when it is set, then ~/.local/bin under
// %USERPROFILE%. A home folder that is not an absolute path is skipped, so
// the result is always absolute.
func (l Locator) candidates() []string {
	name := "claude"
	if l.OS == "windows" {
		name = "claude.exe"
	}
	var out []string
	if home := l.Getenv("HOME"); filepath.IsAbs(home) || isWindowsAbs(l.OS, home) {
		out = append(out,
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, ".claude", "local", name))
	}
	if l.OS == "windows" {
		if profile := l.Getenv("USERPROFILE"); filepath.IsAbs(profile) || isWindowsAbs(l.OS, profile) {
			out = append(out, filepath.Join(profile, ".local", "bin", name))
		}
	}
	return out
}

// isWindowsAbs reports whether p starts with a drive letter and a slash,
// so a Windows home folder still counts as absolute when the check runs
// on another OS, as it does in tests.
func isWindowsAbs(goos, p string) bool {
	return goos == "windows" && len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}

// IsExecutable reports whether path is a regular file this account could
// run: on Windows any regular file, elsewhere one with an execute bit set.
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

// RealLocator is a Locator wired to this machine.
func RealLocator() Locator {
	return Locator{OS: runtime.GOOS, LookPath: exec.LookPath, Getenv: os.Getenv, IsExecutable: IsExecutable}
}

// FindCLI returns the absolute path to this machine's claude CLI, or an
// empty string when there is none.
func FindCLI() string {
	p, _ := RealLocator().Find()
	return p
}
