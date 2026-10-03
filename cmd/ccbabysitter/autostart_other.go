//go:build !darwin && !windows && !linux

package main

// This platform has no autostart mechanism CC Babysitter can install, so
// autostartInstaller is left nil and the page reports autostart as
// unsupported.
