//go:build dev

// Package web provides a no-op FS in dev mode.
// In dev mode the Vite dev server (localhost:5173) handles all frontend assets
// with HMR; the Go backend only serves the API on localhost:8088.
package web

import "io/fs"

// FS returns nil in dev mode. The server skips static file serving when
// staticFS is nil, leaving the Vite dev server to handle the frontend.
func FS() fs.FS {
	return nil
}
