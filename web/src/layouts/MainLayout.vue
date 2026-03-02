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
      <div class="logo-container" @click="router.push('/')">
        <n-icon :component="PawOutline" :size="32" color="#18a058" />
        <span v-if="!collapsed" class="logo-text">GoPaw</span>
      </div>
      
      <n-menu
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuSelect"
        class="side-menu"
      />
    </n-layout-sider>
    
    <!-- 主内容区 -->
    <n-layout>
      <!-- 顶部导航栏 -->
      <n-layout-header bordered class="header">
        <div class="header-left">
          <n-breadcrumb>
            <n-breadcrumb-item>{{ t('nav.chat') }}</n-breadcrumb-item>
            <n-breadcrumb-item v-if="currentRouteName">{{ currentRouteName }}</n-breadcrumb-item>
          </n-breadcrumb>
        </div>
        
        <div class="header-right">
          <!-- LLM 配置状态指示器 -->
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-tag
                :type="appStore.isLLMConfigured ? 'success' : 'warning'"
                round
                size="small"
                class="status-tag"
              >
                <template #icon>
                  <n-icon :component="appStore.isLLMConfigured ? CheckmarkCircleOutline : AlertCircleOutline" />
                </template>
                {{ appStore.isLLMConfigured ? 'LLM OK' : 'LLM Missing' }}
              </n-tag>
            </template>
            {{ appStore.isLLMConfigured ? 'LLM 已配置' : '请先配置 LLM' }}
          </n-tooltip>

          <n-divider vertical />
          
          <n-button quaternary circle @click="appStore.toggleTheme">
            <template #icon>
              <n-icon :component="appStore.isDark ? SunnyOutline : MoonOutline" />
            </template>
          </n-button>
        </div>
      </n-layout-header>

      <n-layout-content class="content-wrapper" :native-scrollbar="false">
        <div class="content-container">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent,
  NMenu, NIcon, NBreadcrumb, NBreadcrumbItem, NButton,
  NTooltip, NTag, NDivider
} from 'naive-ui'
import {
  PawOutline,
  ChatboxEllipsesOutline,
  SettingsOutline,
  TimeOutline,
  MoonOutline,
  SunnyOutline,
  CheckmarkCircleOutline,
  AlertCircleOutline,
  RocketOutline,
  PersonOutline,
  HardwareChipOutline,
  DocumentTextOutline,
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

// 当前路由名称用于面包屑
const currentRouteName = computed(() => {
  if (route.path === '/chat') return null
  if (route.path.includes('/settings/providers')) return t('nav.providers')
  if (route.path.includes('/settings/channels')) return t('nav.channels')
  if (route.path.includes('/settings/agent')) return t('nav.agent')
  if (route.path.includes('/settings/skills')) return t('nav.skills')
  if (route.path.includes('/cron')) return t('nav.cron')
  if (route.path.includes('/logs')) return t('nav.logs')
  return null
})

// 渲染图标函数
function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

// 菜单选项
const menuOptions = computed(() => [
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
    label: t('nav.logs'),
    key: '/logs',
    icon: renderIcon(DocumentTextOutline)
  },
  {
    label: t('nav.settings'),
    key: '/settings',
    icon: renderIcon(SettingsOutline),
    children: [
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
        label: t('settings.skills.title'),
        key: '/settings/skills',
        icon: renderIcon(BulbOutline)
      }
    ]
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
  background-color: #1a1a2e; // 深色侧边栏
  height: 100vh;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
  
  :deep(.n-layout-sider-scroll-container) {
    display: flex;
    flex-direction: column;
  }
}

.logo-container {
  display: flex;
  align-items: center;
  padding: 20px 24px;
  gap: 12px;
  cursor: pointer;
  transition: all 0.3s;
  
  &:hover {
    opacity: 0.8;
  }
}

.logo-text {
  font-size: 22px;
  font-weight: 800;
  color: #fff;
  letter-spacing: 1px;
}

.side-menu {
  margin-top: 12px;
  
  // 侧边栏菜单深色适配
  :deep(.n-menu-item-content-header) {
    color: rgba(255, 255, 255, 0.8) !important;
  }
  
  :deep(.n-menu-item-content--selected) {
    .n-menu-item-content-header {
      color: #fff !important;
      font-weight: 600;
    }
  }
  
  :deep(.n-menu-item-content:hover) {
    .n-menu-item-content-header {
      color: #fff !important;
    }
  }
}

.header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: #fff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  z-index: 10;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.status-tag {
  cursor: default;
  font-weight: 600;
}

.content-wrapper {
  background-color: #f3f4f6;
  height: calc(100vh - 64px);
}

.content-container {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
  min-height: 100%;
}
</style>
