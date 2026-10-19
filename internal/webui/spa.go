// Package webui serves a built single-page app from a file system.
package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPA serves the files in fsys, and index.html for every path that names no
// file, so the app's own router can draw it. It answers only GET and HEAD.
//
// Files under _app/immutable/ have a content hash in their name and are cached
// for a year; everything else — index.html above all, which names the current
// hashes — is revalidated on every use, so a deploy is seen at once.
func SPA(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		setPageHeaders(w.Header())

		// fs.FS never resolves ".." out of fsys, and the file server answers a
		// request path that contains one with a 400; the name is cleaned only
		// to look it up.
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

		if isFile(fsys, name) {
			if strings.HasPrefix(name, "_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}

		// A missing file is a 404, not the app: a tab still running the last
		// deploy asks for chunks that are gone, and an HTML page in their place
		// fails to load as a script with an error that names nothing useful.
		// Routes never have an extension; files always do.
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}

// setPageHeaders are the app's own, for whatever middleware set before: the
// API's Content-Security-Policy forbids everything a page needs, and the
// app's real policy is in the <meta> tag SvelteKit writes into index.html.
// What a <meta> cannot say is said here.
func setPageHeaders(h http.Header) {
	h.Set("Content-Security-Policy", "frame-ancestors 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	// Nothing here uses a camera, a microphone or a payment sheet, so no
	// page may, even an injected one.
	h.Set("Permissions-Policy", permissionsPolicy)
}

const permissionsPolicy = "accelerometer=(), autoplay=(), camera=(), display-capture=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), screen-wake-lock=(), usb=(), xr-spatial-tracking=()"

func isFile(fsys fs.FS, name string) bool {
	if name == "" {
		return false
	}
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}
