// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

import axios from 'axios'

const API_BASE = '/api'

export interface ModeInfo {
  mode: 'solo' | 'team'
  require_auth: boolean
  is_multi_user: boolean
}

export interface LoginResult {
  user_id: string
  username: string
  role: string
  access_token: string
}

/** GET /api/mode — 公开，无需 auth */
export async function getMode(): Promise<ModeInfo> {
  const response = await axios.get(`${API_BASE}/mode`)
  return response.data
}

/** POST /api/auth/login — team 模式用户名+密码登录 */
export async function loginWithPassword(username: string, password: string): Promise<LoginResult> {
  const response = await axios.post(`${API_BASE}/auth/login`, { username, password })
  return response.data
}

/** GET /api/auth/status — 检查当前 session 是否有效 */
export async function checkAuthStatus(): Promise<boolean> {
  try {
    const response = await axios.get(`${API_BASE}/auth/status`)
    return response.status === 200 && response.data?.authenticated === true
  } catch {
    return false
  }
}

/** POST /api/auth/logout */
export async function logout(): Promise<void> {
  await axios.post(`${API_BASE}/auth/logout`)
  localStorage.removeItem('access_token')
}

/** 设置 axios 请求拦截器：读 localStorage access_token，自动加 Bearer header */
export function setupAxiosInterceptors(): void {
  axios.interceptors.request.use(
    (config) => {
      const token = localStorage.getItem('access_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    },
    (error) => Promise.reject(error)
  )

  axios.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.response?.status === 401) {
        localStorage.removeItem('access_token')
        window.location.href = '/login'
      }
      return Promise.reject(error)
    }
  )
}
