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
      :title="t('cron.add')"
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
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted, reactive, h } from 'vue'
import {
  NSpace, NH2, NText, NButton, NIcon, NCard, NDataTable,
  NModal, NForm, NFormItem, NInput, NSelect, NSwitch,
  NGrid, NGi, NTimePicker, NPopconfirm, useMessage, NTag
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, FlashOutline, TrashOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getCronJobs, createCronJob, deleteCronJob, triggerCronJob } from '@/api/cron'
import type { CronJob } from '@/types'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const saving = ref(false)
const jobs = ref<CronJob[]>([])
const showModal = ref(false)
const formRef = ref<any>(null)

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

// 中文：定义表格列
// English: Define table columns
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
    render(row) {
      return h(NSpace, null, {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              quaternary: true,
              type: 'primary',
              onClick: () => handleTrigger(row.id)
            },
            { icon: () => h(NIcon, null, { default: () => h(FlashOutline) }) }
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

// 中文：加载任务列表
// English: Load job list
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

function showAddModal() {
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

// 中文：提交表单
// English: Submit form
async function handleSubmit() {
  try {
    await formRef.value?.validate()
    saving.value = true
    await createCronJob(formData)
    message.success(t('common.success'))
    showModal.value = false
    loadJobs()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = false
  }
}

// 中文：立即触发
// English: Trigger now
async function handleTrigger(id: string) {
  try {
    await triggerCronJob(id)
    message.success('已触发执行 / Triggered successfully')
  } catch (error) {
    message.error(t('common.error'))
  }
}

// 中文：删除任务
// English: Delete task
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

.cron-modal {
  border-radius: 12px;
}
</style>
