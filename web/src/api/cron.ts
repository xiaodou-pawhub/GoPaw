import api from './index'
import type { CronJob, CronRun } from '@/types'

/**
 * 解析标准响应格式 { code, message, data }
 */
function parseData<T>(res: any): T {
  if (res && res.data !== undefined) {
    return res.data as T
  }
  return res as T
}

// 获取所有定时任务
export async function getCronJobs(): Promise<CronJob[]> {
	const res = await api.get('/cron')
	return parseData<CronJob[]>(res)
}

// 创建定时任务
export async function createCronJob(data: Partial<CronJob>): Promise<{ id: string }> {
	const res = await api.post('/cron', data)
	return parseData(res)
}

// 更新定时任务
export async function updateCronJob(id: string, data: Partial<CronJob>): Promise<{ ok: boolean }> {
	const res = await api.put(`/cron/${id}`, data)
	return parseData(res)
}

// 删除定时任务
export async function deleteCronJob(id: string): Promise<{ ok: boolean }> {
	const res = await api.delete(`/cron/${id}`)
	return parseData(res)
}

// 立即触发任务
export async function triggerCronJob(id: string): Promise<{ ok: boolean }> {
	const res = await api.post(`/cron/${id}/trigger`)
	return parseData(res)
}

// 获取任务执行历史
export async function getCronRunHistory(id: string, limit: number = 20): Promise<CronRun[]> {
	const res = await api.get(`/cron/${id}/runs`, {
		params: { limit }
	})
	return parseData<CronRun[]>(res)
}
