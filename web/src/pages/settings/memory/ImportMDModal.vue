<template>
  <n-modal
    :show="show"
    @update:show="emit('update:show', $event)"
    preset="card"
    :title="t('settings.memory.importTitle')"
    style="width: 640px"
  >
    <div class="import-body">
      <div class="form-item">
        <label class="form-label">{{ t('settings.memory.importContent') }}</label>
        <n-input
          v-model:value="content"
          type="textarea"
          :placeholder="t('settings.memory.importContentPlaceholder')"
          :autosize="{ minRows: 8, maxRows: 15 }"
          style="font-family: var(--font-mono); font-size: 13px"
        />
      </div>
      <div class="form-row">
        <div class="form-item">
          <label class="form-label">{{ t('settings.memory.importCategory') }}</label>
          <n-select
            v-model:value="category"
            :options="categoryOptions"
            style="width: 160px"
          />
        </div>
      </div>

      <!-- Preview -->
      <div v-if="preview.length > 0" class="preview-section">
        <div class="preview-title">
          {{ t('settings.memory.importPreview', { n: preview.length }) }}
        </div>
        <div class="preview-list">
          <div v-for="sec in preview" :key="sec.key" class="preview-item">
            <span class="preview-key">{{ sec.key }}</span>
            <span class="preview-content">{{ truncate(sec.content, 60) }}</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="modal-footer">
        <n-button @click="emit('update:show', false)">{{ t('settings.memory.cancel') }}</n-button>
        <n-button @click="handlePreview" :disabled="!content.trim()">预览</n-button>
        <n-button
          type="primary"
          :loading="importing"
          :disabled="!content.trim()"
          @click="handleImport"
        >
          {{ t('settings.memory.importConfirm') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { importMarkdown } from '@/api/memory'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'imported'): void
}>()

const { t } = useI18n()
const message = useMessage()

const content = ref('')
const category = ref('core')
const importing = ref(false)
const preview = ref<{ key: string; content: string }[]>([])

const categoryOptions = [
  { label: 'Core', value: 'core' },
  { label: 'Daily', value: 'daily' },
  { label: 'Conversation', value: 'conversation' },
]

function truncate(s: string, n: number) {
  return s.length > n ? s.slice(0, n) + '…' : s
}

function handlePreview() {
  preview.value = splitMarkdown(content.value)
}

function splitMarkdown(md: string): { key: string; content: string }[] {
  const lines = md.split('\n')
  const sections: { key: string; content: string }[] = []
  let currentKey = ''
  let currentLines: string[] = []

  const flush = () => {
    if (!currentKey) return
    const body = currentLines.join('\n').trim()
    if (body) sections.push({ key: currentKey, content: body })
  }

  for (const line of lines) {
    if (line.startsWith('## ')) {
      flush()
      currentKey = line.replace(/^## /, '').trim()
      currentLines = []
    } else if (currentKey) {
      currentLines.push(line)
    }
  }
  flush()
  return sections
}

async function handleImport() {
  if (!content.value.trim()) return
  importing.value = true
  try {
    const res = await importMarkdown({ content: content.value, category: category.value })
    // @ts-ignore
    const n = res.imported || 0
    message.success(t('settings.memory.importSuccess', { n }))
    emit('update:show', false)
    emit('imported')
    content.value = ''
    preview.value = []
  } catch (e: any) {
    message.error(e?.message || t('common.error'))
  } finally {
    importing.value = false
  }
}

watch(() => props.show, (v) => {
  if (!v) {
    content.value = ''
    preview.value = []
  }
})
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;

.import-body {
  display: flex;
  flex-direction: column;
  gap: $spacing-4;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: $spacing-2;
}

.form-row {
  display: flex;
  gap: $spacing-4;
}

.form-label {
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  color: $color-text-primary;
}

.preview-section {
  border: 1px solid $color-border-light;
  border-radius: $radius-md;
  overflow: hidden;
}

.preview-title {
  padding: $spacing-2 $spacing-3;
  background: $color-bg-secondary;
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  color: $color-text-secondary;
  border-bottom: 1px solid $color-border-light;
}

.preview-list {
  max-height: 200px;
  overflow-y: auto;
}

.preview-item {
  display: flex;
  gap: $spacing-3;
  padding: $spacing-2 $spacing-3;
  border-bottom: 1px solid $color-border-light;
  font-size: $font-size-xs;

  &:last-child {
    border-bottom: none;
  }
}

.preview-key {
  font-family: $font-family-mono;
  color: $color-primary;
  flex-shrink: 0;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-content {
  color: $color-text-secondary;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-2;
}
</style>
