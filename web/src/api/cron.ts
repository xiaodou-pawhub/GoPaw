import api from './index'
import type { CronJob } from '@/types'

// 中文：获取所有定时任务
// English: Get all cron jobs
export async function getCronJobs(): Promise<CronJob[]> {
  const res: any = await api.get('/cron') // 后端路由通常直接挂在 /api/cron 下
  return res.jobs || []
}

// 中文：创建新定时任务
// English: Create a new cron job
export async function createCronJob(job: Partial<CronJob>) {
  return await api.post('/cron', job)
}

// 中文：删除定时任务
// English: Delete a cron job
export async function deleteCronJob(id: string) {
  return await api.delete(`/cron/${id}`)
}

// 中文：立即触发定时任务执行
// English: Trigger a cron job immediately
export async function triggerCronJob(id: string) {
  return await api.post(`/cron/${id}/trigger`)
}
