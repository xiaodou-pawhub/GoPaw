import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: MainLayout,
    children: [
      {
        path: '',
        redirect: '/setup'
      },
      {
        path: 'setup',
        name: 'Setup',
        component: () => import('@/pages/Setup.vue')
      },
      {
        path: 'chat/:id?',
        name: 'Chat',
        component: () => import('@/pages/Chat.vue')
      },
      {
        path: 'cron',
        name: 'Cron',
        component: () => import('@/pages/Cron.vue')
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/pages/Logs.vue')
      },
      // 设置页面 - 扁平化路由，直接在 MainLayout 中显示
      {
        path: 'settings/providers',
        name: 'SettingsProviders',
        component: () => import('@/pages/settings/Providers.vue')
      },
      {
        path: 'settings/agent',
        name: 'SettingsAgent',
        component: () => import('@/pages/settings/Agent.vue')
      },
      {
        path: 'settings/channels',
        name: 'SettingsChannels',
        component: () => import('@/pages/settings/Channels.vue')
      },
      {
        path: 'settings/skills',
        name: 'SettingsSkills',
        component: () => import('@/pages/settings/Skills.vue')
      },
      {
        path: 'settings/context',
        name: 'SettingsContext',
        component: () => import('@/pages/settings/Context.vue')
      },
      {
        path: 'settings/memory',
        name: 'SettingsMemory',
        component: () => import('@/pages/settings/Memory.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router