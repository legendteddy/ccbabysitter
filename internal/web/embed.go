package web

import "embed"

// uiFS holds the page exactly as it ships: one HTML document, its
// stylesheets, its scripts and its icon, compiled into the binary so the
// program has no files to find at runtime and no way to serve anything
// else.
//
//go:embed ui
var uiFS embed.FS
