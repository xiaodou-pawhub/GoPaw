// 中文：定义项目通用的 TypeScript 接口
// English: Define common TypeScript interfaces for the project

// LLM 模型能力标签定义
export interface ModelCapability {
  key: string
  label: string
  icon: string
  color: 'default' | 'success' | 'warning' | 'error' | 'info' | 'primary'
  category: 'core' | 'feature'  // core=核心能力，feature=特性
}

// 模型能力标签配置
export const MODEL_CAPABILITIES: Record<string, ModelCapability> = {
  // 核心能力
  multimodal: { 
    key: 'multimodal', 
    label: '多模态', 
    icon: '🎬', 
    color: 'primary',
    category: 'core'
  },
  vision: { 
    key: 'vision', 
    label: '视觉理解', 
    icon: '👁️', 
    color: 'success',
    category: 'core'
  },
  image_generation: { 
    key: 'image_generation', 
    label: '图像生成', 
    icon: '🖼️', 
    color: 'warning',
    category: 'core'
  },
  video: { 
    key: 'video', 
    label: '视频处理', 
    icon: '🎥', 
    color: 'warning',
    category: 'core'
  },
  text: { 
    key: 'text', 
    label: '文本对话', 
    icon: '📝', 
    color: 'default',
    category: 'core'
  },
  
  // 进阶特性
  function_call: { 
    key: 'function_call', 
    label: '工具调用', 
    icon: '🔧', 
    color: 'info',
    category: 'feature'
  },
  reasoning: { 
    key: 'reasoning', 
    label: '深度推理', 
    icon: '🧠', 
    color: 'primary',
    category: 'feature'
  },
  long_context: { 
    key: 'long_context', 
    label: '长上下文', 
    icon: '📚', 
    color: 'default',
    category: 'feature'
  },
  streaming: { 
    key: 'streaming', 
    label: '流式输出', 
    icon: '📡', 
    color: 'default',
    category: 'feature'
  }
}

// 模型能力自动推断规则
export const MODEL_PATTERNS: Record<string, string[]> = {
  // OpenAI - 2025 最新模型
  'gpt-4.5': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'gpt-4o': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'gpt-4o-mini': ['multimodal', 'vision', 'function_call', 'streaming'],
  'gpt-4-turbo': ['multimodal', 'vision', 'function_call', 'streaming'],
  'gpt-4-vision': ['vision', 'function_call', 'streaming'],
  'gpt-4': ['function_call', 'streaming', 'text'],
  'o3': ['reasoning', 'function_call', 'streaming'],
  'o3-mini': ['reasoning', 'function_call', 'streaming'],
  'o1': ['reasoning', 'function_call', 'streaming'],
  'o1-mini': ['reasoning', 'function_call', 'streaming'],
  
  // GPT-3.5 系列
  'gpt-3.5': ['text', 'function_call', 'streaming'],
  
  // Anthropic Claude - 2025 最新模型
  'claude-3-7-sonnet': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'claude-3-5-sonnet': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'claude-3-5-haiku': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'claude-3-opus': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'claude-3-sonnet': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'claude-3-haiku': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'claude-2': ['streaming', 'long_context', 'text'],
  
  // Google Gemini - 2025 最新模型
  'gemini-2.5': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'gemini-2.0': ['multimodal', 'vision', 'function_call', 'streaming'],
  'gemini-1.5-pro': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'gemini-1.5-flash': ['multimodal', 'vision', 'function_call', 'streaming'],
  'gemini-pro-vision': ['multimodal', 'vision', 'streaming'],
  'gemini-pro': ['text', 'streaming'],
  
  // DeepSeek - 2025 最新模型
  'deepseek-v3': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'deepseek-r1': ['reasoning', 'function_call', 'streaming', 'long_context'],
  'deepseek-chat': ['function_call', 'streaming', 'text'],
  'deepseek-coder': ['function_call', 'streaming', 'text'],
  
  // 阿里通义千问 - 2025 最新模型
  'qwen3': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'qwen2.5': ['multimodal', 'vision', 'function_call', 'streaming'],
  'qwen2': ['function_call', 'streaming', 'text'],
  'qwen-max': ['multimodal', 'vision', 'function_call', 'streaming'],
  'qwen-plus': ['function_call', 'streaming', 'text'],
  
  // 月之暗面 Kimi - 2025 最新模型
  'kimi-k2': ['multimodal', 'vision', 'function_call', 'streaming', 'long_context'],
  'kimi-plus': ['function_call', 'streaming', 'long_context'],
  
  // 智谱 AI - 2025 最新模型
  'glm-4': ['multimodal', 'vision', 'function_call', 'streaming'],
  'glm-4-air': ['function_call', 'streaming', 'text'],
  'glm-4-flash': ['function_call', 'streaming', 'text'],
  
  // DALL-E 等图像生成
  'dall-e-3': ['image_generation'],
  'dall-e': ['image_generation'],
  'midjourney': ['image_generation'],
  'stable-diffusion': ['image_generation'],
  'flux': ['image_generation'],
  
  // 视频模型
  'sora': ['video', 'multimodal'],
  'runway': ['video', 'image_generation'],
  'pika': ['video', 'image_generation'],
  'kling': ['video', 'multimodal']  // 可灵
}

/**
 * 根据模型名称自动推断能力标签
 */
export function autoDetectCapabilities(model: string): string[] {
  const modelLower = model.toLowerCase()
  
  // 先匹配预定义模式
  for (const [pattern, caps] of Object.entries(MODEL_PATTERNS)) {
    if (modelLower.includes(pattern)) {
      return caps
    }
  }
  
  // Fallback: 根据关键词推断
  const caps: string[] = []
  if (modelLower.includes('vision')) caps.push('vision')
  if (modelLower.includes('multimodal')) caps.push('multimodal')
  if (modelLower.includes('image') || modelLower.includes('draw') || modelLower.includes('paint')) {
    caps.push('image_generation')
  }
  if (modelLower.includes('video')) caps.push('video')
  if (modelLower.includes('function') || modelLower.includes('tool')) {
    caps.push('function_call')
  }
  if (modelLower.includes('reason') || modelLower.includes('think')) {
    caps.push('reasoning')
  }
  if (modelLower.includes('long') || modelLower.includes('context')) {
    caps.push('long_context')
  }
  if (modelLower.includes('stream')) caps.push('streaming')
  
  // 默认文本模型
  if (caps.length === 0) caps.push('text')
  
  return caps
}

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
  tags: string[]
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
  
  // New fields for priority-based management
  priority: number
  enabled: boolean
  tags: string[]
  
  // Legacy field for backward compatibility
  is_active: boolean
  
  created_at: number
  updated_at: number
}

// 预置厂商
export interface BuiltinProvider {
  id: string
  name: string
  base_url: string
  models: string[]
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
