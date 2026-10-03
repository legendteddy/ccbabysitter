//go:build linux

package main

import "path/filepath"

func init() {
	autostartInstaller = installAutostartLinux
	autostartInstalled = autostartInstalledLinux
}

// installAutostartLinux enables or disables the same systemd user unit the
// install subcommand manages, without ever touching lingering: enabling it
// here only starts CC Babysitter along with the rest of the user's own
// session, and only someone running install themselves is asking for it to
// keep running after they log out too. Disabling follows the same order
// uninstall does: ask systemd to stop managing the unit before the file
// backing it is removed, then reload so systemd forgets it entirely.
func installAutostartLinux(enable bool) (string, error) {
	if !enable {
		_ = runSystemctl("disable", "ccbabysitter")
		path, err := removeUnit()
		if err != nil {
			return "", err
		}
		_ = runSystemctl("daemon-reload")
		return path, nil
	}
	bin, err := resolvedExecutablePath()
	if err != nil {
		return "", err
	}
	path, err := writeUnit(bin)
	if err != nil {
		return "", err
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		return "", err
	}
	if err := runSystemctl("enable", "ccbabysitter"); err != nil {
		return "", err
	}
	return path, nil
}

// autostartInstalledLinux reports whether the user unit is enabled, judged
// by the link systemctl enable makes for it in the default target's wants
// folder, next to the unit itself. It never runs systemctl, so a look is
// cheap and has no effect.
func autostartInstalledLinux() (bool, error) {
	unit, err := systemdUnitPath()
	if err != nil {
		return false, err
	}
	return pathPresent(filepath.Join(filepath.Dir(unit), "default.target.wants", filepath.Base(unit)))
}
