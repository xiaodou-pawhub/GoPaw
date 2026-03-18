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
  // 编排管理
  list: () => api.get<Orchestration[]>('/orchestrations'),
  
  get: (id: string) => api.get<Orchestration>(`/orchestrations/${id}`),
  
  create: (data: {
    id: string
    name: string
    description?: string
    definition: OrchestrationDefinition
  }) => api.post<Orchestration>('/orchestrations', data),
  
  update: (id: string, data: {
    name?: string
    description?: string
    status?: string
    definition?: OrchestrationDefinition
  }) => api.put(`/orchestrations/${id}`, data),
  
  delete: (id: string) => api.delete(`/orchestrations/${id}`),
  
  validate: (definition: OrchestrationDefinition) => 
    api.post('/orchestrations/validate', definition),
  
  // 执行
  execute: (id: string, data: { input: string; variables?: Record<string, any> }) => 
    api.post<{ execution_id: string; status: string }>(`/orchestrations/${id}/execute`, data),
  
  // 执行记录
  listExecutions: (id: string) => 
    api.get<ExecutionContext[]>(`/orchestrations/${id}/executions`),
  
  getExecution: (executionId: string) => 
    api.get<ExecutionContext>(`/executions/${executionId}`),
  
  getExecutionMessages: (executionId: string) => 
    api.get<ExecutionMessage[]>(`/executions/${executionId}/messages`),
  
  submitHumanInput: (executionId: string, input: string) => 
    api.post(`/executions/${executionId}/human-input`, { input }),
}
