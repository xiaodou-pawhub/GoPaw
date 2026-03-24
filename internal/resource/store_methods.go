// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package resource

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CreatePackage creates a new resource package.
func (s *Store) CreatePackage(ctx context.Context, pkg *Package) error {
	if pkg.ID == "" {
		pkg.ID = uuid.New().String()
	}
	pkg.CreatedAt = time.Now()

	query := `
		INSERT INTO resource_packages (id, name, description, created_by, is_global)
		VALUES (?, ?, ?, ?, ?)
	`

	isGlobal := 0
	if pkg.IsGlobal {
		isGlobal = 1
	}

	_, err := s.db.ExecContext(ctx, query, pkg.ID, pkg.Name, pkg.Description, pkg.CreatedBy, isGlobal)
	if err != nil {
		return fmt.Errorf("resource: failed to create package: %w", err)
	}

	s.logger.Info("created resource package", zap.String("id", pkg.ID), zap.String("name", pkg.Name))
	return nil
}

// GetPackage retrieves a package by ID.
func (s *Store) GetPackage(ctx context.Context, id string) (*Package, error) {
	query := `
		SELECT id, name, description, created_by, created_at, is_global
		FROM resource_packages
		WHERE id = ?
	`

	pkg := &Package{}
	isGlobal := 0
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&pkg.ID, &pkg.Name, &pkg.Description, &pkg.CreatedBy, &pkg.CreatedAt, &isGlobal,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("resource: package not found")
		}
		return nil, fmt.Errorf("resource: failed to get package: %w", err)
	}

	pkg.IsGlobal = isGlobal == 1
	return pkg, nil
}

// ListPackages returns all packages.
func (s *Store) ListPackages(ctx context.Context) ([]*Package, error) {
	query := `
		SELECT id, name, description, created_by, created_at, is_global
		FROM resource_packages
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("resource: failed to list packages: %w", err)
	}
	defer rows.Close()

	var packages []*Package
	for rows.Next() {
		pkg := &Package{}
		isGlobal := 0
		err := rows.Scan(&pkg.ID, &pkg.Name, &pkg.Description, &pkg.CreatedBy, &pkg.CreatedAt, &isGlobal)
		if err != nil {
			return nil, fmt.Errorf("resource: failed to scan package: %w", err)
		}
		pkg.IsGlobal = isGlobal == 1
		packages = append(packages, pkg)
	}

	return packages, nil
}

// UpdatePackage updates an existing package.
func (s *Store) UpdatePackage(ctx context.Context, pkg *Package) error {
	query := `
		UPDATE resource_packages
		SET name = ?, description = ?, is_global = ?
		WHERE id = ?
	`

	isGlobal := 0
	if pkg.IsGlobal {
		isGlobal = 1
	}

	_, err := s.db.ExecContext(ctx, query, pkg.Name, pkg.Description, isGlobal, pkg.ID)
	if err != nil {
		return fmt.Errorf("resource: failed to update package: %w", err)
	}

	s.logger.Info("updated resource package", zap.String("id", pkg.ID))
	return nil
}

// DeletePackage deletes a package by ID.
func (s *Store) DeletePackage(ctx context.Context, id string) error {
	query := `DELETE FROM resource_packages WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("resource: failed to delete package: %w", err)
	}

	s.logger.Info("deleted resource package", zap.String("id", id))
	return nil
}

// AddItem adds a resource item to a package.
func (s *Store) AddItem(ctx context.Context, item *Item) error {
	query := `
		INSERT OR REPLACE INTO resource_package_items (package_id, resource_type, resource_id)
		VALUES (?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query, item.PackageID, item.ResourceType, item.ResourceID)
	if err != nil {
		return fmt.Errorf("resource: failed to add item: %w", err)
	}

	s.logger.Info("added resource item",
		zap.String("package_id", item.PackageID),
		zap.String("resource_type", string(item.ResourceType)),
		zap.String("resource_id", item.ResourceID))
	return nil
}

// RemoveItem removes a resource item from a package.
func (s *Store) RemoveItem(ctx context.Context, packageID string, resourceType ResourceType, resourceID string) error {
	query := `DELETE FROM resource_package_items WHERE package_id = ? AND resource_type = ? AND resource_id = ?`

	_, err := s.db.ExecContext(ctx, query, packageID, resourceType, resourceID)
	if err != nil {
		return fmt.Errorf("resource: failed to remove item: %w", err)
	}

	s.logger.Info("removed resource item",
		zap.String("package_id", packageID),
		zap.String("resource_type", string(resourceType)),
		zap.String("resource_id", resourceID))
	return nil
}

// GetItems returns all items in a package.
func (s *Store) GetItems(ctx context.Context, packageID string) ([]*Item, error) {
	query := `
		SELECT package_id, resource_type, resource_id
		FROM resource_package_items
		WHERE package_id = ?
	`

	rows, err := s.db.QueryContext(ctx, query, packageID)
	if err != nil {
		return nil, fmt.Errorf("resource: failed to get items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		item := &Item{}
		err := rows.Scan(&item.PackageID, &item.ResourceType, &item.ResourceID)
		if err != nil {
			return nil, fmt.Errorf("resource: failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GrantToUser grants a package to a user.
func (s *Store) GrantToUser(ctx context.Context, userID, packageID, grantedBy string) error {
	query := `
		INSERT OR REPLACE INTO user_package_grants (user_id, package_id, granted_by, granted_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query, userID, packageID, grantedBy, time.Now())
	if err != nil {
		return fmt.Errorf("resource: failed to grant package to user: %w", err)
	}

	s.logger.Info("granted package to user",
		zap.String("user_id", userID),
		zap.String("package_id", packageID))
	return nil
}

// RevokeUserGrant revokes a package grant from a user.
func (s *Store) RevokeUserGrant(ctx context.Context, userID, packageID string) error {
	query := `DELETE FROM user_package_grants WHERE user_id = ? AND package_id = ?`

	_, err := s.db.ExecContext(ctx, query, userID, packageID)
	if err != nil {
		return fmt.Errorf("resource: failed to revoke user grant: %w", err)
	}

	s.logger.Info("revoked package grant from user",
		zap.String("user_id", userID),
		zap.String("package_id", packageID))
	return nil
}

// GetUserPackages returns all packages granted to a user.
func (s *Store) GetUserPackages(ctx context.Context, userID string) ([]*Package, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_by, p.created_at, p.is_global
		FROM resource_packages p
		INNER JOIN user_package_grants g ON p.id = g.package_id
		WHERE g.user_id = ?
		ORDER BY g.granted_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("resource: failed to get user packages: %w", err)
	}
	defer rows.Close()

	var packages []*Package
	for rows.Next() {
		pkg := &Package{}
		isGlobal := 0
		err := rows.Scan(&pkg.ID, &pkg.Name, &pkg.Description, &pkg.CreatedBy, &pkg.CreatedAt, &isGlobal)
		if err != nil {
			return nil, fmt.Errorf("resource: failed to scan package: %w", err)
		}
		pkg.IsGlobal = isGlobal == 1
		packages = append(packages, pkg)
	}

	return packages, nil
}

// SetAgentPermission sets fine-grained permissions for an agent.
func (s *Store) SetAgentPermission(ctx context.Context, userID, agentID string, canUse, canModify, canDelete bool) error {
	query := `
		INSERT OR REPLACE INTO user_agent_permissions (user_id, agent_id, can_use, can_modify, can_delete)
		VALUES (?, ?, ?, ?, ?)
	`

	cu, cm, cd := 0, 0, 0
	if canUse {
		cu = 1
	}
	if canModify {
		cm = 1
	}
	if canDelete {
		cd = 1
	}

	_, err := s.db.ExecContext(ctx, query, userID, agentID, cu, cm, cd)
	if err != nil {
		return fmt.Errorf("resource: failed to set agent permission: %w", err)
	}

	s.logger.Info("set agent permission",
		zap.String("user_id", userID),
		zap.String("agent_id", agentID),
		zap.Bool("can_use", canUse),
		zap.Bool("can_modify", canModify),
		zap.Bool("can_delete", canDelete))
	return nil
}

// GetAgentPermission returns permissions for a user-agent pair.
func (s *Store) GetAgentPermission(ctx context.Context, userID, agentID string) (canUse, canModify, canDelete bool, err error) {
	query := `
		SELECT can_use, can_modify, can_delete
		FROM user_agent_permissions
		WHERE user_id = ? AND agent_id = ?
	`

	cu, cm, cd := 0, 0, 0
	err = s.db.QueryRowContext(ctx, query, userID, agentID).Scan(&cu, &cm, &cd)
	if err != nil {
		if err == sql.ErrNoRows {
			// Default: can use but not modify/delete
			return true, false, false, nil
		}
		return false, false, false, fmt.Errorf("resource: failed to get agent permission: %w", err)
	}

	return cu == 1, cm == 1, cd == 1, nil
}

// SetAgentVisibility sets the visibility of an agent.
func (s *Store) SetAgentVisibility(ctx context.Context, agentID, visibility string, ownerID *string) error {
	query := `
		INSERT OR REPLACE INTO agent_visibility (agent_id, visibility, owner_id)
		VALUES (?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query, agentID, visibility, ownerID)
	if err != nil {
		return fmt.Errorf("resource: failed to set agent visibility: %w", err)
	}

	s.logger.Info("set agent visibility",
		zap.String("agent_id", agentID),
		zap.String("visibility", visibility))
	return nil
}

// GetAgentVisibility returns the visibility of an agent.
func (s *Store) GetAgentVisibility(ctx context.Context, agentID string) (visibility string, ownerID *string, err error) {
	query := `
		SELECT visibility, owner_id
		FROM agent_visibility
		WHERE agent_id = ?
	`

	err = s.db.QueryRowContext(ctx, query, agentID).Scan(&visibility, &ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Default: private
			return "private", nil, nil
		}
		return "", nil, fmt.Errorf("resource: failed to get agent visibility: %w", err)
	}

	return visibility, ownerID, nil
}
