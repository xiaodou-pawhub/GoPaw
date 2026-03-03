<template>
  <div class="page-container narrow">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('settings.memory.title') }}</h1>
        <p class="page-description">{{ t('settings.memory.description') }}</p>
      </div>
      <n-button type="primary" size="medium" round :loading="saving" @click="handleSave">
        {{ t('common.save') }}
      </n-button>
    </div>

    <div v-if="isEmpty" class="page-empty">
      <n-empty :description="t('settings.memory.emptyTip')" />
    </div>

    <div v-else class="page-card">
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
        :placeholder="t('settings.memory.placeholder')"
        :autosize="{ minRows: 15, maxRows: 30 }"
        @input="isModified = true"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { NButton, NInput, NEmpty, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getAgentMemory, saveAgentMemory } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const content = ref('')
const saving = ref(false)
const isModified = ref(false)
const loading = ref(false)

const isEmpty = computed(() => !content.value.trim())

async function loadData() {
  loading.value = true
  try {
    const res = await getAgentMemory()
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
    await saveAgentMemory(content.value)
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
</style>