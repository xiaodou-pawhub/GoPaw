// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package user

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// ErrBadCredentials is returned when username or password is wrong.
var ErrBadCredentials = errors.New("user: invalid credentials")

// ErrInactive is returned when the account exists but is disabled.
var ErrInactive = errors.New("user: account is inactive")

// Service wraps Store with business logic.
type Service struct {
	store *Store
}

// NewService creates a Service backed by the given Store.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// CreateUser hashes the password and persists the user.
func (svc *Service) CreateUser(username, email, password string, role Role) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("user service: hash password: %w", err)
	}
	u := &User{
		Username:     username,
		Email:        email,
		Role:         role,
		PasswordHash: string(hash),
		IsActive:     true,
	}
	if err := svc.store.Create(u); err != nil {
		return nil, err
	}
	return u, nil
}

// Authenticate verifies credentials and returns the user on success.
// Returns ErrBadCredentials or ErrInactive on failure.
func (svc *Service) Authenticate(username, password string) (*User, error) {
	u, err := svc.store.GetByUsername(username)
	if errors.Is(err, ErrNotFound) {
		// Constant-time comparison to resist timing attacks even on missing users.
		bcrypt.CompareHashAndPassword([]byte("$2a$10$invalid"), []byte(password)) //nolint
		return nil, ErrBadCredentials
	}
	if err != nil {
		return nil, err
	}

	if !u.IsActive {
		return nil, ErrInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrBadCredentials
	}
	return u, nil
}

// EnsureAdminExists creates the default admin user if no users exist.
// Used on first startup in team mode.
func (svc *Service) EnsureAdminExists(username, password string) (*User, error) {
	n, err := svc.store.Count()
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return svc.store.GetByUsername(username)
	}
	return svc.CreateUser(username, "", password, RoleAdmin)
}

// GetByID returns the user with the given ID.
func (svc *Service) GetByID(id string) (*User, error) {
	return svc.store.GetByID(id)
}

// List returns all users.
func (svc *Service) List() ([]*User, error) {
	return svc.store.List()
}

// SetRole updates the user's role.
func (svc *Service) SetRole(userID string, role Role) error {
	u, err := svc.store.GetByID(userID)
	if err != nil {
		return err
	}
	u.Role = role
	return svc.store.Update(u)
}

// SetPassword updates the user's password.
func (svc *Service) SetPassword(userID, newPassword string) error {
	u, err := svc.store.GetByID(userID)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("user service: hash password: %w", err)
	}
	u.PasswordHash = string(hash)
	return svc.store.Update(u)
}

// SetActive enables or disables the account.
func (svc *Service) SetActive(userID string, active bool) error {
	u, err := svc.store.GetByID(userID)
	if err != nil {
		return err
	}
	u.IsActive = active
	return svc.store.Update(u)
}

// Delete removes the user from the store.
func (svc *Service) Delete(userID string) error {
	return svc.store.Delete(userID)
}
