//go:build darwin

package power

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type darwinKeepAwake struct{}

// New returns the keep-awake request for this platform.
func New() KeepAwake { return darwinKeepAwake{} }

func (darwinKeepAwake) Supported() bool {
	_, err := exec.LookPath("caffeinate")
	return err == nil
}

func (darwinKeepAwake) Acquire() (func(), bool) {
	// -i prevents idle system sleep only. -w ties caffeinate's own
	// lifetime to our pid, so a hard kill of this process ends the
	// request without any cleanup on our part.
	cmd := exec.Command("caffeinate", "-i", "-w", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return nil, false
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			cmd.Process.Kill()
			cmd.Wait()
		})
	}
	return release, true
}
