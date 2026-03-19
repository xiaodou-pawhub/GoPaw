<template>
  <div class="triggers-page">
    <div class="page-header">
      <h1 class="text-h5">Triggers</h1>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog">
        新建 Trigger
      </v-btn>
    </div>

    <v-card class="mt-4">
      <v-card-text>
        <v-data-table
          :headers="headers"
          :items="triggers"
          :loading="loading"
          item-value="id"
        >
          <template #item.is_enabled="{ item }">
            <v-switch
              v-model="item.is_enabled"
              density="compact"
              hide-details
              @change="toggleEnabled(item)"
            />
          </template>

          <template #item.type="{ item }">
            <v-chip
              :color="getTypeColor(item.type)"
              size="small"
            >
              {{ item.type }}
            </v-chip>
          </template>

          <template #item.last_fired_at="{ item }">
            <span v-if="item.last_fired_at">
              {{ formatDate(item.last_fired_at) }}
            </span>
            <span v-else class="text-grey">从未触发</span>
          </template>

          <template #item.actions="{ item }">
            <v-btn
              icon="mdi-play"
              size="small"
              variant="text"
              color="success"
              @click="fireTrigger(item)"
            />
            <v-btn
              icon="mdi-history"
              size="small"
              variant="text"
              @click="showHistory(item)"
            />
            <v-btn
              icon="mdi-pencil"
              size="small"
              variant="text"
              @click="openEditDialog(item)"
            />
            <v-btn
              icon="mdi-delete"
              size="small"
              variant="text"
              color="error"
              @click="confirmDelete(item)"
            />
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Create/Edit Dialog -->
    <v-dialog v-model="dialog.show" max-width="600">
      <v-card>
        <v-card-title>
          {{ dialog.isEdit ? '编辑 Trigger' : '新建 Trigger' }}
        </v-card-title>
        <v-card-text>
          <v-form ref="form" v-model="dialog.valid">
            <v-text-field
              v-model="dialog.data.id"
              label="ID"
              :disabled="dialog.isEdit"
              :rules="[(v: string) => !!v || 'ID 不能为空']"
              required
            />
            <v-text-field
              v-model="dialog.data.name"
              label="名称"
              :rules="[(v: string) => !!v || '名称不能为空']"
              required
            />
            <v-text-field
              v-model="dialog.data.description"
              label="描述"
            />
            <v-select
              v-model="dialog.data.agent_id"
              :items="agents"
              item-title="name"
              item-value="id"
              label="目标 Agent"
              :rules="[(v: string) => !!v || '请选择目标 Agent']"
              required
            />
            <v-select
              v-model="dialog.data.type"
              :items="triggerTypes"
              label="类型"
              :disabled="dialog.isEdit"
              :rules="[(v: string) => !!v || '请选择类型']"
              required
            />

            <!-- Cron Config -->
            <template v-if="dialog.data.type === 'cron'">
              <v-text-field
                v-model="dialog.data.config.expression"
                label="Cron 表达式"
                placeholder="0 9 * * 1-5"
                :rules="[(v: string) => !!v || 'Cron 表达式不能为空']"
                required
              />
              <v-btn
                size="small"
                variant="text"
                color="info"
                @click="validateCron"
                :loading="validating"
              >
                验证表达式
              </v-btn>
              <div v-if="cronValidation.description" class="text-caption text-grey mt-1">
                {{ cronValidation.description }}
                <span v-if="cronValidation.next_run">下次执行: {{ formatDate(cronValidation.next_run) }}</span>
              </div>
            </template>

            <!-- Webhook Config -->
            <template v-if="dialog.data.type === 'webhook'">
              <v-text-field
                v-model="dialog.data.config.secret"
                label="Webhook Secret (可选)"
                type="password"
              />
              <div class="text-caption text-grey">
                Webhook URL: {{ webhookUrl }}
              </div>
            </template>

            <!-- Message Config -->
            <template v-if="dialog.data.type === 'message'">
              <v-select
                v-model="dialog.data.config.from_agent"
                :items="agents"
                item-title="name"
                item-value="id"
                label="来源 Agent (可选)"
                clearable
              />
            </template>

            <v-text-field
              v-model="dialog.data.reason"
              label="触发原因"
              placeholder="触发时传递给 Agent 的上下文"
            />
            <v-text-field
              v-model.number="dialog.data.cooldown_seconds"
              label="冷却时间 (秒)"
              type="number"
              min="0"
            />
            <v-text-field
              v-model.number="dialog.data.max_fires"
              label="最大触发次数 (可选)"
              type="number"
              min="1"
              clearable
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="dialog.show = false">取消</v-btn>
          <v-btn color="primary" :disabled="!dialog.valid" @click="saveTrigger">
            保存
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- History Dialog -->
    <v-dialog v-model="historyDialog.show" max-width="800">
      <v-card>
        <v-card-title>
          Trigger 历史: {{ historyDialog.trigger?.name }}
        </v-card-title>
        <v-card-text>
          <v-data-table
            :headers="historyHeaders"
            :items="historyDialog.history"
            :loading="historyDialog.loading"
          >
            <template #item.success="{ item }">
              <v-icon
                :icon="item.success ? 'mdi-check-circle' : 'mdi-close-circle'"
                :color="item.success ? 'success' : 'error'"
              />
            </template>
            <template #item.fired_at="{ item }">
              {{ formatDate(item.fired_at) }}
            </template>
            <template #item.payload="{ item }">
              <pre v-if="item.payload" class="text-caption">{{ JSON.parse(item.payload) }}</pre>
              <span v-else>-</span>
            </template>
          </v-data-table>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="historyDialog.show = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation -->
    <v-dialog v-model="deleteDialog.show" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除 Trigger "{{ deleteDialog.trigger?.name }}" 吗？
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog.show = false">取消</v-btn>
          <v-btn color="error" @click="deleteTrigger">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { triggersApi, type Trigger, type TriggerHistory } from '@/api/triggers'
import { listAgents, type Agent } from '@/api/agents'

const showSnackbar = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
  console.log(`[${type}] ${message}`)
  // TODO: Implement proper snackbar notification
}

const loading = ref(false)
const triggers = ref<Trigger[]>([])
const agents = ref<Agent[]>([])
const validating = ref(false)

const headers = [
  { title: 'ID', key: 'id' },
  { title: '名称', key: 'name' },
  { title: 'Agent', key: 'agent_id' },
  { title: '类型', key: 'type' },
  { title: '启用', key: 'is_enabled', sortable: false },
  { title: '触发次数', key: 'fire_count' },
  { title: '上次触发', key: 'last_fired_at' },
  { title: '操作', key: 'actions', sortable: false },
]

const triggerTypes = [
  { title: 'Cron (定时)', value: 'cron' },
  { title: 'Webhook (外部事件)', value: 'webhook' },
  { title: 'Message (Agent 消息)', value: 'message' },
]

const dialog = reactive({
  show: false,
  isEdit: false,
  valid: false,
  data: {
    id: '',
    agent_id: '',
    name: '',
    description: '',
    type: 'cron' as 'cron' | 'webhook' | 'message',
    config: {} as any,
    reason: '',
    is_enabled: true,
    cooldown_seconds: 0,
    max_fires: null as number | null,
  },
})

const cronValidation = reactive({
  valid: false,
  description: '',
  next_run: '',
  error: '',
})

const historyDialog = reactive({
  show: false,
  trigger: null as Trigger | null,
  history: [] as TriggerHistory[],
  loading: false,
})

const historyHeaders = [
  { title: '时间', key: 'fired_at' },
  { title: '成功', key: 'success' },
  { title: '数据', key: 'payload' },
  { title: '错误', key: 'error_message' },
]

const deleteDialog = reactive({
  show: false,
  trigger: null as Trigger | null,
})

const webhookUrl = computed(() => {
  if (!dialog.data.id) return ''
  return `${window.location.origin}/webhook/${dialog.data.id}`
})

function getTypeColor(type: string) {
  switch (type) {
    case 'cron': return 'primary'
    case 'webhook': return 'success'
    case 'message': return 'warning'
    default: return 'grey'
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

async function loadTriggers() {
  loading.value = true
  try {
    const response = await triggersApi.list()
    triggers.value = response
  } catch (error) {
    showSnackbar('加载 Triggers 失败', 'error')
  } finally {
    loading.value = false
  }
}

async function loadAgents() {
  try {
    const response = await listAgents()
    agents.value = response.agents
  } catch (error) {
    showSnackbar('加载 Agents 失败', 'error')
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.data = {
    id: '',
    agent_id: '',
    name: '',
    description: '',
    type: 'cron',
    config: { expression: '' },
    reason: '',
    is_enabled: true,
    cooldown_seconds: 0,
    max_fires: null,
  }
  cronValidation.description = ''
  dialog.show = true
}

function openEditDialog(trigger: Trigger) {
  dialog.isEdit = true
  dialog.data = {
    id: trigger.id,
    agent_id: trigger.agent_id,
    name: trigger.name,
    description: trigger.description,
    type: trigger.type,
    config: JSON.parse(JSON.stringify(trigger.config)),
    reason: trigger.reason,
    is_enabled: trigger.is_enabled,
    cooldown_seconds: trigger.cooldown_seconds,
    max_fires: trigger.max_fires,
  }
  cronValidation.description = ''
  dialog.show = true
}

async function validateCron() {
  if (!dialog.data.config.expression) return
  validating.value = true
  try {
    const response = await triggersApi.validateCron(dialog.data.config.expression)
    cronValidation.valid = response.valid
    cronValidation.description = response.description || ''
    cronValidation.next_run = response.next_run || ''
    cronValidation.error = response.error || ''
    if (!response.valid) {
      showSnackbar(response.error || '表达式无效', 'error')
    }
  } catch (error) {
    showSnackbar('验证失败', 'error')
  } finally {
    validating.value = false
  }
}

async function saveTrigger() {
  try {
    const maxFires = dialog.data.max_fires === null ? undefined : dialog.data.max_fires
    if (dialog.isEdit) {
      await triggersApi.update(dialog.data.id, {
        name: dialog.data.name,
        description: dialog.data.description,
        config: dialog.data.config,
        reason: dialog.data.reason,
        is_enabled: dialog.data.is_enabled,
        cooldown_seconds: dialog.data.cooldown_seconds,
        max_fires: maxFires,
      })
      showSnackbar('Trigger 更新成功', 'success')
    } else {
      await triggersApi.create({
        id: dialog.data.id,
        agent_id: dialog.data.agent_id,
        name: dialog.data.name,
        description: dialog.data.description,
        type: dialog.data.type,
        config: dialog.data.config,
        reason: dialog.data.reason,
        is_enabled: dialog.data.is_enabled,
        cooldown_seconds: dialog.data.cooldown_seconds,
        max_fires: maxFires,
      })
      showSnackbar('Trigger 创建成功', 'success')
    }
    dialog.show = false
    loadTriggers()
  } catch (error: any) {
    showSnackbar(error.response?.data?.error || '保存失败', 'error')
  }
}

async function toggleEnabled(trigger: Trigger) {
  try {
    if (trigger.is_enabled) {
      await triggersApi.enable(trigger.id)
    } else {
      await triggersApi.disable(trigger.id)
    }
    showSnackbar(`Trigger ${trigger.is_enabled ? '启用' : '禁用'}成功`, 'success')
  } catch (error) {
    showSnackbar('操作失败', 'error')
    trigger.is_enabled = !trigger.is_enabled
  }
}

async function fireTrigger(trigger: Trigger) {
  try {
    await triggersApi.fire(trigger.id)
    showSnackbar('Trigger 手动触发成功', 'success')
    loadTriggers()
  } catch (error: any) {
    showSnackbar(error.response?.data?.error || '触发失败', 'error')
  }
}

async function showHistory(trigger: Trigger) {
  historyDialog.trigger = trigger
  historyDialog.show = true
  historyDialog.loading = true
  try {
    const response = await triggersApi.getHistory(trigger.id)
    historyDialog.history = response
  } catch (error) {
    showSnackbar('加载历史失败', 'error')
  } finally {
    historyDialog.loading = false
  }
}

function confirmDelete(trigger: Trigger) {
  deleteDialog.trigger = trigger
  deleteDialog.show = true
}

async function deleteTrigger() {
  if (!deleteDialog.trigger) return
  try {
    await triggersApi.delete(deleteDialog.trigger!.id)
    showSnackbar('Trigger 删除成功', 'success')
    loadTriggers()
  } catch (error) {
    showSnackbar('删除失败', 'error')
  } finally {
    deleteDialog.show = false
    deleteDialog.trigger = null
  }
}

onMounted(() => {
  loadTriggers()
  loadAgents()
})
</script>

<style scoped>
.triggers-page {
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
