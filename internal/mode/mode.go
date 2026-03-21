// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package mode defines the three deployment modes supported by GoPaw:
//
//   - solo:  single-user personal use, no authentication required.
//   - team:  small team (≤50 users), JWT auth, admin-managed accounts.
//   - cloud: SaaS, JWT auth + invite codes, open or restricted registration.
package mode

// Mode represents the deployment mode of a GoPaw instance.
type Mode string

const (
	// Solo is the default single-user mode. Authentication is disabled so
	// the web UI is immediately accessible on first launch.
	Solo Mode = "solo"

	// Team enables multi-user support for small groups. Users are created by
	// an administrator. JWT auth is enforced.
	Team Mode = "team"

	// Cloud enables full SaaS features: open registration, invite codes, and
	// usage metering. JWT auth is enforced.
	Cloud Mode = "cloud"
)

// Parse converts a string to Mode, defaulting to Solo for unknown values.
func Parse(s string) Mode {
	switch Mode(s) {
	case Team:
		return Team
	case Cloud:
		return Cloud
	default:
		return Solo
	}
}

// IsMultiUser reports whether this mode requires per-user data isolation.
func (m Mode) IsMultiUser() bool {
	return m == Team || m == Cloud
}

// RequireAuth reports whether HTTP requests must be authenticated.
// Solo mode is always unauthenticated.
func (m Mode) RequireAuth() bool {
	return m == Team || m == Cloud
}

// RequireInvite reports whether new user registration requires an invite code.
func (m Mode) RequireInvite() bool {
	return m == Cloud
}

// String implements fmt.Stringer.
func (m Mode) String() string {
	return string(m)
}
