<template>
  <div class="settings-container">
    <div class="settings-sidebar">
      <div class="settings-sidebar-header">
        <n-h3 class="title">{{ t('nav.settings') }}</n-h3>
        <n-text depth="3" class="subtitle">{{ t('settings.description') }}</n-text>
      </div>
      
      <n-menu
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuSelect"
        class="settings-menu"
      />
    </div>
    
    <div class="settings-main">
      <div class="settings-content">
        <transition name="fade-slide" mode="out-in">
          <router-view />
        </transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NH3, NText, NMenu, NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { HardwareChipOutline, PersonOutline, RocketOutline, BulbOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()

const activeKey = computed(() => route.path)

function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
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
  }
]

function handleMenuSelect(key: string) {
  router.push(key)
}
</script>

<style scoped lang="scss">
.settings-container {
  display: flex;
  height: 100%;
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.04);
}

.settings-sidebar {
  width: 280px;
  background: #fafafa;
  border-right: 1px solid rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  padding: 32px 12px;

  &-header {
    padding: 0 20px 24px;
    
    .title {
      margin: 0;
      font-weight: 800;
      letter-spacing: -0.5px;
    }
    .subtitle {
      font-size: 12px;
    }
  }
}

.settings-menu {
  :deep(.n-menu-item) {
    margin-bottom: 4px;
    
    .n-menu-item-content {
      padding-left: 20px !important;
      border-radius: 10px;
      
      &--selected {
        background-color: #fff !important;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
        
        &::before {
          display: none;
        }
      }
      
      &:hover {
        background-color: rgba(0, 0, 0, 0.02);
      }
    }
  }
}

.settings-main {
  flex: 1;
  overflow-y: auto;
  background: #fff;
  display: flex;
  flex-direction: column;
}

.settings-content {
  padding: 48px 64px;
  max-width: 900px;
  width: 100%;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.3s ease;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
