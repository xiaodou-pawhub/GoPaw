import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import { useAppStore } from '@/stores/app'
import { getMode } from '@/api/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/Login.vue'),
    meta: { public: true }
  },
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
        path: 'models',
        name: 'Models',
        component: () => import('@/pages/Models.vue')
      },
      {
        path: 'market',
        redirect: '/skills'
      },
      {
        path: 'skills',
        name: 'Skills',
        component: () => import('@/pages/Skills.vue')
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
        path: 'mcp',
        name: 'MCP',
        component: () => import('@/pages/MCP.vue')
      },
      {
        path: 'workflows',
        name: 'Workflows',
        component: () => import('@/pages/Workflows.vue')
      },
      {
        path: 'queue',
        name: 'Queue',
        component: () => import('@/pages/Queue.vue')
      },
      {
        path: 'metrics',
        name: 'Metrics',
        component: () => import('@/pages/Metrics.vue')
      },
      {
        path: 'knowledge',
        name: 'Knowledge',
        component: () => import('@/pages/Knowledge.vue')
      },
      {
        path: 'orchestrations',
        name: 'Orchestrations',
        component: () => import('@/pages/Orchestrations.vue')
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/pages/Users.vue')
      },
      {
        path: 'audit-logs',
        name: 'AuditLogs',
        component: () => import('@/pages/AuditLogs.vue')
      },
      {
        path: 'memory',
        name: 'Memory',
        component: () => import('@/pages/Memory.vue')
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/pages/Logs.vue')
      },
      {
        path: 'cron',
        name: 'Cron',
        component: () => import('@/pages/Cron.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true

  const appStore = useAppStore()
  if (!appStore.modeInfo) {
    try {
      const info = await getMode()
      appStore.setModeInfo(info)
    } catch {
      return true
    }
  }

  if (appStore.isSoloMode) return true

  const token = localStorage.getItem('access_token')
  if (!token) return '/login'

  return true
})

export default router
