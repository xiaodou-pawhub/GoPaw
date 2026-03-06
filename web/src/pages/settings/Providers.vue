<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('settings.providers.title') }}</h1>
        <p class="page-description">{{ t('settings.providers.description') }}</p>
      </div>
      <n-button type="primary" size="medium" round @click="openModal('create')" class="add-button">
        <template #icon><n-icon :component="AddOutline" :size="18" /></template>
        {{ t('settings.providers.add') }}
      </n-button>
    </div>

    <div v-if="providers.length === 0" class="page-empty">
      <n-empty :description="t('settings.providers.noProviders')" size="large">
        <template #extra>
          <n-button type="primary" round @click="openModal('create')" style="margin-top: 24px;">
            {{ t('settings.providers.addFirst') }}
          </n-button>
        </template>
      </n-empty>
    </div>

    <div v-else class="provider-grid">
      <div
        v-for="provider in providers"
        :key="provider.id"
        class="provider-card"
        :class="{ 
          active: provider.is_active,
          'status-cooldown': getHealth(provider.id)?.status === 'cooldown',
          'status-degraded': getHealth(provider.id)?.status === 'degraded'
        }"
      >
        <div class="card-content">
          <div class="card-header">
            <div class="provider-info">
              <h3 class="provider-name">{{ provider.name }}</h3>
              <div class="tag-row">
                <n-text depth="3" class="model-label">{{ provider.model }}</n-text>
                <div class="capability-tags">
                  <n-tooltip v-if="hasTag(provider, 'fc')" trigger="hover">
                    <template #trigger><n-icon :component="BuildOutline" class="cap-icon fc" /></template>
                    支持工具调用 (Function Calling)
                  </n-tooltip>
                  <n-tooltip v-if="hasTag(provider, 'vision')" trigger="hover">
                    <template #trigger><n-icon :component="EyeOutline" class="cap-icon vision" /></template>
                    支持视觉理解 (Vision)
                  </n-tooltip>
                  <n-tooltip v-if="hasTag(provider, 'reasoning')" trigger="hover">
                    <template #trigger><n-icon :component="ExtensionPuzzleOutline" class="cap-icon reasoning" /></template>
                    深度思考/推理模型 (Reasoning)
                  </n-tooltip>
                </div>
              </div>
            </div>
            
            <div class="header-status">
              <n-tag v-if="provider.is_active" type="success" size="small" round ghost>
                {{ t('settings.providers.active') }}
              </n-tag>
              
              <!-- 健康状态指示灯 -->
              <n-tooltip trigger="hover">
                <template #trigger>
                  <div class="health-dot" :class="getHealth(provider.id)?.status || 'healthy'" />
                </template>
                {{ getHealthLabel(provider.id) }}
              </n-tooltip>
            </div>
          </div>
          
          <div class="card-body">
            <div class="url-badge">
              <n-icon :component="LinkOutline" />
              <span class="url-text">{{ provider.base_url }}</span>
            </div>
            
            <!-- 冷却/错误详细信息展示 -->
            <div v-if="getHealth(provider.id)?.status !== 'healthy'" class="health-banner">
              <n-alert :type="getHealth(provider.id)?.status === 'cooldown' ? 'warning' : 'error'" size="small" :show-icon="false">
                <div class="health-msg">
                  <span class="msg-text">{{ getHealth(provider.id)?.last_error }}</span>
                  <span v-if="getHealth(provider.id)?.status === 'cooldown'" class="cooldown-timer">
                    恢复于: {{ formatCooldown(getHealth(provider.id)?.cooldown_until) }}
                  </span>
                </div>
              </n-alert>
            </div>
          </div>

          <div class="card-footer">
            <n-space>
              <n-button quaternary circle size="small" @click="openModal('edit', provider)" class="action-btn">
                <template #icon><n-icon :component="CreateOutline" :size="16" /></template>
              </n-button>
              <n-popconfirm @positive-click="handleDelete(provider.id)">
                <template #trigger>
                  <n-button quaternary circle size="small" type="error" class="action-btn">
                    <template #icon><n-icon :component="TrashOutline" :size="16" /></template>
                  </n-button>
                </template>
                {{ t('settings.providers.deleteConfirm') }}
              </n-popconfirm>
            </n-space>
            
            <n-button
              v-if="!provider.is_active"
              secondary
              size="small"
              round
              @click="handleSetActive(provider.id)"
              class="activate-btn"
              :disabled="getHealth(provider.id)?.status === 'degraded'"
            >
              {{ t('settings.providers.setActive') }}
            </n-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="modalType === 'create' ? t('settings.providers.add') : t('settings.providers.edit')"
      style="width: 560px;"
      :bordered="false"
      size="medium"
    >
      <n-form :model="formModel" label-placement="top" label-width="auto">
        <n-form-item label="模型厂商 (可选)">
          <n-select
            v-model:value="selectedVendorId"
            :options="vendorOptions"
            placeholder="选择厂商以快速填充配置"
            clearable
            @update:value="handleVendorChange"
          />
        </n-form-item>

        <n-form-item :label="t('settings.providers.name')">
          <n-input v-model:value="formModel.name" :placeholder="t('settings.providers.placeholder.name')" />
        </n-form-item>

        <n-form-item :label="t('settings.providers.baseURL')">
          <n-input v-model:value="formModel.base_url" :placeholder="t('settings.providers.placeholder.baseURL')" />
        </n-form-item>

        <n-form-item :label="t('settings.providers.apiKey')">
          <n-input
            v-model:value="formModel.api_key"
            type="password"
            show-password-on="mousedown"
            :placeholder="t('settings.providers.placeholder.apiKey')"
          />
        </n-form-item>

        <n-form-item :label="t('settings.providers.model')">
          <n-auto-complete
            v-model:value="formModel.model"
            :options="modelOptions"
            :placeholder="t('settings.providers.placeholder.model')"
            clearable
          />
        </n-form-item>

        <n-form-item label="能力标签">
          <div class="selectable-tags">
            <n-tag
              round
              :checkable="true"
              :checked="formModel.tags.includes('fc')"
              @update:checked="(val) => toggleTag('fc', val)"
              class="tag-item fc"
            >
              <template #icon><n-icon :component="BuildOutline" /></template>
              工具调用
            </n-tag>
            <n-tag
              round
              :checkable="true"
              :checked="formModel.tags.includes('vision')"
              @update:checked="(val) => toggleTag('vision', val)"
              class="tag-item vision"
            >
              <template #icon><n-icon :component="EyeOutline" /></template>
              视觉理解
            </n-tag>
            <n-tag
              round
              :checkable="true"
              :checked="formModel.tags.includes('reasoning')"
              @update:checked="(val) => toggleTag('reasoning', val)"
              class="tag-item reasoning"
            >
              <template #icon><n-icon :component="ExtensionPuzzleOutline" /></template>
              深度思考
            </n-tag>
          </div>
          <div class="tag-tip">系统已根据模型名自动为您勾选，您也可以手动调整。</div>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit" round>
            {{ t('common.save') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, reactive, computed, watch } from 'vue'
import {
  NButton, NIcon, NEmpty, NTag, NSpace, NPopconfirm, NModal, NAlert,
  NForm, NFormItem, NInput, NSelect, NAutoComplete, NTooltip, NText, useMessage
} from 'naive-ui'
import {
  AddOutline, LinkOutline, CreateOutline, TrashOutline,
  BuildOutline, EyeOutline, ExtensionPuzzleOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getProviders, saveProvider, deleteProvider, setActiveProvider, getBuiltinProviders, getProvidersHealth } from '@/api/settings'
import type { BackendProvider, BuiltinProvider } from '@/types'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

const providers = ref<BackendProvider[]>([])
const builtinProviders = ref<BuiltinProvider[]>([])
const healthData = ref<any[]>([])
const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const submitting = ref(false)

const selectedVendorId = ref<string | null>(null)
let healthTimer: any = null

const formModel = reactive({
  id: '',
  name: '',
  base_url: '',
  api_key: '',
  model: '',
  tags: [] as string[]
})

// 厂商下拉选项
const vendorOptions = computed(() => 
  builtinProviders.value.map(v => ({ label: v.name, value: v.id }))
)

// 模型自动完成选项
const modelOptions = computed(() => {
  const vendor = builtinProviders.value.find(v => v.id === selectedVendorId.value)
  return vendor ? vendor.models : []
})

async function loadData() {
  try {
    const [backendProviders, builtins, health] = await Promise.all([
      getProviders(),
      getBuiltinProviders(),
      getProvidersHealth()
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
  try {
    healthData.value = await getProvidersHealth()
  } catch (e) { /* ignore */ }
}

function getHealth(id: string) {
  return healthData.value.find(h => h.id === id)
}

function getHealthLabel(id: string) {
  const h = getHealth(id)
  if (!h || h.status === 'healthy') return '运行正常'
  if (h.status === 'cooldown') return '正在冷却 (API 暂时不可用)'
  return '配置失效 (鉴权失败/欠费)'
}

function formatCooldown(until: number) {
  if (!until) return ''
  return new Date(until).toLocaleTimeString()
}

function hasTag(provider: BackendProvider, tag: string): boolean {
  return provider.tags?.includes(tag) || false
}

// 切换标签状态
function toggleTag(tag: string, checked: boolean) {
  if (checked) {
    if (!formModel.tags.includes(tag)) formModel.tags.push(tag)
  } else {
    formModel.tags = formModel.tags.filter(t => t !== tag)
  }
}

// 手动模型名称变化时触发自动推断 (前端预览)
watch(() => formModel.model, (newModel) => {
  if (modalType.value === 'create' && newModel) {
    const m = newModel.toLowerCase()
    const tags = []
    if (m.includes('gpt-4') || m.includes('gpt-3.5') || m.includes('claude-3') || m.includes('qwen-') || m.includes('gemini') || m.includes('deepseek-chat')) {
      tags.push('fc')
    }
    if (m.includes('vision') || m.includes('gpt-4o') || m.includes('claude-3-5-sonnet')) {
      tags.push('vision')
    }
    if (m.includes('r1') || m.includes('reasoner') || m.includes('o1-')) {
      tags.push('reasoning')
    }
    formModel.tags = tags
  }
})

function handleVendorChange(vendorId: string | null) {
  if (!vendorId) return
  const vendor = builtinProviders.value.find(v => v.id === vendorId)
  if (vendor) {
    formModel.name = vendor.name
    formModel.base_url = vendor.base_url
    if (vendor.models.length > 0) {
      formModel.model = vendor.models[0]
    }
  }
}

function openModal(type: 'create' | 'edit', data?: BackendProvider) {
  modalType.value = type
  selectedVendorId.value = null
  if (type === 'edit' && data) {
    Object.assign(formModel, {
      ...data,
      tags: data.tags || []
    })
  } else {
    Object.assign(formModel, {
      id: '',
      name: '',
      base_url: '',
      api_key: '',
      model: '',
      tags: []
    })
  }
  showModal.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    await saveProvider(formModel)
    message.success(t('common.success'))
    showModal.value = false
    loadData()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: string) {
  try {
    await deleteProvider(id)
    message.success(t('common.success'))
    loadData()
  } catch (error) {
    message.error(t('common.error'))
  }
}

async function handleSetActive(id: string) {
  try {
    await setActiveProvider(id)
    message.success(t('common.success'))
    await loadData()
  } catch (error) {
    message.error(t('common.error'))
  }
}

onMounted(() => {
  loadData()
  healthTimer = setInterval(refreshHealth, 10000) // 10秒刷新一次健康状态
})

onUnmounted(() => {
  if (healthTimer) clearInterval(healthTimer)
})
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;
@use '@/styles/page-layout' as *;

.provider-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: $spacing-6;
}

.provider-card {
  background: $color-bg-primary;
  border-radius: $radius-xl;
  padding: $spacing-6;
  border: 1px solid $color-border-light;
  transition: $transition-normal;
  overflow: hidden;
  position: relative;

  &:hover {
    transform: translateY(-4px);
    box-shadow: $shadow-lg;
    border-color: $color-primary-light;
  }

  &.active {
    border-color: $color-success;
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.02) 0%, $color-bg-primary 100%);
  }

  &.status-cooldown { border-color: $color-warning; }
  &.status-degraded { border-color: $color-error; opacity: 0.8; }
}

.tag-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
}

.selectable-tags {
  display: flex;
  gap: 12px;
  margin-top: 4px;

  .tag-item {
    cursor: pointer;
    padding: 6px 16px;
    font-weight: 500;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    
    // 未选中状态
    &:not(.n-tag--checked) {
      background-color: #f3f4f6;
      color: #9ca3af;
      border-color: #e5e7eb;
      filter: grayscale(1);
      opacity: 0.7;
    }

    &.fc.n-tag--checked { 
      background-color: #eff6ff; 
      color: #2563eb; 
      border-color: #bfdbfe; 
    }
    &.vision.n-tag--checked { 
      background-color: #fffbeb; 
      color: #d97706; 
      border-color: #fde68a; 
    }
    &.reasoning.n-tag--checked { 
      background-color: #f0fdf4; 
      color: #16a34a; 
      border-color: #bbf7d0; 
    }
  }
}

.tag-tip {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 8px;
}

.capability-tags {
  display: flex;
  gap: 8px;
  align-items: center;
}

.cap-icon {
  font-size: 16px;
  cursor: help;
  
  &.fc { color: #2080f0; }
  &.vision { color: #f0a020; }
  &.reasoning { color: #18a058; }
}

.header-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.health-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  
  &.healthy { background-color: $color-success; box-shadow: 0 0 8px rgba(16, 185, 129, 0.4); }
  &.cooldown { background-color: $color-warning; animation: blink 1.5s infinite; }
  &.degraded { background-color: $color-error; }
}

@keyframes blink {
  0% { opacity: 1; }
  50% { opacity: 0.4; }
  100% { opacity: 1; }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: $spacing-5;

  .provider-name {
    font-size: $font-size-h4;
    font-weight: $font-weight-semibold;
    color: $color-text-primary;
    margin: 0;
  }
}

.card-body {
  margin-bottom: $spacing-6;

  .url-badge {
    display: inline-flex;
    align-items: center;
    gap: $spacing-2;
    background: $color-bg-tertiary;
    padding: $spacing-2 $spacing-3;
    border-radius: $radius-md;
    font-family: $font-family-mono;
    font-size: $font-size-xs;
    color: $color-text-secondary;
    max-width: 100%;
    margin-bottom: 8px;

    .url-text {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}

.health-banner {
  margin-top: 8px;
  .health-msg {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 12px;
    .msg-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
    .cooldown-timer { opacity: 0.8; font-family: monospace; }
  }
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid $color-border-light;
  padding-top: $spacing-5;
}

.empty-state {
  padding: $spacing-20 0;
  text-align: center;
}
</style>
