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
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/layouts/SettingsLayout.vue'),
        redirect: '/settings/providers',
        children: [
          {
            path: 'providers',
            name: 'SettingsProviders',
            component: () => import('@/pages/settings/Providers.vue')
          },
          {
            path: 'agent',
            name: 'SettingsAgent',
            component: () => import('@/pages/settings/Agent.vue')
          },
          {
            path: 'channels',
            name: 'SettingsChannels',
            component: () => import('@/pages/settings/Channels.vue')
          },
          {
            path: 'skills',
            name: 'SettingsSkills',
            component: () => import('@/pages/settings/Skills.vue')
          }
        ]
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
