// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package permission provides resource permission checking for team mode.
package permission

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gopaw/gopaw/internal/audit"
	"go.uber.org/zap"
)

// Checker provides permission checking for resources.
type Checker struct {
	db       *sql.DB
	auditMgr *audit.Manager
	logger   *zap.Logger
}

// NewChecker creates a new permission checker.
func NewChecker(db *sql.DB, auditMgr *audit.Manager, logger *zap.Logger) *Checker {
	return &Checker{
		db:       db,
		auditMgr: auditMgr,
		logger:   logger.Named("permission"),
	}
}

// CanUseResource checks if a user can use a specific resource.
// resourceType: agent, skill, knowledge, model
func (c *Checker) CanUseResource(ctx context.Context, userID, resourceType, resourceID string) (bool, error) {
	// For team mode, check resource package grants
	query := `
		SELECT COUNT(*) > 0
		FROM user_package_grants g
		JOIN resource_package_items i ON g.package_id = i.package_id
		WHERE g.user_id = ? AND i.resource_type = ? AND i.resource_id = ?
	`

	var hasAccess bool
	err := c.db.QueryRowContext(ctx, query, userID, resourceType, resourceID).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("permission: failed to check resource access: %w", err)
	}

	if hasAccess {
		return true, nil
	}

	// For agents, also check visibility
	if resourceType == "agent" {
		return c.canUseAgent(ctx, userID, resourceID)
	}

	// For other resources, check if they are in a global package
	return c.hasGlobalAccess(ctx, resourceType, resourceID)
}

// canUseAgent checks agent-specific permissions
func (c *Checker) canUseAgent(ctx context.Context, userID, agentID string) (bool, error) {
	// Check agent visibility
	query := `
		SELECT visibility, owner_id
		FROM agents
		WHERE id = ?
	`

	var visibility string
	var ownerID sql.NullString
	err := c.db.QueryRowContext(ctx, query, agentID).Scan(&visibility, &ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("permission: agent not found")
		}
		return false, fmt.Errorf("permission: failed to get agent visibility: %w", err)
	}

	// Global agents are available to everyone
	if visibility == "global" {
		return true, nil
	}

	// Owner can always use their own agents
	if ownerID.Valid && ownerID.String == userID {
		return true, nil
	}

	// Check explicit permission
	var canUse int
	err = c.db.QueryRowContext(ctx, `
		SELECT can_use
		FROM user_agent_permissions
		WHERE user_id = ? AND agent_id = ?
	`, userID, agentID).Scan(&canUse)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("permission: failed to get agent permission: %w", err)
	}

	return canUse == 1, nil
}

// hasGlobalAccess checks if a resource is in a global package
func (c *Checker) hasGlobalAccess(ctx context.Context, resourceType, resourceID string) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM resource_package_items i
		JOIN resource_packages p ON i.package_id = p.id
		WHERE p.is_global = 1 AND i.resource_type = ? AND i.resource_id = ?
	`

	var hasAccess bool
	err := c.db.QueryRowContext(ctx, query, resourceType, resourceID).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("permission: failed to check global access: %w", err)
	}

	return hasAccess, nil
}

// GetAccessibleResources returns all resources of a type that a user can access.
func (c *Checker) GetAccessibleResources(ctx context.Context, userID, resourceType string) ([]string, error) {
	// Get resources from granted packages
	query := `
		SELECT DISTINCT i.resource_id
		FROM user_package_grants g
		JOIN resource_package_items i ON g.package_id = i.package_id
		WHERE g.user_id = ? AND i.resource_type = ?
	`

	rows, err := c.db.QueryContext(ctx, query, userID, resourceType)
	if err != nil {
		return nil, fmt.Errorf("permission: failed to get accessible resources: %w", err)
	}
	defer rows.Close()

	var resourceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		resourceIDs = append(resourceIDs, id)
	}

	// For agents, also get global and owned agents
	if resourceType == "agent" {
		agentIDs, err := c.getAccessibleAgents(ctx, userID)
		if err != nil {
			return nil, err
		}
		// Merge and deduplicate
		existing := make(map[string]bool)
		for _, id := range resourceIDs {
			existing[id] = true
		}
		for _, id := range agentIDs {
			if !existing[id] {
				resourceIDs = append(resourceIDs, id)
			}
		}
	} else {
		// For other resources, also get global package resources
		globalIDs, err := c.getGlobalResources(ctx, resourceType)
		if err != nil {
			return nil, err
		}
		// Merge and deduplicate
		existing := make(map[string]bool)
		for _, id := range resourceIDs {
			existing[id] = true
		}
		for _, id := range globalIDs {
			if !existing[id] {
				resourceIDs = append(resourceIDs, id)
			}
		}
	}

	return resourceIDs, nil
}

// getAccessibleAgents returns agents that a user can access
func (c *Checker) getAccessibleAgents(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT id
		FROM agents
		WHERE visibility = 'global'
		   OR owner_id = ?
		   OR id IN (SELECT agent_id FROM user_agent_permissions WHERE user_id = ? AND can_use = 1)
	`

	rows, err := c.db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("permission: failed to get accessible agents: %w", err)
	}
	defer rows.Close()

	var agentIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		agentIDs = append(agentIDs, id)
	}

	return agentIDs, nil
}

// getGlobalResources returns resources in global packages
func (c *Checker) getGlobalResources(ctx context.Context, resourceType string) ([]string, error) {
	query := `
		SELECT DISTINCT i.resource_id
		FROM resource_package_items i
		JOIN resource_packages p ON i.package_id = p.id
		WHERE p.is_global = 1 AND i.resource_type = ?
	`

	rows, err := c.db.QueryContext(ctx, query, resourceType)
	if err != nil {
		return nil, fmt.Errorf("permission: failed to get global resources: %w", err)
	}
	defer rows.Close()

	var resourceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		resourceIDs = append(resourceIDs, id)
	}

	return resourceIDs, nil
}
