package observe

import (
	"errors"
	"testing"
)

// fakeFSWatcher is a minimal fsWatcher for exercising watchSwitch
// without a real file system watcher. addErrs is consumed in order: each
// call to Add returns the next queued error (nil once the queue is empty).
type fakeFSWatcher struct {
	addErrs     []error
	addCalls    []string
	removeCalls []string
}

func (f *fakeFSWatcher) Add(name string) error {
	f.addCalls = append(f.addCalls, name)
	if len(f.addErrs) == 0 {
		return nil
	}
	err := f.addErrs[0]
	f.addErrs = f.addErrs[1:]
	return err
}

func (f *fakeFSWatcher) Remove(name string) error {
	f.removeCalls = append(f.removeCalls, name)
	return nil
}

func TestWatchSwitchKeepsParentOnFailureThenSwitches(t *testing.T) {
	w := &fakeFSWatcher{addErrs: []error{errors.New("not ready yet"), nil}}
	h := &watchSwitch{parent: "/home/dev/ws/sessions-parent", dir: "/home/dev/ws/sessions"}
	var logged []string
	logErr := func(msg string) { logged = append(logged, msg) }

	if h.try(w, logErr) {
		t.Fatal("expected the first attempt to fail")
	}
	if len(w.removeCalls) != 0 {
		t.Fatalf("parent watch must stay in place after a failed switch, got removes %v", w.removeCalls)
	}
	if len(logged) != 1 {
		t.Fatalf("expected exactly one logged error after the failure, got %d: %v", len(logged), logged)
	}

	if !h.try(w, logErr) {
		t.Fatal("expected the second attempt to switch over")
	}
	if len(w.removeCalls) != 1 || w.removeCalls[0] != h.parent {
		t.Fatalf("expected the parent watch to be removed once, got %v", w.removeCalls)
	}
	if len(logged) != 1 {
		t.Fatalf("expected still exactly one logged error after the success, got %d: %v", len(logged), logged)
	}
}
