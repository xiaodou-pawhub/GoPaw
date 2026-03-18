<template>
  <div class="orchestrations-page">
    <div class="page-header">
      <h1 class="text-h5">Agent 编排器</h1>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog">
        新建编排
      </v-btn>
    </div>

    <v-row class="mt-4">
      <!-- 编排列表 -->
      <v-col cols="3">
        <v-card>
          <v-card-title>编排列表</v-card-title>
          <v-list density="compact">
            <v-list-item
              v-for="orch in orchestrations"
              :key="orch.id"
              :active="selectedOrch?.id === orch.id"
              @click="selectOrch(orch)"
            >
              <v-list-item-title>{{ orch.name }}</v-list-item-title>
              <v-list-item-subtitle>
                <v-chip :color="getStatusColor(orch.status)" size="x-small" class="mr-2">
                  {{ orch.status }}
                </v-chip>
                {{ orch.id }}
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <!-- 编排详情 -->
      <v-col cols="9" v-if="selectedOrch">
        <v-card>
          <v-card-title class="d-flex align-center">
            <span>{{ selectedOrch.name }}</span>
            <v-spacer />
            <v-btn
              prepend-icon="mdi-play"
              color="success"
              size="small"
              @click="openExecuteDialog"
            >
              执行
            </v-btn>
            <v-btn
              icon="mdi-pencil"
              size="small"
              @click="openEditDialog"
            />
            <v-btn
              icon="mdi-delete"
              size="small"
              color="error"
              @click="confirmDelete"
            />
          </v-card-title>
          
          <v-card-text>
            <p class="text-body-2 text-grey">{{ selectedOrch.description }}</p>
            
            <v-tabs v-model="activeTab" class="mt-4">
              <v-tab value="designer">设计器</v-tab>
              <v-tab value="json">JSON</v-tab>
              <v-tab value="executions">执行记录</v-tab>
            </v-tabs>

            <v-window v-model="activeTab">
              <!-- 设计器 -->
              <v-window-item value="designer">
                <div class="designer-container mt-4">
                  <OrchestrationDesigner
                    :definition="selectedOrch.definition"
                    :agents="availableAgents"
                    @save="onDesignerSave"
                    @validate="onDesignerValidate"
                  />
                </div>
              </v-window-item>

              <!-- JSON -->
              <v-window-item value="json">
                <v-textarea
                  v-model="definitionText"
                  label="编排定义 (JSON)"
                  rows="20"
                  readonly
                  class="mt-4"
                />
              </v-window-item>

              <!-- 执行记录 -->
              <v-window-item value="executions">
                <v-data-table
                  :headers="executionHeaders"
                  :items="executions"
                  density="compact"
                  class="mt-4"
                >
                  <template #item.status="{ item }">
                    <v-chip :color="getExecutionStatusColor(item.status)" size="small">
                      {{ item.status }}
                    </v-chip>
                  </template>
                  <template #item.started_at="{ item }">
                    {{ formatDate(item.started_at) }}
                  </template>
                  <template #item.actions="{ item }">
                    <v-btn
                      icon="mdi-eye"
                      size="x-small"
                      @click="viewExecution(item)"
                    />
                  </template>
                </v-data-table>
              </v-window-item>
            </v-window>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 创建/编辑对话框 -->
    <v-dialog v-model="dialog.show" max-width="800">
      <v-card>
        <v-card-title>{{ dialog.isEdit ? '编辑编排' : '新建编排' }}</v-card-title>
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
            <v-textarea
              v-model="dialog.data.description"
              label="描述"
              rows="2"
            />
            <v-textarea
              v-model="dialog.definitionText"
              label="编排定义 (JSON)"
              rows="15"
              :rules="[(v: string) => {
                if (!v) return '定义不能为空'
                try {
                  JSON.parse(v)
                  return true
                } catch {
                  return '无效的 JSON'
                }
              }]"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="dialog.show = false">取消</v-btn>
          <v-btn color="primary" :disabled="!dialog.valid" @click="saveOrchestration">
            保存
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 执行对话框 -->
    <v-dialog v-model="executeDialog.show" max-width="500">
      <v-card>
        <v-card-title>执行编排</v-card-title>
        <v-card-text>
          <v-textarea
            v-model="executeDialog.input"
            label="输入"
            rows="5"
            placeholder="请输入初始输入..."
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="executeDialog.show = false">取消</v-btn>
          <v-btn color="primary" @click="executeOrchestration">执行</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="deleteDialog.show" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除编排 "{{ deleteDialog.orch?.name }}" 吗？
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog.show = false">取消</v-btn>
          <v-btn color="error" @click="deleteOrchestration">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { orchestrationApi, type Orchestration, type ExecutionContext } from '@/api/orchestration'
import OrchestrationDesigner from '@/components/orchestration/OrchestrationDesigner.vue'

const orchestrations = ref<Orchestration[]>([])
const selectedOrch = ref<Orchestration | null>(null)
const executions = ref<ExecutionContext[]>([])
const activeTab = ref('definition')

const executionHeaders = [
  { title: 'ID', key: 'id' },
  { title: '状态', key: 'status' },
  { title: '开始时间', key: 'started_at' },
  { title: '操作', key: 'actions', sortable: false },
]

const dialog = reactive({
  show: false,
  isEdit: false,
  valid: false,
  data: {
    id: '',
    name: '',
    description: '',
  },
  definitionText: '',
})

const executeDialog = reactive({
  show: false,
  input: '',
})

const deleteDialog = reactive({
  show: false,
  orch: null as Orchestration | null,
})

// 可用 Agent 列表（用于设计器）
const availableAgents = ref([
  { id: 'default', name: '默认 Agent' },
  { id: 'product_manager', name: '产品经理' },
  { id: 'developer', name: '开发工程师' },
  { id: 'tester', name: '测试工程师' },
])

const definitionText = computed(() => {
  if (!selectedOrch.value) return ''
  return JSON.stringify(selectedOrch.value.definition, null, 2)
})

onMounted(() => {
  loadOrchestrations()
})

async function loadOrchestrations() {
  try {
    const response = await orchestrationApi.list()
    orchestrations.value = response.data
  } catch (error: any) {
    console.error('Failed to load orchestrations:', error)
    alert('加载编排列表失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

async function selectOrch(orch: Orchestration) {
  selectedOrch.value = orch
  loadExecutions(orch.id)
}

async function loadExecutions(orchId: string) {
  try {
    const response = await orchestrationApi.listExecutions(orchId)
    executions.value = response.data
  } catch (error: any) {
    console.error('Failed to load executions:', error)
    alert('加载执行记录失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.data = {
    id: '',
    name: '',
    description: '',
  }
  dialog.definitionText = JSON.stringify({
    nodes: [
      {
        id: 'agent_1',
        type: 'agent',
        agent_id: 'default',
        name: 'Agent',
        position: { x: 100, y: 100 },
      },
    ],
    edges: [],
    start_node_id: 'agent_1',
  }, null, 2)
  dialog.show = true
}

function openEditDialog() {
  if (!selectedOrch.value) return
  dialog.isEdit = true
  dialog.data = {
    id: selectedOrch.value.id,
    name: selectedOrch.value.name,
    description: selectedOrch.value.description,
  }
  dialog.definitionText = JSON.stringify(selectedOrch.value.definition, null, 2)
  dialog.show = true
}

async function saveOrchestration() {
  try {
    const definition = JSON.parse(dialog.definitionText)
    
    // 验证 JSON 结构
    if (!definition.nodes || !Array.isArray(definition.nodes)) {
      alert('编排定义必须包含 nodes 数组')
      return
    }
    if (!definition.edges || !Array.isArray(definition.edges)) {
      alert('编排定义必须包含 edges 数组')
      return
    }

    if (dialog.isEdit) {
      await orchestrationApi.update(dialog.data.id, {
        name: dialog.data.name,
        description: dialog.data.description,
        definition,
      })
    } else {
      await orchestrationApi.create({
        id: dialog.data.id,
        name: dialog.data.name,
        description: dialog.data.description,
        definition,
      })
    }
    dialog.show = false
    loadOrchestrations()
  } catch (error: any) {
    console.error('Failed to save orchestration:', error)
    if (error instanceof SyntaxError) {
      alert('JSON 格式错误: ' + error.message)
    } else {
      alert('保存失败: ' + (error.response?.data?.error || error.message || '未知错误'))
    }
  }
}

function confirmDelete() {
  if (!selectedOrch.value) return
  deleteDialog.orch = selectedOrch.value
  deleteDialog.show = true
}

async function deleteOrchestration() {
  if (!deleteDialog.orch) return
  try {
    await orchestrationApi.delete(deleteDialog.orch.id)
    deleteDialog.show = false
    selectedOrch.value = null
    loadOrchestrations()
  } catch (error: any) {
    console.error('Failed to delete orchestration:', error)
    alert('删除失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function openExecuteDialog() {
  executeDialog.input = ''
  executeDialog.show = true
}

async function executeOrchestration() {
  if (!selectedOrch.value) return
  try {
    const response = await orchestrationApi.execute(selectedOrch.value.id, {
      input: executeDialog.input,
    })
    executeDialog.show = false
    alert(`执行已开始，执行ID: ${response.data.execution_id}`)
    loadExecutions(selectedOrch.value.id)
  } catch (error: any) {
    console.error('Failed to execute orchestration:', error)
    alert('执行失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function viewExecution(exec: ExecutionContext) {
  // TODO: 打开执行详情对话框
  console.log('View execution:', exec)
}

// 设计器保存
async function onDesignerSave(definition: any) {
  if (!selectedOrch.value) return
  try {
    await orchestrationApi.update(selectedOrch.value.id, {
      definition,
    })
    // 刷新数据
    const response = await orchestrationApi.get(selectedOrch.value.id)
    selectedOrch.value = response.data
    alert('编排已保存')
  } catch (error: any) {
    console.error('Failed to save orchestration:', error)
    alert('保存失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

// 设计器验证
async function onDesignerValidate(definition: any) {
  try {
    await orchestrationApi.validate(definition)
    alert('编排定义有效')
  } catch (error: any) {
    console.error('Validation failed:', error)
    alert('验证失败: ' + (error.response?.data?.error || error.message || '未知错误'))
  }
}

function getStatusColor(status: string) {
  switch (status) {
    case 'active': return 'success'
    case 'draft': return 'warning'
    case 'disabled': return 'grey'
    default: return 'grey'
  }
}

function getExecutionStatusColor(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'running': return 'info'
    case 'paused': return 'warning'
    case 'failed': return 'error'
    default: return 'grey'
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}
</script>

<style scoped>
.orchestrations-page {
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.designer-container {
  height: 600px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
}
</style>
