//go:build linux

package power

import (
	"os/exec"
	"sync"
)

type linuxKeepAwake struct{}

// New returns the keep-awake request for this platform.
func New() KeepAwake { return linuxKeepAwake{} }

func (linuxKeepAwake) Supported() bool {
	_, err := exec.LookPath("systemd-inhibit")
	return err == nil
}

func (linuxKeepAwake) Acquire() (func(), bool) {
	// --what=idle blocks idle system sleep only, never display sleep.
	// The child's own command is "cat" reading our stdin pipe: closing
	// that pipe on release makes cat exit, which ends the inhibitor even
	// if we are hard-killed before release runs.
	cmd := exec.Command("systemd-inhibit",
		"--what=idle",
		"--who=CC Babysitter",
		"--why=keep Claude Code sessions reachable",
		"--mode=block",
		"cat",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, false
	}
	if err := cmd.Start(); err != nil {
		return nil, false
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			stdin.Close()
			cmd.Process.Kill()
			cmd.Wait()
		})
	}
	return release, true
}
