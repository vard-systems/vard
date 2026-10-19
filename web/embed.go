// Package web holds the two apps, built, for the server to serve: the id app
// on the public listener and the console on the admin one.
//
// The builds are not in the repository. Run `make web` first: until then
// `go build` stops here with "pattern all:id/build: no matching files found".
package web

import (
	"embed"
	"io/fs"
)

// all: so that files SvelteKit names with a leading underscore (_app/) come
// too. Only build/ is embedded, never the sources or node_modules.
var (
	//go:embed all:id/build
	idBuild embed.FS

	//go:embed all:console/build
	consoleBuild embed.FS
)

// ID and Console are the built apps, rooted at their build directory.
var (
	ID      = mustSub(idBuild, "id/build")
	Console = mustSub(consoleBuild, "console/build")
)

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err) // dir is a constant that go:embed has already checked
	}
	return sub
}
