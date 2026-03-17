<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h3>{{ server.name }} - 工具列表</h3>
        <button class="close-btn" @click="$emit('close')">
          <XIcon :size="18" />
        </button>
      </div>

      <div class="modal-body">
        <div v-if="tools.length === 0" class="empty-state">
          <WrenchIcon :size="32" class="empty-icon" />
          <p>暂无可用工具</p>
          <p class="empty-hint">服务器可能未启动或不支持工具</p>
        </div>

        <div v-else class="tools-list">
          <div
            v-for="tool in tools"
            :key="tool.name"
            class="tool-item"
          >
            <div class="tool-header">
              <span class="tool-name">{{ tool.name }}</span>
            </div>
            <p class="tool-desc">{{ tool.description }}</p>
            <div v-if="tool.inputSchema" class="tool-schema">
              <details>
                <summary>查看参数</summary>
                <pre>{{ JSON.stringify(tool.inputSchema, null, 2) }}</pre>
              </details>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { XIcon, WrenchIcon } from 'lucide-vue-next'
import type { MCPServer, MCPTool } from '@/api/mcp'

// ---- Props ----
defineProps<{
  server: MCPServer
  tools: MCPTool[]
}>()

// ---- Emits ----
defineEmits<{
  close: []
}>()
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  background: var(--bg-card);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
}

.close-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-secondary);
}

.modal-body {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px;
  color: var(--text-tertiary);
}

.empty-icon {
  opacity: 0.5;
}

.empty-hint {
  font-size: 12px;
}

.tools-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tool-item {
  padding: 16px;
  background: var(--bg-overlay);
  border-radius: 8px;
}

.tool-header {
  margin-bottom: 8px;
}

.tool-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  font-family: monospace;
}

.tool-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.tool-schema details {
  font-size: 12px;
}

.tool-schema summary {
  color: var(--accent);
  cursor: pointer;
  user-select: none;
}

.tool-schema pre {
  margin-top: 8px;
  padding: 12px;
  background: var(--bg-input);
  border-radius: 6px;
  font-size: 11px;
  overflow-x: auto;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid var(--border);
}

.btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn.secondary {
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-secondary);
}

.btn.secondary:hover {
  background: var(--bg-overlay);
}
</style>
