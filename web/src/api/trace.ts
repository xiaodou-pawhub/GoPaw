import api from './index'

// ---- Types ----

export interface Trace {
  id: string
  session_id: string
  started_at: number
  ended_at?: number
  status: 'running' | 'completed' | 'error'
  error_message?: string
  duration_ms: number
  step_count: number
}

export interface TraceStep {
  id: number
  step_number: number
  step_type: string
  started_at: number
  ended_at?: number
  duration_ms: number
  input?: unknown
  output?: unknown
  metadata?: unknown
}

export interface TraceDetail extends Trace {
  steps: TraceStep[]
}

export interface TraceStats {
  total_traces: number
  completed: number
  errors: number
  avg_duration_ms: number
  step_types: Record<string, number>
}

// ---- API Functions ----

// 解析标准响应格式
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

export async function listTraces(params?: {
  session_id?: string
  status?: string
  limit?: number
}): Promise<{ traces: Trace[]; total: number }> {
  const p: Record<string, string> = {}
  if (params?.session_id) p.session_id = params.session_id
  if (params?.status) p.status = params.status
  if (params?.limit) p.limit = String(params.limit)
  const res = await api.get('/traces', { params: p })
  return parseData<{ traces: Trace[]; total: number }>(res)
}

export async function getTrace(id: string): Promise<TraceDetail> {
  const res = await api.get(`/traces/${encodeURIComponent(id)}`)
  return parseData<TraceDetail>(res)
}

export async function getTraceStats(): Promise<TraceStats> {
  const res = await api.get('/traces/stats')
  return parseData<TraceStats>(res)
}

// ---- Helpers ----

export function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`
  }
  const minutes = Math.floor(ms / 60000)
  const seconds = Math.floor((ms % 60000) / 1000)
  return `${minutes}m${seconds}s`
}

export function formatTimestamp(ts: number): string {
  const date = new Date(ts)
  return date.toLocaleString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

export function getStepTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    'context_build': '上下文构建',
    'llm_call': 'LLM 调用',
    'tool_execution': '工具执行',
    'hook_execution': '钩子执行',
    'final_answer': '最终回答'
  }
  return labels[type] || type
}

export function getStepTypeColor(type: string): string {
  const colors: Record<string, string> = {
    'context_build': 'var(--blue)',
    'llm_call': 'var(--purple)',
    'tool_execution': 'var(--green)',
    'hook_execution': 'var(--orange)',
    'final_answer': 'var(--cyan)'
  }
  return colors[type] || 'var(--text-secondary)'
}

export function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    'running': '运行中',
    'completed': '已完成',
    'error': '错误'
  }
  return labels[status] || status
}

export function getStatusColor(status: string): string {
  const colors: Record<string, string> = {
    'running': 'var(--yellow)',
    'completed': 'var(--green)',
    'error': 'var(--red)'
  }
  return colors[status] || 'var(--text-secondary)'
}
