package observe

// fsWatcher is the subset of *fsnotify.Watcher needed to move a watch from
// a directory's parent to the directory itself once it exists. Keeping the
// surface this small lets the switch-over be unit tested without a real
// file system watcher.
type fsWatcher interface {
	Add(name string) error
	Remove(name string) error
}

// watchSwitch moves a watch from a not-yet-existing directory's parent
// onto the directory itself once it exists. It always tries the child
// watch first and only gives up the parent watch once the child watch is
// confirmed to work, so a failed attempt never leaves nothing watched. A
// failure is logged once rather than on every retry; a later success
// clears that memory.
type watchSwitch struct {
	parent, dir string
	logged      bool
}

// try attempts to move the watch onto dir. It reports whether the watch
// now covers dir. On failure the parent watch is left untouched, so the
// caller can simply retry on the next event it sees on the parent.
func (h *watchSwitch) try(w fsWatcher, logErr func(string)) bool {
	if err := w.Add(h.dir); err != nil {
		if !h.logged {
			logErr("file watcher: could not start watching " + h.dir + " after it appeared, still watching its parent: " + err.Error())
			h.logged = true
		}
		return false
	}
	_ = w.Remove(h.parent)
	h.logged = false
	return true
}
