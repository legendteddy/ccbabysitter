package state

import (
	"errors"
	"os"
	"path/filepath"
)

// lingerFile is there when CC Babysitter itself turned lingering on for
// the user, and only then: setting up the service on Linux turns
// lingering on when it was off, so the service keeps running after the
// user logs out, and writes this file. Uninstall turns lingering off again
// only when the file is there, so lingering the user had turned on for
// other services of their own is left as it was. It is a file of its own
// in this program's folder, holding one line that says what it is for.
const lingerFile = "lingering-turned-on"

// lingerNote is what the file says, for anyone who finds it.
const lingerNote = "CC Babysitter turned lingering on for this user. Uninstall turns it off again."

// MarkLingeringTurnedOn remembers in dir that CC Babysitter turned
// lingering on.
func MarkLingeringTurnedOn(dir string) error {
	if err := EnsureDir(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, lingerFile)
	tmp := path + ".tmp"
	if err := writeSynced(tmp, []byte(lingerNote+"\n")); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LingeringTurnedOn reports whether dir remembers that CC Babysitter
// turned lingering on.
func LingeringTurnedOn(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, lingerFile))
	return err == nil && info.Mode().IsRegular()
}

// ForgetLingeringTurnedOn removes what MarkLingeringTurnedOn wrote. A file
// that is not there is not an error.
func ForgetLingeringTurnedOn(dir string) error {
	err := os.Remove(filepath.Join(dir, lingerFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
