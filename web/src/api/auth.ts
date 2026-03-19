// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

import axios from 'axios'

const API_BASE = '/api'

// Legacy token storage key (for admin token auth)
const TOKEN_KEY = 'gopaw_token'

// ============================================
// Legacy Auth Functions (for admin token auth)
// ============================================

/**
 * Check if user is authenticated (legacy admin token)
 * Uses cookie-based authentication, so we just check the status endpoint
 */
export async function checkAuthStatus(): Promise<boolean> {
  try {
    const response = await axios.get(`${API_BASE}/auth/status`)
    return response.status === 200 && response.data?.authenticated === true
  } catch {
    return false
  }
}

/**
 * Login with admin token (legacy)
 */
export async function login(token: string): Promise<void> {
  const response = await axios.post(`${API_BASE}/auth/login`, { token })
  if (response.status === 200) {
    localStorage.setItem(TOKEN_KEY, token)
  } else {
    throw new Error('Login failed')
  }
}

/**
 * Logout (legacy)
 */
export function logout(): void {
  localStorage.removeItem(TOKEN_KEY)
}

/**
 * Get stored token (legacy)
 */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

// ============================================
// Types
export interface User {
  id: string
  username: string
  email: string
  display_name: string
  avatar: string
  status: string
  last_login_at?: string
  created_at: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export interface AuthResponse {
  code: number
  message: string
  data: {
    user: User
    tokens: TokenPair
  }
}

export interface Team {
  id: string
  name: string
  slug: string
  description: string
  avatar: string
  owner_id: string
  settings: string
  status: string
  created_at: string
  updated_at: string
}

export interface TeamMember {
  id: string
  team_id: string
  user_id: string
  role: string
  joined_at: string
  invited_by?: string
  user?: User
}

// Auth API
export const authApi = {
  // Register a new user
  async register(data: {
    username: string
    email: string
    password: string
    display_name?: string
  }): Promise<AuthResponse> {
    const response = await axios.post(`${API_BASE}/auth/register`, data)
    return response.data
  },

  // Login
  async login(data: { username: string; password: string }): Promise<AuthResponse> {
    const response = await axios.post(`${API_BASE}/auth/login`, data)
    return response.data
  },

  // Refresh token
  async refreshToken(refreshToken: string): Promise<{ code: number; message: string; data: { tokens: TokenPair } }> {
    const response = await axios.post(`${API_BASE}/auth/refresh`, { refresh_token: refreshToken })
    return response.data
  },

  // Get current user profile
  async getProfile(): Promise<{ code: number; message: string; data: User }> {
    const response = await axios.get(`${API_BASE}/auth/profile`)
    return response.data
  },

  // Update profile
  async updateProfile(data: { display_name?: string; avatar?: string }): Promise<{ code: number; message: string; data: User }> {
    const response = await axios.put(`${API_BASE}/auth/profile`, data)
    return response.data
  },

  // Change password
  async changePassword(data: { current_password: string; new_password: string }): Promise<{ code: number; message: string }> {
    const response = await axios.post(`${API_BASE}/auth/change-password`, data)
    return response.data
  },
}

// Team API
export const teamApi = {
  // Create a new team
  async create(data: { name: string; slug?: string; description?: string; avatar?: string }): Promise<{ code: number; message: string; data: Team }> {
    const response = await axios.post(`${API_BASE}/teams`, data)
    return response.data
  },

  // List user's teams
  async list(): Promise<{ code: number; message: string; data: Team[] }> {
    const response = await axios.get(`${API_BASE}/teams`)
    return response.data
  },

  // Get team by ID
  async get(teamId: string): Promise<{ code: number; message: string; data: Team }> {
    const response = await axios.get(`${API_BASE}/teams/${teamId}`)
    return response.data
  },

  // Update team
  async update(teamId: string, data: { name?: string; description?: string; avatar?: string; settings?: string }): Promise<{ code: number; message: string; data: Team }> {
    const response = await axios.put(`${API_BASE}/teams/${teamId}`, data)
    return response.data
  },

  // Delete team
  async delete(teamId: string): Promise<{ code: number; message: string }> {
    const response = await axios.delete(`${API_BASE}/teams/${teamId}`)
    return response.data
  },

  // Get team members
  async getMembers(teamId: string): Promise<{ code: number; message: string; data: TeamMember[] }> {
    const response = await axios.get(`${API_BASE}/teams/${teamId}/members`)
    return response.data
  },

  // Add member
  async addMember(teamId: string, data: { user_id: string; role: string }): Promise<{ code: number; message: string }> {
    const response = await axios.post(`${API_BASE}/teams/${teamId}/members`, data)
    return response.data
  },

  // Remove member
  async removeMember(teamId: string, userId: string): Promise<{ code: number; message: string }> {
    const response = await axios.delete(`${API_BASE}/teams/${teamId}/members/${userId}`)
    return response.data
  },

  // Invite member
  async inviteMember(teamId: string, data: { email: string; role: string; expires_in?: number }): Promise<{ code: number; message: string; data: { invitation_id: string; token: string; expires_at?: string } }> {
    const response = await axios.post(`${API_BASE}/teams/${teamId}/invite`, data)
    return response.data
  },

  // Accept invitation
  async acceptInvitation(token: string): Promise<{ code: number; message: string }> {
    const response = await axios.post(`${API_BASE}/teams/accept-invitation?token=${token}`)
    return response.data
  },
}

// Token storage utilities
export const tokenStorage = {
  getAccessToken(): string | null {
    return localStorage.getItem('access_token')
  },

  getRefreshToken(): string | null {
    return localStorage.getItem('refresh_token')
  },

  setTokens(tokens: TokenPair): void {
    localStorage.setItem('access_token', tokens.access_token)
    localStorage.setItem('refresh_token', tokens.refresh_token)
    localStorage.setItem('token_expires_at', String(Date.now() + tokens.expires_in * 1000))
  },

  clearTokens(): void {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('token_expires_at')
    localStorage.removeItem('user')
  },

  isTokenExpired(): boolean {
    const expiresAt = localStorage.getItem('token_expires_at')
    if (!expiresAt) return true
    return Date.now() > parseInt(expiresAt, 10)
  },
}

// Setup axios interceptors
export function setupAxiosInterceptors() {
  // Request interceptor - add auth header
  axios.interceptors.request.use(
    (config) => {
      const token = tokenStorage.getAccessToken()
      if (token && !tokenStorage.isTokenExpired()) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    },
    (error) => Promise.reject(error)
  )

  // Response interceptor - handle token refresh
  axios.interceptors.response.use(
    (response) => response,
    async (error) => {
      const originalRequest = error.config

      // If 401 and we have a refresh token, try to refresh
      if (error.response?.status === 401 && !originalRequest._retry) {
        originalRequest._retry = true

        const refreshToken = tokenStorage.getRefreshToken()
        if (refreshToken) {
          try {
            const response = await authApi.refreshToken(refreshToken)
            tokenStorage.setTokens(response.data.tokens)
            originalRequest.headers.Authorization = `Bearer ${response.data.tokens.access_token}`
            return axios(originalRequest)
          } catch {
            // Refresh failed, clear tokens and redirect to login
            tokenStorage.clearTokens()
            window.location.href = '/login'
          }
        }
      }

      return Promise.reject(error)
    }
  )
}