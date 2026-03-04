<template>
  <div class="panel">
    <div class="panel-header">
      <h2 class="panel-title">{{ t('settings.memory.archives') }}</h2>
      <span class="readonly-badge">{{ t('settings.memory.readOnly') }}</span>
    </div>

    <div class="panel-body">
      <div class="archives-layout">
        <!-- Archive file list -->
        <div class="archive-list">
          <div class="list-header">{{ archives.length }} 个归档文件</div>
          <n-empty v-if="archives.length === 0" description="暂无归档文件" size="small" />
          <div
            v-for="arch in archives"
            :key="arch.name"
            class="archive-item"
            :class="{ active: selectedArchive === arch.name }"
            @click="selectArchive(arch.name)"
          >
            <span class="arch-name">{{ arch.name }}</span>
            <span class="arch-size">{{ formatSize(arch.size) }}</span>
          </div>
        </div>

        <!-- Archive content viewer -->
        <div class="archive-viewer">
          <div v-if="!selectedArchive" class="empty-state">
            <n-empty description="选择左侧归档文件查看内容" />
          </div>
          <div v-else>
            <div class="viewer-header">
              <span class="viewer-name">{{ selectedArchive }}</span>
              <span class="viewer-hint">只读模式</span>
            </div>
            <pre class="viewer-content">{{ archiveContent }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { listArchives, getArchive, type ArchiveFileInfo } from '@/api/memory'

const { t } = useI18n()

const archives = ref<ArchiveFileInfo[]>([])
const selectedArchive = ref('')
const archiveContent = ref('')

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes}B`
  return `${(bytes / 1024).toFixed(1)}KB`
}

async function selectArchive(name: string) {
  selectedArchive.value = name
  try {
    const res = await getArchive(name)
    // @ts-ignore
    archiveContent.value = res.content || ''
  } catch {
    archiveContent.value = '加载失败'
  }
}

onMounted(async () => {
  try {
    const res = await listArchives()
    // @ts-ignore
    archives.value = res.archives || []
  } catch {}
})
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
  gap: $spacing-3;
  padding: $spacing-4 $spacing-6;
  border-bottom: 1px solid $color-border-light;
  flex-shrink: 0;
}

.panel-title {
  font-size: $font-size-lg;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
  margin: 0;
}

.readonly-badge {
  font-size: $font-size-xs;
  color: $color-text-tertiary;
  background: $color-bg-secondary;
  padding: 2px $spacing-2;
  border-radius: $radius-sm;
  border: 1px solid $color-border-light;
}

.panel-body {
  flex: 1;
  overflow: hidden;
}

.archives-layout {
  display: flex;
  height: 100%;
}

.archive-list {
  width: 200px;
  flex-shrink: 0;
  border-right: 1px solid $color-border-light;
  overflow-y: auto;
  padding: $spacing-2;
}

.list-header {
  font-size: $font-size-xs;
  color: $color-text-secondary;
  padding: $spacing-2;
}

.archive-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-2 $spacing-3;
  border-radius: $radius-md;
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: $color-bg-tertiary;
  }

  &.active {
    background: $color-gray-100;
    color: $color-primary;
  }
}

.arch-name {
  font-size: $font-size-xs;
  font-family: $font-family-mono;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.arch-size {
  font-size: $font-size-xs;
  color: $color-text-tertiary;
  flex-shrink: 0;
}

.archive-viewer {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-4 $spacing-6;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
}

.viewer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-3;
}

.viewer-name {
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  font-family: $font-family-mono;
  color: $color-text-primary;
}

.viewer-hint {
  font-size: $font-size-xs;
  color: $color-text-tertiary;
}

.viewer-content {
  font-family: $font-family-mono;
  font-size: $font-size-xs;
  line-height: $line-height-relaxed;
  color: $color-text-secondary;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
