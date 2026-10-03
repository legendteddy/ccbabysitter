package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"ccbabysitter.dev/ccbabysitter/internal/procs"
	"ccbabysitter.dev/ccbabysitter/internal/state"
)

// resetFlow deletes CC Babysitter's own state folder after asking for
// confirmation. It refuses outright while the folder's lock is held: an
// instance that is still running owns that folder, and deleting it out
// from under one would corrupt whatever it writes next. isHeld and the
// reader and writer are parameters so this can be tested without a real
// running process or a real terminal.
func resetFlow(dir string, in io.Reader, out io.Writer, isHeld func(string) bool) int {
	if isHeld(dir) {
		fmt.Fprintln(out, "CC Babysitter is still running against this folder. Stop it before running reset.")
		return 1
	}

	store := &state.Store{Dir: dir}
	st, ok := store.Peek()

	fmt.Fprintf(out, "This deletes the state folder at %s.\n", dir)
	if ok && len(st.Watches) > 0 {
		fmt.Fprintln(out, "It is currently watching:")
		for _, w := range st.Watches {
			fmt.Fprintf(out, "  %s, %s\n", w.Name, w.ShortID)
		}
	}
	fmt.Fprint(out, "Continue? [y/N] ")

	line, _ := bufio.NewReader(in).ReadString('\n')
	line = strings.TrimSpace(line)
	if !strings.EqualFold(line, "y") {
		fmt.Fprintln(out, "Nothing changed.")
		return 0
	}

	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(out, "could not delete the folder:", err)
		return 1
	}
	fmt.Fprintln(out, "The state folder is gone. If you are done with CC Babysitter, you can also delete its binary.")
	return 0
}

// runReset is the reset subcommand: it wires resetFlow to the real state
// folder, the real terminal, and a lock check backed by real process
// identity.
func runReset() int {
	state.SetPIDChecker(procs.NewReal().Exists)
	return resetFlow(state.DefaultDir(), os.Stdin, os.Stdout, state.IsHeld)
}
