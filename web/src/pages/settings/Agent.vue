<template>
  <div class="agent-view">
    <div class="view-header">
      <div class="header-main">
        <n-h2 class="title">{{ t('settings.agent.title') }}</n-h2>
        <n-text depth="3" class="description">{{ t('settings.agent.description') }}</n-text>
      </div>
      <n-button type="primary" size="large" round :loading="saving" @click="handleSave">
        {{ t('common.save') }}
      </n-button>
    </div>

    <div class="editor-container">
      <div class="editor-header">
        <div class="status-indicator">
          <div class="dot" :class="{ modified: isModified }"></div>
          <span>{{ isModified ? t('settings.modifiedStatus') : t('settings.syncStatus') }}</span>
        </div>
      </div>
      
      <n-input
        v-model:value="content"
        type="textarea"
        class="markdown-editor"
        :placeholder="t('settings.agent.placeholder')"
        :autosize="{ minRows: 15, maxRows: 30 }"
        @input="isModified = true"
      />
      
      <div class="editor-footer">
        <n-text depth="3">{{ t('settings.markdownTip') }}</n-text>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NH2, NText, NButton, NInput, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getAgentConfig, saveAgentConfig } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const content = ref('')
const saving = ref(false)
const isModified = ref(false)

async function loadData() {
  try {
    const res = await getAgentConfig()
    content.value = res.content || ''
    isModified.value = false
  } catch (error) {
    console.error(error)
  }
}

async function handleSave() {
  saving.value = true
  try {
    await saveAgentConfig(content.value)
    message.success(t('common.success'))
    isModified.value = false
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped lang="scss">
.agent-view { display: flex; flex-direction: column; gap: 40px; }
.view-header { display: flex; justify-content: space-between; align-items: flex-start; .title { margin: 0 0 8px; font-weight: 800; font-size: 32px; letter-spacing: -1px; } }
.editor-container { background: #fff; border-radius: 24px; border: 1px solid rgba(0, 0, 0, 0.06); box-shadow: 0 12px 40px rgba(0, 0, 0, 0.03); overflow: hidden; transition: all 0.3s; &:focus-within { border-color: rgba(24, 160, 88, 0.3); box-shadow: 0 12px 40px rgba(24, 160, 88, 0.05); } }
.editor-header { padding: 12px 24px; background: #fafafa; border-bottom: 1px solid rgba(0, 0, 0, 0.04); display: flex; justify-content: flex-end; .status-indicator { display: flex; align-items: center; gap: 8px; font-size: 12px; color: #888; .dot { width: 6px; height: 6px; border-radius: 50%; background: #18a058; &.modified { background: #f0a020; box-shadow: 0 0 8px rgba(240, 160, 32, 0.4); } } } }
.markdown-editor { :deep(.n-input-wrapper) { padding: 32px; } :deep(.n-input__border), :deep(.n-input__state-border) { border: none !important; } :deep(textarea) { font-family: 'Fira Code', 'PingFang SC', monospace; font-size: 15px; line-height: 1.8; color: #2c3e50; } }
.editor-footer { padding: 16px 32px; background: #fff; border-top: 1px solid rgba(0, 0, 0, 0.02); font-size: 13px; }
</style>
