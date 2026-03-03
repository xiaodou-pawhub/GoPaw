// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

import "errors"

// ErrMissingCredentials is returned by Init when required credentials are absent.
// The channel manager treats this as "unconfigured, skip silently" and logs at Info level.
var ErrMissingCredentials = errors.New("missing required credentials")
