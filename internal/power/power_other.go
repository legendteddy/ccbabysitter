//go:build !windows && !darwin && !linux

package power

type otherKeepAwake struct{}

// New returns the keep-awake request for this platform.
func New() KeepAwake { return otherKeepAwake{} }

func (otherKeepAwake) Supported() bool { return false }

func (otherKeepAwake) Acquire() (func(), bool) { return nil, false }
