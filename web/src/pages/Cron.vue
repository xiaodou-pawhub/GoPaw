<template>
  <div class="cron-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <div class="header-left">
          <n-h2>{{ t('cron.title') }}</n-h2>
          <n-text depth="3">自动化定时触发 Agent 执行预设任务 / Automate Agent execution with scheduled tasks</n-text>
        </div>
        <n-button type="primary" @click="showAddModal">
          <template #icon>
            <n-icon :component="AddOutline" />
          </template>
          {{ t('cron.add') }}
        </n-button>
      </div>

      <n-card bordered class="list-card">
        <n-data-table
          :columns="columns"
          :data="jobs"
          :loading="loading"
          :bordered="false"
          remote
        />
      </n-card>
    </n-space>

    <!-- 中文：新增/编辑任务对话框 / English: Add/Edit job modal -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="isEdit ? t('cron.edit') : t('cron.add')"
      class="cron-modal"
      style="width: 600px"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item :label="t('cron.name')" path="name">
          <n-input v-model:value="formData.name" placeholder="例如：每日早报" />
        </n-form-item>
        
        <n-form-item :label="t('cron.expr')" path="cron_expr">
          <n-input v-model:value="formData.cron_expr" :placeholder="t('cron.helper.expr')" />
        </n-form-item>

        <n-form-item :label="t('cron.channel')" path="channel">
          <n-select
            v-model:value="formData.channel"
            :options="channelOptions"
          />
        </n-form-item>

        <n-form-item :label="t('cron.prompt')" path="prompt">
          <n-input
            v-model:value="formData.prompt"
            type="textarea"
            :placeholder="t('chat.placeholder')"
            :autosize="{ minRows: 3 }"
          />
        </n-form-item>

        <n-grid :cols="2" :x-gap="12">
          <n-gi>
            <n-form-item label="Active From">
              <n-time-picker v-model:formatted-value="formData.active_from" value-format="HH:mm" format="HH:mm" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="Active Until">
              <n-time-picker v-model:formatted-value="formData.active_until" value-format="HH:mm" format="HH:mm" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('cron.status')">
          <n-switch v-model:value="formData.enabled" />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="handleSubmit">
            {{ t('common.save') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 中文：执行历史对话框 / English: Execution history modal -->
    <n-modal
      v-model:show="showHistoryModal"
      preset="card"
      :title="t('cron.history')"
      class="history-modal"
      style="width: 800px"
    >
      <n-data-table
        :columns="historyColumns"
        :data="historyRuns"
        :loading="loadingHistory"
        :bordered="false"
        size="small"
      />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted, reactive, h } from 'vue'
import {
  NSpace, NH2, NText, NButton, NIcon, NCard, NDataTable,
  NModal, NForm, NFormItem, NInput, NSelect, NSwitch,
  NGrid, NGi, NTimePicker, NPopconfirm, useMessage, NTag, NTooltip
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, FlashOutline, TrashOutline, CreateOutline, TimeOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getCronJobs, createCronJob, updateCronJob, deleteCronJob, triggerCronJob, getCronRuns, type CronRun } from '@/api/cron'
import type { CronJob } from '@/types'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const saving = ref(false)
const jobs = ref<CronJob[]>([])
const showModal = ref(false)
const isEdit = ref(false)
const editingJobId = ref<string | null>(null)
const formRef = ref<any>(null)

// 中文：执行历史状态 / English: Execution history state
const showHistoryModal = ref(false)
const loadingHistory = ref(false)
const historyRuns = ref<CronRun[]>([])

const formData = reactive({
  name: '',
  cron_expr: '0 9 * * *',
  channel: 'console',
  prompt: '',
  enabled: true,
  active_from: '00:00',
  active_until: '23:59'
})

const rules = {
  name: { required: true, message: '请输入任务名称', trigger: 'blur' },
  cron_expr: { required: true, message: '请输入 Cron 表达式', trigger: 'blur' },
  prompt: { required: true, message: '请输入触发词', trigger: 'blur' }
}

const channelOptions = [
  { label: 'Web Console', value: 'console' },
  { label: '飞书 / Feishu', value: 'feishu' },
  { label: '钉钉 / DingTalk', value: 'dingtalk' },
  { label: 'Webhook', value: 'webhook' }
]

// 中文：格式化时间戳 / English: Format timestamp
function formatTimestamp(ts: number | null): string {
  if (!ts) return '-'
  const date = new Date(ts * 1000)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 中文：计算耗时 / English: Calculate duration
function calculateDuration(run: CronRun): string {
  if (!run.finished_at) return t('cron.running')
  const seconds = run.finished_at - run.triggered_at
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${minutes}m ${secs}s`
}

// 中文：截断输出 / English: Truncate output
function truncateOutput(output: string, maxLen: number = 100): string {
  if (!output) return '-'
  return output.length > maxLen ? output.slice(0, maxLen) + '...' : output
}

// 中文：执行历史表格列 / English: Execution history table columns
const historyColumns: DataTableColumns<CronRun> = [
  {
    title: t('cron.triggeredAt'),
    key: 'triggered_at',
    width: 120,
    render(row) {
      return formatTimestamp(row.triggered_at)
    }
  },
  {
    title: t('cron.status'),
    key: 'status',
    width: 100,
    render(row) {
      const statusMap: Record<string, { type: 'success' | 'error' | 'info', label: string }> = {
        success: { type: 'success', label: '✅ ' + t('cron.success') },
        error: { type: 'error', label: '❌ ' + t('cron.failed') },
        running: { type: 'info', label: '⏳ ' + t('cron.running') }
      }
      const s = statusMap[row.status] || { type: 'info', label: row.status }
      return h(NTag, { type: s.type, size: 'small', round: true }, { default: () => s.label })
    }
  },
  {
    title: t('cron.duration'),
    key: 'duration',
    width: 80,
    render(row) {
      return calculateDuration(row)
    }
  },
  {
    title: t('cron.output'),
    key: 'output',
    render(row) {
      const output = row.status === 'error' ? row.error_msg : row.output
      const truncated = truncateOutput(output)
      if (truncated === '-') return truncated
      return h(NTooltip, { trigger: 'hover' }, {
        trigger: () => h('span', { class: 'output-cell' }, truncated),
        default: () => output
      })
    }
  }
]

// 中文：定义表格列 / English: Define table columns
const columns: DataTableColumns<CronJob> = [
  { title: t('cron.name'), key: 'name' },
  { title: t('cron.expr'), key: 'cron_expr' },
  {
    title: t('cron.channel'),
    key: 'channel',
    render(row) {
      return h(NTag, { type: 'info', size: 'small', round: true }, { default: () => row.channel })
    }
  },
  {
    title: t('cron.status'),
    key: 'enabled',
    render(row) {
      return h(NTag, { type: row.enabled ? 'success' : 'default', size: 'small' }, { default: () => row.enabled ? 'Enabled' : 'Disabled' })
    }
  },
  {
    title: t('cron.action'),
    key: 'actions',
    width: 180,
    render(row) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(
            NTooltip,
            { trigger: 'hover' },
            {
              trigger: () => h(
                NButton,
                {
                  size: 'small',
                  quaternary: true,
                  type: 'info',
                  onClick: () => showJobHistory(row.id)
                },
                { icon: () => h(NIcon, null, { default: () => h(TimeOutline) }) }
              ),
              default: () => t('cron.history')
            }
          ),
          h(
            NTooltip,
            { trigger: 'hover' },
            {
              trigger: () => h(
                NButton,
                {
                  size: 'small',
                  quaternary: true,
                  type: 'primary',
                  onClick: () => handleTrigger(row.id)
                },
                { icon: () => h(NIcon, null, { default: () => h(FlashOutline) }) }
              ),
              default: () => t('cron.trigger')
            }
          ),
          h(
            NTooltip,
            { trigger: 'hover' },
            {
              trigger: () => h(
                NButton,
                {
                  size: 'small',
                  quaternary: true,
                  type: 'warning',
                  onClick: () => showEditModal(row)
                },
                { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }
              ),
              default: () => t('common.edit')
            }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDelete(row.id)
            },
            {
              trigger: () => h(
                NButton,
                { size: 'small', quaternary: true, type: 'error' },
                { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }
              ),
              default: () => '确认删除该任务吗？ / Delete this task?'
            }
          )
        ]
      })
    }
  }
]

// 中文：加载任务列表 / English: Load job list
async function loadJobs() {
  loading.value = true
  try {
    jobs.value = await getCronJobs()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    loading.value = false
  }
}

// 中文：显示添加对话框 / English: Show add modal
function showAddModal() {
  isEdit.value = false
  editingJobId.value = null
  Object.assign(formData, {
    name: '',
    cron_expr: '0 9 * * *',
    channel: 'console',
    prompt: '',
    enabled: true,
    active_from: '00:00',
    active_until: '23:59'
  })
  showModal.value = true
}

// 中文：显示编辑对话框 / English: Show edit modal
function showEditModal(job: CronJob) {
  isEdit.value = true
  editingJobId.value = job.id
  Object.assign(formData, {
    name: job.name,
    cron_expr: job.cron_expr,
    channel: job.channel,
    prompt: job.prompt,
    enabled: job.enabled,
    active_from: job.active_from || '00:00',
    active_until: job.active_until || '23:59'
  })
  showModal.value = true
}

// 中文：显示执行历史 / English: Show execution history
async function showJobHistory(jobId: string) {
  showHistoryModal.value = true
  loadingHistory.value = true
  try {
    historyRuns.value = await getCronRuns(jobId, 20)
  } catch (error) {
    message.error(t('common.error'))
    historyRuns.value = []
  } finally {
    loadingHistory.value = false
  }
}

// 中文：提交表单 / English: Submit form
async function handleSubmit() {
  try {
    await formRef.value?.validate()
    saving.value = true
    
    if (isEdit.value && editingJobId.value) {
      await updateCronJob(editingJobId.value, formData)
    } else {
      await createCronJob(formData)
    }
    
    message.success(t('common.success'))
    showModal.value = false
    loadJobs()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = false
  }
}

// 中文：立即触发 / English: Trigger now
async function handleTrigger(id: string) {
  try {
    await triggerCronJob(id)
    message.success('已触发执行 / Triggered successfully')
  } catch (error) {
    message.error(t('common.error'))
  }
}

// 中文：删除任务 / English: Delete task
async function handleDelete(id: string) {
  try {
    await deleteCronJob(id)
    message.success(t('common.success'))
    loadJobs()
  } catch (error) {
    message.error(t('common.error'))
  }
}

onMounted(loadJobs)
</script>

<style scoped lang="scss">
.cron-page {
  padding: 12px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 8px;
}

.list-card {
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
}

.cron-modal,
.history-modal {
  border-radius: 12px;
}

.output-cell {
  cursor: pointer;
  color: #666;
}
</style>
