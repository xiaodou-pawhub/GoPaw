// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

import axios from 'axios'

const API_BASE = '/api'

export interface ResourcePackage {
  id: string
  name: string
  description: string
  created_by: string
  created_at: string
  is_global: boolean
}

export interface ResourceItem {
  package_id: string
  resource_type: 'agent' | 'skill' | 'knowledge' | 'model'
  resource_id: string
}

export interface CreatePackageRequest {
  name: string
  description: string
  is_global: boolean
}

export interface AddItemRequest {
  resource_type: string
  resource_id: string
}

export interface GrantRequest {
  user_id: string
}

/**
 * Resource Package API
 */
export const resourcePackageApi = {
  // List all packages
  async listPackages(): Promise<ResourcePackage[]> {
    const response = await axios.get(`${API_BASE}/resource-packages`)
    return response.data.packages || []
  },

  // Get package by ID
  async getPackage(id: string): Promise<{ package: ResourcePackage; items: ResourceItem[] }> {
    const response = await axios.get(`${API_BASE}/resource-packages/${id}`)
    return response.data
  },

  // Create package
  async createPackage(req: CreatePackageRequest): Promise<ResourcePackage> {
    const response = await axios.post(`${API_BASE}/resource-packages`, req)
    return response.data.package
  },

  // Update package
  async updatePackage(id: string, req: CreatePackageRequest): Promise<void> {
    await axios.put(`${API_BASE}/resource-packages/${id}`, req)
  },

  // Delete package
  async deletePackage(id: string): Promise<void> {
    await axios.delete(`${API_BASE}/resource-packages/${id}`)
  },

  // Add item to package
  async addItem(packageId: string, item: AddItemRequest): Promise<void> {
    await axios.post(`${API_BASE}/resource-packages/${packageId}/items`, item)
  },

  // Remove item from package
  async removeItem(packageId: string, resourceType: string, resourceId: string): Promise<void> {
    await axios.delete(`${API_BASE}/resource-packages/${packageId}/items/${resourceType}/${resourceId}`)
  },

  // Grant package to user
  async grantToUser(packageId: string, userId: string): Promise<void> {
    await axios.post(`${API_BASE}/resource-packages/${packageId}/grant`, { user_id: userId })
  },

  // Revoke grant from user
  async revokeGrant(packageId: string, userId: string): Promise<void> {
    await axios.delete(`${API_BASE}/resource-packages/${packageId}/grant/${userId}`)
  },

  // Get package grants
  async getPackageGrants(packageId: string): Promise<any[]> {
    const response = await axios.get(`${API_BASE}/resource-packages/${packageId}/grants`)
    return response.data.grants || []
  },

  // Get user packages
  async getUserPackages(userId: string): Promise<ResourcePackage[]> {
    const response = await axios.get(`${API_BASE}/users/${userId}/packages`)
    return response.data.packages || []
  },
}

/**
 * Agent Permission API
 */
export const agentPermissionApi = {
  // Set agent permission
  async setPermission(agentId: string, userId: string, canUse: boolean, canModify: boolean, canDelete: boolean): Promise<void> {
    await axios.post(`${API_BASE}/agents/${agentId}/permissions`, {
      user_id: userId,
      can_use: canUse,
      can_modify: canModify,
      can_delete: canDelete,
    })
  },

  // Set agent visibility
  async setVisibility(agentId: string, visibility: 'global' | 'private' | 'shared', ownerId: string | null): Promise<void> {
    await axios.put(`${API_BASE}/agents/${agentId}/visibility`, {
      visibility,
      owner_id: ownerId,
    })
  },

  // Check agent permission
  async checkPermission(agentId: string, userId: string): Promise<{ can_use: boolean; can_modify: boolean }> {
    const response = await axios.get(`${API_BASE}/agents/${agentId}/permission`, {
      params: { user_id: userId },
    })
    return {
      can_use: response.data.can_use === 'true',
      can_modify: response.data.can_modify === 'true',
    }
  },
}
