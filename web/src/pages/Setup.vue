<template>
  <div class="setup-page">
    <n-card class="setup-card" :bordered="false">
      <div class="setup-content">
        <div class="logo-hero">
          <img src="/assets/logo.png" alt="GoPaw Logo" class="logo-icon" />
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
import { NCard, NResult, NButton, NH1, useMessage } from 'naive-ui'
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
@use '@/styles/variables.scss' as *;

.setup-page {
  display: flex;
  align-items: center;
  justify-content: center;
  height: calc(100vh - 64px);
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.setup-card {
  max-width: 600px;
  width: 100%;
  border-radius: $radius-2xl;
  box-shadow: $shadow-xl;
  animation: slideUp 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(40px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.setup-content {
  padding: $spacing-12 0;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.logo-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: $spacing-6;
  animation: bounceIn 0.8s cubic-bezier(0.68, -0.55, 0.265, 1.55);
}

.logo-icon {
  width: 80px;
  height: 80px;
  object-fit: contain;
  border-radius: 12px;
  transition: all 0.3s ease;

  &:hover {
    transform: scale(1.1) rotate(-5deg);
  }
}

@keyframes bounceIn {
  0% {
    opacity: 0;
    transform: scale(0.3);
  }
  50% {
    opacity: 1;
    transform: scale(1.05);
  }
  70% {
    transform: scale(0.9);
  }
  100% {
    transform: scale(1);
  }
}

.hero-title {
  margin-top: $spacing-4;
  font-size: 36px;
  font-weight: $font-weight-extrabold;
  letter-spacing: -1px;
  color: $color-primary;
}

.action-btn {
  min-width: 160px;
  border-radius: $radius-lg;
  transition: all 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: $shadow-lg;
  }

  &:active {
    transform: translateY(0);
  }
}
</style>
