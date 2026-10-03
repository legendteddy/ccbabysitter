package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic title records, shaped as the CLI writes them.
func aiTitleLine(v string) string {
	return `{"type":"ai-title","aiTitle":"` + v + `","sessionId":"s"}` + "\n"
}

func customTitleLine(v string) string {
	return `{"type":"custom-title","customTitle":"` + v + `","sessionId":"s"}` + "\n"
}

func agentNameLine(v string) string {
	return `{"type":"agent-name","agentName":"` + v + `","sessionId":"s"}` + "\n"
}

const promptLine = `{"type":"user","timestamp":"2026-09-26T10:00:00Z","message":{"content":"go"}}` + "\n"

// titleOf reads a whole transcript once and returns the title it found.
func titleOf(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "11111111-2222-4333-8444-555555555501.jsonl")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := (&StatsReader{}).Update(p)
	if err != nil {
		t.Fatal(err)
	}
	return s.Title
}

func TestTitleIsTheLatestAITitle(t *testing.T) {
	got := titleOf(t, aiTitleLine("First idea")+promptLine+aiTitleLine("Hooora test")+promptLine)
	if got != "Hooora test" {
		t.Fatalf("%q", got)
	}
}

func TestTitleIsEmptyWithoutAnyTitleRecord(t *testing.T) {
	if got := titleOf(t, promptLine); got != "" {
		t.Fatalf("%q", got)
	}
}

// The name Claude gives by itself when Remote Control connects is a
// custom-title followed at once by an agent-name with the same value. It
// is not the person's, so the AI title still shows.
func TestTheAutomaticNameIsSkipped(t *testing.T) {
	body := aiTitleLine("Hooora test") + promptLine +
		customTitleLine("my-app-2-94") + agentNameLine("my-app-2-94") + promptLine
	if got := titleOf(t, body); got != "Hooora test" {
		t.Fatalf("%q", got)
	}
}

// A custom-title followed by an agent-name with another value is a name
// the person gave.
func TestACustomTitleFollowedByAnotherAgentNameCounts(t *testing.T) {
	body := aiTitleLine("Hooora test") + customTitleLine("Checkout fixes") + agentNameLine("my-app-2-94") + promptLine
	if got := titleOf(t, body); got != "Checkout fixes" {
		t.Fatalf("%q", got)
	}
}

// A rename after the automatic name wins, and a later AI title does not
// take its place.
func TestARenameAfterTheAutomaticNameWins(t *testing.T) {
	body := aiTitleLine("Hooora test") + customTitleLine("my-app-2-94") + agentNameLine("my-app-2-94") +
		promptLine + customTitleLine("Checkout fixes") + promptLine + aiTitleLine("Later idea") + promptLine
	if got := titleOf(t, body); got != "Checkout fixes" {
		t.Fatalf("%q", got)
	}
}

// A second automatic name after a rename leaves the rename in place.
func TestAnAutomaticNameAfterARenameLeavesTheRename(t *testing.T) {
	body := customTitleLine("Checkout fixes") + promptLine +
		customTitleLine("my-app-2-95") + agentNameLine("my-app-2-95") + promptLine
	if got := titleOf(t, body); got != "Checkout fixes" {
		t.Fatalf("%q", got)
	}
}

// The sidecar title file VS Code keeps beside the transcript wins over
// everything in the transcript.
func TestTheSidecarTitleWins(t *testing.T) {
	dir := t.TempDir()
	const id = "11111111-2222-4333-8444-555555555502"
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte(aiTitleLine("Hooora test")+customTitleLine("Checkout fixes")+promptLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	side := filepath.Join(dir, id, "custom-title.json")
	if err := os.WriteFile(side, []byte(`{"customTitle":"  Panel   name  "}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &StatsReader{}
	s, err := r.Update(p)
	if err != nil || s.Title != "Panel name" {
		t.Fatalf("%q %v", s.Title, err)
	}

	// An empty one counts as none.
	if err := os.WriteFile(side, []byte(`{"customTitle":"   "}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if s, _ = r.Update(p); s.Title != "Checkout fixes" {
		t.Fatalf("an empty sidecar is none: %q", s.Title)
	}

	// One far larger than any title is not read at all.
	big := `{"customTitle":"` + strings.Repeat("x", 8<<10) + `"}`
	if err := os.WriteFile(side, []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	if s, _ = r.Update(p); s.Title != "Checkout fixes" {
		t.Fatalf("an oversized sidecar is ignored: %q", s.Title)
	}

	// A change to it is picked up without anything new in the transcript.
	if err := os.WriteFile(side, []byte(`{"customTitle":"Renamed in the panel"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if s, _ = r.Update(p); s.Title != "Renamed in the panel" {
		t.Fatalf("%q", s.Title)
	}
}

// Reading only what was appended still finds a later rename. A rename
// that is the very last line is taken on the read after, so an automatic
// name whose agent-name line has not landed yet never shows.
func TestIncrementalReadingPicksUpALaterRename(t *testing.T) {
	p := filepath.Join(t.TempDir(), "11111111-2222-4333-8444-555555555503.jsonl")
	if err := os.WriteFile(p, []byte(aiTitleLine("Hooora test")+promptLine), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &StatsReader{}
	if s, _ := r.Update(p); s.Title != "Hooora test" {
		t.Fatalf("%q", s.Title)
	}

	appendLine(t, p, customTitleLine("my-app-2-94"))
	if s, _ := r.Update(p); s.Title != "Hooora test" {
		t.Fatalf("a custom-title with nothing after it yet is not shown: %q", s.Title)
	}
	appendLine(t, p, agentNameLine("my-app-2-94"))
	if s, _ := r.Update(p); s.Title != "Hooora test" {
		t.Fatalf("the automatic name must never show: %q", s.Title)
	}
	if s, _ := r.Update(p); s.Title != "Hooora test" {
		t.Fatalf("%q", s.Title)
	}

	appendLine(t, p, customTitleLine("Checkout fixes"))
	if s, _ := r.Update(p); s.Title != "Hooora test" {
		t.Fatalf("%q", s.Title)
	}
	if s, _ := r.Update(p); s.Title != "Checkout fixes" {
		t.Fatalf("a rename left alone for a read is taken: %q", s.Title)
	}

	appendLine(t, p, customTitleLine("Checkout fixes two")+promptLine)
	if s, _ := r.Update(p); s.Title != "Checkout fixes two" {
		t.Fatalf("a rename with a line after it is taken at once: %q", s.Title)
	}

	// A custom-title left pending by one read and followed in the next by
	// an automatic name that ends that read is not taken early.
	appendLine(t, p, customTitleLine("Third name"))
	if s, _ := r.Update(p); s.Title != "Checkout fixes two" {
		t.Fatalf("%q", s.Title)
	}
	appendLine(t, p, promptLine+customTitleLine("my-app-2-96"))
	if s, _ := r.Update(p); s.Title != "Third name" {
		t.Fatalf("%q", s.Title)
	}
	appendLine(t, p, agentNameLine("my-app-2-96"))
	if s, _ := r.Update(p); s.Title != "Third name" {
		t.Fatalf("an automatic name after a pending one must never show: %q", s.Title)
	}
}

// A title is trimmed, has its whitespace collapsed and anything that does
// not print dropped, and is cut to 120 characters.
func TestTitlesAreCleaned(t *testing.T) {
	cases := map[string]string{
		`  Fix\tthe\n  checkout  `: "Fix the checkout",
		`Bell\u0007 ring`:          "Bell ring",
		`\u200b`:                   "",
	}
	for raw, want := range cases {
		if got := titleOf(t, aiTitleLine(raw)+promptLine); got != want {
			t.Errorf("%s: got %q, want %q", raw, got, want)
		}
	}
	long := strings.Repeat("ab ", 100)
	got := titleOf(t, aiTitleLine(long)+promptLine)
	if n := len([]rune(got)); n > 120 || n < 110 || strings.HasSuffix(got, " ") {
		t.Fatalf("capped at 120 without a trailing space: %d %q", n, got)
	}
	if got := CleanTitle(strings.Repeat(string(rune(0xe9)), 130)); len([]rune(got)) != 120 {
		t.Fatalf("the cap counts characters: %d", len([]rune(got)))
	}
}

// Only a regular file is read as the sidecar title. Anything else there,
// a folder or a pipe that would never answer, is left alone.
func TestASidecarThatIsNotAFileIsIgnored(t *testing.T) {
	dir := t.TempDir()
	const id = "11111111-2222-4333-8444-555555555504"
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte(aiTitleLine("Hooora test")+promptLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, id, "custom-title.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := sidecarTitle(p); got != "" {
		t.Fatalf("%q", got)
	}
	if s, err := (&StatsReader{}).Update(p); err != nil || s.Title != "Hooora test" {
		t.Fatalf("%q %v", s.Title, err)
	}
}
