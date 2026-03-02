//go:build !dev

// Package web embeds the compiled Vue frontend assets into the Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the embedded frontend filesystem with the "dist/" prefix stripped,
// so callers can open "index.html" directly instead of "dist/index.html".
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic("web: failed to create sub-filesystem: " + err.Error())
	}
	return sub
}
