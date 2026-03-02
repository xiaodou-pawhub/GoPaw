<template>
  <div class="agent-page">
    <n-card :title="t('settings.agent.title')">
      <n-space vertical>
        <n-text depth="3">{{ t('settings.agent.description') }}</n-text>
        
        <n-input
          v-model:value="agentContent"
          type="textarea"
          :placeholder="t('settings.agent.placeholder')"
          :rows="20"
        />
        
        <n-button type="primary" :loading="saving" @click="saveAgent">
          {{ t('save') }}
        </n-button>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted } from 'vue'
import { NCard, NSpace, NText, NInput, NButton, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getAgent, saveAgent as apiSaveAgent } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const agentContent = ref('')
const saving = ref(false)

// 中文：加载 Agent 配置
// English: Load Agent config
async function loadAgent() {
  try {
    const res = await getAgent()
    agentContent.value = res.content || ''
  } catch (error) {
    message.error('Failed to load agent config')
  }
}

// 中文：保存 Agent 配置
// English: Save Agent config
async function saveAgent() {
  try {
    saving.value = true
    await apiSaveAgent(agentContent.value)
    message.success(t('success'))
  } catch (error) {
    message.error(t('error'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadAgent()
})
</script>

<style scoped>
.agent-page {
  padding: 24px;
}
</style>
