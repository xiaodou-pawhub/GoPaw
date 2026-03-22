<template>
  <div class="memory-page">
    <div class="memory-page-header">
      <div class="header-left">
        <h2 class="page-title">记忆管理</h2>
        <span class="page-subtitle">管理 Agent 的结构化记忆、每日笔记和记忆文件</span>
      </div>
      <div class="namespace-selector">
        <label class="ns-label">Agent 空间</label>
        <select v-model="selectedNamespace" class="ns-select">
          <option value="">全局（默认）</option>
          <option v-for="agent in agents" :key="agent.id" :value="agent.config?.memory?.namespace || agent.id">
            {{ agent.name }}
          </option>
        </select>
      </div>
    </div>
    <div class="memory-page-body">
      <MemoryTab :namespace="selectedNamespace || undefined" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import MemoryTab from '@/components/settings/MemoryTab.vue'
import { listAgents, type Agent } from '@/api/agents'

const agents = ref<Agent[]>([])
const selectedNamespace = ref('')

onMounted(async () => {
  try {
    const res = await listAgents()
    agents.value = res.agents
  } catch {}
})
</script>

<style scoped>
.memory-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.memory-page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px 16px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-subtle);
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
}

.namespace-selector {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ns-label {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.ns-select {
  padding: 5px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
  cursor: pointer;
  min-width: 150px;
}

.ns-select:focus {
  border-color: var(--accent);
}

.memory-page-body {
  flex: 1;
  overflow: hidden;
  padding: 16px 24px 24px;
  display: flex;
  flex-direction: column;
}

.memory-page-body :deep(.tab-root) {
  flex: 1;
  overflow: hidden;
}

.memory-page-body :deep(.tab-header) {
  display: none;
}

.memory-page-body :deep(.memory-layout) {
  flex: 1;
  min-height: 0;
}
</style>
