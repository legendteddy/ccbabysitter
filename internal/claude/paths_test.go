package claude

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSlugAndTranscriptPath(t *testing.T) {
	if got := Slug(`C:\Users\dev.name\source\repos\app`); got != "C--Users-dev-name-source-repos-app" {
		t.Fatal(got)
	}
	if got := Slug("/home/dev/my ws"); got != "-home-dev-my-ws" {
		t.Fatal(got)
	}
	if got := Slug("/Users/jos\u00e9/ws"); got != "-Users-jos--ws" {
		t.Fatal(got)
	}
	want := filepath.Join("/proj", "-home-dev", "abc.jsonl")
	if got := transcriptPath("/proj", "/home/dev", "abc"); got != want {
		t.Fatal(got)
	}
}

func TestFindTranscriptSlugPath(t *testing.T) {
	tmp := t.TempDir()
	cwd := "/home/dev/ws"
	id := "11111111-2222-4333-8444-555555555501"
	dir := filepath.Join(tmp, Slug(cwd))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(want, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := FindTranscript(tmp, cwd, id)
	if !ok || got != want {
		t.Fatalf("got %q %v want %q", got, ok, want)
	}
}

func TestFindTranscriptGlobFallback(t *testing.T) {
	tmp := t.TempDir()
	id := "22222222-3333-4444-8888-555555555502"
	dir := filepath.Join(tmp, "some-other-project-slug")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(want, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := FindTranscript(tmp, "/home/dev/does-not-match", id)
	if !ok || got != want {
		t.Fatalf("got %q %v want %q", got, ok, want)
	}
}

func TestFindTranscriptMissing(t *testing.T) {
	tmp := t.TempDir()
	if _, ok := FindTranscript(tmp, "/home/dev/nope", "33333333-4444-4444-8444-555555555503"); ok {
		t.Fatal("expected no transcript found")
	}
}

// Only a finished search that found nothing says the transcript is not
// there: a missing projects folder, an empty one, folders without it, a
// stray file among them, and an id no transcript could have.
func TestLookupTranscriptNotFoundIsDefinite(t *testing.T) {
	id := "33333333-4444-4444-8444-555555555504"
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "-home-dev-other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct{ dir, id string }{
		"missing projects folder": {filepath.Join(tmp, "nowhere"), id},
		"empty projects folder":   {t.TempDir(), id},
		"folders without it":      {tmp, id},
		"not a session id":        {tmp, "*"},
	}
	for name, c := range cases {
		if _, err := LookupTranscript(c.dir, "/home/dev/ws", c.id); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: got %v, want not found", name, err)
		}
	}
}

// A folder that cannot be read leaves the search unfinished, which is not
// the same as the transcript not being there.
func TestLookupTranscriptUnreadableIsNotNotFound(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("folder permissions do not stop this user here")
	}
	id := "33333333-4444-4444-8444-555555555505"

	tmp := t.TempDir()
	locked := filepath.Join(tmp, "-home-dev-locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if _, err := LookupTranscript(tmp, "/home/dev/ws", id); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a project folder that cannot be read: got %v", err)
	}
	if _, ok := FindTranscript(tmp, "/home/dev/ws", id); ok {
		t.Fatal("nothing was found")
	}

	projects := filepath.Join(t.TempDir(), "projects")
	if err := os.MkdirAll(projects, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(projects, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(projects, 0o755) })
	if _, err := LookupTranscript(projects, "/home/dev/ws", id); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a projects folder that cannot be read: got %v", err)
	}
}

func TestFindTranscriptRejectsInvalidID(t *testing.T) {
	tmp := t.TempDir()
	cwd := "/home/dev/ws"
	// A transcript exists in the temp projects dir, reachable by a valid id,
	// so a bad id must never match it via a glob wildcard or path escape.
	dir := filepath.Join(tmp, Slug(cwd))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	realID := "11111111-2222-4333-8444-555555555501"
	if err := os.WriteFile(filepath.Join(dir, realID+".jsonl"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"*", "../../x", "a/b"} {
		if got, ok := FindTranscript(tmp, cwd, id); ok {
			t.Fatalf("FindTranscript(%q) = %q, true; want no match", id, got)
		}
	}
}

func TestValidID(t *testing.T) {
	valid := []string{
		"11111111-2222-4333-8444-555555555501",
		"AAAAAAAA-BBBB-4CCC-8DDD-EEEEEEEEEEEE",
	}
	for _, id := range valid {
		if !ValidID(id) {
			t.Errorf("ValidID(%q) = false, want true", id)
		}
	}
	invalid := []string{
		"",
		"11111111-2222-4333-8444-55555555550",
		"11111111-2222-4333-8444-5555555555011",
		"11111111222243338444555555555501",
		"../../etc/passwd",
		"11111111-2222-4333-8444-55555555550g",
		"11111111-2222-4333-8444-555555555501 ",
		" 11111111-2222-4333-8444-555555555501",
		"11111111-2222-4333-8444-555555555501/../x",
		"11111111-2222-4333-8444-555555555501.jsonl",
	}
	for _, id := range invalid {
		if ValidID(id) {
			t.Errorf("ValidID(%q) = true, want false", id)
		}
	}
}
