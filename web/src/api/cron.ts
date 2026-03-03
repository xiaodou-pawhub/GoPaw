import api from './index'
import type { CronJob, CronRun } from '@/types'

// 获取所有定时任务
export async function getCronJobs(): Promise<CronJob[]> {
	const res = await api.get<{ jobs: CronJob[] }>('/cron')
	// @ts-ignore - 响应拦截器返回response.data，所以res直接就是响应体
	return res.jobs || []
}

// 创建定时任务
export async function createCronJob(data: Partial<CronJob>): Promise<{ id: string }> {
	// @ts-ignore - 响应拦截器返回response.data
	return await api.post<{ id: string }>('/cron', data)
}

// 更新定时任务
export async function updateCronJob(id: string, data: Partial<CronJob>): Promise<{ ok: boolean }> {
	// @ts-ignore - 响应拦截器返回response.data
	return await api.put<{ ok: boolean }>(`/cron/${id}`, data)
}

// 删除定时任务
export async function deleteCronJob(id: string): Promise<{ ok: boolean }> {
	// @ts-ignore - 响应拦截器返回response.data
	return await api.delete<{ ok: boolean }>(`/cron/${id}`)
}

// 立即触发任务
export async function triggerCronJob(id: string): Promise<{ ok: boolean }> {
	// @ts-ignore - 响应拦截器返回response.data
	return await api.post<{ ok: boolean }>(`/cron/${id}/trigger`)
}

// 获取任务执行历史
export async function getCronRunHistory(id: string, limit: number = 20): Promise<CronRun[]> {
	const res = await api.get<{ runs: CronRun[] }>(`/cron/${id}/runs`, {
		params: { limit }
	})
	// @ts-ignore - 响应拦截器返回response.data
	return res.runs || []
}
