// localStorage 数据持久化工具
// 用于存储用户偏好和会话状态

const STORAGE_PREFIX = 'gopaw_'

export interface StoredState {
  // 聊天状态
  currentSessionId?: string
  recentSessions?: string[]

  // 用户偏好
  theme?: 'light' | 'dark' | 'system'
  language?: string

  // 时间戳
  lastActive?: number
}

/**
 * 存储状态
 */
export function saveState(state: Partial<StoredState>): void {
  try {
    const existing = getState()
    const merged = { ...existing, ...state, lastActive: Date.now() }
    localStorage.setItem(`${STORAGE_PREFIX}state`, JSON.stringify(merged))
  } catch (error) {
    console.error('Failed to save state:', error)
  }
}

/**
 * 获取状态
 */
export function getState(): StoredState {
  try {
    const data = localStorage.getItem(`${STORAGE_PREFIX}state`)
    if (!data) return {}
    return JSON.parse(data)
  } catch (error) {
    console.error('Failed to get state:', error)
    return {}
  }
}

/**
 * 删除状态
 */
export function clearState(): void {
  try {
    localStorage.removeItem(`${STORAGE_PREFIX}state`)
  } catch (error) {
    console.error('Failed to clear state:', error)
  }
}

/**
 * 存储当前会话 ID
 */
export function saveCurrentSession(sessionId: string): void {
  saveState({ currentSessionId: sessionId })
}

/**
 * 获取当前会话 ID
 */
export function getCurrentSession(): string | undefined {
  return getState().currentSessionId
}

/**
 * 存储最近会话列表
 */
export function saveRecentSessions(sessionIds: string[]): void {
  saveState({ recentSessions: sessionIds })
}

/**
 * 获取最近会话列表
 */
export function getRecentSessions(): string[] {
  return getState().recentSessions || []
}

/**
 * 检查数据是否过期（7 天）
 */
export function isDataExpired(timestamp: number, ttl: number = 7 * 24 * 60 * 60 * 1000): boolean {
  return Date.now() - timestamp > ttl
}

/**
 * 清理过期数据
 */
export function cleanupExpiredData(): void {
  const state = getState()
  if (state.lastActive && isDataExpired(state.lastActive)) {
    // 数据过期，可以选择清理
    console.log('Stored data is expired')
  }
}
