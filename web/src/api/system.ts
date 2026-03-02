import api from './index'

// 中文：获取系统日志
// English: Get system logs
export async function getSystemLogs(): Promise<any> {
  return await api.get('/system/logs')
}

// 中文：获取系统版本信息
// English: Get system version info
export async function getSystemVersion(): Promise<any> {
  return await api.get('/system/version')
}
