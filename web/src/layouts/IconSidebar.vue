<template>
  <nav class="icon-sidebar">
    <!-- Logo -->
    <div class="sidebar-logo" @click="router.push('/chat')">
      <img src="/assets/logo.png" alt="GoPaw" class="logo-img" />
    </div>

    <div class="sidebar-divider" />

    <!-- 导航图标 -->
    <div class="sidebar-nav">
      <button
        v-for="item in navItems"
        :key="item.to"
        class="nav-item"
        :class="{ active: isActive(item.to) }"
        :title="item.label"
        @click="router.push(item.to)"
      >
        <component :is="item.icon" :size="18" />
      </button>
    </div>

    <div class="sidebar-spacer" />

    <!-- LLM 状态点 -->
    <div class="sidebar-bottom">
      <div
        class="status-dot"
        :class="appStore.isLLMConfigured ? 'status-ok' : 'status-warn'"
        :title="appStore.isLLMConfigured ? 'LLM 已连接' : 'LLM 未配置'"
      />
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { MessageSquare, Store, Activity, Bot, Server, Settings, Zap } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()

const navItems = [
  { to: '/chat', label: '聊天', icon: MessageSquare },
  { to: '/market', label: '市场', icon: Store },
  { to: '/traces', label: '轨迹', icon: Activity },
  { to: '/agents', label: 'Agents', icon: Bot },
  { to: '/mcp', label: 'MCP', icon: Server },
  { to: '/triggers', label: 'Triggers', icon: Zap },
  { to: '/settings', label: '设置', icon: Settings },
]

function isActive(to: string): boolean {
  if (to === '/chat') return route.path.startsWith('/chat')
  if (to === '/market') return route.path.startsWith('/market')
  if (to === '/traces') return route.path.startsWith('/traces')
  if (to === '/agents') return route.path.startsWith('/agents')
  if (to === '/mcp') return route.path.startsWith('/mcp')
  if (to === '/triggers') return route.path.startsWith('/triggers')
  if (to === '/settings') return route.path.startsWith('/settings')
  return false
}
</script>

<style scoped>
.icon-sidebar {
  width: 48px;
  height: 100%;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 0;
  flex-shrink: 0;
}

.sidebar-logo {
  width: 32px;
  height: 32px;
  cursor: pointer;
  border-radius: 8px;
  overflow: hidden;
  transition: opacity 0.15s;
}

.sidebar-logo:hover {
  opacity: 0.8;
}

.logo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.sidebar-divider {
  width: 28px;
  height: 1px;
  background: var(--border);
  margin: 10px 0;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex: 1;
}

.nav-item {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.nav-item:hover {
  background: var(--bg-overlay);
  color: var(--text-secondary);
}

.nav-item.active {
  background: var(--accent-dim);
  color: var(--accent);
}

.sidebar-spacer {
  flex: 1;
}

.sidebar-bottom {
  padding-bottom: 4px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  cursor: default;
}

.status-ok {
  background: var(--green);
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
}

.status-warn {
  background: var(--yellow);
  animation: pulse-warn 2s infinite;
}

@keyframes pulse-warn {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
