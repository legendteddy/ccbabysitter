package web

import "net/http"

// registerStatic wires the routes that serve the page out of
// the embedded filesystem, each with the content type it is, and nothing
// else: any other path falls through the mux and is a 404. The version is
// no longer needed here, since the page reads it from the view like every
// other fact it shows.
func registerStatic(mux *http.ServeMux, _ string) {
	mux.HandleFunc("GET /{$}", serveEmbedded("ui/index.html", "text/html; charset=utf-8"))
	mux.HandleFunc("GET /app.css", serveEmbedded("ui/app.css", "text/css; charset=utf-8"))
	mux.HandleFunc("GET /app.js", serveEmbedded("ui/app.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /theme.js", serveEmbedded("ui/theme.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /art.js", serveEmbedded("ui/art.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /art.css", serveEmbedded("ui/art.css", "text/css; charset=utf-8"))
	mux.HandleFunc("GET /favicon.svg", serveEmbedded("ui/favicon.svg", "image/svg+xml"))
}

// serveEmbedded answers with one file from the embedded filesystem. The
// content type is written out rather than guessed, because the responses
// carry X-Content-Type-Options: nosniff and a wrong guess would leave the
// browser with a stylesheet or a script it refuses to use.
func serveEmbedded(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := uiFS.ReadFile(name)
		if err != nil {
			http.Error(w, "that part of the page is missing from this build", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}
}
