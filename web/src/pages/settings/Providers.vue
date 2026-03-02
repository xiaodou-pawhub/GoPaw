<template>
  <div class="providers-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <n-h2>{{ t('settings.providers.title') }}</n-h2>
        <n-text depth="3">配置对话所使用的语言模型提供商 / Configure language model providers for chat</n-text>
      </div>

      <n-card bordered class="list-card">
        <n-space vertical :size="16">
          <n-alert v-if="providers.length === 0" type="info">
            {{ t('settings.providers.addFirst') }}
          </n-alert>
          
          <n-list v-else hoverable clickable>
            <n-list-item v-for="provider in providers" :key="provider.id" class="provider-item">
              <template #prefix>
                <div class="status-indicator">
                  <n-tag :type="provider.isActive ? 'success' : 'default'" round size="small">
                    {{ provider.isActive ? t('settings.providers.active') : 'Inactive' }}
                  </n-tag>
                </div>
              </template>
              
              <n-space vertical :size="4">
                <div class="provider-name">{{ provider.name }}</div>
                <div class="provider-info">
                  <n-text depth="3">{{ provider.baseURL }}</n-text>
                  <n-divider vertical />
                  <n-tag size="small" quaternary>{{ provider.model }}</n-tag>
                </div>
              </n-space>
              
              <template #suffix>
                <n-space>
                  <n-button
                    v-if="!provider.isActive"
                    size="small"
                    secondary
                    @click="setActive(provider.id)"
                  >
                    {{ t('settings.providers.setActive') }}
                  </n-button>
                  <n-button size="small" quaternary @click="editProvider(provider)">
                    {{ t('common.edit') }}
                  </n-button>
                  <n-button size="small" quaternary type="error" @click="handleDeleteProvider(provider.id)">
                    {{ t('common.delete') }}
                  </n-button>
                </n-space>
              </template>
            </n-list-item>
          </n-list>
          
          <div class="card-actions">
            <n-button type="primary" @click="showAddModal">
              <template #icon>
                <n-icon :component="AddOutline" />
              </template>
              {{ t('settings.providers.add') }}
            </n-button>
          </div>
        </n-space>
      </n-card>
    </n-space>
    
    <!-- 中文：添加/编辑提供商对话框 / English: Add/Edit provider modal -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      style="width: 500px"
      :title="isEdit ? t('settings.providers.edit') : t('settings.providers.add')"
      class="provider-modal"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-placement="top"
      >
        <n-form-item :label="t('settings.providers.name')" path="name">
          <n-input
            v-model:value="formData.name"
            :placeholder="t('settings.providers.placeholder.name')"
          />
        </n-form-item>
        
        <n-form-item :label="t('settings.providers.baseURL')" path="baseURL">
          <n-input
            v-model:value="formData.baseURL"
            :placeholder="t('settings.providers.placeholder.baseURL')"
          />
        </n-form-item>
        
        <n-form-item :label="t('settings.providers.apiKey')" path="apiKey">
          <n-input
            v-model:value="formData.apiKey"
            type="password"
            show-password-on="click"
            :placeholder="t('settings.providers.placeholder.apiKey')"
          />
        </n-form-item>
        
        <n-form-item :label="t('settings.providers.model')" path="model">
          <n-input
            v-model:value="formData.model"
            :placeholder="t('settings.providers.placeholder.model')"
          />
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
import { ref, reactive, onMounted } from 'vue'
import {
  NCard, NList, NListItem, NButton, NSpace, NTag, NText,
  NModal, NForm, NFormItem, NInput, NAlert, NDivider, NH2,
  NIcon, useMessage
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import type { FormRules, FormInst } from 'naive-ui'
import type { Provider } from '@/types'
import {
  getProviders,
  saveProvider,
  setActiveProvider,
  deleteProvider as apiDeleteProvider
} from '@/api/settings'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

const providers = ref<Provider[]>([])
const showModal = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const formRef = ref<FormInst | null>(null)

const formData = reactive<Partial<Provider>>({
  name: '',
  baseURL: '',
  apiKey: '',
  model: '',
  isActive: false
})

const formRules: FormRules = {
  name: { required: true, message: t('settings.providers.placeholder.name'), trigger: 'blur' },
  baseURL: { required: true, message: t('settings.providers.placeholder.baseURL'), trigger: 'blur' },
  apiKey: { required: true, message: t('settings.providers.placeholder.apiKey'), trigger: 'blur' },
  model: { required: true, message: t('settings.providers.placeholder.model'), trigger: 'blur' }
}

// 中文：加载提供商列表
// English: Load provider list
async function loadProviders() {
  try {
    providers.value = await getProviders()
    appStore.setProviders(providers.value)
  } catch (error) {
    message.error(t('common.error'))
  }
}

// 中文：显示添加对话框
// English: Show add modal
function showAddModal() {
  isEdit.value = false
  Object.assign(formData, {
    id: '',
    name: '',
    baseURL: '',
    apiKey: '',
    model: '',
    isActive: false
  })
  showModal.value = true
}

// 中文：编辑提供商
// English: Edit provider
function editProvider(provider: Provider) {
  isEdit.value = true
  Object.assign(formData, {
    ...provider,
    apiKey: '' // 中文：编辑时不回填密码 / Do not backfill password when editing
  })
  showModal.value = true
}

// 中文：设置活跃提供商
// English: Set active provider
async function setActive(id: string) {
  try {
    await setActiveProvider(id)
    message.success(t('common.success'))
    loadProviders()
  } catch (error) {
    message.error(t('common.error'))
  }
}

// 中文：删除提供商
// English: Delete provider
async function handleDeleteProvider(id: string) {
  try {
    await apiDeleteProvider(id)
    message.success(t('common.success'))
    loadProviders()
  } catch (error) {
    message.error(t('common.error'))
  }
}

// 中文：提交表单
// English: Submit form
async function handleSubmit() {
  try {
    await formRef.value?.validate()
    saving.value = true
    await saveProvider(formData)
    message.success(t('common.success'))
    showModal.value = false
    loadProviders()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadProviders()
})
</script>

<style scoped lang="scss">
.providers-page {
  padding: 12px;
}

.page-header {
  margin-bottom: 8px;
}

.list-card {
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
}

.provider-item {
  padding: 16px 0;
}

.provider-name {
  font-weight: 700;
  font-size: 17px;
}

.provider-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.card-actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.provider-modal {
  border-radius: 12px;
}
</style>
