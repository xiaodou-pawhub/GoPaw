<template>
  <div class="memory-layout">
    <!-- Left Sidebar -->
    <aside class="memory-sidebar">
      <div class="sidebar-section">
        <div class="sidebar-label">{{ t('settings.memory.structured') }}</div>
        <nav class="sidebar-nav">
          <button
            v-for="cat in structuredCategories"
            :key="cat.key"
            class="sidebar-item"
            :class="{ active: selectedView === cat.key }"
            @click="selectView(cat.key)"
          >
            <span class="item-label">{{ cat.label }}</span>
            <span v-if="stats" class="item-badge">{{ getCatCount(cat.key) }}</span>
          </button>
        </nav>
      </div>

      <div class="sidebar-divider" />

      <div class="sidebar-section">
        <div class="sidebar-label">{{ t('settings.memory.memoryFiles') }}</div>
        <nav class="sidebar-nav">
          <button
            class="sidebar-item"
            :class="{ active: selectedView === 'memory-md' }"
            @click="selectView('memory-md')"
          >
            <span class="item-icon">📝</span>
            <span class="item-label">{{ t('settings.memory.memoryMD') }}</span>
          </button>
          <button
            class="sidebar-item"
            :class="{ active: selectedView === 'daily-notes' }"
            @click="selectView('daily-notes')"
          >
            <span class="item-icon">📅</span>
            <span class="item-label">{{ t('settings.memory.dailyNotes') }}</span>
          </button>
          <button
            class="sidebar-item"
            :class="{ active: selectedView === 'archives' }"
            @click="selectView('archives')"
          >
            <span class="item-icon">📦</span>
            <span class="item-label">{{ t('settings.memory.archives') }}</span>
          </button>
        </nav>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="memory-main">
      <!-- Structured Memory List -->
      <StructuredMemoryPanel
        v-if="isStructuredView"
        :category="currentCategory"
        @stats-change="loadStats"
      />

      <!-- MEMORY.md Editor -->
      <MemoryMDPanel v-else-if="selectedView === 'memory-md'" />

      <!-- Daily Notes -->
      <DailyNotesPanel v-else-if="selectedView === 'daily-notes'" />

      <!-- Archives -->
      <ArchivesPanel v-else-if="selectedView === 'archives'" />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMemoryStats } from '@/api/memory'
import type { MemoryStats } from '@/api/memory'
import StructuredMemoryPanel from './memory/StructuredMemoryPanel.vue'
import MemoryMDPanel from './memory/MemoryMDPanel.vue'
import DailyNotesPanel from './memory/DailyNotesPanel.vue'
import ArchivesPanel from './memory/ArchivesPanel.vue'

const { t } = useI18n()

const selectedView = ref<string>('all')
const stats = ref<MemoryStats | null>(null)

const structuredCategories = computed(() => [
  { key: 'all', label: t('settings.memory.all') },
  { key: 'core', label: t('settings.memory.core') },
  { key: 'daily', label: t('settings.memory.daily') },
  { key: 'conversation', label: t('settings.memory.conversation') },
])

const isStructuredView = computed(() =>
  ['all', 'core', 'daily', 'conversation'].includes(selectedView.value)
)

const currentCategory = computed(() =>
  selectedView.value === 'all' ? '' : selectedView.value
)

function selectView(key: string) {
  selectedView.value = key
}

function getCatCount(key: string): number {
  if (!stats.value) return 0
  if (key === 'all') return stats.value.total
  return (stats.value as any)[key] ?? 0
}

async function loadStats() {
  try {
    const res = await getMemoryStats()
    // @ts-ignore
    stats.value = res.stats
  } catch {
    // ignore
  }
}

onMounted(loadStats)
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;

.memory-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.memory-sidebar {
  width: 220px;
  flex-shrink: 0;
  background: $color-bg-primary;
  border-right: 1px solid $color-border-light;
  overflow-y: auto;
  padding: $spacing-4 0;
}

.sidebar-section {
  padding: 0 $spacing-3;
}

.sidebar-label {
  font-size: $font-size-xs;
  font-weight: $font-weight-semibold;
  color: $color-text-secondary;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: $spacing-2 $spacing-2;
  margin-bottom: $spacing-1;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: $spacing-2;
  width: 100%;
  padding: $spacing-2 $spacing-3;
  border: none;
  background: none;
  border-radius: $radius-md;
  cursor: pointer;
  color: $color-text-primary;
  font-size: $font-size-sm;
  text-align: left;
  transition: background 0.15s;

  &:hover {
    background: $color-bg-tertiary;
  }

  &.active {
    background: $color-gray-100;
    color: $color-primary;
    font-weight: $font-weight-medium;
  }
}

.item-icon {
  font-size: 14px;
}

.item-label {
  flex: 1;
}

.item-badge {
  font-size: $font-size-xs;
  color: $color-text-tertiary;
  background: $color-bg-secondary;
  border-radius: 10px;
  padding: 0 $spacing-2;
  min-width: 20px;
  text-align: center;
  line-height: 18px;
}

.sidebar-divider {
  height: 1px;
  background: $color-border-light;
  margin: $spacing-3 $spacing-3;
}

.memory-main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
