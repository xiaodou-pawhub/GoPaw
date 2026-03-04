import api from './index'

export async function login(token: string): Promise<{ ok: boolean }> {
  return await api.post('/auth/login', { token })
}

export async function logout(): Promise<{ ok: boolean }> {
  return await api.post('/auth/logout')
}

// 返回 true = 已登录（cookie 有效），false = 未登录（401）
export async function checkAuthStatus(): Promise<boolean> {
  try {
    await api.get('/auth/status')
    return true
  } catch {
    return false
  }
}
