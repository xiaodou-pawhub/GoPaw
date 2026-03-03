<template>
  <n-layout has-sider class="layout">
    <!-- 深色侧边栏 -->
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      v-model:collapsed="collapsed"
      class="sidebar"
      :native-scrollbar="false"
    >
      <div class="logo-container" @click="router.push('/chat')">
        <n-icon :component="PawOutline" :size="28" color="#18a058" />
        <div v-if="!collapsed" class="logo-text-wrapper">
          <span class="logo-text">GoPaw</span>
          <span class="logo-status" :class="{ 'status-ok': appStore.isLLMConfigured, 'status-warning': !appStore.isLLMConfigured }">
            {{ appStore.isLLMConfigured ? 'LLM Ready' : '待配置' }}
          </span>
        </div>
      </div>
      
      <n-menu
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuSelect"
        class="side-menu"
      />

      <!-- 底部状态区域 -->
      <div v-if="!collapsed" class="sidebar-footer">
        <n-tooltip trigger="hover" placement="right">
          <template #trigger>
            <div class="status-indicator" :class="{ 'connected': appStore.isLLMConfigured, 'disconnected': !appStore.isLLMConfigured }">
              <n-icon :component="appStore.isLLMConfigured ? CheckmarkCircleOutline : AlertCircleOutline" />
              <span>{{ appStore.isLLMConfigured ? 'LLM 已连接' : 'LLM 未配置' }}</span>
            </div>
          </template>
          {{ appStore.isLLMConfigured ? '大语言模型服务已就绪' : '请先在"模型配置"中添加 API Key' }}
        </n-tooltip>
      </div>
    </n-layout-sider>
    
    <!-- 主内容区 -->
    <n-layout>
      <n-layout-content class="content-wrapper" :native-scrollbar="false">
        <div class="content-container">
          <div class="page-layout">
            <div class="right-content full-width">
              <transition name="fade-slide" mode="out-in">
                <router-view />
              </transition>
            </div>
          </div>
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutContent,
  NMenu, NIcon, NTooltip
} from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import {
  PawOutline,
  ChatboxEllipsesOutline,
  TimeOutline,
  CheckmarkCircleOutline,
  AlertCircleOutline,
  DocumentTextOutline,
  HardwareChipOutline,
  PersonOutline,
  RocketOutline,
  BulbOutline
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()

const collapsed = ref(false)

// 当前激活的菜单项
const activeKey = computed(() => route.path)

// 渲染图标函数
function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

// 一级菜单选项
const menuOptions = computed<MenuOption[]>(() => [
  {
    label: t('nav.chat'),
    key: '/chat',
    icon: renderIcon(ChatboxEllipsesOutline)
  },
  {
    label: t('nav.cron'),
    key: '/cron',
    icon: renderIcon(TimeOutline)
  },
  {
    label: t('settings.providers.title'),
    key: '/settings/providers',
    icon: renderIcon(HardwareChipOutline)
  },
  {
    label: t('settings.agent.title'),
    key: '/settings/agent',
    icon: renderIcon(PersonOutline)
  },
  {
    label: t('settings.channels.title'),
    key: '/settings/channels',
    icon: renderIcon(RocketOutline)
  },
  {
    label: t('nav.skills'),
    key: '/settings/skills',
    icon: renderIcon(BulbOutline)
  },
  {
    label: t('nav.logs'),
    key: '/logs',
    icon: renderIcon(DocumentTextOutline)
  }
])

// 处理菜单选择
function handleMenuSelect(key: string) {
  router.push(key)
}
</script>

<style scoped lang="scss">
.layout {
  height: 100vh;
  background-color: #f9fafb;
}

.sidebar {
  background-color: #ffffff;
  height: 100vh;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  
  :deep(.n-layout-sider-scroll-container) {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
}

.logo-container {
  display: flex;
  align-items: center;
  padding: 20px;
  gap: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  border-bottom: 1px solid #f3f4f6;
  
  &:hover {
    background-color: #f9fafb;
  }
}

.logo-text-wrapper {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #1f2937;
  letter-spacing: -0.5px;
}

.logo-status {
  font-size: 11px;
  font-weight: 500;
  
  &.status-ok {
    color: #10b981;
  }
  
  &.status-warning {
    color: #f59e0b;
  }
}

.side-menu {
  margin-top: 8px;
  padding: 0 8px;
  flex: 1;
  
  :deep(.n-menu-item-content) {
    border-radius: 8px;
    margin: 4px 0;
  }
  
  :deep(.n-menu-item-content-header) {
    color: #6b7280 !important;
    font-weight: 500;
  }
  
  :deep(.n-menu-item-content--selected) {
    background-color: #f0fdf4 !important;
    
    .n-menu-item-content-header {
      color: #16a34a !important;
      font-weight: 600;
    }
  }
  
  :deep(.n-menu-item-content:hover) {
    background-color: #f9fafb !important;
    
    .n-menu-item-content-header {
      color: #374151 !important;
    }
  }
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid #f3f4f6;
  margin-top: auto;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  
  &.connected {
    background-color: #f0fdf4;
    color: #16a34a;
  }
  
  &.disconnected {
    background-color: #fffbeb;
    color: #d97706;
  }
}

.content-wrapper {
  background-color: #f9fafb;
  height: 100vh;
}

.content-container {
  padding: 0;
  max-width: 100%;
  margin: 0;
  min-height: 100%;
}

.page-layout {
  display: flex;
  height: 100vh;
  background: #ffffff;
}

.right-content {
  flex: 1;
  overflow-y: auto;
  background: #f9fafb;
  display: flex;
  flex-direction: column;
}

// 过渡动画
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}
</style>
