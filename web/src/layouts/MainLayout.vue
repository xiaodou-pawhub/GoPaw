<template>
  <n-layout has-sider class="layout">
    <!-- 中文：侧边栏 / English: Sidebar -->
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      class="sidebar"
    >
      <div class="logo">
        <n-icon :component="Logo" :size="32" color="#18a058" />
        <span v-if="!collapsed" class="logo-text">GoPaw</span>
      </div>
      
      <n-menu
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>
    
    <!-- 中文：主内容区 / English: Main content -->
    <n-layout class="main-content">
      <router-view />
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon, NText } from 'naive-ui'
import { Logo, ChatboxEllipsesOutline, SettingsOutline, PersonOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const collapsed = ref(false)

// 中文：当前激活的菜单项
// English: Current active menu item
const activeKey = computed(() => route.path)

// 中文：渲染图标函数
// English: Render icon function
function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

// 中文：菜单选项
// English: Menu options
const menuOptions = computed(() => [
  {
    label: t('nav.chat'),
    key: '/chat',
    icon: renderIcon(ChatboxEllipsesOutline)
  },
  {
    label: t('nav.settings'),
    key: '/settings',
    icon: renderIcon(SettingsOutline),
    children: [
      {
        label: t('settings.providers.title'),
        key: '/settings/providers'
      },
      {
        label: t('settings.agent.title'),
        key: '/settings/agent'
      },
      {
        label: t('settings.channels.title'),
        key: '/settings/channels'
      }
    ]
  }
])

// 中文：处理菜单选择
// English: Handle menu select
function handleMenuSelect(key: string) {
  router.push(key)
}
</script>

<style scoped>
.layout {
  height: 100vh;
}

.sidebar {
  display: flex;
  flex-direction: column;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 60px;
  gap: 12px;
  border-bottom: 1px solid #eee;
}

.logo-text {
  font-size: 20px;
  font-weight: bold;
  color: #18a058;
}

.main-content {
  background: #f5f5f5;
}
</style>
