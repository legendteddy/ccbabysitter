//go:build windows

package hosts

import (
	"os/exec"
	"syscall"
)

// createNewProcessGroup is the Windows CREATE_NEW_PROCESS_GROUP creation
// flag, which lets cmd survive this process exiting instead of receiving
// the same console events.
const createNewProcessGroup = 0x00000200

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}
