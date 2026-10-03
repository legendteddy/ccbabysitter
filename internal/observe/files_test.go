package observe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSessionFilesFiltersToJSONFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("11111111.json", `{"pid":1,"sessionId":"11111111-2222-4333-8444-555555555501","cwd":"/home/dev/ws","procStart":"1","kind":"interactive","entrypoint":"cli"}`)
	write("22222222.json", `{"pid":2,"sessionId":"22222222-2222-4333-8444-555555555502","cwd":"/home/dev/ws","procStart":"1","kind":"interactive","entrypoint":"cli"}`)
	write("11111111.abcd1234.key", "not-json-do-not-read-me")
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	read := ReadSessionFiles(dir)
	got := read()
	if len(got) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(got), got)
	}
	for _, b := range got {
		if string(b) == "not-json-do-not-read-me" {
			t.Fatal("key file must never be read")
		}
	}
}

func TestReadSessionFilesMissingDirYieldsNil(t *testing.T) {
	read := ReadSessionFiles(filepath.Join(t.TempDir(), "does-not-exist"))
	if got := read(); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// TestReadSessionFilesSkipsOversizedFile checks the read itself is bounded,
// not just a size check against a directory listing taken earlier: a file
// that has grown past the limit by the time it is actually read must still
// be skipped.
func TestReadSessionFilesSkipsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, maxSessionFileSize+1024)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(dir, "huge.json"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "normal.json"), []byte(`{"pid":1,"sessionId":"11111111-2222-4333-8444-555555555501","cwd":"/home/dev/ws","procStart":"1","kind":"interactive","entrypoint":"cli"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ReadSessionFiles(dir)()
	if len(got) != 1 {
		t.Fatalf("expected the oversized file to be skipped, got %d files", len(got))
	}
}
