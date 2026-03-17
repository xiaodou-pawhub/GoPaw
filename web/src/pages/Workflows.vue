<template>
  <div class="workflows-page">
    <div class="page-header">
      <h1 class="text-h5">工作流</h1>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog">
        新建工作流
      </v-btn>
    </div>

    <v-row class="mt-4">
      <!-- 工作流列表 -->
      <v-col cols="4">
        <v-card>
          <v-card-title>工作流列表</v-card-title>
          <v-list density="compact">
            <v-list-item
              v-for="wf in workflows"
              :key="wf.id"
              :active="selectedWorkflow?.id === wf.id"
              @click="selectWorkflow(wf)"
            >
              <v-list-item-title>{{ wf.name }}</v-list-item-title>
              <v-list-item-subtitle>
                <v-chip
                  :color="getStatusColor(wf.status)"
                  size="x-small"
                  class="mr-2"
                >
                  {{ wf.status }}
                </v-chip>
                {{ wf.id }}
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <!-- 工作流详情 -->
      <v-col cols="8" v-if="selectedWorkflow">
        <v-card>
          <v-card-title class="d-flex align-center">
            <span>{{ selectedWorkflow.name }}</span>
            <v-spacer />
            <v-btn
              icon="mdi-play"
              size="small"
              color="success"
              @click="executeWorkflow"
              :disabled="selectedWorkflow.status !== 'active'"
            />
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
            <p class="text-body-2 text-grey">{{ selectedWorkflow.description }}</p>
            <v-divider class="my-4" />
            
            <!-- 统计信息 -->
            <div v-if="stats" class="d-flex justify-space-around mb-4">
              <div class="text-center">
                <div class="text-h6">{{ stats.total_executions }}</div>
                <div class="text-caption">总执行</div>
              </div>
              <div class="text-center">
                <div class="text-h6 text-success">{{ stats.completed_count }}</div>
                <div class="text-caption">成功</div>
              </div>
              <div class="text-center">
                <div class="text-h6 text-error">{{ stats.failed_count }}</div>
                <div class="text-caption">失败</div>
              </div>
              <div class="text-center">
                <div class="text-h6 text-info">{{ stats.running_count }}</div>
                <div class="text-caption">运行中</div>
              </div>
            </div>

            <!-- 步骤列表 -->
            <h3 class="text-subtitle-1 mb-2">步骤</h3>
            <v-timeline density="compact" side="end">
              <v-timeline-item
                v-for="step in selectedWorkflow.definition.steps"
                :key="step.id"
                size="small"
              >
                <template #icon>
                  <v-icon :icon="getStepIcon(step.action)" />
                </template>
                <div>
                  <div class="font-weight-medium">{{ step.name || step.id }}</div>
                  <div class="text-caption text-grey">
                    Agent: {{ step.agent }} | Action: {{ step.action }}
                  </div>
                  <div v-if="step.depends_on?.length" class="text-caption">
                    依赖: {{ step.depends_on.join(', ') }}
                  </div>
                </div>
              </v-timeline-item>
            </v-timeline>
          </v-card-text>
        </v-card>

        <!-- 执行历史 -->
        <v-card class="mt-4">
          <v-card-title>执行历史</v-card-title>
          <v-card-text>
            <v-data-table
              :headers="executionHeaders"
              :items="executions"
              :loading="loadingExecutions"
              density="compact"
            >
              <template #item.status="{ item }">
                <v-chip
                  :color="getExecutionStatusColor(item.status)"
                  size="small"
                >
                  {{ item.status }}
                </v-chip>
              </template>
              <template #item.started_at="{ item }">
                {{ item.started_at ? formatDate(item.started_at) : '-' }}
              </template>
              <template #item.actions="{ item }">
                <v-btn
                  icon="mdi-eye"
                  size="x-small"
                  variant="text"
                  @click="viewExecution(item)"
                />
                <v-btn
                  v-if="item.status === 'running'"
                  icon="mdi-stop"
                  size="x-small"
                  variant="text"
                  color="error"
                  @click="cancelExecution(item)"
                />
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 创建/编辑对话框 -->
    <v-dialog v-model="dialog.show" max-width="1200" fullscreen>
      <v-card>
        <v-card-title class="d-flex align-center">
          <span>{{ dialog.isEdit ? '编辑工作流' : '新建工作流' }}</span>
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" @click="dialog.show = false" />
        </v-card-title>
        <v-card-text class="pa-0">
          <v-form ref="form" v-model="dialog.valid">
            <v-row class="ma-0">
              <!-- 左侧：基本信息 -->
              <v-col cols="3" class="pa-4" style="border-right: 1px solid #e0e0e0;">
                <v-text-field
                  v-model="dialog.data.id"
                  label="ID"
                  :disabled="dialog.isEdit"
                  :rules="[(v: string) => !!v || 'ID 不能为空']"
                  required
                  density="compact"
                  class="mb-4"
                />
                <v-text-field
                  v-model="dialog.data.name"
                  label="名称"
                  :rules="[(v: string) => !!v || '名称不能为空']"
                  required
                  density="compact"
                  class="mb-4"
                />
                <v-textarea
                  v-model="dialog.data.description"
                  label="描述"
                  rows="3"
                  density="compact"
                  class="mb-4"
                />
                <v-select
                  v-model="dialog.data.status"
                  :items="['draft', 'active', 'disabled']"
                  label="状态"
                  density="compact"
                  class="mb-4"
                />
              </v-col>
              
              <!-- 右侧：设计器/JSON -->
              <v-col cols="9" class="pa-0">
                <v-tabs v-model="dialog.activeTab">
                  <v-tab value="designer">设计器</v-tab>
                  <v-tab value="json">JSON</v-tab>
                </v-tabs>
                
                <v-window v-model="dialog.activeTab" class="fill-height" @update:model-value="onTabChange">
                  <v-window-item value="designer" class="fill-height">
                    <WorkflowDesigner
                      :workflow="dialog.data.definition"
                      :agents="agents"
                      @save="onDesignerSave"
                      @validate="onDesignerValidate"
                      style="height: 600px;"
                    />
                  </v-window-item>
                  
                  <v-window-item value="json" class="fill-height">
                    <v-textarea
                      v-model="dialog.definitionText"
                      label="定义 (JSON)"
                      rows="25"
                      :rules="[(v: string) => {
                        if (!v) return '定义不能为空'
                        try {
                          JSON.parse(v)
                          return true
                        } catch {
                          return '无效的 JSON'
                        }
                      }]"
                      class="ma-4"
                    />
                  </v-window-item>
                </v-window>
              </v-col>
            </v-row>
          </v-form>
        </v-card-text>
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="dialog.show = false">取消</v-btn>
          <v-btn color="primary" :disabled="!dialog.valid" @click="saveWorkflow">
            保存
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 执行对话框 -->
    <v-dialog v-model="executeDialog.show" max-width="500">
      <v-card>
        <v-card-title>执行工作流</v-card-title>
        <v-card-text>
          <v-textarea
            v-model="executeDialog.inputText"
            label="输入参数 (JSON)"
            rows="5"
            placeholder='{"key": "value"}'
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="executeDialog.show = false">取消</v-btn>
          <v-btn color="primary" @click="confirmExecute">执行</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 执行详情对话框 -->
    <v-dialog v-model="executionDialog.show" max-width="700">
      <v-card v-if="executionDialog.execution">
        <v-card-title>执行详情</v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="6">
              <div class="text-caption text-grey">ID</div>
              <div>{{ executionDialog.execution.id }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">状态</div>
              <v-chip
                :color="getExecutionStatusColor(executionDialog.execution.status)"
                size="small"
              >
                {{ executionDialog.execution.status }}
              </v-chip>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">开始时间</div>
              <div>{{ executionDialog.execution.started_at ? formatDate(executionDialog.execution.started_at) : '-' }}</div>
            </v-col>
            <v-col cols="6">
              <div class="text-caption text-grey">完成时间</div>
              <div>{{ executionDialog.execution.completed_at ? formatDate(executionDialog.execution.completed_at) : '-' }}</div>
            </v-col>
            <v-col v-if="executionDialog.execution.error" cols="12">
              <div class="text-caption text-grey">错误</div>
              <div class="text-error">{{ executionDialog.execution.error }}</div>
            </v-col>
          </v-row>

          <!-- 步骤执行状态 -->
          <h3 class="text-subtitle-1 mt-4 mb-2">步骤执行</h3>
          <v-list density="compact">
            <v-list-item
              v-for="step in executionDialog.execution.steps"
              :key="step.id"
            >
              <template #prepend>
                <v-icon
                  :icon="getStepStatusIcon(step.status)"
                  :color="getStepStatusColor(step.status)"
                />
              </template>
              <v-list-item-title>{{ step.step_id }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ step.agent_id }} | {{ step.status }}
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="executionDialog.show = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="deleteDialog.show" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除工作流 "{{ deleteDialog.workflow?.name }}" 吗？
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog.show = false">取消</v-btn>
          <v-btn color="error" @click="deleteWorkflow">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { workflowsApi, type Workflow, type Execution, type ExecutionStats } from '@/api/workflows'
import WorkflowDesigner from '@/components/workflow/WorkflowDesigner.vue'

const showSnackbar = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
  console.log(`[${type}] ${message}`)
}

const workflows = ref<Workflow[]>([])
const selectedWorkflow = ref<Workflow | null>(null)
const executions = ref<Execution[]>([])
const stats = ref<ExecutionStats | null>(null)
const loadingExecutions = ref(false)

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
  activeTab: 'designer' as 'designer' | 'json',
  data: {
    id: '',
    name: '',
    description: '',
    status: 'draft' as 'draft' | 'active' | 'disabled',
    definition: {
      steps: [],
    } as any,
  },
  definitionText: '',
})

// Agent 列表（用于设计器）
const agents = ref<{ id: string; name: string }[]>([])

async function loadAgents() {
  try {
    // 这里应该从 agentApi 获取，暂时返回空数组
    agents.value = []
  } catch (error) {
    console.error('Failed to load agents:', error)
  }
}

const executeDialog = reactive({
  show: false,
  inputText: '{}',
})

const executionDialog = reactive({
  show: false,
  execution: null as Execution | null,
})

const deleteDialog = reactive({
  show: false,
  workflow: null as Workflow | null,
})

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
    case 'pending': return 'warning'
    case 'failed': return 'error'
    case 'cancelled': return 'grey'
    default: return 'grey'
  }
}

function getStepIcon(action: string) {
  switch (action) {
    case 'task': return 'mdi-play-circle'
    case 'notify': return 'mdi-bell'
    case 'query': return 'mdi-help-circle'
    default: return 'mdi-circle'
  }
}

function getStepStatusIcon(status: string) {
  switch (status) {
    case 'completed': return 'mdi-check-circle'
    case 'running': return 'mdi-loading'
    case 'pending': return 'mdi-clock'
    case 'failed': return 'mdi-close-circle'
    case 'skipped': return 'mdi-skip-next'
    default: return 'mdi-circle'
  }
}

function getStepStatusColor(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'running': return 'info'
    case 'pending': return 'warning'
    case 'failed': return 'error'
    case 'skipped': return 'grey'
    default: return 'grey'
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString('zh-CN')
}

async function loadWorkflows() {
  try {
    const response = await workflowsApi.list()
    workflows.value = response.data
  } catch (error) {
    showSnackbar('加载工作流失败', 'error')
  }
}

async function selectWorkflow(wf: Workflow) {
  selectedWorkflow.value = wf
  loadExecutions(wf.id)
  loadStats(wf.id)
}

async function loadExecutions(workflowId: string) {
  loadingExecutions.value = true
  try {
    const response = await workflowsApi.listExecutions(workflowId, 10)
    executions.value = response.data
  } catch (error) {
    showSnackbar('加载执行历史失败', 'error')
  } finally {
    loadingExecutions.value = false
  }
}

async function loadStats(workflowId: string) {
  try {
    const response = await workflowsApi.getStats(workflowId)
    stats.value = response.data
  } catch (error) {
    showSnackbar('加载统计失败', 'error')
  }
}

function openCreateDialog() {
  dialog.isEdit = false
  dialog.activeTab = 'designer'
  dialog.data = {
    id: '',
    name: '',
    description: '',
    status: 'draft',
    definition: { steps: [] },
  }
  dialog.definitionText = JSON.stringify({
    steps: [
      {
        id: 'step1',
        name: '第一步',
        agent: '',
        action: 'task',
        input: {},
      },
    ],
  }, null, 2)
  dialog.show = true
  loadAgents()
}

function openEditDialog() {
  if (!selectedWorkflow.value) return
  dialog.isEdit = true
  dialog.activeTab = 'designer'
  dialog.data = {
    id: selectedWorkflow.value.id,
    name: selectedWorkflow.value.name,
    description: selectedWorkflow.value.description,
    status: selectedWorkflow.value.status,
    definition: selectedWorkflow.value.definition,
  }
  dialog.definitionText = JSON.stringify(selectedWorkflow.value.definition, null, 2)
  dialog.show = true
  loadAgents()
}

// 设计器保存
function onDesignerSave(definition: any) {
  dialog.data.definition = definition
  dialog.definitionText = JSON.stringify(definition, null, 2)
  showSnackbar('工作流定义已更新', 'success')
}

// 设计器验证
function onDesignerValidate(definition: any) {
  dialog.data.definition = definition
  dialog.definitionText = JSON.stringify(definition, null, 2)
  showSnackbar('工作流定义已验证', 'success')
}

// 标签页切换
function onTabChange(tab: string) {
  if (tab === 'designer') {
    // 从 JSON 同步到设计器
    try {
      const definition = JSON.parse(dialog.definitionText)
      dialog.data.definition = definition
    } catch (e) {
      showSnackbar('JSON 格式无效，无法切换到设计器', 'error')
      dialog.activeTab = 'json'
    }
  } else if (tab === 'json') {
    // 从设计器同步到 JSON
    dialog.definitionText = JSON.stringify(dialog.data.definition, null, 2)
  }
}

async function saveWorkflow() {
  try {
    const definition = JSON.parse(dialog.definitionText)
    
    if (dialog.isEdit) {
      await workflowsApi.update(dialog.data.id, {
        name: dialog.data.name,
        description: dialog.data.description,
        definition,
        status: dialog.data.status,
      })
      showSnackbar('工作流更新成功', 'success')
    } else {
      await workflowsApi.create({
        id: dialog.data.id,
        name: dialog.data.name,
        description: dialog.data.description,
        definition,
      })
      showSnackbar('工作流创建成功', 'success')
    }
    dialog.show = false
    loadWorkflows()
  } catch (error: any) {
    showSnackbar(error.response?.data?.error || '保存失败', 'error')
  }
}

function executeWorkflow() {
  executeDialog.inputText = '{}'
  executeDialog.show = true
}

async function confirmExecute() {
  if (!selectedWorkflow.value) return
  try {
    let input = {}
    if (executeDialog.inputText) {
      try {
        input = JSON.parse(executeDialog.inputText)
      } catch {
        showSnackbar('输入 JSON 格式错误', 'error')
        return
      }
    }
    
    await workflowsApi.execute(selectedWorkflow.value.id, { input })
    showSnackbar('工作流执行已启动', 'success')
    executeDialog.show = false
    loadExecutions(selectedWorkflow.value.id)
  } catch (error: any) {
    showSnackbar(error.response?.data?.error || '执行失败', 'error')
  }
}

async function viewExecution(exec: Execution) {
  try {
    const response = await workflowsApi.getExecution(exec.id)
    executionDialog.execution = response.data
    executionDialog.show = true
  } catch (error) {
    showSnackbar('加载执行详情失败', 'error')
  }
}

async function cancelExecution(exec: Execution) {
  try {
    await workflowsApi.cancelExecution(exec.id)
    showSnackbar('执行已取消', 'success')
    if (selectedWorkflow.value) {
      loadExecutions(selectedWorkflow.value.id)
    }
  } catch (error) {
    showSnackbar('取消失败', 'error')
  }
}

function confirmDelete() {
  if (!selectedWorkflow.value) return
  deleteDialog.workflow = selectedWorkflow.value
  deleteDialog.show = true
}

async function deleteWorkflow() {
  if (!deleteDialog.workflow) return
  try {
    await workflowsApi.delete(deleteDialog.workflow.id)
    showSnackbar('工作流删除成功', 'success')
    deleteDialog.show = false
    selectedWorkflow.value = null
    loadWorkflows()
  } catch (error) {
    showSnackbar('删除失败', 'error')
  }
}

onMounted(() => {
  loadWorkflows()
})
</script>

<style scoped>
.workflows-page {
  padding: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
