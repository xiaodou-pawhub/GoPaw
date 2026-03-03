<template>
  <div class="providers-view">
    <div class="view-header">
      <div class="header-main">
        <n-h2 class="title">{{ t('settings.providers.title') }}</n-h2>
        <n-text depth="3" class="description">{{ t('settings.providers.description') }}</n-text>
      </div>
      <n-button type="primary" size="large" round @click="openModal('create')">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('settings.providers.add') }}
      </n-button>
    </div>

    <div v-if="providers.length === 0" class="empty-state">
      <n-empty :description="t('settings.providers.noProviders')">
        <template #extra>
          <n-button quaternary @click="openModal('create')">{{ t('settings.providers.addFirst') }}</n-button>
        </template>
      </n-empty>
    </div>

    <div v-else class="provider-grid">
      <div
        v-for="provider in providers"
        :key="provider.id"
        class="provider-card"
        :class="{ active: provider.isActive }"
      >
        <div class="card-glow"></div>
        <div class="card-content">
          <div class="card-header">
            <div class="provider-info">
              <div class="provider-name">{{ provider.name }}</div>
              <div class="provider-model">{{ provider.model }}</div>
            </div>
            <n-tag v-if="provider.isActive" type="success" size="small" round ghost>
              {{ t('settings.providers.active') }}
            </n-tag>
          </div>
          
          <div class="card-body">
            <div class="url-badge">
              <n-icon :component="LinkOutline" />
              <span class="url-text">{{ provider.baseURL }}</span>
            </div>
          </div>

          <div class="card-footer">
            <n-space>
              <n-button quaternary circle size="small" @click="openModal('edit', provider)">
                <template #icon><n-icon :component="CreateOutline" /></template>
              </n-button>
              <n-popconfirm @positive-click="handleDelete(provider.id)">
                <template #trigger>
                  <n-button quaternary circle size="small" type="error">
                    <template #icon><n-icon :component="TrashOutline" /></template>
                  </n-button>
                </template>
                {{ t('settings.providers.deleteConfirm') }}
              </n-popconfirm>
            </n-space>
            
            <n-button
              v-if="!provider.isActive"
              secondary
              size="small"
              round
              @click="handleSetActive(provider.id)"
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
      style="width: 500px; border-radius: 16px;"
    >
      <n-form :model="formModel" label-placement="top">
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
          <n-button type="primary" :loading="submitting" @click="handleSubmit" round style="padding: 0 24px;">
            {{ t('common.save') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { NH2, NText, NButton, NIcon, NEmpty, NTag, NSpace, NPopconfirm, NModal, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { AddOutline, LinkOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getProviders, saveProvider, deleteProvider, setActiveProvider } from '@/api/settings'
import type { BackendProvider } from '@/types'

const { t } = useI18n()
const message = useMessage()

const providers = ref<BackendProvider[]>([])
const showModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const submitting = ref(false)

const formModel = reactive({ id: '', name: '', base_url: '', api_key: '', model: '' })

async function loadData() {
  try {
    providers.value = await getProviders()
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
    loadData()
  } catch (error) {
    message.error(t('common.error'))
  }
}

onMounted(loadData)
</script>

<style scoped lang="scss">
.providers-view { display: flex; flex-direction: column; gap: 40px; }
.view-header { display: flex; justify-content: space-between; align-items: flex-start; .title { margin: 0 0 8px; font-weight: 800; font-size: 32px; letter-spacing: -1px; } .description { font-size: 15px; } }
.provider-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 24px; }
.provider-card { position: relative; background: #fff; border-radius: 20px; padding: 24px; border: 1px solid rgba(0, 0, 0, 0.06); transition: all 0.4s cubic-bezier(0.165, 0.84, 0.44, 1); overflow: hidden; &:hover { transform: translateY(-4px); box-shadow: 0 12px 32px rgba(0, 0, 0, 0.08); border-color: rgba(24, 160, 88, 0.2); } &.active { background: #fdfdfd; border-color: rgba(24, 160, 88, 0.4); box-shadow: 0 8px 24px rgba(24, 160, 88, 0.06); .card-glow { opacity: 1; } } }
.card-glow { position: absolute; top: -50%; left: -50%; width: 200%; height: 200%; background: radial-gradient(circle, rgba(24, 160, 88, 0.03) 0%, transparent 70%); pointer-events: none; opacity: 0; transition: opacity 0.4s; }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; .provider-name { font-size: 20px; font-weight: 700; color: #1a1a1a; } .provider-model { font-size: 13px; color: #888; margin-top: 2px; } }
.card-body { margin-bottom: 24px; .url-badge { display: inline-flex; align-items: center; gap: 6px; background: #f5f5f5; padding: 6px 12px; border-radius: 8px; font-family: monospace; font-size: 12px; color: #666; max-width: 100%; .url-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } } }
.card-footer { display: flex; justify-content: space-between; align-items: center; border-top: 1px solid rgba(0, 0, 0, 0.04); padding-top: 20px; }
.empty-state { padding: 80px 0; }
</style>
