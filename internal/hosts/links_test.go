package hosts

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestStartDetachedReapsChild checks that a process started through
// startDetached does not linger as a zombie: something must call Wait on
// it even though the caller never does, since a long running program that
// opens many pages over its lifetime would otherwise accumulate one zombie
// per launch.
func TestStartDetachedReapsChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("zombie states are a unix concept; ps -o stat is used to observe them")
	}

	cmd := exec.Command("true")
	if err := startDetached(cmd); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	deadline := time.Now().Add(2 * time.Second)
	for {
		out, _ := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
		if strings.TrimSpace(string(out)) == "" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d still present after reap window: stat=%q", pid, out)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
