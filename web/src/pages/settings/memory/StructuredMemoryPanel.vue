<template>
  <div class="panel">
    <!-- Header -->
    <div class="panel-header">
      <div class="header-left">
        <h2 class="panel-title">
          {{ categoryLabel }}
          <span class="count-badge">{{ total }}</span>
        </h2>
      </div>
      <div class="header-actions">
        <n-input
          v-model:value="searchQuery"
          :placeholder="t('settings.memory.search')"
          size="small"
          clearable
          style="width: 220px"
          @update:value="onSearch"
        >
          <template #prefix>
            <n-icon><SearchOutline /></n-icon>
          </template>
        </n-input>
        <n-button size="small" @click="showImport = true">
          {{ t('settings.memory.importMD') }}
        </n-button>
        <n-button type="primary" size="small" @click="openCreate">
          + {{ t('settings.memory.newMemory') }}
        </n-button>
      </div>
    </div>

    <!-- List -->
    <div class="panel-body" v-loading="loading">
      <n-empty v-if="!loading && memories.length === 0" :description="t('settings.memory.noResults')" />

      <div v-else class="memory-list">
        <div
          v-for="entry in memories"
          :key="entry.id"
          class="memory-card"
          @click="openEdit(entry)"
        >
          <div class="card-header">
            <div class="card-key">
              <n-tag size="tiny" :type="catType(entry.category)" round>{{ entry.category }}</n-tag>
              <span class="key-text">{{ entry.key }}</span>
            </div>
            <div class="card-actions" @click.stop>
              <n-button text size="small" @click="openEdit(entry)">
                <template #icon><n-icon><PencilOutline /></n-icon></template>
              </n-button>
              <n-button text size="small" type="error" @click="handleDelete(entry)">
                <template #icon><n-icon><TrashOutline /></n-icon></template>
              </n-button>
            </div>
          </div>
          <div class="card-content">{{ truncate(entry.content, 120) }}</div>
          <div class="card-meta">
            {{ t('settings.memory.updatedAt') }} {{ formatTime(entry.updated_at) }}
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Drawer -->
    <n-drawer v-model:show="showDrawer" :width="480" placement="right">
      <n-drawer-content :title="isEditing ? t('settings.memory.edit') : t('settings.memory.newMemory')" closable>
        <div class="drawer-form">
          <div class="form-item">
            <label class="form-label">{{ t('settings.memory.keyLabel') }}</label>
            <n-input
              v-model:value="form.key"
              :placeholder="t('settings.memory.keyPlaceholder')"
              :disabled="isEditing"
            />
            <p v-if="!isEditing && keyExists" class="form-hint warn">
              {{ t('settings.memory.keyExists') }}
            </p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ t('settings.memory.categoryLabel') }}</label>
            <n-select
              v-model:value="form.category"
              :options="categoryOptions"
            />
          </div>
          <div class="form-item">
            <label class="form-label">{{ t('settings.memory.contentLabel') }}</label>
            <n-input
              v-model:value="form.content"
              type="textarea"
              :placeholder="t('settings.memory.contentPlaceholder')"
              :autosize="{ minRows: 8, maxRows: 20 }"
              style="font-family: var(--font-mono)"
            />
          </div>
        </div>

        <template #footer>
          <div class="drawer-footer">
            <n-button @click="showDrawer = false">{{ t('settings.memory.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ t('settings.memory.save') }}
            </n-button>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- Import Modal -->
    <ImportMDModal
      v-model:show="showImport"
      @imported="loadMemories"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import { SearchOutline, PencilOutline, TrashOutline } from '@vicons/ionicons5'
import {
  listMemories, createMemory, updateMemory, deleteMemory,
  type MemoryEntry
} from '@/api/memory'
import ImportMDModal from './ImportMDModal.vue'

const props = defineProps<{ category: string }>()
const emit = defineEmits<{ (e: 'stats-change'): void }>()

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const memories = ref<MemoryEntry[]>([])
const total = ref(0)
const loading = ref(false)
const searchQuery = ref('')
const showDrawer = ref(false)
const showImport = ref(false)
const saving = ref(false)
const isEditing = ref(false)
const keyExists = ref(false)

const form = ref({ key: '', content: '', category: 'core' })

const categoryLabel = computed(() => {
  if (!props.category) return t('settings.memory.all')
  return t(`settings.memory.${props.category}`)
})

const categoryOptions = [
  { label: 'Core', value: 'core' },
  { label: 'Daily', value: 'daily' },
  { label: 'Conversation', value: 'conversation' },
]

function catType(cat: string) {
  if (cat === 'core') return 'success'
  if (cat === 'daily') return 'info'
  if (cat === 'conversation') return 'warning'
  return 'default'
}

function truncate(s: string, n: number) {
  return s.length > n ? s.slice(0, n) + '…' : s
}

function formatTime(ms: number) {
  if (!ms) return ''
  const d = new Date(ms)
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

let searchTimer: ReturnType<typeof setTimeout> | null = null
function onSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(loadMemories, 300)
}

async function loadMemories() {
  loading.value = true
  try {
    const res = await listMemories({
      category: props.category || undefined,
      q: searchQuery.value || undefined,
      limit: 100,
    })
    // @ts-ignore
    memories.value = res.memories || []
    // @ts-ignore
    total.value = res.total || 0
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEditing.value = false
  keyExists.value = false
  form.value = { key: '', content: '', category: props.category || 'core' }
  showDrawer.value = true
}

function openEdit(entry: MemoryEntry) {
  isEditing.value = true
  form.value = { key: entry.key, content: entry.content, category: entry.category }
  showDrawer.value = true
}

async function handleSave() {
  if (!form.value.key.trim() || !form.value.content.trim()) {
    message.warning('Key 和内容不能为空')
    return
  }
  saving.value = true
  try {
    if (isEditing.value) {
      await updateMemory(form.value.key, { content: form.value.content, category: form.value.category })
      message.success(t('settings.memory.updateSuccess'))
    } else {
      await createMemory(form.value)
      message.success(t('settings.memory.createSuccess'))
    }
    showDrawer.value = false
    await loadMemories()
    emit('stats-change')
  } catch (e: any) {
    message.error(e?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

function handleDelete(entry: MemoryEntry) {
  dialog.warning({
    title: t('settings.memory.delete'),
    content: t('settings.memory.deleteConfirm', { key: entry.key }),
    positiveText: t('settings.memory.delete'),
    negativeText: t('settings.memory.cancel'),
    onPositiveClick: async () => {
      try {
        await deleteMemory(entry.key)
        message.success(t('settings.memory.deleteSuccess'))
        await loadMemories()
        emit('stats-change')
      } catch (e: any) {
        message.error(e?.message || t('common.error'))
      }
    },
  })
}

watch(() => props.category, loadMemories)
onMounted(loadMemories)
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
  gap: $spacing-3;
  flex-wrap: wrap;
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
  display: flex;
  align-items: center;
  gap: $spacing-2;
}

.count-badge {
  font-size: $font-size-sm;
  color: $color-text-secondary;
  font-weight: $font-weight-normal;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: $spacing-2;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-4 $spacing-6;
}

.memory-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-3;
}

.memory-card {
  padding: $spacing-4;
  border: 1px solid $color-border-light;
  border-radius: $radius-lg;
  background: $color-bg-primary;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;

  &:hover {
    border-color: $color-primary;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  }
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-2;
}

.card-key {
  display: flex;
  align-items: center;
  gap: $spacing-2;
}

.key-text {
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  color: $color-text-primary;
  font-family: $font-family-mono;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: $spacing-1;
  opacity: 0;
  transition: opacity 0.15s;

  .memory-card:hover & {
    opacity: 1;
  }
}

.card-content {
  font-size: $font-size-sm;
  color: $color-text-secondary;
  line-height: $line-height-relaxed;
  margin-bottom: $spacing-2;
  white-space: pre-wrap;
}

.card-meta {
  font-size: $font-size-xs;
  color: $color-text-tertiary;
}

.drawer-form {
  display: flex;
  flex-direction: column;
  gap: $spacing-5;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: $spacing-2;
}

.form-label {
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  color: $color-text-primary;
}

.form-hint {
  font-size: $font-size-xs;
  color: $color-text-secondary;

  &.warn {
    color: $color-warning;
  }
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-2;
}
</style>
