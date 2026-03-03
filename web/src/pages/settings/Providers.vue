<template>
  <div class="providers-view">
    <div class="view-header">
      <div class="header-main">
        <h1 class="title">{{ t('settings.providers.title') }}</h1>
        <p class="description">{{ t('settings.providers.description') }}</p>
      </div>
      <n-button type="primary" size="medium" round @click="openModal('create')" class="add-button">
        <template #icon><n-icon :component="AddOutline" :size="18" /></template>
        {{ t('settings.providers.add') }}
      </n-button>
    </div>

    <div v-if="providers.length === 0" class="empty-state">
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
        :class="{ active: provider.is_active }"
      >
        <div class="card-content">
          <div class="card-header">
            <div class="provider-info">
              <h3 class="provider-name">{{ provider.name }}</h3>
              <p class="provider-model">{{ provider.model }}</p>
            </div>
            <n-tag v-if="provider.is_active" type="success" size="small" round>
              {{ t('settings.providers.active') }}
            </n-tag>
          </div>
          
          <div class="card-body">
            <div class="url-badge">
              <n-icon :component="LinkOutline" />
              <span class="url-text">{{ provider.base_url }}</span>
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
            >
              {{ t('settings.providers.setActive') }}
            </n-button>
          </div>
        </div>
      </div>
    </div>

    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="modalType === 'create' ? t('settings.providers.add') : t('settings.providers.edit')"
      style="width: 560px;"
      :bordered="false"
      size="medium"
    >
      <n-form :model="formModel" label-placement="top" label-width="auto">
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
          <n-input v-model:value="formModel.model" :placeholder="t('settings.providers.placeholder.model')" />
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
import { ref, onMounted, reactive } from 'vue'
import { NButton, NIcon, NEmpty, NTag, NSpace, NPopconfirm, NModal, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { AddOutline, LinkOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getProviders, saveProvider, deleteProvider, setActiveProvider } from '@/api/settings'
import type { BackendProvider } from '@/types'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

const providers = ref<BackendProvider[]>([])
const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const submitting = ref(false)

const formModel = reactive({ id: '', name: '', base_url: '', api_key: '', model: '' })

async function loadData() {
  try {
    const backendProviders = await getProviders()
    providers.value = backendProviders
    // 修复 P1: 直接同步后端原始数据（snake_case）
    appStore.setProviders(backendProviders)
  } catch (error) {
    console.error(error)
  }
}

function openModal(type: 'create' | 'edit', data?: BackendProvider) {
  modalType.value = type
  if (type === 'edit' && data) Object.assign(formModel, data)
  else Object.assign(formModel, { id: '', name: '', base_url: '', api_key: '', model: '' })
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
    await loadData()  // 重新加载数据并同步全局状态
  } catch (error) {
    message.error(t('common.error'))
  }
}

onMounted(loadData)
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;

.providers-view {
  display: flex;
  flex-direction: column;
  gap: 32px;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding-bottom: 24px;
  border-bottom: 1px solid $color-border-light;

  .title {
    margin: 0 0 8px;
    font-weight: $font-weight-bold;
    font-size: $font-size-h1;
    color: $color-text-primary;
    letter-spacing: -0.5px;
  }

  .description {
    font-size: $font-size-base;
    color: $color-text-secondary;
    margin: 0;
  }
}

.add-button {
  transition: all 0.2s ease;

  &:hover {
    transform: scale(1.02);
  }

  &:active {
    transform: scale(0.98);
  }
}

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
  animation: slideUp 0.4s ease-out;
  animation-fill-mode: both;

  @for $i from 1 through 10 {
    &:nth-child(#{$i}) {
      animation-delay: #{$i * 0.05}s;
    }
  }

  &:hover {
    transform: translateY(-4px);
    box-shadow: $shadow-lg;
    border-color: $color-primary-light;
  }

  &.active {
    border-color: $color-success;
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.02) 0%, $color-bg-primary 100%);

    &:hover {
      box-shadow: 0 12px 32px rgba(16, 185, 129, 0.15);
    }
  }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
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
    margin: 0 0 4px;
  }

  .provider-model {
    font-size: $font-size-sm;
    color: $color-text-secondary;
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

    .url-text {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid $color-border-light;
  padding-top: $spacing-5;

  .action-btn {
    transition: all 0.2s ease;

    &:hover {
      transform: scale(1.1);
    }

    &:active {
      transform: scale(0.95);
    }
  }

  .activate-btn {
    transition: all 0.2s ease;

    &:hover {
      transform: translateY(-1px);
      box-shadow: $shadow-md;
    }
  }
}

.empty-state {
  padding: $spacing-20 0;
  text-align: center;
}
</style>
