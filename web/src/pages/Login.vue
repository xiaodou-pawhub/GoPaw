<template>
  <div class="login-page">
    <div class="login-container">
      <!-- Logo and Title -->
      <div class="login-header">
        <div class="logo">
          <span class="logo-icon">🐾</span>
          <span class="logo-text">GoPaw</span>
        </div>
        <h1 class="title">{{ isLogin ? '登录' : '注册' }}</h1>
        <p class="subtitle">{{ isLogin ? '欢迎回来' : '创建您的账户' }}</p>
      </div>

      <!-- Login Form -->
      <form v-if="isLogin" class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="username">用户名或邮箱</label>
          <input
            id="username"
            v-model="loginForm.username"
            type="text"
            placeholder="请输入用户名或邮箱"
            required
            autocomplete="username"
          />
        </div>

        <div class="form-group">
          <label for="password">密码</label>
          <div class="password-input">
            <input
              id="password"
              v-model="loginForm.password"
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

        <div class="form-options">
          <label class="remember-me">
            <input v-model="loginForm.remember" type="checkbox" />
            <span>记住我</span>
          </label>
          <a href="#" class="forgot-password">忘记密码？</a>
        </div>

        <button type="submit" class="submit-btn" :disabled="loading">
          <LoadingSpinner v-if="loading" size="small" />
          <span v-else>登录</span>
        </button>

        <div class="form-footer">
          <span>还没有账户？</span>
          <a href="#" @click.prevent="isLogin = false">立即注册</a>
        </div>
      </form>

      <!-- Register Form -->
      <form v-else class="login-form" @submit.prevent="handleRegister">
        <div class="form-group">
          <label for="reg-username">用户名</label>
          <input
            id="reg-username"
            v-model="registerForm.username"
            type="text"
            placeholder="请输入用户名"
            required
            minlength="3"
            maxlength="50"
            autocomplete="username"
          />
        </div>

        <div class="form-group">
          <label for="reg-email">邮箱</label>
          <input
            id="reg-email"
            v-model="registerForm.email"
            type="email"
            placeholder="请输入邮箱"
            required
            autocomplete="email"
          />
        </div>

        <div class="form-group">
          <label for="reg-display-name">显示名称（可选）</label>
          <input
            id="reg-display-name"
            v-model="registerForm.display_name"
            type="text"
            placeholder="请输入显示名称"
            autocomplete="name"
          />
        </div>

        <div class="form-group">
          <label for="reg-password">密码</label>
          <div class="password-input">
            <input
              id="reg-password"
              v-model="registerForm.password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="请输入密码（至少8位）"
              required
              minlength="8"
              autocomplete="new-password"
            />
            <button type="button" class="toggle-password" @click="showPassword = !showPassword">
              <EyeIcon v-if="!showPassword" :size="18" />
              <EyeOffIcon v-else :size="18" />
            </button>
          </div>
        </div>

        <div class="form-group">
          <label for="reg-confirm-password">确认密码</label>
          <div class="password-input">
            <input
              id="reg-confirm-password"
              v-model="registerForm.confirmPassword"
              :type="showConfirmPassword ? 'text' : 'password'"
              placeholder="请再次输入密码"
              required
              autocomplete="new-password"
            />
            <button type="button" class="toggle-password" @click="showConfirmPassword = !showConfirmPassword">
              <EyeIcon v-if="!showConfirmPassword" :size="18" />
              <EyeOffIcon v-else :size="18" />
            </button>
          </div>
          <p v-if="passwordMismatch" class="error-text">两次输入的密码不一致</p>
        </div>

        <button type="submit" class="submit-btn" :disabled="loading || passwordMismatch">
          <LoadingSpinner v-if="loading" size="small" />
          <span v-else>注册</span>
        </button>

        <div class="form-footer">
          <span>已有账户？</span>
          <a href="#" @click.prevent="isLogin = true">立即登录</a>
        </div>
      </form>

      <!-- Error Message -->
      <div v-if="error" class="error-message">
        <AlertCircleIcon :size="18" />
        <span>{{ error }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { EyeIcon, EyeOffIcon, AlertCircleIcon } from 'lucide-vue-next'
import { authApi, tokenStorage, setupAxiosInterceptors } from '@/api/auth'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const isLogin = ref(true)
const loading = ref(false)
const error = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)

const loginForm = ref({
  username: '',
  password: '',
  remember: false,
})

const registerForm = ref({
  username: '',
  email: '',
  display_name: '',
  password: '',
  confirmPassword: '',
})

const passwordMismatch = computed(() => {
  return !!(registerForm.value.password && registerForm.value.confirmPassword &&
         registerForm.value.password !== registerForm.value.confirmPassword)
})

async function handleLogin() {
  if (!loginForm.value.username || !loginForm.value.password) return

  loading.value = true
  error.value = ''

  try {
    const response = await authApi.login({
      username: loginForm.value.username,
      password: loginForm.value.password,
    })

    if (response.code === 200) {
      // Save tokens
      tokenStorage.setTokens(response.data.tokens)
      
      // Save user info
      userStore.setUser(response.data.user)
      
      // Setup axios interceptors
      setupAxiosInterceptors()
      
      // Redirect to home
      router.push('/')
    } else {
      error.value = response.message || '登录失败'
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '登录失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}

async function handleRegister() {
  if (!registerForm.value.username || !registerForm.value.email || !registerForm.value.password) return
  if (passwordMismatch.value) return

  loading.value = true
  error.value = ''

  try {
    const response = await authApi.register({
      username: registerForm.value.username,
      email: registerForm.value.email,
      password: registerForm.value.password,
      display_name: registerForm.value.display_name || undefined,
    })

    if (response.code === 201) {
      // Save tokens
      tokenStorage.setTokens(response.data.tokens)
      
      // Save user info
      userStore.setUser(response.data.user)
      
      // Setup axios interceptors
      setupAxiosInterceptors()
      
      // Redirect to home
      router.push('/')
    } else {
      error.value = response.message || '注册失败'
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '注册失败，请检查网络连接'
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-container {
  width: 100%;
  max-width: 420px;
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  padding: 40px;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 16px;
}

.logo-icon {
  font-size: 32px;
}

.logo-text {
  font-size: 24px;
  font-weight: 700;
  color: #333;
}

.title {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px;
}

.subtitle {
  font-size: 14px;
  color: #666;
  margin: 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.form-group input {
  height: 44px;
  padding: 0 14px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.2s;
}

.form-group input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-group input::placeholder {
  color: #aaa;
}

.password-input {
  position: relative;
}

.password-input input {
  width: 100%;
  padding-right: 44px;
}

.toggle-password {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  color: #666;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.toggle-password:hover {
  color: #333;
}

.error-text {
  font-size: 12px;
  color: #e53935;
  margin: 0;
}

.form-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
}

.remember-me {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #666;
}

.remember-me input {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.forgot-password {
  color: #667eea;
  text-decoration: none;
}

.forgot-password:hover {
  text-decoration: underline;
}

.submit-btn {
  height: 48px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.form-footer {
  text-align: center;
  font-size: 14px;
  color: #666;
}

.form-footer a {
  color: #667eea;
  text-decoration: none;
  font-weight: 500;
}

.form-footer a:hover {
  text-decoration: underline;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #ffebee;
  border-radius: 8px;
  color: #c62828;
  font-size: 14px;
  margin-top: 16px;
}

@media (max-width: 480px) {
  .login-container {
    padding: 24px;
  }

  .title {
    font-size: 20px;
  }
}
</style>