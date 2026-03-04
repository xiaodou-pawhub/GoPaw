<template>
  <div class="panel">
    <div class="panel-header">
      <div class="header-left">
        <h2 class="panel-title">MEMORY.md</h2>
        <span class="char-count" :class="{ warn: content.length > 2000 }">
          {{ content.length }} / 2000
        </span>
      </div>
      <div class="header-actions">
        <span v-if="isModified" class="modified-hint">● 未保存</span>
        <n-button
          type="primary"
          size="small"
          :loading="saving"
          :disabled="!isModified"
          @click="handleSave"
        >
          {{ t('settings.memory.save') }}
        </n-button>
      </div>
    </div>
    <div class="panel-body">
      <n-input
        v-model:value="content"
        type="textarea"
        class="md-editor"
        :placeholder="t('settings.memory.placeholder')"
        :autosize="{ minRows: 20 }"
        @input="isModified = true"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { getMemoryMD, putMemoryMD } from '@/api/memory'

const { t } = useI18n()
const message = useMessage()

const content = ref('')
const saving = ref(false)
const isModified = ref(false)

async function load() {
  try {
    const res = await getMemoryMD()
    // @ts-ignore
    content.value = res.content || ''
    isModified.value = false
  } catch (e) {
    console.error(e)
  }
}

async function handleSave() {
  saving.value = true
  try {
    await putMemoryMD(content.value)
    message.success(t('common.success'))
    isModified.value = false
  } catch (e: any) {
    message.error(e?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;

.panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-4 $spacing-6;
  border-bottom: 1px solid $color-border-light;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: $spacing-3;
}

.panel-title {
  font-size: $font-size-lg;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
  margin: 0;
  font-family: $font-family-mono;
}

.char-count {
  font-size: $font-size-xs;
  color: $color-text-tertiary;

  &.warn {
    color: $color-warning;
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: $spacing-3;
}

.modified-hint {
  font-size: $font-size-xs;
  color: $color-warning;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-4 $spacing-6;
}

.md-editor {
  :deep(.n-input__border),
  :deep(.n-input__state-border) {
    border: none !important;
  }

  :deep(textarea) {
    font-family: $font-family-mono;
    font-size: $font-size-sm;
    line-height: $line-height-relaxed;
  }
}
</style>
