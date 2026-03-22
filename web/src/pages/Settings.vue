<template>
  <div class="settings-root">
    <!-- 左侧 Tab 导航 -->
    <aside class="settings-nav">
      <div class="nav-section">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="nav-tab"
          :class="{ active: activeTab === tab.key }"
          @click="activeTab = tab.key"
        >
          <component :is="tab.icon" :size="14" />
          <span>{{ tab.label }}</span>
        </button>
      </div>
    </aside>

    <!-- 右侧内容区 -->
    <main class="settings-content">
      <Suspense>
        <template #default>
          <component :is="currentTabComponent" />
        </template>
        <template #fallback>
          <LoadingSpinner text="加载中..." />
        </template>
      </Suspense>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, defineAsyncComponent } from 'vue'
import {
  ScrollTextIcon
} from 'lucide-vue-next'
import LoadingSpinner from '@/components/LoadingSpinner.vue'

const tabs = [
  { key: 'logs', label: '系统日志', icon: ScrollTextIcon },
]

const activeTab = ref('logs')

// 懒加载 Tab 组件（代码分割）
const tabComponents: Record<string, any> = {
  logs: defineAsyncComponent(() => import('@/components/settings/LogsTab.vue')),
}

const currentTabComponent = computed(() => tabComponents[activeTab.value])
</script>

<style scoped>
.settings-root {
  flex: 1;
  display: flex;
  overflow: hidden;
  height: 100%;
}

.settings-nav {
  width: 180px;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  padding: 12px 8px;
  overflow-y: auto;
  flex-shrink: 0;
}

.nav-section {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.nav-section-title {
  font-size: 10px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 8px 8px 4px;
}

.nav-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 8px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
  text-align: left;
}

.nav-tab:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.nav-tab.active {
  background: var(--accent-dim);
  color: var(--accent);
}

.settings-content {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-app);
  padding: 24px 28px;
}
</style>
