<template>
  <div class="node-properties">
    <!-- 节点名称 -->
    <div class="form-group">
      <label>节点名称</label>
      <input v-model="localData.name" type="text" @input="onUpdate" />
    </div>

    <!-- Agent 节点 -->
    <template v-if="nodeType === 'agent'">
      <div class="form-group">
        <label>数字员工</label>
        <Combobox
          v-model="localData.agent_id"
          :options="agentOptions"
          placeholder="请选择数字员工..."
          @change="onAgentChange"
        />
      </div>
      <div class="form-group">
        <label>角色描述</label>
        <input v-model="localData.role" type="text" placeholder="如：负责需求分析" @input="onUpdate" />
      </div>
      <div class="form-group">
        <label>Prompt 模板</label>
        <textarea v-model="localData.prompt" rows="4" placeholder="输入 Prompt 模板，可用 {{变量名}}" @input="onUpdate" />
      </div>
      <div class="form-group">
        <label>输出变量名 <span class="hint-inline">（可选，默认用节点 ID）</span></label>
        <input v-model="localData.output_var" type="text" placeholder="如：result，后续节点可用 {{result}}" @input="onUpdate" />
      </div>

      <!-- 输入映射 -->
      <div class="mapping-section">
        <div class="mapping-header">
          <label>输入映射</label>
          <button class="btn-add-small" @click="addInputMapping">+ 添加</button>
        </div>
        <p class="mapping-hint">将上游节点的输出映射为本节点的输入变量</p>
        <div v-if="getInputMappings().length === 0" class="mapping-empty">
          暂无输入映射，将使用默认输入
        </div>
        <div v-else class="mapping-list">
          <div v-for="(mapping, index) in getInputMappings()" :key="'in_'+index" class="mapping-item">
            <input v-model="mapping.localName" type="text" class="mapping-name" placeholder="本节点变量名" @input="onUpdate" />
            <span class="mapping-arrow">←</span>
            <input v-model="mapping.sourceExpr" type="text" class="mapping-source" placeholder="{{上游变量}}" @input="onUpdate" />
            <button class="icon-btn icon-danger" @click="removeInputMapping(index)">×</button>
          </div>
        </div>
      </div>

      <!-- 输出映射 -->
      <div class="mapping-section">
        <div class="mapping-header">
          <label>输出映射</label>
          <button class="btn-add-small" @click="addOutputMapping">+ 添加</button>
        </div>
        <p class="mapping-hint">将本节点的输出存储为全局变量</p>
        <div v-if="getOutputMappings().length === 0" class="mapping-empty">
          暂无输出映射
        </div>
        <div v-else class="mapping-list">
          <div v-for="(mapping, index) in getOutputMappings()" :key="'out_'+index" class="mapping-item">
            <input v-model="mapping.outputName" type="text" class="mapping-name" placeholder="输出字段名" @input="onUpdate" />
            <span class="mapping-arrow">→</span>
            <input v-model="mapping.storeName" type="text" class="mapping-source" placeholder="存储变量名" @input="onUpdate" />
            <button class="icon-btn icon-danger" @click="removeOutputMapping(index)">×</button>
          </div>
        </div>
      </div>
    </template>

    <!-- 人工节点 -->
    <template v-if="nodeType === 'human'">
      <div class="form-group">
        <label>提示模板</label>
        <textarea v-model="localData.prompt" rows="3" placeholder="如：请确认以下方案：{{content}}" @input="onUpdate" />
      </div>
      <div class="form-group">
        <label>快捷选项</label>
        <div v-for="(_opt, index) in getOptions()" :key="index" class="option-row">
          <input v-model="(getOptions() as string[])[index]" type="text" @input="onUpdate" />
          <button class="icon-btn icon-danger" @click="removeOption(index)">×</button>
        </div>
        <button class="btn-add" @click="addOption">+ 添加选项</button>
      </div>
      <div class="form-group">
        <label>超时 (秒): {{ getConfig().timeout || 0 }}</label>
        <input v-model.number="getConfig().timeout" type="range" min="0" max="3600" step="60" @input="onUpdate" />
      </div>
    </template>

    <!-- 条件节点 -->
    <template v-if="nodeType === 'condition'">
      <div class="form-group">
        <label>条件类型</label>
        <Combobox
          v-model="getConfig().condition_type"
          :options="conditionTypeOptions"
          placeholder="请选择条件类型..."
          @change="onConditionTypeChange"
        />
      </div>

      <!-- 二分模式：是/否 -->
      <template v-if="getConfig().condition_type !== 'switch'">
        <div v-if="getConfig().condition_type === 'expression'" class="form-group">
          <label>表达式</label>
          <input v-model="getConfig().expression" type="text" placeholder="{{score}} > 80" @input="onUpdate" />
          <span class="hint">支持变量替换和简单比较</span>
        </div>
        <div v-if="getConfig().condition_type === 'intent'" class="form-group">
          <label>意图关键词</label>
          <input v-model="getConfig().intent" type="text" placeholder="查询,订单" @input="onUpdate" />
          <span class="hint">多个关键词用逗号分隔</span>
        </div>
        <div v-if="getConfig().condition_type === 'llm'" class="form-group">
          <label>LLM 判断提示</label>
          <textarea v-model="getConfig().llm_query" rows="3" placeholder="根据用户情绪判断..." @input="onUpdate" />
        </div>
        <!-- 二分模式分支预览 -->
        <div v-if="conditionBranches" class="branch-preview">
          <div class="branch-title">分支走向</div>
          <div class="branch-item branch-true">
            <span class="branch-badge true">是</span>
            <span class="branch-arrow">→</span>
            <span class="branch-target">{{ conditionBranches.true || '未连接' }}</span>
          </div>
          <div class="branch-item branch-false">
            <span class="branch-badge false">否</span>
            <span class="branch-arrow">→</span>
            <span class="branch-target">{{ conditionBranches.false || '未连接' }}</span>
          </div>
        </div>
      </template>

      <!-- Switch 模式：多分支 -->
      <template v-if="getConfig().condition_type === 'switch'">
        <div class="branch-list">
          <div class="branch-list-header">
            <span>分支配置</span>
            <button class="btn-add-small" @click="addBranch">+ 添加分支</button>
          </div>
          <div v-for="(branch, index) in getBranches()" :key="index" class="branch-config-item">
            <div class="branch-config-header">
              <input
                v-model="branch.label"
                type="text"
                class="branch-label-input"
                placeholder="分支名称"
                @input="onUpdate"
              />
              <button v-if="getBranches().length > 1" class="icon-btn icon-danger" @click="removeBranch(index)">×</button>
            </div>
            <div class="branch-condition-row">
              <Combobox
                v-model="branch.condition_type"
                :options="branchConditionTypeOptions"
                placeholder="条件类型"
                @change="onUpdate"
              />
            </div>
            <div v-if="branch.condition_type === 'expression'" class="branch-condition-row">
              <input v-model="branch.expression" type="text" placeholder="{{var}} == 'value'" @input="onUpdate" />
            </div>
            <div v-if="branch.condition_type === 'intent'" class="branch-condition-row">
              <input v-model="branch.intent" type="text" placeholder="关键词1,关键词2" @input="onUpdate" />
            </div>
            <div v-if="branch.condition_type === 'llm'" class="branch-condition-row">
              <textarea v-model="branch.llm_query" rows="2" placeholder="LLM 判断提示..." @input="onUpdate" />
            </div>
            <div v-if="branch.condition_type === 'always'" class="branch-condition-row hint">
              无条件执行（默认分支）
            </div>
          </div>
        </div>
        <!-- Switch 模式分支预览 -->
        <div v-if="switchBranches.length > 0" class="branch-preview">
          <div class="branch-title">分支走向</div>
          <div v-for="(branch, index) in switchBranches" :key="index" class="branch-item">
            <span class="branch-badge" :class="getBranchBadgeClass(index)">{{ branch.label }}</span>
            <span class="branch-arrow">→</span>
            <span class="branch-target">{{ branch.target || '未连接' }}</span>
          </div>
        </div>
      </template>
    </template>

    <!-- 并行节点 -->
    <template v-if="nodeType === 'parallel'">
      <div class="form-group">
        <label>最大并发数</label>
        <input v-model.number="getConfig().max_concurrent" type="number" min="1" max="10" @input="onUpdate" />
        <span class="hint">0 表示不限制</span>
      </div>
    </template>

    <!-- 循环节点 -->
    <template v-if="nodeType === 'loop'">
      <div class="form-group">
        <label>循环条件</label>
        <input v-model="getConfig().condition" type="text" placeholder="{{continue}} == true" @input="onUpdate" />
        <span class="hint">条件为 true 时继续循环</span>
      </div>
      <div class="form-group">
        <label>最大循环次数</label>
        <input v-model.number="getConfig().max_loop" type="number" min="1" max="100" @input="onUpdate" />
      </div>
    </template>

    <!-- 子流程节点 -->
    <template v-if="nodeType === 'subflow'">
      <div class="form-group">
        <label>子流程</label>
        <Combobox
          v-model="getConfig().flow_id"
          :options="flowOptions"
          placeholder="请选择子流程..."
          @change="onUpdate"
        />
      </div>
    </template>

    <!-- Webhook 节点 -->
    <template v-if="nodeType === 'webhook'">
      <div class="form-group">
        <label>Webhook 路径</label>
        <input v-model="getConfig().path" type="text" placeholder="/webhook/xxx" @input="onUpdate" />
        <span class="hint">自定义路径标识</span>
      </div>
      <div class="form-group">
        <label>超时 (秒)</label>
        <input v-model.number="getConfig().timeout" type="number" min="60" max="86400" @input="onUpdate" />
        <span class="hint">等待外部回调的最长时间</span>
      </div>
    </template>

    <!-- 结束节点 -->
    <template v-if="nodeType === 'end'">
      <div class="form-group">
        <label>输出模板</label>
        <textarea v-model="getConfig().output_template" rows="3" placeholder="最终答案：{{content}}" @input="onUpdate" />
      </div>
    </template>

    <div class="divider" />

    <button class="btn-danger" @click="emit('delete')">
      <Trash2Icon :size="14" /> 删除节点
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Trash2Icon } from 'lucide-vue-next'
import Combobox from '@/components/common/Combobox.vue'

interface Agent { id: string; name: string }
interface Flow { id: string; name: string }
interface EdgeInfo { id: string; source: string; target: string; sourceHandle?: string | null; label?: string }
interface NodeInfo { id: string; data?: { name?: string }; type?: string }

const props = defineProps<{
  nodeType: string
  nodeData: Record<string, any>
  nodeId?: string
  agents?: Agent[]
  flows?: Flow[]
  edges?: EdgeInfo[]
  nodes?: NodeInfo[]
}>()

const emit = defineEmits<{
  update: [data: Record<string, any>]
  delete: []
}>()

const localData = ref<Record<string, any>>({ name: '', config: {} })

// Agent 选项
const agentOptions = computed(() => {
  return (props.agents || []).map(a => ({ value: a.id, label: a.name }))
})

// 流程选项
const flowOptions = computed(() => {
  return (props.flows || []).map(f => ({ value: f.id, label: f.name }))
})

// 条件类型选项
const conditionTypeOptions = [
  { value: 'expression', label: '表达式' },
  { value: 'intent', label: '意图匹配' },
  { value: 'llm', label: 'LLM 判断' },
  { value: 'switch', label: '多分支 (Switch)' }
]

// 分支条件类型选项
const branchConditionTypeOptions = [
  { value: 'always', label: '默认（无条件）' },
  { value: 'expression', label: '表达式' },
  { value: 'intent', label: '意图匹配' },
  { value: 'llm', label: 'LLM 判断' }
]

// 条件节点分支预览
const conditionBranches = computed(() => {
  if (props.nodeType !== 'condition' || !props.nodeId) return null
  const outEdges = (props.edges || []).filter(e => e.source === props.nodeId)
  const getNodeName = (id: string) => {
    const n = (props.nodes || []).find(n => n.id === id)
    return n?.data?.name || id
  }
  const trueBranch = outEdges.find(e => e.sourceHandle === 'true' || e.sourceHandle === '是' || e.label === 'true' || e.label === '是')
  const falseBranch = outEdges.find(e => e.sourceHandle === 'false' || e.sourceHandle === '否' || e.label === 'false' || e.label === '否')
  return {
    true: trueBranch ? getNodeName(trueBranch.target) : null,
    false: falseBranch ? getNodeName(falseBranch.target) : null,
  }
})

// Switch 模式分支预览
const switchBranches = computed(() => {
  if (props.nodeType !== 'condition' || !props.nodeId) return []
  const branches = getBranches()
  const outEdges = (props.edges || []).filter(e => e.source === props.nodeId)
  const getNodeName = (id: string) => {
    const n = (props.nodes || []).find(n => n.id === id)
    return n?.data?.name || id
  }
  return branches.map((branch, index) => {
    const edge = outEdges.find(e => e.sourceHandle === branch.label || e.label === branch.label)
    return {
      label: branch.label || `分支${index + 1}`,
      target: edge ? getNodeName(edge.target) : null
    }
  })
})

// 获取分支列表
function getBranches(): Array<{ label: string; condition_type: string; expression?: string; intent?: string; llm_query?: string }> {
  const cfg = getConfig()
  if (!cfg.branches) {
    // 默认两个分支
    cfg.branches = [
      { label: '是', condition_type: 'always' },
      { label: '否', condition_type: 'always' }
    ]
  }
  return cfg.branches
}

// 添加分支
function addBranch() {
  const branches = getBranches()
  branches.push({
    label: `分支${branches.length + 1}`,
    condition_type: 'always'
  })
  onUpdate()
}

// 删除分支
function removeBranch(index: number) {
  getBranches().splice(index, 1)
  onUpdate()
}

// 条件类型变更
function onConditionTypeChange(value: string | number | null) {
  if (value === 'switch') {
    // 初始化多分支配置
    const cfg = getConfig()
    if (!cfg.branches) {
      cfg.branches = [
        { label: '分支1', condition_type: 'always' },
        { label: '默认', condition_type: 'always' }
      ]
    }
  }
  onUpdate()
}

// 获取分支徽章样式
function getBranchBadgeClass(index: number): string {
  const colors = ['branch-1', 'branch-2', 'branch-3', 'branch-4', 'branch-5']
  return colors[index % colors.length]
}

function getConfig(): Record<string, any> {
  if (!localData.value.config) localData.value.config = {}
  return localData.value.config
}

function getOptions(): string[] {
  const cfg = getConfig()
  if (!cfg.options) cfg.options = []
  return cfg.options
}

// 输入映射相关
interface InputMapping {
  localName: string
  sourceExpr: string
}

function getInputMappings(): InputMapping[] {
  if (!localData.value.inputs) localData.value.inputs = {}
  // 将对象转换为数组便于编辑
  const mappings: InputMapping[] = []
  const inputs = localData.value.inputs
  for (const [localName, sourceExpr] of Object.entries(inputs)) {
    mappings.push({ localName, sourceExpr: String(sourceExpr) })
  }
  return mappings
}

function addInputMapping() {
  if (!localData.value.inputs) localData.value.inputs = {}
  const mappings = getInputMappings()
  mappings.push({ localName: '', sourceExpr: '' })
  // 重新构建 inputs 对象
  localData.value.inputs = {}
  for (const m of mappings) {
    if (m.localName) {
      localData.value.inputs[m.localName] = m.sourceExpr
    }
  }
  onUpdate()
}

function removeInputMapping(index: number) {
  const mappings = getInputMappings()
  mappings.splice(index, 1)
  // 重新构建 inputs 对象
  localData.value.inputs = {}
  for (const m of mappings) {
    if (m.localName) {
      localData.value.inputs[m.localName] = m.sourceExpr
    }
  }
  onUpdate()
}

// 输出映射相关
interface OutputMapping {
  outputName: string
  storeName: string
}

function getOutputMappings(): OutputMapping[] {
  if (!localData.value.outputs) localData.value.outputs = {}
  const mappings: OutputMapping[] = []
  const outputs = localData.value.outputs
  for (const [outputName, storeName] of Object.entries(outputs)) {
    mappings.push({ outputName, storeName: String(storeName) })
  }
  return mappings
}

function addOutputMapping() {
  if (!localData.value.outputs) localData.value.outputs = {}
  const mappings = getOutputMappings()
  mappings.push({ outputName: '', storeName: '' })
  localData.value.outputs = {}
  for (const m of mappings) {
    if (m.outputName) {
      localData.value.outputs[m.outputName] = m.storeName
    }
  }
  onUpdate()
}

function removeOutputMapping(index: number) {
  const mappings = getOutputMappings()
  mappings.splice(index, 1)
  localData.value.outputs = {}
  for (const m of mappings) {
    if (m.outputName) {
      localData.value.outputs[m.outputName] = m.storeName
    }
  }
  onUpdate()
}

watch(() => props.nodeData, (newData) => {
  localData.value = JSON.parse(JSON.stringify(newData || { name: '', config: {} }))
  if (!localData.value.config) localData.value.config = {}
}, { immediate: true, deep: true })

function onUpdate() {
  emit('update', JSON.parse(JSON.stringify(localData.value)))
}

function onAgentChange(value: string | number | null) {
  // 自动填充 Agent 名称作为节点名称
  const agent = (props.agents || []).find(a => a.id === value)
  if (agent && !localData.value.name) {
    localData.value.name = agent.name
  }
  onUpdate()
}

function addOption() {
  getOptions().push('')
  onUpdate()
}

function removeOption(index: number) {
  getOptions().splice(index, 1)
  onUpdate()
}
</script>

<style scoped>
.node-properties { padding: 12px; }
.form-group { margin-bottom: 12px; }
.form-group label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 4px;
}
.form-group input,
.form-group textarea {
  width: 100%;
  padding: 6px 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
}
.form-group textarea { resize: vertical; }
.hint { font-size: 10px; color: var(--text-muted); margin-top: 2px; display: block; }
.option-row { display: flex; gap: 4px; margin-bottom: 4px; }
.option-row input { flex: 1; }
.icon-btn {
  width: 24px; height: 24px;
  border: none; border-radius: 4px;
  cursor: pointer;
  background: var(--bg-overlay);
}
.icon-danger:hover { background: #fee2e2; color: #ef4444; }
.btn-add {
  width: 100%;
  padding: 6px;
  background: var(--bg-app);
  border: 1px dashed var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}
.btn-add:hover { border-color: var(--accent); color: var(--accent); }
.hint-inline { font-size: 10px; color: var(--text-muted); font-weight: 400; }
.branch-preview { margin-bottom: 12px; padding: 10px; background: var(--bg-app); border: 1px solid var(--border); border-radius: 6px; }
.branch-title { font-size: 11px; font-weight: 500; color: var(--text-secondary); margin-bottom: 8px; }
.branch-item { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; font-size: 12px; }
.branch-badge { padding: 1px 6px; border-radius: 3px; font-size: 10px; font-weight: 600; }
.branch-badge.true { background: #dcfce7; color: #16a34a; }
.branch-badge.false { background: #fee2e2; color: #dc2626; }
.branch-badge.branch-1 { background: #dbeafe; color: #2563eb; }
.branch-badge.branch-2 { background: #fef3c7; color: #d97706; }
.branch-badge.branch-3 { background: #f3e8ff; color: #9333ea; }
.branch-badge.branch-4 { background: #e0f2fe; color: #0891b2; }
.branch-badge.branch-5 { background: #fce7f3; color: #db2777; }
.branch-arrow { color: var(--text-muted); }
.branch-target { color: var(--text-primary); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.divider { height: 1px; background: var(--border); margin: 12px 0; }
.btn-danger {
  width: 100%;
  padding: 8px;
  background: transparent;
  border: 1px solid #ef4444;
  border-radius: 4px;
  color: #ef4444;
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
.btn-danger:hover { background: #fee2e2; }

/* 多分支配置样式 */
.branch-list { margin-bottom: 12px; }
.branch-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}
.btn-add-small {
  padding: 2px 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--accent);
  font-size: 11px;
  cursor: pointer;
}
.btn-add-small:hover { background: var(--bg-overlay); }
.branch-config-item {
  padding: 8px;
  margin-bottom: 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
}
.branch-config-header {
  display: flex;
  gap: 4px;
  margin-bottom: 6px;
}
.branch-label-input {
  flex: 1;
  padding: 4px 6px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 500;
}
.branch-condition-row {
  margin-bottom: 4px;
}
.branch-condition-row:last-child { margin-bottom: 0; }
.branch-condition-row input,
.branch-condition-row textarea {
  width: 100%;
  padding: 4px 6px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 11px;
}

/* 输入输出映射样式 */
.mapping-section {
  margin-top: 12px;
  padding: 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.mapping-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.mapping-header label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

.mapping-hint {
  font-size: 10px;
  color: var(--text-muted);
  margin: 0 0 8px 0;
}

.mapping-empty {
  font-size: 11px;
  color: var(--text-muted);
  text-align: center;
  padding: 8px;
}

.mapping-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mapping-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.mapping-name,
.mapping-source {
  flex: 1;
  min-width: 0;
  padding: 4px 6px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 11px;
}

.mapping-arrow {
  font-size: 12px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.icon-btn {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 14px;
  flex-shrink: 0;
}

.icon-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.icon-btn.icon-danger {
  border-color: #fecaca;
  color: #dc2626;
}

.icon-btn.icon-danger:hover {
  background: #fee2e2;
}

.branch-condition-row textarea { resize: vertical; min-height: 40px; }
</style>