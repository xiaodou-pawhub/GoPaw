import api from './index'

// 获取系统日志
export async function getSystemLogs(): Promise<any> {
  return await api.get('/system/logs')
}

// 获取系统版本信息
export async function getSystemVersion(): Promise<any> {
  return await api.get('/system/version')
}
