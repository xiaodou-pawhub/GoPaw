<template>
  <div class="page-container narrow">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('settings.context.title') }}</h1>
        <p class="page-description">{{ t('settings.context.description') }}</p>
      </div>
      <n-button type="primary" size="medium" round :loading="saving" @click="handleSave">
        {{ t('common.save') }}
      </n-button>
    </div>

    <div class="page-card">
      <div class="editor-header">
        <div class="status-indicator" :class="{ modified: isModified }">
          <div class="status-dot"></div>
          <span class="status-text">{{ isModified ? t('settings.modifiedStatus') : t('settings.syncStatus') }}</span>
        </div>
      </div>
      
      <n-input
        v-model:value="content"
        type="textarea"
        class="markdown-editor"
        :placeholder="t('settings.context.placeholder')"
        :autosize="{ minRows: 15, maxRows: 30 }"
        @input="isModified = true"
      />
      
      <div class="editor-footer">
        <p class="footer-tip">{{ t('settings.markdownTip') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NButton, NInput, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getWorkspaceContext, saveWorkspaceContext } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const content = ref('')
const saving = ref(false)
const isModified = ref(false)
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    const res = await getWorkspaceContext()
    content.value = res.content || ''
    isModified.value = false
  } catch (error) {
    console.error(error)
    content.value = ''
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    await saveWorkspaceContext(content.value)
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
@use '@/styles/variables.scss' as *;
@use '@/styles/page-layout' as *;

.markdown-editor {
  :deep(.n-input-wrapper) {
    padding: $spacing-8;
  }

  :deep(.n-input__border),
  :deep(.n-input__state-border) {
    border: none !important;
  }

  :deep(textarea) {
    font-family: $font-family-mono;
    font-size: $font-size-sm;
    line-height: $line-height-relaxed;
    color: $color-text-primary;
  }
}

.editor-header {
  padding: $spacing-4 $spacing-5;
  background: $color-bg-secondary;
  border-bottom: 1px solid $color-border-light;
  display: flex;
  justify-content: flex-end;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: $spacing-2;
  font-size: $font-size-xs;
  color: $color-text-secondary;

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: $color-success;
    animation: pulse 2s infinite;
  }

  &.modified .status-dot {
    background: $color-warning;
    animation: pulse-warning 1.5s infinite;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

@keyframes pulse-warning {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.8;
    transform: scale(1.2);
  }
}

.status-text {
  font-weight: $font-weight-medium;
}

.editor-footer {
  padding: $spacing-4 $spacing-6;
  background: $color-bg-primary;
  border-top: 1px solid $color-border-light;
}

.footer-tip {
  margin: 0;
  font-size: $font-size-xs;
  color: $color-text-secondary;
}
</style>