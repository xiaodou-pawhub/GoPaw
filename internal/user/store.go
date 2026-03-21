// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a user does not exist.
var ErrNotFound = errors.New("user: not found")

// ErrDuplicate is returned when a username already exists.
var ErrDuplicate = errors.New("user: username already taken")

// Store persists user accounts in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the users database at dbPath and runs migrations.
// Pass the shared gopaw.db *sql.DB handle to reuse the existing connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("user store: migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    username    TEXT NOT NULL UNIQUE,
    email       TEXT,
    role        TEXT NOT NULL DEFAULT 'member',
    password_hash TEXT NOT NULL,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
`)
	return err
}

// Create inserts a new user. Returns ErrDuplicate if username is taken.
func (s *Store) Create(u *User) error {
	u.ID = uuid.New().String()
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := s.db.Exec(
		`INSERT INTO users (id, username, email, role, password_hash, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.Email, string(u.Role), u.PasswordHash, boolToInt(u.IsActive), u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("user store: create: %w", err)
	}
	return nil
}

// GetByUsername retrieves a user by username. Returns ErrNotFound if absent.
func (s *Store) GetByUsername(username string) (*User, error) {
	row := s.db.QueryRow(
		`SELECT id, username, email, role, password_hash, is_active, created_at, updated_at
		 FROM users WHERE username = ?`, username)
	return scanUser(row)
}

// GetByID retrieves a user by ID. Returns ErrNotFound if absent.
func (s *Store) GetByID(id string) (*User, error) {
	row := s.db.QueryRow(
		`SELECT id, username, email, role, password_hash, is_active, created_at, updated_at
		 FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// List returns all users ordered by created_at DESC.
func (s *Store) List() ([]*User, error) {
	rows, err := s.db.Query(
		`SELECT id, username, email, role, password_hash, is_active, created_at, updated_at
		 FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("user store: list: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Update saves changes to username, email, role, password_hash, and is_active.
func (s *Store) Update(u *User) error {
	u.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(
		`UPDATE users SET username=?, email=?, role=?, password_hash=?, is_active=?, updated_at=?
		 WHERE id=?`,
		u.Username, u.Email, string(u.Role), u.PasswordHash, boolToInt(u.IsActive), u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("user store: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes the user with the given ID.
func (s *Store) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("user store: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Count returns the total number of users.
func (s *Store) Count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(s scanner) (*User, error) {
	var u User
	var active int
	var email sql.NullString
	err := s.Scan(&u.ID, &u.Username, &email, &u.Role, &u.PasswordHash, &active, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user store: scan: %w", err)
	}
	u.Email = email.String
	u.IsActive = active == 1
	return &u, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isUniqueConstraint(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
