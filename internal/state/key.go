package state

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// keyFile holds the page's access key: 32 random bytes as 64 hex
// characters. Every request to the page and its API must carry it, so
// another account on the same machine, which can reach 127.0.0.1 too,
// cannot read or drive this one's sessions. It is kept across restarts so
// an open or bookmarked page keeps working, and goes with the rest of the
// folder on reset.
const keyFile = "page-key"

// ValidPageKey reports whether k looks like a key PageKey writes: exactly
// 64 hex characters.
func ValidPageKey(k string) bool {
	if len(k) != 64 {
		return false
	}
	_, err := hex.DecodeString(k)
	return err == nil
}

// ReadPageKey returns the key saved in dir, or "" when there is none or it
// is not a key this program wrote.
func ReadPageKey(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, keyFile))
	if err != nil {
		return ""
	}
	k := strings.TrimSpace(string(b))
	if !ValidPageKey(k) {
		return ""
	}
	return k
}

// PageKey returns the key saved in dir, writing a new one first when there
// is none or the file does not hold a valid one, with writePrivate.
//
// An existing key file is tightened to owner-only, so one an earlier build
// or a person left readable by other accounts does not stay that way. On
// Windows, Chmod only sets or clears the read-only attribute, and the
// account's private %LOCALAPPDATA% is what keeps the file to itself; a
// failure to tighten only leaves the file as it was, so it is ignored.
func PageKey(dir string) (string, error) {
	if k := ReadPageKey(dir); k != "" {
		_ = os.Chmod(filepath.Join(dir, keyFile), fileMode)
		return k, nil
	}
	if err := EnsureDir(dir); err != nil {
		return "", err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	k := hex.EncodeToString(b)
	if err := writePrivate(filepath.Join(dir, keyFile), []byte(k+"\n")); err != nil {
		return "", err
	}
	return k, nil
}

// writePrivate writes data to path through a temporary file with
// owner-only permissions that is renamed into place, so the file is never
// readable by another account, not even for a moment, and a reader never
// sees half of it.
func writePrivate(path string, data []byte) error {
	tmp := path + ".tmp"
	// A leftover temporary file keeps its own permissions when it is opened
	// again, so it is removed first and created anew with fileMode.
	_ = os.Remove(tmp)
	if err := writeSynced(tmp, data); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// KeyedURL is the page address a person opens: the plain address with the
// key, which the page swaps for a cookie on the first visit. With no key
// it is the plain address, since a "?token=" with nothing after it would
// only be refused.
func KeyedURL(pageURL, key string) string {
	if key == "" {
		return pageURL
	}
	return pageURL + "/?token=" + key
}
