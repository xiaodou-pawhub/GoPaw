<template>
  <div class="setup-page">
    <n-card class="setup-card" title="🐾 欢迎使用 GoPaw">
      <div class="setup-content">
        <n-result
          v-if="!status.llm_configured"
          status="info"
          :title="t('setup.title')"
          :description="t('setup.description')"
        >
          <template #footer>
            <n-button type="primary" @click="goToSettings">
              {{ t('setup.getStarted') }}
            </n-button>
          </template>
        </n-result>
        
        <n-result
          v-else
          status="success"
          title="LLM 已配置"
          description="您可以开始使用聊天功能了"
        >
          <template #footer>
            <n-button type="primary" @click="goToChat">
              {{ t('setup.configured') }}
            </n-button>
          </template>
        </n-result>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NResult, NButton } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getSetupStatus } from '@/api/settings'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()

const status = ref({
  llm_configured: false,
  setup_required: true,
  hint: ''
})

// 中文：检查设置状态
// English: Check setup status
async function checkSetupStatus() {
  try {
    const res = await getSetupStatus()
    status.value = res
    appStore.isLLMConfigured = res.llm_configured
  } catch (error) {
    console.error('Failed to get setup status:', error)
  }
}

// 中文：跳转到设置页
// English: Navigate to settings
function goToSettings() {
  router.push('/settings/providers')
}

// 中文：跳转到聊天页
// English: Navigate to chat
function goToChat() {
  router.push('/chat')
}

onMounted(() => {
  checkSetupStatus()
})
</script>

<style scoped>
.setup-page {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 24px;
}

.setup-card {
  max-width: 600px;
  width: 100%;
}

.setup-content {
  padding: 24px 0;
}
</style>
