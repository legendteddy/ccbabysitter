package claude

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Dir returns the user's Claude configuration directory, ~/.claude.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// SessionsDir returns the directory holding per-process session files.
func SessionsDir() string { return filepath.Join(Dir(), "sessions") }

// ProjectsDir returns the directory holding per-project transcript folders.
func ProjectsDir() string { return filepath.Join(Dir(), "projects") }

// Slug reproduces Claude's project folder name for a working directory:
// every ASCII letter or digit is kept, and everything else becomes a dash.
func Slug(cwd string) string {
	var b strings.Builder
	for _, r := range cwd {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// transcriptPath returns where Claude would keep the transcript for a
// session with the given id, started in cwd, under projectsDir. It is not
// exported: FindTranscript is the way in, because it checks the id before
// the value ever reaches a file path.
func transcriptPath(projectsDir, cwd, id string) string {
	return filepath.Join(projectsDir, Slug(cwd), id+".jsonl")
}

// FindTranscript locates a session's transcript file. It checks the slug
// path first, then falls back to scanning every project folder for a
// transcript with a matching id, since the cwd used to start a session can
// differ from the one used to look it up later. It rejects anything that is
// not a canonical id before touching the filesystem, since id ends up in a
// file path and an unchecked "*" or ".." would reach outside the single
// transcript it is meant to find. A search that could not be finished
// counts as not found here; LookupTranscript tells the two apart.
func FindTranscript(projectsDir, cwd, id string) (string, bool) {
	path, err := LookupTranscript(projectsDir, cwd, id)
	return path, err == nil
}

// LookupTranscript is FindTranscript that says why nothing was found. The
// error wraps fs.ErrNotExist only when the search was finished and the
// transcript is definitely not there, or when id is not one a transcript
// could have. Any other error means the search could not be finished, such
// as a folder that could not be read, and says nothing either way.
func LookupTranscript(projectsDir, cwd, id string) (string, error) {
	if !ValidID(id) {
		return "", fmt.Errorf("not a session id: %w", fs.ErrNotExist)
	}
	slugPath := transcriptPath(projectsDir, cwd, id)
	if _, err := os.Stat(slugPath); err == nil {
		return slugPath, nil
	} else if !notThere(err) {
		return "", err
	}
	// A projects folder that is not there holds nothing, and its error
	// already says so; any other failure to read it leaves the search
	// unfinished.
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", err
	}
	var unsure error
	for _, e := range entries {
		// Only a folder, or a link that may lead to one, can hold a
		// transcript.
		if !e.IsDir() && e.Type()&fs.ModeSymlink == 0 {
			continue
		}
		path := filepath.Join(projectsDir, e.Name(), id+".jsonl")
		_, err := os.Stat(path)
		switch {
		case err == nil:
			return path, nil
		case !notThere(err) && unsure == nil:
			unsure = err
		}
	}
	if unsure != nil {
		return "", unsure
	}
	return "", fmt.Errorf("no transcript for %s: %w", id, fs.ErrNotExist)
}

// notThere reports whether err from looking at a path means the file is
// definitely not there: it does not exist, or something on the way to it
// is a file rather than a folder.
func notThere(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// ValidID reports whether id is a canonical 8-4-4-4-12 hex UUID, case
// insensitive. A raw CLI argument used to build a file path must never be
// trusted without this check.
func ValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !isHexDigit(r) {
				return false
			}
		}
	}
	return true
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// ConfigFile returns the Claude Code CLI's own settings file, ~/.claude.json.
// This program only ever reads it.
func ConfigFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude.json")
}
