package observe

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxSessionFileSize bounds how large a session file is allowed to be
// before it is skipped. A well formed session file is a few hundred bytes;
// anything past this is treated as unreadable rather than risking a large
// read on every sweep.
const maxSessionFileSize = 1 << 20

// ReadSessionFiles returns a reader that lists dir and returns the contents
// of every session file in it. Only regular files whose name ends in
// ".json" are read; the key files Claude Code writes alongside them never
// are. A file that cannot be read, or that turns out to hold more than the
// size limit once actually read, is skipped rather than failing the whole
// read. A missing directory yields no files, since the sessions folder
// does not exist until Claude Code has run at least once.
func ReadSessionFiles(dir string) func() [][]byte {
	return func() [][]byte {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		var out [][]byte
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			if data, ok := readBoundedFile(filepath.Join(dir, e.Name())); ok {
				out = append(out, data)
			}
		}
		return out
	}
}

// readBoundedFile reads path's contents through a reader capped one byte
// past the size limit, so a file that keeps growing while it is being read
// is caught and skipped rather than read without bound. Checking the size
// a directory listing reported earlier would not catch this, since the
// file can grow between that listing and this read.
func readBoundedFile(path string) ([]byte, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}

	data, err := io.ReadAll(io.LimitReader(f, maxSessionFileSize+1))
	if err != nil || len(data) > maxSessionFileSize {
		return nil, false
	}
	return data, true
}
