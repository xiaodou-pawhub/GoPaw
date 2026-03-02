<template>
  <div class="setup-page">
    <n-card class="setup-card" :bordered="false">
      <div class="setup-content">
        <div class="logo-hero">
          <n-icon :component="PawOutline" :size="80" color="#18a058" />
          <n-h1 class="hero-title">GoPaw</n-h1>
        </div>

        <n-result
          v-if="!status.llm_configured"
          status="info"
          :title="t('setup.title')"
          :description="t('setup.description')"
        >
          <template #footer>
            <n-button type="primary" size="large" @click="goToSettings" class="action-btn">
              {{ t('setup.getStarted') }}
            </n-button>
          </template>
        </n-result>
        
        <n-result
          v-else
          status="success"
          title="LLM 已就绪 / LLM Ready"
          description="系统已配置完成，开启您的 AI 助手之旅吧 / System configured, start your AI journey"
        >
          <template #footer>
            <n-button type="primary" size="large" @click="goToChat" class="action-btn">
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
import { NCard, NResult, NButton, NIcon, NH1, useMessage } from 'naive-ui'
import { PawOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getSetupStatus } from '@/api/settings'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const { t } = useI18n()
const message = useMessage()
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
    message.error(t('common.error'))
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

<style scoped lang="scss">
.setup-page {
  display: flex;
  align-items: center;
  justify-content: center;
  height: calc(100vh - 112px);
}

.setup-card {
  max-width: 600px;
  width: 100%;
  border-radius: 20px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.08);
}

.setup-content {
  padding: 40px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.logo-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
}

.hero-title {
  margin-top: 16px;
  font-size: 36px;
  font-weight: 800;
  letter-spacing: -1px;
}

.action-btn {
  min-width: 160px;
  border-radius: 8px;
}
</style>
