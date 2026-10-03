package hosts

import (
	"os/exec"
	"runtime"
)

// OpenURL hands a URL to the operating system's default handler for it,
// detached from this process and reaped once it finishes so it never
// leaves a zombie behind.
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return startDetached(exec.Command("rundll32", "url.dll,FileProtocolHandler", url))
	case "darwin":
		return startDetached(exec.Command("open", url))
	default:
		return startDetached(exec.Command("xdg-open", url))
	}
}

// startDetached puts cmd in its own process group so it survives this
// process exiting, starts it, and reaps it in the background once it
// finishes. Without that reap, every page this package opens would leave a
// zombie behind for as long as our own process keeps running, since
// nothing else ever calls Wait on a detached child.
func startDetached(cmd *exec.Cmd) error {
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}
