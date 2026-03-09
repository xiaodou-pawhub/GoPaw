<template>
  <div class="tab-root">
    <div class="tab-header">
      <h2 class="tab-title">频道集成</h2>
      <p class="tab-desc">配置第三方平台接入凭证，实现自动化推送与交互</p>
    </div>

    <div class="channel-list">
      <!-- 飞书 -->
      <div class="channel-card">
        <div class="channel-brand">
          <div class="brand-badge feishu">飞</div>
          <div>
            <div class="brand-name">飞书 (Feishu)</div>
            <div class="brand-status">
              <span class="status-dot" :class="getChannelHealth('feishu').running ? 'ok' : 'off'" />
              <span class="status-text">{{ getChannelHealth('feishu').running ? '运行中' : '未启用' }}</span>
            </div>
          </div>
        </div>
        <div class="channel-form">
          <div class="form-row">
            <div class="form-field">
              <label>App ID</label>
              <input v-model="feishuForm.app_id" placeholder="cli_xxx" class="form-input" />
            </div>
            <div class="form-field">
              <label>App Secret</label>
              <input v-model="feishuForm.app_secret" type="password" placeholder="App Secret" class="form-input" />
            </div>
          </div>
          <div class="form-actions">
            <button class="btn-primary" :disabled="saving === 'feishu'" @click="saveConfig('feishu', feishuForm)">
              {{ saving === 'feishu' ? '保存中...' : '保存' }}
            </button>
            <button class="btn-secondary" :disabled="testing === 'feishu'" @click="testChannelConn('feishu')">
              {{ testing === 'feishu' ? '测试中...' : '测试连接' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 钉钉 -->
      <div class="channel-card">
        <div class="channel-brand">
          <div class="brand-badge dingtalk">钉</div>
          <div>
            <div class="brand-name">钉钉 (DingTalk)</div>
            <div class="brand-status">
              <span class="status-dot" :class="getChannelHealth('dingtalk').running ? 'ok' : 'off'" />
              <span class="status-text">{{ getChannelHealth('dingtalk').running ? '运行中' : '未启用' }}</span>
            </div>
          </div>
        </div>
        <div class="channel-form">
          <div class="form-row">
            <div class="form-field">
              <label>Client ID</label>
              <input v-model="dingtalkForm.client_id" placeholder="Suite Key" class="form-input" />
            </div>
            <div class="form-field">
              <label>Client Secret</label>
              <input v-model="dingtalkForm.client_secret" type="password" placeholder="Secret" class="form-input" />
            </div>
          </div>
          <div class="form-actions">
            <button class="btn-primary" :disabled="saving === 'dingtalk'" @click="saveConfig('dingtalk', dingtalkForm)">
              {{ saving === 'dingtalk' ? '保存中...' : '保存' }}
            </button>
            <button class="btn-secondary" :disabled="testing === 'dingtalk'" @click="testChannelConn('dingtalk')">
              {{ testing === 'dingtalk' ? '测试中...' : '测试连接' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Webhook -->
      <div class="channel-card">
        <div class="channel-brand">
          <div class="brand-badge webhook">W</div>
          <div>
            <div class="brand-name">Webhook</div>
            <div class="brand-status">
              <span class="status-dot" :class="getChannelHealth('webhook').running ? 'ok' : 'off'" />
              <span class="status-text">{{ getChannelHealth('webhook').running ? '运行中' : '未启用' }}</span>
            </div>
          </div>
        </div>
        <div class="channel-form">
          <div class="form-field">
            <label>Webhook URL</label>
            <input v-model="webhookForm.url" placeholder="https://your-server.com/webhook/gopaw" class="form-input" />
          </div>
          <p class="form-tip">GoPaw 将通过此 URL 推送消息到第三方系统</p>
          <div class="form-actions">
            <button class="btn-primary" :disabled="saving === 'webhook'" @click="saveConfig('webhook', webhookForm)">
              {{ saving === 'webhook' ? '保存中...' : '保存' }}
            </button>
            <button class="btn-secondary" :disabled="testing === 'webhook'" @click="testChannelConn('webhook')">
              {{ testing === 'webhook' ? '测试中...' : '测试连接' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Email -->
      <div class="channel-card">
        <div class="channel-brand">
          <div class="brand-badge email">@</div>
          <div>
            <div class="brand-name">Email 邮件助手</div>
            <div class="brand-status">
              <span class="status-text">配置 SMTP 以发送通知邮件</span>
            </div>
          </div>
        </div>
        <div class="channel-form">
          <div class="form-row">
            <div class="form-field" style="flex: 2">
              <label>SMTP Host</label>
              <input v-model="emailForm.host" placeholder="smtp.gmail.com" class="form-input" />
            </div>
            <div class="form-field" style="flex: 1">
              <label>Port</label>
              <input v-model.number="emailForm.port" type="number" placeholder="465" class="form-input" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-field">
              <label>Username</label>
              <input v-model="emailForm.username" placeholder="your@email.com" class="form-input" />
            </div>
            <div class="form-field">
              <label>Password / Token</label>
              <input v-model="emailForm.password" type="password" placeholder="App Password" class="form-input" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-field">
              <label>From Name</label>
              <input v-model="emailForm.from" placeholder="GoPaw Assistant" class="form-input" />
            </div>
            <div class="form-field" style="justify-content: center; align-items: flex-start; flex-direction: row; gap: 8px; padding-top: 20px;">
              <label class="toggle" title="SSL">
                <input type="checkbox" v-model="emailForm.ssl" />
                <span class="toggle-slider" />
              </label>
              <span style="font-size: 12px; color: var(--text-secondary); line-height: 18px;">Use SSL</span>
            </div>
          </div>
          <div class="form-actions">
            <button class="btn-primary" :disabled="saving === 'email'" @click="saveConfig('email', emailForm)">
              {{ saving === 'email' ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, reactive } from 'vue'
import { toast } from 'vue-sonner'
import { getChannelConfig, saveChannelConfig, getChannelsHealth, testChannel } from '@/api/settings'
import type { ChannelStatus } from '@/types'

const saving = ref<string | null>(null)
const testing = ref<string | null>(null)
const healthData = ref<ChannelStatus[]>([])
let healthTimer: ReturnType<typeof setInterval>

const feishuForm = reactive({ app_id: '', app_secret: '' })
const dingtalkForm = reactive({ client_id: '', client_secret: '' })
const webhookForm = reactive({ url: '' })
const emailForm = reactive({ host: '', port: 465, username: '', password: '', from: '', ssl: true })

async function loadConfigs() {
  try {
    const [fs, dt, wh, em] = await Promise.all([
      getChannelConfig('feishu'), getChannelConfig('dingtalk'),
      getChannelConfig('webhook'), getChannelConfig('email')
    ])
    Object.assign(feishuForm, fs)
    Object.assign(dingtalkForm, dt)
    Object.assign(webhookForm, wh)
    Object.assign(emailForm, em)
  } catch {}
}

async function loadHealth() {
  try { healthData.value = await getChannelsHealth() } catch {}
}

function getChannelHealth(name: string): Partial<ChannelStatus> {
  return healthData.value.find(h => h.name === name) || { running: false }
}

async function saveConfig(name: string, data: Record<string, any>) {
  saving.value = name
  try {
    await saveChannelConfig(name, data)
    toast.success('保存成功')
    loadHealth()
  } catch {
    toast.error('保存失败')
  } finally {
    saving.value = null
  }
}

async function testChannelConn(name: string) {
  testing.value = name
  try {
    const result = await testChannel(name)
    if (result.success) toast.success(result.message)
    else toast.error(result.message)
  } catch (error: any) {
    toast.error(error?.message || '测试失败')
  } finally {
    testing.value = null
  }
}

onMounted(() => {
  loadConfigs()
  loadHealth()
  healthTimer = setInterval(loadHealth, 10000)
})
onUnmounted(() => { if (healthTimer) clearInterval(healthTimer) })
</script>

<style scoped>
.tab-root { display: flex; flex-direction: column; gap: 20px; }
.tab-header { display: flex; flex-direction: column; gap: 4px; }
.tab-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0; }
.tab-desc { font-size: 12px; color: var(--text-secondary); margin: 0; }

.channel-list { display: flex; flex-direction: column; gap: 12px; }

.channel-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.channel-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-badge {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.brand-badge.feishu { background: linear-gradient(135deg, #22c55e, #16a34a); }
.brand-badge.dingtalk { background: linear-gradient(135deg, #3b82f6, #1d4ed8); }
.brand-badge.webhook { background: linear-gradient(135deg, #f59e0b, #d97706); }
.brand-badge.email { background: linear-gradient(135deg, #ef4444, #b91c1c); }

.brand-name { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.brand-status { display: flex; align-items: center; gap: 5px; margin-top: 2px; }

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-dot.ok { background: var(--green); }
.status-dot.off { background: var(--text-tertiary); }

.status-text { font-size: 11px; color: var(--text-secondary); }

.channel-form { display: flex; flex-direction: column; gap: 10px; }

.form-row { display: flex; gap: 10px; }

.form-field { display: flex; flex-direction: column; gap: 5px; flex: 1; }

.form-field label { font-size: 12px; font-weight: 500; color: var(--text-secondary); }

.form-input {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
  transition: border-color 0.15s;
}

.form-input:focus { border-color: var(--accent); }

.form-tip { font-size: 11px; color: var(--text-tertiary); margin: 0; }

.form-actions { display: flex; gap: 8px; justify-content: flex-end; }

/* Toggle */
.toggle {
  position: relative;
  width: 34px;
  height: 18px;
  cursor: pointer;
  display: inline-block;
}

.toggle input { display: none; }

.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 9px;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 12px;
  height: 12px;
  background: var(--text-tertiary);
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: transform 0.2s, background 0.2s;
}

.toggle input:checked + .toggle-slider { background: rgba(124, 106, 247, 0.2); border-color: var(--accent); }
.toggle input:checked + .toggle-slider::before { transform: translateX(16px); background: var(--accent); }

.btn-primary {
  padding: 7px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }

.btn-secondary {
  padding: 7px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-secondary:hover { background: var(--bg-elevated); }
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
