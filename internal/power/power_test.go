package power

import "testing"

func TestWanted(t *testing.T) {
	cases := []struct {
		mode   Mode
		active int
		want   bool
	}{
		{ModeAlways, 0, true}, {ModeAlways, 3, true},
		{ModeBabysitting, 0, false}, {ModeBabysitting, 1, true},
		{ModeOff, 5, false}, {Mode("garbage"), 5, false},
	}
	for _, c := range cases {
		if got := Wanted(c.mode, c.active); got != c.want {
			t.Errorf("Wanted(%q,%d)=%v want %v", c.mode, c.active, got, c.want)
		}
	}
}

func TestAcquireReleaseIsIdempotent(t *testing.T) {
	k := New()
	if !k.Supported() {
		t.Skip("keep-awake unsupported on this machine")
	}
	release, ok := k.Acquire()
	if !ok {
		t.Fatal("acquire failed")
	}
	release()
	release() // second release must be a no-op
}
