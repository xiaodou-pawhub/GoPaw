<template>
  <div class="agent-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <n-h2>{{ t('settings.agent.title') }}</n-h2>
        <n-text depth="3">{{ t('settings.agent.description') }}</n-text>
      </div>

      <n-card bordered class="editor-card">
        <n-space vertical :size="20">
          <n-alert type="info" title="AGENT.md">
            中文：此内容将作为所有对话的基础系统提示词（System Prompt），即时生效。
            <br/>
            English: This content serves as the base system prompt for all chats, taking effect immediately.
          </n-alert>
          
          <n-input
            v-model:value="agentContent"
            type="textarea"
            :placeholder="t('settings.agent.placeholder')"
            :rows="20"
            class="agent-editor"
          />
          
          <div class="form-actions">
            <n-button type="primary" size="large" :loading="saving" @click="saveAgent">
              {{ t('common.save') }}
            </n-button>
          </div>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted } from 'vue'
import { NCard, NSpace, NText, NInput, NButton, NH2, NAlert, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getAgentMD, saveAgentMD as apiSaveAgent } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const agentContent = ref('')
const saving = ref(false)

// 中文：加载 Agent 配置
// English: Load Agent config
async function loadAgent() {
  try {
    const res = await getAgentMD()
    agentContent.value = res.content || ''
  } catch (error) {
    message.error(t('common.error'))
  }
}

// 中文：保存 Agent 配置
// English: Save Agent config
async function saveAgent() {
  try {
    saving.value = true
    await apiSaveAgent(agentContent.value)
    message.success(t('common.success'))
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadAgent()
})
</script>

<style scoped lang="scss">
.agent-page {
  padding: 12px;
}

.page-header {
  margin-bottom: 8px;
}

.editor-card {
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
}

.agent-editor {
  font-family: 'Fira Code', 'Courier New', Courier, monospace;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
