import api from './index'

export interface Orchestration {
  id: string
  name: string
  description: string
  status: string
  definition: OrchestrationDefinition
  created_at: string
  updated_at: string
}

export interface OrchestrationDefinition {
  nodes: OrchestrationNode[]
  edges: OrchestrationEdge[]
  variables: Record<string, any>
  start_node_id: string
}

export interface OrchestrationNode {
  id: string
  type: 'agent' | 'human' | 'condition' | 'workflow' | 'end'
  agent_id?: string
  name: string
  role?: string
  prompt?: string
  config?: Record<string, any>
  position: { x: number; y: number }
}

export interface OrchestrationEdge {
  id: string
  source: string
  target: string
  message_type: string
  condition?: EdgeCondition
  transform?: MessageTransform
  label?: string
}

export interface EdgeCondition {
  type: 'expression' | 'intent' | 'llm'
  expression?: string
  intent?: string
  llm_query?: string
}

export interface MessageTransform {
  template: string
}

export interface ExecutionContext {
  id: string
  orchestration: Orchestration
  status: string
  current_node_id: string
  variables: Record<string, any>
  messages: ExecutionMessage[]
  start_time: string
  update_time: string
}

export interface ExecutionMessage {
  id: string
  execution_id: string
  from_node_id: string
  to_node_id: string
  message_type: string
  content: string
  created_at: string
}

export const orchestrationApi = {
  // 解析标准响应格式
  parseData<T>(res: any): T {
    if (res && res.data !== undefined) {
      return res.data as T
    }
    return res as T
  },

  // 编排管理
  list: async () => {
    const res = await api.get('/orchestrations')
    return orchestrationApi.parseData<Orchestration[]>(res)
  },

  get: async (id: string) => {
    const res = await api.get(`/orchestrations/${id}`)
    return orchestrationApi.parseData<Orchestration>(res)
  },

  create: async (data: {
    id: string
    name: string
    description?: string
    definition: OrchestrationDefinition
  }) => {
    const res = await api.post('/orchestrations', data)
    return orchestrationApi.parseData<Orchestration>(res)
  },

  update: async (id: string, data: {
    name?: string
    description?: string
    status?: string
    definition?: OrchestrationDefinition
  }) => {
    const res = await api.put(`/orchestrations/${id}`, data)
    return orchestrationApi.parseData<any>(res)
  },

  delete: async (id: string) => {
    const res = await api.delete(`/orchestrations/${id}`)
    return orchestrationApi.parseData<any>(res)
  },

  validate: async (definition: OrchestrationDefinition) => {
    const res = await api.post('/orchestrations/validate', definition)
    return orchestrationApi.parseData<any>(res)
  },

  // 执行
  execute: async (id: string, data: { input: string; variables?: Record<string, any> }) => {
    const res = await api.post(`/orchestrations/${id}/execute`, data)
    return orchestrationApi.parseData<{ execution_id: string; status: string }>(res)
  },

  // 执行记录
  listExecutions: async (id: string) => {
    const res = await api.get(`/orchestrations/${id}/executions`)
    return orchestrationApi.parseData<ExecutionContext[]>(res)
  },

  getExecution: async (executionId: string) => {
    const res = await api.get(`/executions/${executionId}`)
    return orchestrationApi.parseData<ExecutionContext>(res)
  },

  getExecutionMessages: async (executionId: string) => {
    const res = await api.get(`/executions/${executionId}/messages`)
    return orchestrationApi.parseData<ExecutionMessage[]>(res)
  },

  submitHumanInput: async (executionId: string, input: string) => {
    const res = await api.post(`/executions/${executionId}/human-input`, { input })
    return orchestrationApi.parseData<any>(res)
  },
}
