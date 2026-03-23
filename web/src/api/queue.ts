import api from './index'

// 任务状态
export type TaskStatus = 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled' | 'retry'

// 任务优先级
export type TaskPriority = 1 | 5 | 10 | 20  // low, normal, high, urgent

// 任务
export interface Task {
  id: string
  type: string
  priority: TaskPriority
  status: TaskStatus
  payload: Record<string, any>
  result?: Record<string, any>
  error?: string
  retry_count: number
  max_retries: number
  scheduled_at?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
  timeout: number
  worker_id?: string
  metadata?: Record<string, string>
}

// 队列统计
export interface QueueStats {
  pending: number      // 等待中的任务
  delayed: number      // 延迟任务
  workers: number      // Worker 总数
  busy_workers: number // 忙碌的 Worker
}

// 入队请求
export interface EnqueueTaskRequest {
  type: string
  payload?: Record<string, any>
  priority?: number
  delay?: number       // 延迟秒数
  max_retries?: number
  timeout?: number
}

// 任务类型
export const TaskTypes = {
  FLOW_EXECUTE: 'flow_execute',
  WEBHOOK_CALLBACK: 'webhook_callback',
  SUBFLOW_EXECUTE: 'subflow_execute',
  RETRY_EXECUTION: 'retry_execution',
} as const

// 优先级标签
export const PriorityLabels: Record<number, { label: string; color: string }> = {
  1: { label: '低', color: 'neutral' },
  5: { label: '普通', color: 'info' },
  10: { label: '高', color: 'warning' },
  20: { label: '紧急', color: 'error' },
}

// 状态标签
export const StatusLabels: Record<TaskStatus, { label: string; color: string }> = {
  pending: { label: '等待中', color: 'neutral' },
  queued: { label: '已入队', color: 'info' },
  running: { label: '执行中', color: 'primary' },
  completed: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  cancelled: { label: '已取消', color: 'neutral' },
  retry: { label: '重试中', color: 'warning' },
}

export const taskQueueApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // 获取队列统计
  getStats: async () => {
    const res = await api.get('/flows/queue/stats')
    return taskQueueApi.parseData<QueueStats>(res)
  },

  // 列出任务
  listTasks: async (status?: TaskStatus, limit?: number) => {
    const res = await api.get('/flows/queue/tasks', { params: { status, limit } })
    return taskQueueApi.parseData<Task[]>(res)
  },

  // 获取任务
  getTask: async (taskId: string) => {
    const res = await api.get(`/flows/queue/tasks/${taskId}`)
    return taskQueueApi.parseData<Task>(res)
  },

  // 入队任务
  enqueue: async (data: EnqueueTaskRequest) => {
    const res = await api.post('/flows/queue/tasks', data)
    return taskQueueApi.parseData<Task>(res)
  },

  // 取消任务
  cancel: async (taskId: string) => {
    const res = await api.delete(`/flows/queue/tasks/${taskId}`)
    return taskQueueApi.parseData<{ status: string; message: string }>(res)
  },
}

// 兼容旧 API 名称
export const queueApi = taskQueueApi

// 兼容旧类型
export type Message = Task
export type MessageStatus = TaskStatus