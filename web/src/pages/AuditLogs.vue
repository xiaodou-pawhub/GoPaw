<template>
  <div class="audit-logs-page">
    <div class="page-header">
      <h1 class="text-h5">审计日志</h1>
      <div class="header-actions">
        <v-btn color="primary" prepend-icon="mdi-download" @click="showExportDialog">
          导出
        </v-btn>
        <v-btn color="error" prepend-icon="mdi-delete" @click="showCleanupDialog">
          清理
        </v-btn>
      </div>
    </div>

    <v-row class="mt-4">
      <!-- 筛选面板 -->
      <v-col cols="3">
        <v-card>
          <v-card-title>筛选</v-card-title>
          <v-card-text>
            <v-select
              v-model="filters.category"
              :items="categories"
              label="分类"
              clearable
              density="compact"
            />
            <v-select
              v-model="filters.action"
              :items="actions"
              label="动作"
              clearable
              density="compact"
              class="mt-2"
            />
            <v-select
              v-model="filters.status"
              :items="statuses"
              label="状态"
              clearable
              density="compact"
              class="mt-2"
            />
            <v-text-field
              v-model="filters.user_id"
              label="用户 ID"
              clearable
              density="compact"
              class="mt-2"
            />
            <v-text-field
              v-model="filters.resource_type"
              label="资源类型"
              clearable
              density="compact"
              class="mt-2"
            />
            <v-text-field
              v-model="filters.resource_id"
              label="资源 ID"
              clearable
              density="compact"
              class="mt-2"
            />
            <v-text-field
              v-model="filters.start_time"
              label="开始时间"
              type="datetime-local"
              density="compact"
              class="mt-2"
            />
            <v-text-field
              v-model="filters.end_time"
              label="结束时间"
              type="datetime-local"
              density="compact"
              class="mt-2"
            />
            <v-btn color="primary" block class="mt-4" @click="applyFilters">
              应用筛选
            </v-btn>
            <v-btn variant="text" block class="mt-2" @click="resetFilters">
              重置
            </v-btn>
          </v-card-text>
        </v-card>

        <!-- 统计信息 -->
        <v-card class="mt-4" v-if="stats">
          <v-card-title>统计</v-card-title>
          <v-card-text>
            <div class="d-flex justify-space-between">
              <span>总计:</span>
              <span class="font-weight-bold">{{ stats.total_count }}</span>
            </div>
            <div class="d-flex justify-space-between mt-2">
              <span class="text-success">成功:</span>
              <span class="font-weight-bold text-success">{{ stats.success_count }}</span>
            </div>
            <div class="d-flex justify-space-between mt-2">
              <span class="text-error">失败:</span>
              <span class="font-weight-bold text-error">{{ stats.failed_count }}</span>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- 日志列表 -->
      <v-col cols="9">
        <v-card>
          <v-card-text>
            <v-data-table
              :headers="headers"
              :items="logs"
              :loading="loading"
              :items-per-page="50"
              density="compact"
            >
              <template #item.timestamp="{ item }">
                {{ formatDate(item.timestamp) }}
              </template>
              <template #item.category="{ item }">
                <v-chip :color="getCategoryColor(item.category)" size="small">
                  {{ item.category }}
                </v-chip>
              </template>
              <template #item.action="{ item }">
                <span class="text-caption">{{ item.action }}</span>
              </template>
              <template #item.status="{ item }">
                <v-chip :color="getStatusColor(item.status)" size="small">
                  {{ item.status }}
                </v-chip>
              </template>
              <template #item.user_id="{ item }">
                <span class="text-caption">{{ item.user_id || '-' }}</span>
              </template>
              <template #item.resource="{ item }">
                <span class="text-caption">
                  {{ item.resource_type }}:{{ item.resource_id }}
                </span>
              </template>
              <template #item.actions="{ item }">
                <v-btn
                  icon="mdi-eye"
                  size="x-small"
                  variant="text"
                  @click="viewDetails(item)"
                />
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 详情对话框 -->
    <v-dialog v-model="detailDialog.show" max-width="700">
      <v-card v-if="detailDialog.log">
        <v-card-title>日志详情</v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="6">
              <div class="text-caption text-grey">ID</div>
              <div class="text-body-2">{{ detailDialog.log.id }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">时间</div>
              <div class="text-body-2">{{ formatDate(detailDialog.log.timestamp) }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">分类</div>
              <v-chip :color="getCategoryColor(detailDialog.log.category)" size="small">
                {{ detailDialog.log.category }}
              </v-chip>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">动作</div>
              <div class="text-body-2">{{ detailDialog.log.action }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">用户</div>
              <div class="text-body-2">{{ detailDialog.log.user_id || '-' }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">IP</div>
              <div class="text-body-2">{{ detailDialog.log.user_ip || '-' }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">资源</div>
              <div class="text-body-2">
                {{ detailDialog.log.resource_type }}:{{ detailDialog.log.resource_id }}
              </div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">状态</div>
              <v-chip :color="getStatusColor(detailDialog.log.status)" size="small">
                {{ detailDialog.log.status }}
              </v-chip>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">耗时</div>
              <div class="text-body-2">{{ detailDialog.log.duration }}ms</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">请求 ID</div>
              <div class="text-body-2">{{ detailDialog.log.request_id || '-' }}</div>
            </v-col>
            <v-col v-if="detailDialog.log.error" cols="12">
              <div class="text-caption text-grey">错误</div>
              <div class="text-body-2 text-error">{{ detailDialog.log.error }}</div>
            </v-col>
            <v-col cols="12">
              <div class="text-caption text-grey">详情</div>
              <pre class="details-pre">{{ JSON.stringify(detailDialog.log.details, null, 2) }}</pre>
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="detailDialog.show = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 导出对话框 -->
    <v-dialog v-model="exportDialog.show" max-width="500">
      <v-card>
        <v-card-title>导出审计日志</v-card-title>
        <v-card-text>
          <v-select
            v-model="exportDialog.format"
            :items="['csv', 'json']"
            label="格式"
          />
          <v-select
            v-model="exportDialog.category"
            :items="categories"
            label="分类（可选）"
            clearable
          />
          <v-text-field
            v-model="exportDialog.user_id"
            label="用户 ID（可选）"
            clearable
          />
          <v-text-field
            v-model="exportDialog.start_time"
            label="开始时间"
            type="datetime-local"
          />
          <v-text-field
            v-model="exportDialog.end_time"
            label="结束时间"
            type="datetime-local"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="exportDialog.show = false">取消</v-btn>
          <v-btn color="primary" @click="confirmExport">导出</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 清理对话框 -->
    <v-dialog v-model="cleanupDialog.show" max-width="400">
      <v-card>
        <v-card-title>清理审计日志</v-card-title>
        <v-card-text>
          <v-alert type="warning" class="mb-4">
            此操作将永久删除指定天数之前的审计日志，无法恢复！
          </v-alert>
          <v-text-field
            v-model="cleanupDialog.days"
            label="删除多少天前的日志"
            type="number"
            min="1"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="cleanupDialog.show = false">取消</v-btn>
          <v-btn color="error" @click="confirmCleanup">确认删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { auditApi, type AuditLog, type AuditStats, type AuditCategory, type AuditStatus } from '@/api/audit'

const showSnackbar = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
  console.log(`[${type}] ${message}`)
}

const logs = ref<AuditLog[]>([])
const stats = ref<AuditStats | null>(null)
const loading = ref(false)

const categories: AuditCategory[] = ['auth', 'agent', 'workflow', 'trigger', 'mcp', 'message', 'system', 'config', 'http']
const actions = [
  'login', 'logout', 'token_refresh', 'password_change',
  'agent_create', 'agent_update', 'agent_delete', 'agent_switch', 'agent_execute',
  'workflow_create', 'workflow_update', 'workflow_delete', 'workflow_execute', 'workflow_cancel',
  'trigger_create', 'trigger_update', 'trigger_delete', 'trigger_fire',
  'mcp_create', 'mcp_update', 'mcp_delete', 'mcp_connect',
  'message_send', 'message_receive',
  'system_start', 'system_stop', 'system_error', 'system_warning',
  'config_update', 'http_request'
]
const statuses: AuditStatus[] = ['success', 'failed', 'pending']

const headers = [
  { title: '时间', key: 'timestamp' },
  { title: '分类', key: 'category' },
  { title: '动作', key: 'action' },
  { title: '用户', key: 'user_id' },
  { title: '资源', key: 'resource' },
  { title: '状态', key: 'status' },
  { title: '操作', key: 'actions', sortable: false },
]

const filters = reactive({
  category: undefined as AuditCategory | undefined,
  action: undefined as string | undefined,
  status: undefined as AuditStatus | undefined,
  user_id: '',
  resource_type: '',
  resource_id: '',
  start_time: '',
  end_time: '',
})

const detailDialog = reactive({
  show: false,
  log: null as AuditLog | null,
})

const exportDialog = reactive({
  show: false,
  format: 'csv' as 'csv' | 'json',
  category: undefined as AuditCategory | undefined,
  user_id: '',
  start_time: '',
  end_time: '',
})

const cleanupDialog = reactive({
  show: false,
  days: 30,
})

function getCategoryColor(category: string) {
  const colors: Record<string, string> = {
    auth: 'purple',
    agent: 'blue',
    workflow: 'green',
    trigger: 'orange',
    mcp: 'cyan',
    message: 'pink',
    system: 'grey',
    config: 'brown',
    http: 'indigo',
  }
  return colors[category] || 'grey'
}

function getStatusColor(status: string) {
  switch (status) {
    case 'success': return 'success'
    case 'failed': return 'error'
    case 'pending': return 'warning'
    default: return 'grey'
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

async function loadLogs() {
  loading.value = true
  try {
    const params: any = {}
    if (filters.category) params.category = filters.category
    if (filters.action) params.action = filters.action
    if (filters.status) params.status = filters.status
    if (filters.user_id) params.user_id = filters.user_id
    if (filters.resource_type) params.resource_type = filters.resource_type
    if (filters.resource_id) params.resource_id = filters.resource_id
    if (filters.start_time) params.start_time = new Date(filters.start_time).toISOString()
    if (filters.end_time) params.end_time = new Date(filters.end_time).toISOString()

    const response = await auditApi.list(params)
    logs.value = response
  } catch (error) {
    showSnackbar('加载审计日志失败', 'error')
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const response = await auditApi.getStats()
    stats.value = response
  } catch (error) {
    showSnackbar('加载统计信息失败', 'error')
  }
}

function applyFilters() {
  loadLogs()
}

function resetFilters() {
  filters.category = undefined
  filters.action = undefined
  filters.status = undefined
  filters.user_id = ''
  filters.resource_type = ''
  filters.resource_id = ''
  filters.start_time = ''
  filters.end_time = ''
  loadLogs()
}

function viewDetails(log: AuditLog) {
  detailDialog.log = log
  detailDialog.show = true
}

function showExportDialog() {
  exportDialog.format = 'csv'
  exportDialog.category = undefined
  exportDialog.user_id = ''
  exportDialog.start_time = ''
  exportDialog.end_time = ''
  exportDialog.show = true
}

async function confirmExport() {
  try {
    const data = {
      format: exportDialog.format,
      category: exportDialog.category,
      user_id: exportDialog.user_id || undefined,
      start_time: exportDialog.start_time ? new Date(exportDialog.start_time).toISOString() : undefined,
      end_time: exportDialog.end_time ? new Date(exportDialog.end_time).toISOString() : undefined,
    }
    const response = await auditApi.export(data)
    
    // Download file
    const blob = new Blob([response.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `audit_logs.${exportDialog.format}`
    link.click()
    window.URL.revokeObjectURL(url)
    
    showSnackbar('导出成功', 'success')
    exportDialog.show = false
  } catch (error) {
    showSnackbar('导出失败', 'error')
  }
}

function showCleanupDialog() {
  cleanupDialog.days = 30
  cleanupDialog.show = true
}

async function confirmCleanup() {
  try {
    await auditApi.cleanup({ older_than_days: cleanupDialog.days })
    showSnackbar('清理成功', 'success')
    cleanupDialog.show = false
    loadLogs()
    loadStats()
  } catch (error) {
    showSnackbar('清理失败', 'error')
  }
}

onMounted(() => {
  loadLogs()
  loadStats()
})
</script>

<style scoped>
.audit-logs-page {
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.details-pre {
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  overflow-x: auto;
  font-size: 12px;
}
</style>
