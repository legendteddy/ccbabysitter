package state

import (
	"os"
	"path/filepath"
	"strings"
)

// addressFile holds the address this machine was last reached at over
// ssh, as the server's own side of SSH_CONNECTION reported it. It is a
// file of its own in this program's folder, one line with nothing else in
// it, rather than a field in state.json: it is written from wherever the
// address is seen, including before the engine that owns state.json has
// started, and the page never needs it. A run with no ssh connection of
// its own, such as the systemd service, reads it back to name the address
// in the commands it prints.
const addressFile = "ssh-address"

// ValidServerAddress reports whether addr can stand as one word in a
// command line: not empty, and made only of letters, digits and the
// characters a host name or an IP address uses.
func ValidServerAddress(addr string) bool {
	if addr == "" || len(addr) > 255 {
		return false
	}
	for _, r := range addr {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune(".:-_%", r):
		default:
			return false
		}
	}
	return true
}

// SaveServerAddress remembers addr in dir, leaving the file alone when it
// already says the same thing. An address that could not stand as one
// word in a command line is not saved.
func SaveServerAddress(dir, addr string) error {
	if !ValidServerAddress(addr) || ServerAddress(dir) == addr {
		return nil
	}
	if err := EnsureDir(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, addressFile)
	tmp := path + ".tmp"
	if err := writeSynced(tmp, []byte(addr+"\n")); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ServerAddress returns the address SaveServerAddress last remembered in
// dir, or an empty string when there is none or the file does not hold
// one.
func ServerAddress(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, addressFile))
	if err != nil {
		return ""
	}
	addr := strings.TrimSpace(string(data))
	if !ValidServerAddress(addr) {
		return ""
	}
	return addr
}
