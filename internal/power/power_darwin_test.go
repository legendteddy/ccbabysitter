//go:build darwin

package power

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func caffeinatePresent(pid int) bool {
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("caffeinate -i -w %d", pid))
	return cmd.Run() == nil
}

func TestAcquireStartsCaffeinateForOurPID(t *testing.T) {
	if _, err := exec.LookPath("caffeinate"); err != nil {
		t.Skip("caffeinate not installed")
	}
	k := New()
	if !k.Supported() {
		t.Skip("keep-awake unsupported on this machine")
	}
	release, ok := k.Acquire()
	if !ok {
		t.Fatal("acquire failed")
	}
	pid := os.Getpid()
	if !caffeinatePresent(pid) {
		t.Fatal("expected a caffeinate process holding our pid")
	}

	release()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !caffeinatePresent(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("caffeinate process still present 2s after release")
}
