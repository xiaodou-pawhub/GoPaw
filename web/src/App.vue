<template>
  <n-config-provider>
    <n-message-provider>
      <n-dialog-provider>
        <!-- 未认证：全屏登录遮罩 -->
        <div v-if="!authenticated" class="auth-overlay">
          <div class="auth-card">
            <img src="/assets/logo.png" alt="GoPaw Logo" class="logo-icon" />
            <h2 class="auth-title">GoPaw</h2>
            <p class="auth-subtitle">请输入访问 Token</p>
            <n-input
              v-model:value="tokenInput"
              type="password"
              show-password-on="click"
              placeholder="粘贴访问 Token..."
              size="large"
              :disabled="logging"
              @keydown.enter="handleLogin"
            />
            <p v-if="loginError" class="auth-error">{{ loginError }}</p>
            <n-button
              type="primary"
              size="large"
              block
              :loading="logging"
              style="margin-top: 12px"
              @click="handleLogin"
            >
              进入
            </n-button>
            <p class="auth-hint">
              Token 在服务启动日志中查看，或在 config.yaml 中配置 <code>app.admin_token</code>
            </p>
          </div>
        </div>

        <!-- 已认证：正常渲染页面 -->
        <router-view v-else />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NConfigProvider, NMessageProvider, NDialogProvider, NInput, NButton } from 'naive-ui'
import { checkAuthStatus, login } from '@/api/auth'

const authenticated = ref(false)
const tokenInput = ref('')
const loginError = ref('')
const logging = ref(false)

onMounted(async () => {
  authenticated.value = await checkAuthStatus()
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
  } catch {
    loginError.value = 'Token 不正确，请重试'
  } finally {
    logging.value = false
  }
}
</script>

<style>
#app {
  width: 100%;
  height: 100vh;
}

body {
  margin: 0;
  padding: 0;
}
</style>

<style scoped>
.auth-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f0f0f;
  z-index: 9999;
}

.auth-card {
  width: 360px;
  padding: 40px 36px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.auth-logo {
  text-align: center;
  margin-bottom: 4px;
}

.logo-icon {
  width: 80px;
  height: 80px;
  object-fit: contain;
  border-radius: 12px;
}

.auth-title {
  margin: 0;
  text-align: center;
  color: #fff;
  font-size: 22px;
  font-weight: 600;
}

.auth-subtitle {
  margin: 0 0 8px;
  text-align: center;
  color: #888;
  font-size: 14px;
}

.auth-error {
  margin: 0;
  color: #f56c6c;
  font-size: 13px;
  text-align: center;
}

.auth-hint {
  margin: 8px 0 0;
  color: #555;
  font-size: 12px;
  text-align: center;
  line-height: 1.5;
}

.auth-hint code {
  color: #777;
  background: #222;
  padding: 1px 4px;
  border-radius: 3px;
}
</style>
