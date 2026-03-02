// 中文：定义项目通用的 TypeScript 接口
// English: Define common TypeScript interfaces for the project

// 中文：LLM 提供商配置
// English: LLM Provider configuration
export interface Provider {
  id: string
  name: string
  baseURL: string
  apiKey: string
  model: string
  maxTokens: number
  timeoutSec: number
  isActive: boolean
  createdAt?: number
  updatedAt?: number
}

// 中文：后端返回的提供商结构（snake_case）
// English: Provider structure returned by backend (snake_case)
export interface BackendProvider {
  id: string
  name: string
  base_url: string
  api_key: string
  model: string
  max_tokens: number
  timeout_sec: number
  is_active: boolean
  created_at: number
  updated_at: number
}

// 中文：频道健康状态
// English: Channel health status
export interface ChannelStatus {
  name: string
  running: boolean
  message: string
  since: number
}

// 中文：定时任务（Cron）
// English: Cron Job
export interface CronJob {
  id: string
  name: string
  description: string
  cron_expr: string
  channel: string
  session_id: string
  prompt: string
  enabled: boolean
  active_from: string // "HH:MM"
  active_until: string // "HH:MM"
  last_run?: number
  next_run?: number
  created_at?: number
}

// 中文：聊天消息
// English: Chat Message
export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  time: string
}

// 中文：会话信息
// English: Session information
export interface SessionInfo {
  id: string
  user_id: string
  channel: string
  created_at: number
  updated_at: number
}
