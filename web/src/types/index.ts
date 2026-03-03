// 中文：定义项目通用的 TypeScript 接口
// English: Define common TypeScript interfaces for the project

// LLM 提供商配置
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

// 后端返回的提供商结构（snake_case）
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

// 频道健康状态
export interface ChannelStatus {
  name: string
  running: boolean
  message: string
  since: number
}

// 定时任务（Cron）
export interface CronJob {
  id: string
  name: string
  description: string
  cron_expr: string
  channel: string
  session_id: string
  prompt: string
  enabled: boolean
  active_from?: string | null // "HH:MM"
  active_until?: string | null // "HH:MM"
  last_run?: number
  next_run?: number
  created_at?: number
}

// 定时任务执行历史
export interface CronRun {
  id: string
  job_id: string
  triggered_at: number
  finished_at: number | null
  status: 'success' | 'error' | 'running'
  output: string
  error_msg: string
}

// 聊天消息
export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  time: string
}

// 会话统计信息
export interface SessionStats {
  session_id: string
  message_count: number
  total_tokens: number
  user_tokens: number
  assist_tokens: number
}

// 会话信息
export interface SessionInfo {
  id: string
  name: string
  user_id: string
  channel: string
  created_at: number
  updated_at: number
}
