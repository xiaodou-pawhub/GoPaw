// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package resource

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Service provides business logic for resource packages.
type Service struct {
	store  *Store
	logger *zap.Logger
}

// NewService creates a new resource service.
func NewService(store *Store, logger *zap.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger.Named("resource"),
	}
}

// CreatePackage creates a new resource package.
func (s *Service) CreatePackage(ctx context.Context, name, description, createdBy string, isGlobal bool) (*Package, error) {
	if name == "" {
		return nil, fmt.Errorf("resource: package name is required")
	}

	pkg := &Package{
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		IsGlobal:    isGlobal,
	}

	if err := s.store.CreatePackage(ctx, pkg); err != nil {
		return nil, err
	}

	return pkg, nil
}

// GetPackage retrieves a package by ID.
func (s *Service) GetPackage(ctx context.Context, id string) (*Package, error) {
	return s.store.GetPackage(ctx, id)
}

// ListPackages returns all packages.
func (s *Service) ListPackages(ctx context.Context) ([]*Package, error) {
	return s.store.ListPackages(ctx)
}

// UpdatePackage updates an existing package.
func (s *Service) UpdatePackage(ctx context.Context, id, name, description string, isGlobal bool) error {
	pkg, err := s.store.GetPackage(ctx, id)
	if err != nil {
		return err
	}

	pkg.Name = name
	pkg.Description = description
	pkg.IsGlobal = isGlobal

	return s.store.UpdatePackage(ctx, pkg)
}

// DeletePackage deletes a package by ID.
func (s *Service) DeletePackage(ctx context.Context, id string) error {
	return s.store.DeletePackage(ctx, id)
}

// AddItem adds a resource item to a package.
func (s *Service) AddItem(ctx context.Context, packageID string, resourceType ResourceType, resourceID string) error {
	item := &Item{
		PackageID:    packageID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
	return s.store.AddItem(ctx, item)
}

// RemoveItem removes a resource item from a package.
func (s *Service) RemoveItem(ctx context.Context, packageID string, resourceType ResourceType, resourceID string) error {
	return s.store.RemoveItem(ctx, packageID, resourceType, resourceID)
}

// GetItems returns all items in a package.
func (s *Service) GetItems(ctx context.Context, packageID string) ([]*Item, error) {
	return s.store.GetItems(ctx, packageID)
}

// GrantToUser grants a package to a user.
func (s *Service) GrantToUser(ctx context.Context, userID, packageID, grantedBy string) error {
	// Verify package exists
	_, err := s.store.GetPackage(ctx, packageID)
	if err != nil {
		return fmt.Errorf("resource: package not found")
	}

	return s.store.GrantToUser(ctx, userID, packageID, grantedBy)
}

// RevokeUserGrant revokes a package grant from a user.
func (s *Service) RevokeUserGrant(ctx context.Context, userID, packageID string) error {
	return s.store.RevokeUserGrant(ctx, userID, packageID)
}

// GetUserPackages returns all packages granted to a user.
func (s *Service) GetUserPackages(ctx context.Context, userID string) ([]*Package, error) {
	return s.store.GetUserPackages(ctx, userID)
}

// SetAgentPermission sets fine-grained permissions for an agent.
func (s *Service) SetAgentPermission(ctx context.Context, userID, agentID string, canUse, canModify, canDelete bool) error {
	return s.store.SetAgentPermission(ctx, userID, agentID, canUse, canModify, canDelete)
}

// CanUseAgent checks if a user can use an agent.
func (s *Service) CanUseAgent(ctx context.Context, userID, agentID string) (bool, error) {
	// Check agent visibility first
	visibility, ownerID, err := s.store.GetAgentVisibility(ctx, agentID)
	if err != nil {
		return false, err
	}

	// Global agents are available to everyone
	if visibility == "global" {
		return true, nil
	}

	// If user is the owner, they can use it
	if ownerID != nil && *ownerID == userID {
		return true, nil
	}

	// Check explicit permission
	canUse, _, _, err := s.store.GetAgentPermission(ctx, userID, agentID)
	return canUse, err
}

// CanModifyAgent checks if a user can modify an agent.
func (s *Service) CanModifyAgent(ctx context.Context, userID, agentID string) (bool, error) {
	// Check agent visibility
	visibility, ownerID, err := s.store.GetAgentVisibility(ctx, agentID)
	if err != nil {
		return false, err
	}

	// Only owner can modify private agents
	if visibility == "private" {
		return ownerID != nil && *ownerID == userID, nil
	}

	// Check explicit permission
	_, canModify, _, err := s.store.GetAgentPermission(ctx, userID, agentID)
	return canModify, err
}

// SetAgentVisibility sets the visibility of an agent.
func (s *Service) SetAgentVisibility(ctx context.Context, agentID, visibility string, ownerID *string) error {
	if visibility != "global" && visibility != "private" && visibility != "shared" {
		return fmt.Errorf("resource: invalid visibility type")
	}
	return s.store.SetAgentVisibility(ctx, agentID, visibility, ownerID)
}
