package observe

import (
	"fmt"
	"testing"
	"time"
)

func file(pid int, sid, kind, entry string) []byte {
	return []byte(fmt.Sprintf(`{"pid":%d,"sessionId":"%s","cwd":"/home/dev/ws","procStart":"7","kind":"%s","entrypoint":"%s"}`, pid, sid, kind, entry))
}

func TestBuildDropsDeadAndMalformedKeepsDuplicates(t *testing.T) {
	files := [][]byte{
		file(1, "11111111-2222-4333-8444-555555555501", "interactive", "cli"),
		file(2, "22222222-2222-4333-8444-555555555502", "interactive", "cli"),
		[]byte("{bad"),
		file(3, "33333333-2222-4333-8444-555555555503", "bg", "cli"),
		file(4, "33333333-2222-4333-8444-555555555503", "interactive", "claude-desktop"),
	}
	snap := Build(files, func(pid int, _ string) bool { return pid != 2 }, time.Now())
	if len(snap.Sessions) != 3 {
		t.Fatalf("%d", len(snap.Sessions))
	}
	if _, ok := snap.Find("22222222-2222-4333-8444-555555555502"); ok {
		t.Fatal("dead dropped")
	}
	if len(snap.All("33333333-2222-4333-8444-555555555503")) != 2 {
		t.Fatal("duplicates kept so a session open in two apps shows as such")
	}
}
