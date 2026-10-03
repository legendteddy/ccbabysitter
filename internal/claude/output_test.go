package claude

import "testing"

func TestParseBackgrounded(t *testing.T) {
	real := "Starting background service\u2026\nbackgrounded \u00b7 77778888 \u00b7 demo-a1 (idle \u2014 send a prompt to start)\n  claude agents             list sessions\n  claude attach 77778888    open in this terminal"
	if s, ok := ParseBackgrounded(real); !ok || s != "77778888" {
		t.Fatalf("%q %v", s, ok)
	}
	if s, ok := ParseBackgrounded("backgrounded \u00b7 33333333 (idle)"); !ok || s != "33333333" {
		t.Fatalf("%q %v", s, ok)
	}
	if _, ok := ParseBackgrounded("error: x"); ok {
		t.Fatal("no match expected")
	}
}

func TestParseCopyAndWoke(t *testing.T) {
	copyNote := "note: background session 12121212 keeps its own saved options, so the flags you passed started a copy as 33334444. Without flags, the same command continues 55556666 itself.\nbackgrounded \u00b7 33334444 (idle)"
	if s, ok := ParseCopy(copyNote); !ok || s != "33334444" {
		t.Fatalf("%q %v", s, ok)
	}
	altCopy := "note: session 77778888 is already running in the background, so this started a copy as 99990000. `claude attach 77778888` opens the original."
	if s, ok := ParseCopy(altCopy); !ok || s != "99990000" {
		t.Fatalf("%q %v", s, ok)
	}
	woke := "note: woke session 12121212 with its saved options (--name, --remote-control, --model).\nbackgrounded \u00b7 12121212"
	if _, ok := ParseCopy(woke); ok {
		t.Fatal("woke is not a copy")
	}
	if !IsWoke(woke) || IsWoke("backgrounded \u00b7 x") {
		t.Fatal("IsWoke")
	}
}
