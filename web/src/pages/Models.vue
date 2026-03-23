<template>
  <div class="models-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">模型配置</h2>
        <p class="page-desc">配置大语言模型服务商，支持 OpenAI 格式 API</p>
      </div>
      <button class="btn-primary" @click="openModal('create')">
        <PlusIcon :size="13" /> 添加模型
      </button>
    </div>

    <div v-if="providers.length === 0" class="empty-state">
      <CpuIcon :size="40" class="empty-icon" />
      <p class="empty-text">暂无模型配置</p>
      <button class="btn-primary" @click="openModal('create')">添加第一个模型</button>
    </div>

    <div v-else class="provider-list">
      <div
        v-for="provider in providers"
        :key="provider.id"
        class="provider-card"
        :class="{
          'card-cooldown': getHealth(provider.id)?.status === 'cooldown',
          'card-degraded': getHealth(provider.id)?.status === 'degraded',
          'card-disabled': !provider.enabled
        }"
      >
        <div class="card-main">
          <div class="card-info">
            <div class="card-title-row">
              <span class="provider-name">{{ provider.name }}</span>
              <span class="model-label">{{ provider.model }}</span>
            </div>
            <div class="provider-url">{{ provider.base_url }}</div>
            <div class="capability-tags">
              <span
                v-for="cap in getCoreCapabilities(provider)"
                :key="cap.key"
                class="cap-tag core"
              >{{ cap.icon }} {{ cap.label }}</span>
              <span
                v-for="cap in getFeatureCapabilities(provider)"
                :key="cap.key"
                class="cap-tag feature"
              >{{ cap.icon }} {{ cap.label }}</span>
            </div>
          </div>

          <div class="card-controls">
            <div class="health-dot" :class="getHealth(provider.id)?.status || 'healthy'" :title="getHealthLabel(provider.id)" />
            <label class="toggle" :title="provider.enabled ? '点击禁用' : '点击启用'">
              <input
                type="checkbox"
                :checked="provider.enabled"
                @change="handleToggle(provider)"
              />
              <span class="toggle-slider" />
            </label>
          </div>
        </div>

        <!-- 健康告警 -->
        <div v-if="getHealth(provider.id)?.status !== 'healthy'" class="health-banner" :class="getHealth(provider.id)?.status">
          <AlertCircleIcon :size="12" />
          <span>{{ getHealth(provider.id)?.last_error }}</span>
          <span v-if="getHealth(provider.id)?.status === 'cooldown'" class="cooldown-time">
            恢复于: {{ formatCooldown(getHealth(provider.id)?.cooldown_until) }}
          </span>
        </div>

        <div class="card-footer">
          <button class="action-btn" title="上移" :disabled="provider.priority === 0" @click="handleMoveUp(provider)">
            <ArrowUpIcon :size="13" />
          </button>
          <button class="action-btn" title="下移" @click="handleMoveDown(provider)">
            <ArrowDownIcon :size="13" />
          </button>
          <button class="action-btn" title="编辑" @click="openModal('edit', provider)">
            <PencilIcon :size="13" />
          </button>
          <button class="action-btn danger" title="删除" @click="handleDelete(provider.id)">
            <TrashIcon :size="13" />
          </button>
        </div>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">{{ modalType === 'create' ? '添加模型' : '编辑模型' }}</h3>
          <button class="icon-close" @click="showModal = false"><XIcon :size="16" /></button>
        </div>

        <div class="modal-body">
          <div class="form-field">
            <label>模型厂商 (可选)</label>
            <VendorCombobox
              v-model="selectedVendorId"
              :builtin-providers="builtinProviders"
              placeholder="选择厂商以快速填充"
              @change="handleVendorChange"
            />
          </div>
          <div class="form-field">
            <label>名称</label>
            <input v-model="formModel.name" placeholder="例如：OpenAI" class="form-input" />
          </div>
          <div class="form-field">
            <label>API 地址</label>
            <input v-model="formModel.base_url" placeholder="https://api.openai.com/v1" class="form-input" />
          </div>
          <div class="form-field">
            <label>API Key</label>
            <input v-model="formModel.api_key" type="password" placeholder="sk-..." class="form-input" />
          </div>
          <div class="form-field">
            <label>模型</label>
            <ModelCombobox
              v-model="formModel.model"
              :selected-vendor-id="selectedVendorId"
              :builtin-providers="builtinProviders"
              placeholder="输入或选择模型"
            />
          </div>

          <div class="form-field">
            <label>能力标签</label>
            <div class="tag-section">
              <div class="tag-group-label">核心能力</div>
              <div class="tag-group">
                <button
                  v-for="cap in coreCaps"
                  :key="cap.key"
                  class="cap-toggle"
                  :class="{ selected: formModel.tags.includes(cap.key) }"
                  @click="toggleTag(cap.key)"
                >{{ cap.icon }} {{ cap.label }}</button>
              </div>
              <div class="tag-group-label" style="margin-top: 8px;">进阶特性</div>
              <div class="tag-group">
                <button
                  v-for="cap in featureCaps"
                  :key="cap.key"
                  class="cap-toggle"
                  :class="{ selected: formModel.tags.includes(cap.key) }"
                  @click="toggleTag(cap.key)"
                >{{ cap.icon }} {{ cap.label }}</button>
              </div>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button class="btn-secondary" @click="showModal = false">取消</button>
          <button class="btn-primary" :disabled="submitting" @click="handleSubmit">
            {{ submitting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, reactive, computed, watch } from 'vue'
import {
  PlusIcon, PencilIcon, TrashIcon, ArrowUpIcon, ArrowDownIcon,
  AlertCircleIcon, XIcon, CpuIcon
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { useAppStore } from '@/stores/app'
import { getProviders, saveProvider, deleteProvider, getBuiltinProviders, getProvidersHealth } from '@/api/settings'
import { toggleProvider, reorderProviders } from '@/api/settings-providers'
import type { BackendProvider, BuiltinProvider } from '@/types'
import { MODEL_CAPABILITIES, autoDetectCapabilities } from '@/types'
import ModelCombobox from '@/components/common/ModelCombobox.vue'
import VendorCombobox from '@/components/common/VendorCombobox.vue'

const appStore = useAppStore()

const providers = ref<BackendProvider[]>([])
const builtinProviders = ref<BuiltinProvider[]>([])
const healthData = ref<any[]>([])
const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const submitting = ref(false)
const selectedVendorId = ref('')
let healthTimer: any = null

const formModel = reactive({
  id: '',
  name: '',
  base_url: '',
  api_key: '',
  model: '',
  tags: [] as string[]
})

const coreCaps = computed(() =>
  Object.values(MODEL_CAPABILITIES).filter(c => c.category === 'core')
)

const featureCaps = computed(() =>
  Object.values(MODEL_CAPABILITIES).filter(c => c.category === 'feature')
)

async function loadData() {
  try {
    const [backendProviders, builtins, health] = await Promise.all([
      getProviders(), getBuiltinProviders(), getProvidersHealth()
    ])
    providers.value = backendProviders
    builtinProviders.value = builtins
    healthData.value = health
    appStore.setProviders(backendProviders)
  } catch (error) {
    console.error(error)
  }
}

async function refreshHealth() {
  try { healthData.value = await getProvidersHealth() } catch {}
}

function getHealth(id: string) { return healthData.value.find(h => h.id === id) }

function getHealthLabel(id: string) {
  const h = getHealth(id)
  if (!h || h.status === 'healthy') return '运行正常'
  if (h.status === 'cooldown') return '正在冷却'
  return '配置失效'
}

function formatCooldown(until: number) {
  if (!until) return ''
  return new Date(until).toLocaleTimeString()
}

function getCoreCapabilities(provider: BackendProvider) {
  return provider.tags
    .map(tag => MODEL_CAPABILITIES[tag])
    .filter(cap => cap && cap.category === 'core')
    .slice(0, 3)
}

function getFeatureCapabilities(provider: BackendProvider) {
  return provider.tags
    .map(tag => MODEL_CAPABILITIES[tag])
    .filter(cap => cap && cap.category === 'feature')
    .slice(0, 2)
}

function toggleTag(tag: string) {
  const idx = formModel.tags.indexOf(tag)
  if (idx >= 0) formModel.tags.splice(idx, 1)
  else formModel.tags.push(tag)
}

watch(() => formModel.model, (newModel) => {
  if (modalType.value === 'create' && newModel) {
    formModel.tags = autoDetectCapabilities(newModel)
  }
})

function handleVendorChange(vendorId: string | null) {
  if (!vendorId) return
  const vendor = builtinProviders.value.find(v => v.id === vendorId)
  if (vendor) {
    formModel.name = vendor.name
    formModel.base_url = vendor.base_url
    if (vendor.models.length > 0) formModel.model = vendor.models[0]
  }
}

function openModal(type: 'create' | 'edit', data?: BackendProvider) {
  modalType.value = type
  selectedVendorId.value = ''
  if (type === 'edit' && data) {
    Object.assign(formModel, { ...data, tags: data.tags || [] })
  } else {
    Object.assign(formModel, { id: '', name: '', base_url: '', api_key: '', model: '', tags: [] })
  }
  showModal.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    await saveProvider(formModel)
    toast.success('保存成功')
    showModal.value = false
    loadData()
  } catch {
    toast.error('保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: string) {
  if (!confirm('确认删除此模型？')) return
  try {
    await deleteProvider(id)
    toast.success('已删除')
    loadData()
  } catch {
    toast.error('删除失败')
  }
}

async function handleToggle(provider: BackendProvider) {
  const newEnabled = !provider.enabled
  provider.enabled = newEnabled
  try {
    await toggleProvider(provider.id, newEnabled)
    toast.success(newEnabled ? '已启用' : '已禁用')
  } catch {
    toast.error('操作失败')
    provider.enabled = !newEnabled
  }
}

async function handleMoveUp(provider: BackendProvider) {
  if (provider.priority === 0) return
  const idx = providers.value.findIndex(p => p.id === provider.id)
  if (idx <= 0) return
  const target = providers.value[idx - 1]
  const temp = provider.priority
  provider.priority = target.priority
  target.priority = temp
  providers.value.splice(idx, 1)
  providers.value.splice(idx - 1, 0, provider)
  try {
    await reorderProviders(providers.value.map(p => p.id))
    toast.success('优先级已更新')
  } catch {
    toast.error('操作失败')
  }
}

async function handleMoveDown(provider: BackendProvider) {
  const idx = providers.value.findIndex(p => p.id === provider.id)
  if (idx < 0 || idx >= providers.value.length - 1) return
  const target = providers.value[idx + 1]
  const temp = provider.priority
  provider.priority = target.priority
  target.priority = temp
  providers.value.splice(idx, 1)
  providers.value.splice(idx + 1, 0, provider)
  try {
    await reorderProviders(providers.value.map(p => p.id))
    toast.success('优先级已更新')
  } catch {
    toast.error('操作失败')
  }
}

onMounted(() => {
  loadData()
  healthTimer = setInterval(refreshHealth, 10000)
})
onUnmounted(() => { if (healthTimer) clearInterval(healthTimer) })
</script>

<style scoped>
.models-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  overflow-y: auto;
  background: var(--bg-app);
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px;
}

.page-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 0;
}

.empty-icon { color: var(--text-tertiary); }
.empty-text { font-size: 13px; color: var(--text-secondary); }

.provider-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.provider-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  transition: border-color 0.15s;
}

.provider-card:hover { border-color: var(--accent); }
.provider-card.card-cooldown { border-color: rgba(245, 158, 11, 0.5); }
.provider-card.card-degraded { border-color: rgba(239, 68, 68, 0.5); }
.provider-card.card-disabled { opacity: 0.5; }

.card-main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 14px 16px 10px;
  gap: 12px;
}

.card-info { flex: 1; min-width: 0; }

.card-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.provider-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.model-label {
  font-size: 11px;
  color: var(--text-tertiary);
  font-family: "SF Mono", monospace;
}

.provider-url {
  font-size: 11px;
  color: var(--text-tertiary);
  font-family: "SF Mono", monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 8px;
}

.capability-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.cap-tag {
  font-size: 11px;
  padding: 2px 7px;
  border-radius: 4px;
}

.cap-tag.core {
  background: rgba(124, 106, 247, 0.1);
  color: var(--accent);
  border: 1px solid rgba(124, 106, 247, 0.2);
}

.cap-tag.feature {
  background: var(--bg-overlay);
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.card-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.health-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  cursor: default;
}

.health-dot.healthy { background: var(--green); box-shadow: 0 0 6px rgba(34, 197, 94, 0.4); }
.health-dot.cooldown { background: var(--yellow); animation: blink 1.5s infinite; }
.health-dot.degraded { background: var(--red); }

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.toggle {
  position: relative;
  width: 34px;
  height: 18px;
  cursor: pointer;
}

.toggle input { display: none; }

.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 9px;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 12px;
  height: 12px;
  background: var(--text-tertiary);
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: transform 0.2s, background 0.2s;
}

.toggle input:checked + .toggle-slider {
  background: rgba(124, 106, 247, 0.2);
  border-color: var(--accent);
}

.toggle input:checked + .toggle-slider::before {
  transform: translateX(16px);
  background: var(--accent);
}

.health-banner {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  font-size: 12px;
}

.health-banner.cooldown {
  background: rgba(245, 158, 11, 0.08);
  color: var(--yellow);
  border-top: 1px solid rgba(245, 158, 11, 0.15);
}

.health-banner.degraded {
  background: rgba(239, 68, 68, 0.08);
  color: var(--red);
  border-top: 1px solid rgba(239, 68, 68, 0.15);
}

.cooldown-time { margin-left: auto; font-family: monospace; }

.card-footer {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 6px 10px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-panel);
}

.action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}

.action-btn:hover { background: var(--bg-overlay); color: var(--text-secondary); }
.action-btn.danger:hover { color: var(--red); }
.action-btn:disabled { opacity: 0.3; cursor: not-allowed; }

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-card {
  width: 520px;
  max-height: 90vh;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.icon-close {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 4px;
  border-radius: 4px;
}

.icon-close:hover { color: var(--text-primary); background: var(--bg-overlay); }

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 20px;
  border-top: 1px solid var(--border-subtle);
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.form-field label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.form-input {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s;
}

.form-input:focus { border-color: var(--accent); }

.tag-section { display: flex; flex-direction: column; gap: 6px; }
.tag-group-label { font-size: 11px; color: var(--text-tertiary); }
.tag-group { display: flex; flex-wrap: wrap; gap: 6px; }

.cap-toggle {
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--bg-overlay);
  color: var(--text-tertiary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.cap-toggle:hover { border-color: var(--accent); color: var(--text-secondary); }
.cap-toggle.selected {
  border-color: var(--accent);
  background: var(--accent-dim);
  color: var(--accent);
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-secondary {
  padding: 7px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-secondary:hover { background: var(--bg-elevated); }
</style>
