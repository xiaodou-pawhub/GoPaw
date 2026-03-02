<template>
  <div class="providers-page">
    <n-card :title="t('settings.providers.title')">
      <n-space vertical>
        <n-alert v-if="providers.length === 0" type="info">
          {{ t('settings.providers.addFirst') }}
        </n-alert>
        
        <n-list v-else>
          <n-list-item v-for="provider in providers" :key="provider.id">
            <template #prefix>
              <n-tag :type="provider.isActive ? 'success' : 'default'">
                {{ provider.isActive ? t('settings.providers.active') : '' }}
              </n-tag>
            </template>
            
            <n-space vertical>
              <div class="provider-name">{{ provider.name }}</div>
              <div class="provider-info">
                <n-text depth="3">{{ provider.baseURL }}</n-text>
                <n-text depth="3"> · </n-text>
                <n-text depth="3">{{ provider.model }}</n-text>
              </div>
            </n-space>
            
            <template #suffix>
              <n-space>
                <n-button
                  v-if="!provider.isActive"
                  size="small"
                  @click="setActive(provider.id)"
                >
                  {{ t('settings.providers.setActive') }}
                </n-button>
                <n-button size="small" @click="editProvider(provider)">
                  {{ t('edit') }}
                </n-button>
                <n-button size="small" type="error" @click="deleteProvider(provider.id)">
                  {{ t('delete') }}
                </n-button>
              </n-space>
            </template>
          </n-list-item>
        </n-list>
        
        <n-divider />
        
        <n-button type="primary" @click="showAddModal">
          {{ t('settings.providers.add') }}
        </n-button>
      </n-space>
    </n-card>
    
    <!-- 中文：添加/编辑提供商对话框 / English: Add/Edit provider modal -->
    <n-modal v-model:show="showModal" :title="isEdit ? t('settings.providers.edit') : t('settings.providers.add')">
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
      
      <template #action>
        <n-button @click="showModal = false">{{ t('cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSubmit">
          {{ t('save') }}
        </n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NCard, NList, NListItem, NButton, NSpace, NTag, NText,
  NModal, NForm, NFormItem, NInput, NAlert, NDivider, useMessage
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { FormRules, FormInst } from 'naive-ui'
import type { Provider } from '@/api/settings'
import { getProviders, saveProvider, setActiveProvider, deleteProvider } from '@/api/settings'
import { useAppStore } from '@/stores/app'

const router = useRouter()
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
    message.error('Failed to load providers')
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
  Object.assign(formData, provider)
  showModal.value = true
}

// 中文：设置活跃提供商
// English: Set active provider
async function setActive(id: string) {
  try {
    await setActiveProvider(id)
    message.success(t('success'))
    loadProviders()
  } catch (error) {
    message.error(t('error'))
  }
}

// 中文：删除提供商
// English: Delete provider
async function deleteProvider(id: string) {
  try {
    await deleteProvider(id)
    message.success(t('success'))
    loadProviders()
  } catch (error) {
    message.error(t('error'))
  }
}

// 中文：提交表单
// English: Submit form
async function handleSubmit() {
  try {
    await formRef.value?.validate()
    saving.value = true
    await saveProvider(formData)
    message.success(t('success'))
    showModal.value = false
    loadProviders()
  } catch (error) {
    // 中文：验证失败或保存失败
    // English: Validation or save failed
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadProviders()
})
</script>

<style scoped>
.providers-page {
  padding: 24px;
}

.provider-name {
  font-weight: 600;
  font-size: 16px;
}

.provider-info {
  font-size: 14px;
}
</style>
