<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('cron.title') }}</h1>
        <p class="page-description">自动化执行 Agent 任务，支持定时推送至不同频道</p>
      </div>
      <n-button type="primary" @click="openModal('create')">
        <template #icon>
          <n-icon :component="AddOutline" />
        </template>
        {{ t('cron.add') }}
      </n-button>
    </div>

    <n-card bordered class="page-card" content-style="padding: 0;">
      <n-data-table
        :columns="columns"
        :data="jobs"
        :loading="loading"
        :bordered="false"
        class="cron-table"
      />
    </n-card>

    <!-- 编辑/创建弹窗 -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="modalType === 'create' ? t('cron.add') : t('cron.edit')"
      class="cron-modal"
      style="width: 600px"
    >
      <n-form
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="100"
      >
        <n-form-item :label="t('cron.name')" path="name">
          <n-input v-model:value="formModel.name" placeholder="请输入任务名称" />
        </n-form-item>
        
        <n-form-item :label="t('cron.expr')" path="cron_expr">
          <n-input v-model:value="formModel.cron_expr" placeholder="例如: 0 9 * * 1-5" />
          <template #feedback>
            {{ t('cron.helper.expr') }}
          </template>
        </n-form-item>

        <n-form-item :label="t('cron.channel')" path="channel">
          <n-select
            v-model:value="formModel.channel"
            :options="channelOptions"
            placeholder="请选择发送频道"
          />
        </n-form-item>

        <n-form-item :label="t('cron.prompt')" path="prompt">
          <n-input
            v-model:value="formModel.prompt"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 6 }"
            placeholder="触发时发给 Agent 的内容"
          />
        </n-form-item>

        <n-grid :cols="2" :x-gap="12">
          <n-gi>
            <n-form-item :label="t('cron.status')">
              <n-switch v-model:value="formModel.enabled" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <div class="active-window-title">{{ t('cron.window') }}</div>
        <n-grid :cols="2" :x-gap="12">
          <n-gi>
            <n-form-item :label="t('cron.windowStart')">
              <n-time-picker v-model:formatted-value="formModel.active_from" value-format="HH:mm" format="HH:mm" clearable />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('cron.windowEnd')">
              <n-time-picker v-model:formatted-value="formModel.active_until" value-format="HH:mm" format="HH:mm" clearable />
            </n-form-item>
          </n-gi>
        </n-grid>
      </n-form>

      <template #footer>
        <div class="modal-footer">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">
            {{ t('common.confirm') }}
          </n-button>
        </div>
      </template>
    </n-modal>

    <!-- 执行历史侧边抽屉 -->
    <n-drawer v-model:show="showHistory" :width="500" placement="right">
      <n-drawer-content :title="`${t('cron.history')} - ${currentJob?.name}`" closable>
        <div v-if="historyLoading" class="history-loading">
          <n-spin size="large" />
        </div>
        <div v-else-if="runHistory.length === 0" class="history-empty">
          <n-empty :description="t('cron.historyEmpty')" />
        </div>
        <div v-else class="history-list">
          <n-card
            v-for="run in runHistory"
            :key="run.id"
            size="small"
            class="history-item"
            :segmented="{ content: true }"
          >
            <template #header>
              <div class="history-header">
                <n-tag :type="getStatusType(run.status)" size="small">
                  {{ run.status.toUpperCase() }}
                </n-tag>
                <span class="history-time">{{ formatTime(run.triggered_at) }}</span>
              </div>
            </template>
            
            <div class="history-content">
              <div v-if="run.output" class="history-output">
                <div class="label">输出:</div>
                <div class="text">{{ run.output }}</div>
              </div>
              <div v-if="run.error_msg" class="history-error">
                <div class="label">错误:</div>
                <div class="text">{{ run.error_msg }}</div>
              </div>
            </div>
          </n-card>
        </div>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, reactive } from 'vue'
import {
  NSpace, NButton, NIcon, NCard, NDataTable, NModal,
  NForm, NFormItem, NInput, NSelect, NSwitch, NGrid, NGi, NTimePicker,
  NDrawer, NDrawerContent, NTag, NEmpty, NSpin, useMessage, useDialog
} from 'naive-ui'
import type { DataTableColumns, FormInst } from 'naive-ui'
import {
  AddOutline,
  PlayOutline,
  TrashOutline,
  CreateOutline,
  TimeOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import {
  getCronJobs, createCronJob, updateCronJob,
  deleteCronJob, triggerCronJob, getCronRunHistory
} from '@/api/cron'
import type { CronJob, CronRun } from '@/types'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const jobs = ref<CronJob[]>([])
const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)
const formRef = ref<FormInst | null>(null)

const showHistory = ref(false)
const currentJob = ref<CronJob | null>(null)
const runHistory = ref<CronRun[]>([])
const historyLoading = ref(false)

const formModel = reactive<Partial<CronJob>>({
  name: '',
  description: '',
  cron_expr: '',
  channel: 'console',
  prompt: '',
  enabled: true,
  active_from: null,
  active_until: null
})

const rules = {
  name: { required: true, message: '请输入任务名称', trigger: 'blur' },
  cron_expr: { required: true, message: '请输入 Cron 表达式', trigger: 'blur' },
  channel: { required: true, message: '请选择发送频道', trigger: 'change' },
  prompt: { required: true, message: '请输入触发提示词', trigger: 'blur' }
}

const channelOptions = [
  { label: '控制台 (Console)', value: 'console' },
  { label: '飞书 (Feishu)', value: 'feishu' },
  { label: '钉钉 (DingTalk)', value: 'dingtalk' },
  { label: 'Webhook', value: 'webhook' }
]

const columns: DataTableColumns<CronJob> = [
  { title: t('cron.name'), key: 'name', width: 150 },
  {
    title: t('cron.expr'),
    key: 'cron_expr',
    render(row) {
      return h(NTag, { size: 'small', type: 'info', ghost: true }, { default: () => row.cron_expr })
    }
  },
  { title: t('cron.channel'), key: 'channel', width: 100 },
  {
    title: t('cron.status'),
    key: 'enabled',
    width: 80,
    render(row) {
      return h(NTag, { size: 'small', type: row.enabled ? 'success' : 'default' }, {
        default: () => row.enabled ? '启用' : '禁用'
      })
    }
  },
  {
    title: t('cron.action'),
    key: 'actions',
    width: 240,
    render(row) {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, {
            size: 'small',
            quaternary: true,
            circle: true,
            onClick: () => handleTrigger(row)
          }, { icon: () => h(NIcon, null, { default: () => h(PlayOutline) }) }),
          h(NButton, {
            size: 'small',
            quaternary: true,
            circle: true,
            onClick: () => openHistory(row)
          }, { icon: () => h(NIcon, null, { default: () => h(TimeOutline) }) }),
          h(NButton, {
            size: 'small',
            quaternary: true,
            circle: true,
            onClick: () => openModal('edit', row)
          }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }),
          h(NButton, {
            size: 'small',
            quaternary: true,
            circle: true,
            type: 'error',
            onClick: () => handleDelete(row)
          }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) })
        ]
      })
    }
  }
]

async function loadJobs() {
  loading.value = true
  try {
    jobs.value = await getCronJobs()
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function openModal(type: 'create' | 'edit', job?: CronJob) {
  modalType.value = type
  if (type === 'edit' && job) {
    editingId.value = job.id
    Object.assign(formModel, { 
      ...job,
      active_from: job.active_from || null,
      active_until: job.active_until || null
    })
  } else {
    editingId.value = null
    Object.assign(formModel, {
      name: '',
      description: '',
      cron_expr: '',
      channel: 'console',
      prompt: '',
      enabled: true,
      active_from: null,
      active_until: null
    })
  }
  showModal.value = true
}

async function handleSubmit() {
  // 闭环 P1：执行表单显式校验
  try {
    await formRef.value?.validate()
  } catch (err) {
    return // 校验失败中断
  }

  submitting.value = true
  try {
    const payload = { ...formModel }
    // 闭环 P1：若用户未显式选择时间，则不发送该字段，保持可选行为
    if (!payload.active_from) payload.active_from = ''
    if (!payload.active_until) payload.active_until = ''

    if (modalType.value === 'create') {
      await createCronJob(payload)
    } else if (editingId.value) {
      await updateCronJob(editingId.value, payload)
    }
    message.success(t('common.success'))
    showModal.value = false
    loadJobs()
  } catch (err: any) {
    // 闭环 P1：透传后端错误信息，提升排障效率
    const errorMsg = err.response?.data?.error || err.message || t('common.error')
    message.error(errorMsg)
  } finally {
    submitting.value = false
  }
}

function handleTrigger(job: CronJob) {
  dialog.info({
    title: t('cron.trigger'),
    content: t('cron.triggerConfirm', { name: job.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await triggerCronJob(job.id)
        message.success('已触发执行请求')
      } catch (err) {
        message.error('触发失败')
      }
    }
  })
}

function handleDelete(job: CronJob) {
  dialog.warning({
    title: t('common.delete'),
    content: t('cron.deleteConfirm', { name: job.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await deleteCronJob(job.id)
        message.success('已删除')
        loadJobs()
      } catch (err) {
        message.error('删除失败')
      }
    }
  })
}

async function openHistory(job: CronJob) {
  currentJob.value = job
  showHistory.value = true
  historyLoading.value = true
  try {
    runHistory.value = await getCronRunHistory(job.id)
  } catch (err) {
    message.error('加载执行历史失败')
  } finally {
    historyLoading.value = false
  }
}

function getStatusType(status: string) {
  switch (status) {
    case 'success': return 'success'
    case 'error': return 'error'
    case 'running': return 'info'
    default: return 'default'
  }
}

function formatTime(ts: number) {
  return new Date(ts * 1000).toLocaleString()
}

onMounted(loadJobs)
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;
@use '@/styles/page-layout';

.cron-table {
  :deep(.n-data-table-td) {
    padding: 12px 16px;
  }
}

.cron-modal {
  border-radius: 12px;
}

.active-window-title {
  font-size: 14px;
  font-weight: 600;
  margin: 16px 0 8px;
  color: #333;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.history-loading, .history-empty {
  height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.history-item {
  border-radius: 8px;
}

.history-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.history-time {
  font-size: 12px;
  color: #999;
}

.history-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.history-output, .history-error {
  .label {
    font-size: 12px;
    font-weight: 600;
    margin-bottom: 4px;
    color: #666;
  }
  .text {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    background: #f9fafb;
    padding: 8px;
    border-radius: 4px;
    white-space: pre-wrap;
    word-break: break-all;
  }
}

.history-error .text {
  background: rgba(240, 68, 68, 0.05);
  color: #f04444;
}
</style>
