//go:build windows

package power

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	powerCreateRequest = kernel32.NewProc("PowerCreateRequest")
	powerSetRequest    = kernel32.NewProc("PowerSetRequest")
	powerClearRequest  = kernel32.NewProc("PowerClearRequest")
	closeHandle        = kernel32.NewProc("CloseHandle")
)

// powerRequestSystemRequired is PowerRequestSystemRequired. It must never be
// PowerRequestDisplayRequired (0): only idle system sleep is prevented, not
// display sleep.
const powerRequestSystemRequired = 1

// reasonContext mirrors the Windows REASON_CONTEXT structure for the simple
// string reason form.
type reasonContext struct {
	Version uint32
	Flags   uint32
	Reason  *uint16
}

// powerRequestContextSimpleString selects the simple string reason form in
// reasonContext.Flags.
const powerRequestContextSimpleString = 1

type winKeepAwake struct{}

// New returns the keep-awake request for this platform.
func New() KeepAwake { return winKeepAwake{} }

func (winKeepAwake) Supported() bool { return powerCreateRequest.Find() == nil }

func (winKeepAwake) Acquire() (func(), bool) {
	reason, err := syscall.UTF16PtrFromString("CC Babysitter: keeping Claude Code sessions reachable")
	if err != nil {
		return nil, false
	}
	ctx := reasonContext{Version: 0, Flags: powerRequestContextSimpleString, Reason: reason}
	h, _, _ := powerCreateRequest.Call(uintptr(unsafe.Pointer(&ctx)))
	// The pointer handed to Windows is not one Go's garbage collector can
	// see, and neither is the reason string the structure points at. Both
	// are kept alive by hand until the call that reads them has returned,
	// so neither can be collected or moved while Windows is still using it.
	runtime.KeepAlive(&ctx)
	runtime.KeepAlive(reason)
	if h == 0 || h == ^uintptr(0) {
		return nil, false
	}
	if r, _, _ := powerSetRequest.Call(h, powerRequestSystemRequired); r == 0 {
		closeHandle.Call(h)
		return nil, false
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			powerClearRequest.Call(h, powerRequestSystemRequired)
			closeHandle.Call(h)
		})
	}, true
}
