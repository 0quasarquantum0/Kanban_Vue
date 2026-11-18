// Package web holds the built frontend, embedded into the server binary.
//
// The files in dist/ are produced by `npm run build:embed` from the repository root. Only a
// placeholder is committed, so the server still compiles before the frontend has been built.
package web

import (
	"embed"
	"io/fs"
)

// The all: prefix includes files whose names start with a dot, such as the .gitkeep placeholder;
// without it, a dist/ holding only that file would fail to compile ("no matching files").
//
//go:embed all:dist
var dist embed.FS

// Files returns the built frontend with dist/ as its root, so index.html is at "index.html".
func Files() fs.FS {
	files, err := fs.Sub(dist, "dist")
	if err != nil {
		// Sub only fails for an invalid path, and "dist" is a constant: this cannot happen.
		panic(err)
	}
	return files
}
