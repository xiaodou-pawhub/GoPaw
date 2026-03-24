<template>
  <div class="login-page">
    <div class="login-container">
      <div class="login-header">
        <div class="logo">
          <img src="/assets/logo.png" alt="GoPaw" class="logo-img" />
          <span class="logo-text">GoPaw</span>
        </div>
        <h1 class="title">登录</h1>
        <p class="subtitle">欢迎回来</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="username">用户名</label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            placeholder="请输入用户名"
            required
            autocomplete="username"
          />
        </div>

        <div class="form-group">
          <label for="password">密码</label>
          <div class="password-input">
            <input
              id="password"
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="请输入密码"
              required
              autocomplete="current-password"
            />
            <button type="button" class="toggle-password" @click="showPassword = !showPassword">
              <EyeIcon v-if="!showPassword" :size="18" />
              <EyeOffIcon v-else :size="18" />
            </button>
          </div>
        </div>

        <button type="submit" class="submit-btn" :disabled="loading">
          <span v-if="loading" class="loading-dot" />
          <span v-else>登录</span>
        </button>

        <div v-if="error" class="error-message">
          <AlertCircleIcon :size="16" />
          <span>{{ error }}</span>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { EyeIcon, EyeOffIcon, AlertCircleIcon } from 'lucide-vue-next'
import { loginWithPassword, getMode } from '@/api/auth'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const appStore = useAppStore()

const form = ref({ username: '', password: '' })
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!form.value.username || !form.value.password) return

  loading.value = true
  error.value = ''

  try {
    const data = await loginWithPassword(form.value.username, form.value.password)
    localStorage.setItem('access_token', data.access_token)
    
    // 登录成功后刷新 mode 信息
    const modeInfo = await getMode()
    appStore.setModeInfo(modeInfo)
    
    router.push('/')
  } catch (err: any) {
    error.value = err.response?.data?.error || '用户名或密码错误'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-app);
  padding: 20px;
}

.login-container {
  width: 100%;
  max-width: 380px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 40px 36px;
}

.login-header {
  text-align: center;
  margin-bottom: 28px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 16px;
}

.logo-img {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  object-fit: contain;
}

.logo-text {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

.title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 6px;
}

.subtitle {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}

.form-group input {
  height: 40px;
  padding: 0 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
  width: 100%;
}

.form-group input:focus {
  border-color: var(--accent);
}

.form-group input::placeholder {
  color: var(--text-tertiary);
}

.password-input {
  position: relative;
}

.password-input input {
  padding-right: 40px;
}

.toggle-password {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-tertiary);
  padding: 4px;
  display: flex;
  align-items: center;
}

.toggle-password:hover {
  color: var(--text-secondary);
}

.submit-btn {
  height: 42px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 4px;
}

.submit-btn:hover:not(:disabled) {
  background: var(--accent-hover);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-dot {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-message {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  background: var(--red-dim, rgba(239, 68, 68, 0.1));
  border-radius: 6px;
  color: var(--red, #ef4444);
  font-size: 13px;
}
</style>
