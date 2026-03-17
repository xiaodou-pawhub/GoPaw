<template>
  <div v-if="!authenticated" class="auth-overlay">
    <div class="auth-card">
      <img src="/assets/logo.png" alt="GoPaw Logo" class="logo-icon" />
      <h2 class="auth-title">GoPaw</h2>
      <p class="auth-subtitle">请输入访问 Token</p>
      <input
        v-model="tokenInput"
        type="password"
        placeholder="粘贴访问 Token..."
        class="auth-input"
        :disabled="logging"
        @keydown.enter="handleLogin"
      />
      <p v-if="loginError" class="auth-error">{{ loginError }}</p>
      <button
        class="auth-btn"
        :disabled="logging"
        @click="handleLogin"
      >
        {{ logging ? '验证中...' : '进入' }}
      </button>
      <p class="auth-hint">
        Token 在服务启动日志中查看，或在 config.yaml 中配置 <code>app.admin_token</code>
      </p>
    </div>
  </div>

  <template v-else>
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
            <span>进入 <strong>设置 → 模型配置</strong></span>
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
          <button class="btn-primary" @click="goToSettings">去配置</button>
          <button class="btn-ghost" @click="showWelcome = false">稍后再说</button>
        </div>
      </div>
    </div>
  </template>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Toaster } from 'vue-sonner'
import { checkAuthStatus, login } from '@/api/auth'
import { getSetupStatus } from '@/api/settings'
import ApprovalDialog from '@/components/ApprovalDialog.vue'
import { useWebSocket } from '@/composables/useWebSocket'

const router = useRouter()
const authenticated = ref(false)
const tokenInput = ref('')
const loginError = ref('')
const logging = ref(false)
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
  authenticated.value = await checkAuthStatus()
  if (authenticated.value) {
    checkSetupStatus()
    registerHotkeys()
  }
})

async function handleLogin() {
  const token = tokenInput.value.trim()
  if (!token) {
    loginError.value = 'Token 不能为空'
    return
  }
  logging.value = true
  loginError.value = ''
  try {
    await login(token)
    authenticated.value = true
    tokenInput.value = ''
    checkSetupStatus()
    registerHotkeys()
  } catch {
    loginError.value = 'Token 不正确，请重试'
  } finally {
    logging.value = false
  }
}

async function checkSetupStatus() {
  try {
    const status = await getSetupStatus()
    if (!status.llm_configured) {
      showWelcome.value = true
    }
  } catch {
    // 接口不可用时忽略
  }
}

function goToSettings() {
  showWelcome.value = false
  router.push('/settings')
}

// 全局键盘快捷键
function handleGlobalKey(e: KeyboardEvent) {
  // 避免在输入框内触发
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return
  }
  
  if (e.metaKey || e.ctrlKey) {
    if (e.key === '1') { e.preventDefault(); router.push('/chat'); return }
    if (e.key === '2') { e.preventDefault(); router.push('/market'); return }
    if (e.key === '3' || e.key === ',') { e.preventDefault(); router.push('/settings'); return }
  }

  // Escape 关闭 Welcome 弹窗
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
.auth-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-app);
  z-index: 9999;
}

.auth-card {
  width: 360px;
  padding: 40px 36px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.logo-icon {
  width: 72px;
  height: 72px;
  object-fit: contain;
  border-radius: 12px;
  margin: 0 auto 4px;
  display: block;
}

.auth-title {
  margin: 0;
  text-align: center;
  color: var(--text-primary);
  font-size: 22px;
  font-weight: 700;
}

.auth-subtitle {
  margin: 0 0 8px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
}

.auth-input {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.auth-input:focus {
  border-color: var(--accent);
}

.auth-error {
  margin: 0;
  color: var(--red);
  font-size: 12px;
  text-align: center;
}

.auth-btn {
  margin-top: 4px;
  width: 100%;
  padding: 11px 0;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.auth-btn:hover:not(:disabled) {
  background: var(--accent-hover);
}

.auth-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.auth-hint {
  margin: 4px 0 0;
  color: var(--text-tertiary);
  font-size: 11px;
  text-align: center;
  line-height: 1.5;
}

.auth-hint code {
  color: var(--text-secondary);
  background: var(--bg-overlay);
  padding: 1px 4px;
  border-radius: 3px;
  font-family: "SF Mono", Menlo, monospace;
}

/* Welcome 弹窗 */
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
