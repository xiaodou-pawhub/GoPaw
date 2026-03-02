import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: MainLayout,
      children: [
        {
          path: '',
          redirect: '/setup'
        },
        {
          path: 'chat',
          name: 'Chat',
          component: () => import('@/pages/Chat.vue')
        },
        {
          path: 'settings',
          name: 'Settings',
          children: [
            {
              path: '',
              redirect: '/settings/providers'
            },
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
            }
          ]
        },
        {
          path: 'setup',
          name: 'Setup',
          component: () => import('@/pages/Setup.vue')
        }
      ]
    }
  ]
})

export default router
