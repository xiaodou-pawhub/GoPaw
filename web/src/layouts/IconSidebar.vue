<template>
  <nav class="sidebar" :class="{ collapsed: isCollapsed }">
    <!-- Logo & Toggle -->
    <div class="sidebar-header">
      <div class="sidebar-logo" @click="router.push('/chat')">
        <img src="/assets/logo.png" alt="GoPaw" class="logo-img" />
        <span v-if="!isCollapsed" class="logo-text">GoPaw</span>
      </div>
      <button class="toggle-btn" @click="isCollapsed = !isCollapsed">
        <ChevronLeftIcon v-if="!isCollapsed" :size="16" />
        <ChevronRightIcon v-else :size="16" />
      </button>
    </div>

    <!-- 导航分组 -->
    <div class="sidebar-content">
      <!-- 核心功能 -->
      <div class="nav-group">
        <div v-if="!isCollapsed" class="nav-group-title">核心</div>
        <div class="nav-items">
          <button
            v-for="item in coreItems"
            :key="item.to"
            class="nav-item"
            :class="{ active: isActive(item.to) }"
            :title="isCollapsed ? item.label : ''"
            @click="router.push(item.to)"
          >
            <component :is="item.icon" :size="18" />
            <span v-if="!isCollapsed" class="nav-label">{{ item.label }}</span>
          </button>
        </div>
      </div>

      <!-- 自动化 -->
      <div class="nav-group">
        <div v-if="!isCollapsed" class="nav-group-title">自动化</div>
        <div class="nav-items">
          <button
            v-for="item in automationItems"
            :key="item.to"
            class="nav-item"
            :class="{ active: isActive(item.to) }"
            :title="isCollapsed ? item.label : ''"
            @click="router.push(item.to)"
          >
            <component :is="item.icon" :size="18" />
            <span v-if="!isCollapsed" class="nav-label">{{ item.label }}</span>
          </button>
        </div>
      </div>

      <!-- 数据与监控 -->
      <div class="nav-group">
        <div v-if="!isCollapsed" class="nav-group-title">数据</div>
        <div class="nav-items">
          <button
            v-for="item in dataItems"
            :key="item.to"
            class="nav-item"
            :class="{ active: isActive(item.to) }"
            :title="isCollapsed ? item.label : ''"
            @click="router.push(item.to)"
          >
            <component :is="item.icon" :size="18" />
            <span v-if="!isCollapsed" class="nav-label">{{ item.label }}</span>
          </button>
        </div>
      </div>

      <!-- 系统设置 -->
      <div class="nav-group">
        <div v-if="!isCollapsed" class="nav-group-title">系统</div>
        <div class="nav-items">
          <button
            v-if="appStore.isMultiUser"
            class="nav-item"
            :class="{ active: isActive('/users') }"
            :title="isCollapsed ? '用户管理' : ''"
            @click="router.push('/users')"
          >
            <UsersIcon :size="18" />
            <span v-if="!isCollapsed" class="nav-label">用户管理</span>
          </button>
          <button
            v-if="appStore.isMultiUser"
            class="nav-item"
            :class="{ active: isActive('/audit-logs') }"
            :title="isCollapsed ? '审计日志' : ''"
            @click="router.push('/audit-logs')"
          >
            <ShieldCheckIcon :size="18" />
            <span v-if="!isCollapsed" class="nav-label">审计日志</span>
          </button>
          <button
            v-for="item in systemItems"
            :key="item.to"
            class="nav-item"
            :class="{ active: isActive(item.to) }"
            :title="isCollapsed ? item.label : ''"
            @click="router.push(item.to)"
          >
            <component :is="item.icon" :size="18" />
            <span v-if="!isCollapsed" class="nav-label">{{ item.label }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 底部状态 -->
    <div class="sidebar-footer">
      <div
        class="status-indicator"
        :class="appStore.isLLMConfigured ? 'status-ok' : 'status-warn'"
      >
        <div class="status-dot" />
        <span v-if="!isCollapsed" class="status-text">
          {{ appStore.isLLMConfigured ? 'LLM 已连接' : 'LLM 未配置' }}
        </span>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  MessageSquare,
  Cpu,
  Bot,
  Store,
  Zap,
  GitBranch,
  Network,
  BookOpen,
  Layers,
  BarChart3,
  Activity,
  Server,
  MessageCircle,
  Settings,
  Users as UsersIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ShieldCheckIcon,
} from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()

// 折叠状态
const isCollapsed = ref(false)

// 核心功能
const coreItems = [
  { to: '/chat', label: '聊天', icon: MessageSquare },
  { to: '/models', label: '模型', icon: Cpu },
  { to: '/agents', label: 'Agents', icon: Bot },
  { to: '/market', label: '技能市场', icon: Store },
]

// 自动化
const automationItems = [
  { to: '/workflows', label: '工作流', icon: GitBranch },
  { to: '/orchestrations', label: '编排器', icon: Network },
  { to: '/triggers', label: '触发器', icon: Zap },
  { to: '/agent-messages', label: 'Agent消息', icon: MessageCircle },
]

// 数据与监控
const dataItems = [
  { to: '/knowledge', label: '知识库', icon: BookOpen },
  { to: '/traces', label: '执行轨迹', icon: Activity },
  { to: '/queue', label: '消息队列', icon: Layers },
  { to: '/metrics', label: '监控面板', icon: BarChart3 },
]

// 系统设置
const systemItems = [
  { to: '/mcp', label: 'MCP服务', icon: Server },
  { to: '/settings', label: '设置', icon: Settings },
]

function isActive(to: string): boolean {
  return route.path.startsWith(to)
}
</script>

<style scoped>
.sidebar {
  width: 200px;
  height: 100%;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width 0.2s ease;
}

.sidebar.collapsed {
  width: 56px;
}

/* Header */
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border-bottom: 1px solid var(--border-subtle);
}

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  border-radius: 8px;
  padding: 4px;
  transition: background 0.15s;
}

.sidebar-logo:hover {
  background: var(--bg-overlay);
}

.logo-img {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: cover;
}

.logo-text {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
}

.toggle-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
}

.toggle-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-secondary);
}

.sidebar.collapsed .sidebar-header {
  justify-content: center;
  padding: 8px;
}

.sidebar.collapsed .toggle-btn {
  display: none;
}

/* Content */
.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

/* Nav Group */
.nav-group {
  margin-bottom: 8px;
}

.nav-group-title {
  font-size: 10px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 8px 16px 4px;
}

.sidebar.collapsed .nav-group-title {
  display: none;
}

.nav-items {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 6px;
}

.sidebar.collapsed .nav-items {
  padding: 0 8px;
}

/* Nav Item */
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
  width: 100%;
}

.nav-item:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.nav-item.active {
  background: var(--accent-dim);
  color: var(--accent);
}

.sidebar.collapsed .nav-item {
  justify-content: center;
  padding: 10px;
}

.nav-label {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
}

/* Footer */
.sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--border-subtle);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--bg-app);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-ok .status-dot {
  background: var(--green);
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
}

.status-warn .status-dot {
  background: var(--yellow);
  animation: pulse-warn 2s infinite;
}

.status-text {
  font-size: 11px;
  color: var(--text-secondary);
}

.sidebar.collapsed .status-indicator {
  justify-content: center;
  padding: 8px;
}

.sidebar.collapsed .status-text {
  display: none;
}

@keyframes pulse-warn {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Scrollbar */
.sidebar-content::-webkit-scrollbar {
  width: 4px;
}

.sidebar-content::-webkit-scrollbar-track {
  background: transparent;
}

.sidebar-content::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

.sidebar-content::-webkit-scrollbar-thumb:hover {
  background: var(--text-tertiary);
}
</style>