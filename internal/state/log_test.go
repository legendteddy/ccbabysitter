package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogWritesActivityAndRolls(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	l.maxBytes = 400
	ch, cancel := l.Subscribe()
	defer cancel()
	l.Auto("demo-a1", "host process exited", "resumed as background 11111111")
	e := <-ch
	if !e.Automatic || e.Reason == "" || e.Session != "demo-a1" {
		t.Fatalf("%+v", e)
	}
	for i := 0; i < 30; i++ {
		l.Info("s", strings.Repeat("x", 30))
	}
	fi, _ := os.Stat(filepath.Join(dir, "log.txt"))
	if fi.Size() > 800 {
		t.Fatalf("log did not roll: %d", fi.Size())
	}
	rec := l.Recent(5, "")
	if len(rec) != 5 || !strings.Contains(rec[0].Message, "x") {
		t.Fatalf("%+v", rec)
	}
	if got := l.Recent(50, "demo-a1"); len(got) != 1 || !got[0].Automatic {
		t.Fatalf("filter by session: %+v", got)
	}
	l.Error("s", "boom")
	if last := l.Recent(1, "")[0]; last.Level != "ERROR" {
		t.Fatalf("%+v", last)
	}
}
