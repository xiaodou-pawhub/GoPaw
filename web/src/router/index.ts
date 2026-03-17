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
        redirect: '/chat'
      },
      {
        path: 'chat/:id?',
        name: 'Chat',
        component: () => import('@/pages/Chat.vue')
      },
      {
        path: 'market',
        name: 'Market',
        component: () => import('@/pages/Market.vue')
      },
      {
        path: 'traces',
        name: 'Traces',
        component: () => import('@/pages/Traces.vue')
      },
      {
        path: 'agents',
        name: 'Agents',
        component: () => import('@/pages/Agents.vue')
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/pages/Settings.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
