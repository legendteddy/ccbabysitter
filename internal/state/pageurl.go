package state

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// pageURLFile holds the address of the page the running copy of this
// program serves, one line of the form http://127.0.0.1:<port> with
// nothing else in it. The serving copy writes it once it is listening and
// removes it when it stops cleanly, so another run of this program, such
// as the one that starts the systemd service, can learn which port the
// service ended up on. The single-instance lock means at most one copy
// serves from this folder at a time, so the file names that one.
const pageURLFile = "page-url"

// ValidPageURL reports whether raw is exactly a loopback page address this
// program builds: http, the host 127.0.0.1, a port from 1 to 65535, and no
// path, query or fragment.
func ValidPageURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" {
		return false
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	return err == nil && port > 0 && port <= 65535
}

// SavePageURL records pageURL in dir as the address of the page being
// served now. Anything other than a loopback page address is refused.
func SavePageURL(dir, pageURL string) error {
	if !ValidPageURL(pageURL) {
		return fmt.Errorf("not a loopback page address: %q", pageURL)
	}
	if err := EnsureDir(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, pageURLFile)
	tmp := path + ".tmp"
	if err := writeSynced(tmp, []byte(pageURL+"\n")); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// PageURL returns the page address SavePageURL last recorded in dir, or an
// empty string when there is none or the file does not hold one.
func PageURL(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, pageURLFile))
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(data))
	if !ValidPageURL(raw) {
		return ""
	}
	return raw
}

// ClearPageURL removes the page address file from dir whatever it names.
// Only the copy that has just taken the state folder's lock calls it: any
// address there was left by a copy that is gone. A file that is not there
// is already the outcome wanted.
func ClearPageURL(dir string) error {
	err := os.Remove(filepath.Join(dir, pageURLFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RemovePageURL removes the page address file from dir, but only while it
// still names pageURL, so a copy that is stopping never removes the
// address a newer copy has already written. A file that is not there is
// already the outcome wanted.
func RemovePageURL(dir, pageURL string) error {
	if PageURL(dir) != pageURL {
		return nil
	}
	err := os.Remove(filepath.Join(dir, pageURLFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
