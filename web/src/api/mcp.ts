import api from './index'

// ---- Types ----

export interface MCPServer {
  id: string
  name: string
  description: string
  command: string
  args: string[]
  env: string[]
  transport: 'stdio' | 'sse'
  url?: string
  is_active: boolean
  is_builtin: boolean
  created_at: number
  updated_at: number
}

export interface MCPTool {
  name: string
  description: string
  inputSchema: Record<string, any>
}

export interface CreateMCPServerRequest {
  id: string
  name: string
  description?: string
  command: string
  args?: string[]
  env?: string[]
  transport?: 'stdio' | 'sse'
  url?: string
}

export interface UpdateMCPServerRequest {
  name?: string
  description?: string
  command?: string
  args?: string[]
  env?: string[]
  transport?: 'stdio' | 'sse'
  url?: string
}

// ---- API Functions ----

// 解析标准响应格式
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

export async function listMCPServers(): Promise<{ servers: MCPServer[] }> {
  const res = await api.get('/mcp/servers')
  const servers = parseData<MCPServer[]>(res)
  return { servers }
}

export async function getMCPServer(id: string): Promise<MCPServer> {
  const res = await api.get(`/mcp/servers/${encodeURIComponent(id)}`)
  return parseData<MCPServer>(res)
}

export async function createMCPServer(data: CreateMCPServerRequest): Promise<MCPServer> {
  const res = await api.post('/mcp/servers', data)
  return parseData<MCPServer>(res)
}

export async function updateMCPServer(id: string, data: UpdateMCPServerRequest): Promise<MCPServer> {
  const res = await api.put(`/mcp/servers/${encodeURIComponent(id)}`, data)
  return parseData<MCPServer>(res)
}

export async function deleteMCPServer(id: string): Promise<{ deleted: string }> {
  const res = await api.delete(`/mcp/servers/${encodeURIComponent(id)}`)
  return parseData<{ deleted: string }>(res)
}

export async function setMCPServerActive(id: string, active: boolean): Promise<{ id: string; active: boolean }> {
  const res = await api.post(`/mcp/servers/${encodeURIComponent(id)}/active`, { active })
  return parseData<{ id: string; active: boolean }>(res)
}

export async function getMCPServerTools(id: string): Promise<{ tools: MCPTool[] }> {
  const res = await api.get(`/mcp/servers/${encodeURIComponent(id)}/tools`)
  const tools = parseData<MCPTool[]>(res)
  return { tools }
}

export async function getAllMCPTools(): Promise<{ tools: MCPTool[] }> {
  const res = await api.get('/mcp/tools')
  const tools = parseData<MCPTool[]>(res)
  return { tools }
}

// ---- Helpers ----

export function getTransportLabel(transport: string): string {
  const labels: Record<string, string> = {
    'stdio': 'Stdio (标准输入输出)',
    'sse': 'SSE (服务器推送)'
  }
  return labels[transport] || transport
}

export function getBuiltinServers(): CreateMCPServerRequest[] {
  return [
    {
      id: 'filesystem',
      name: '文件系统',
      description: '访问本地文件系统',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-filesystem'],
      transport: 'stdio'
    },
    {
      id: 'github',
      name: 'GitHub',
      description: '访问 GitHub API',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-github'],
      transport: 'stdio'
    },
    {
      id: 'postgres',
      name: 'PostgreSQL',
      description: '访问 PostgreSQL 数据库',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-postgres'],
      transport: 'stdio'
    },
    {
      id: 'sqlite',
      name: 'SQLite',
      description: '访问 SQLite 数据库',
      command: 'npx',
      args: ['-y', '@modelcontextprotocol/server-sqlite'],
      transport: 'stdio'
    }
  ]
}
