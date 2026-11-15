// Package web embeds the built frontend. Run `pnpm build` in this directory
// (or `make build` at the repository root) before compiling the server.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built UI rooted at index.html.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
