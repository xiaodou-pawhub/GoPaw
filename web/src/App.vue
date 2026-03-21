<template>
  <router-view />
  <Toaster position="top-right" :theme="'dark'" richColors />

  <!-- Tool Execution Approval Dialog -->
  <ApprovalDialog
    :request="approvalRequest"
    @approve="handleApprove"
    @reject="handleReject"
  />

  <!-- Welcome 弹窗：首次启动无 LLM 配置时引导 -->
  <div v-if="showWelcome" class="modal-overlay" @click.self="showWelcome = false">
    <div class="welcome-card">
      <img src="/assets/logo.png" alt="GoPaw" class="welcome-logo" />
      <h2 class="welcome-title">欢迎使用 GoPaw</h2>
      <p class="welcome-desc">还没有配置 LLM 提供商，请先完成基础配置才能开始对话。</p>
      <div class="welcome-steps">
        <div class="step">
          <span class="step-num">1</span>
          <span>进入 <strong>模型</strong> 页面</span>
        </div>
        <div class="step">
          <span class="step-num">2</span>
          <span>添加 OpenAI / 第三方 LLM 提供商</span>
        </div>
        <div class="step">
          <span class="step-num">3</span>
          <span>返回主界面，开始对话</span>
        </div>
      </div>
      <div class="welcome-actions">
        <button class="btn-primary" @click="goToModels">去配置</button>
        <button class="btn-ghost" @click="showWelcome = false">稍后再说</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Toaster } from 'vue-sonner'
import { getMode, checkAuthStatus, setupAxiosInterceptors } from '@/api/auth'
import { getSetupStatus, getProviders } from '@/api/settings'
import ApprovalDialog from '@/components/ApprovalDialog.vue'
import { useWebSocket } from '@/composables/useWebSocket'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const appStore = useAppStore()
const showWelcome = ref(false)

// Tool approval WebSocket
const { approvalRequest, approve, reject } = useWebSocket()

interface ApprovalRequest {
  id: string
  tool_name: string
  args: string
  level: string
  requested_at: string
  session_id: string
  agent_id?: string
}

const handleApprove = (request: ApprovalRequest, _reason?: string) => {
  approve(request.id)
}

const handleReject = (request: ApprovalRequest, reason?: string) => {
  reject(request.id, reason || '')
}

onMounted(async () => {
  setupAxiosInterceptors()

  try {
    const info = await getMode()
    appStore.setModeInfo(info)

    if (info.mode === 'solo') {
      await checkSetupStatus()
    } else {
      // team 模式：检查 JWT session
      const ok = await checkAuthStatus()
      if (!ok) {
        router.push('/login')
        return
      }
      await checkSetupStatus()
    }
  } catch {
    // 后端不可达时忽略
  }

  registerHotkeys()
})

async function checkSetupStatus() {
  try {
    const [status, providers] = await Promise.all([
      getSetupStatus(),
      getProviders()
    ])
    appStore.setProviders(providers)
    if (!status.llm_configured) {
      showWelcome.value = true
    }
  } catch {
    // 接口不可用时忽略
  }
}

function goToModels() {
  showWelcome.value = false
  router.push('/models')
}

function handleGlobalKey(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return
  }

  if (e.metaKey || e.ctrlKey) {
    if (e.key === '1') { e.preventDefault(); router.push('/chat'); return }
    if (e.key === '2') { e.preventDefault(); router.push('/models'); return }
    if (e.key === '3') { e.preventDefault(); router.push('/market'); return }
    if (e.key === '4' || e.key === ',') { e.preventDefault(); router.push('/settings'); return }
  }

  if (e.key === 'Escape' && showWelcome.value) {
    showWelcome.value = false
  }
}

function registerHotkeys() {
  window.addEventListener('keydown', handleGlobalKey)
}

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKey)
})
</script>

<style>
body {
  margin: 0;
  padding: 0;
  background: var(--bg-app);
}
</style>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9000;
}

.welcome-card {
  width: 420px;
  padding: 36px 32px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.welcome-logo {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  object-fit: contain;
}

.welcome-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

.welcome-desc {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
  text-align: center;
  line-height: 1.6;
}

.welcome-steps {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--bg-app);
  border-radius: 8px;
  padding: 14px 16px;
  margin: 4px 0;
}

.step {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--text-secondary);
}

.step strong {
  color: var(--text-primary);
}

.step-num {
  width: 20px;
  height: 20px;
  background: var(--accent-dim);
  color: var(--accent);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
}

.welcome-actions {
  display: flex;
  gap: 10px;
  width: 100%;
  margin-top: 4px;
}

.btn-primary {
  flex: 1;
  padding: 10px 0;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover {
  background: var(--accent-hover);
}

.btn-ghost {
  flex: 1;
  padding: 10px 0;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.btn-ghost:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}
</style>
