<template>
  <div class="channels-view">
    <div class="view-header">
      <div class="header-main">
        <n-h2 class="title">{{ t('settings.channels.title') }}</n-h2>
        <n-text depth="3" class="description">{{ t('settings.channels.description') }}</n-text>
      </div>
    </div>

    <div class="channel-list">
      <!-- 飞书频道 -->
      <div class="channel-item">
        <div class="channel-brand">
          <div class="brand-icon feishu"><n-icon :component="BusinessOutline" /></div>
          <div class="brand-info">
            <div class="brand-name">{{ t('settings.channels.feishu') }}</div>
            <div class="brand-status">
              <n-badge :type="getChannelHealth('feishu').running ? 'success' : 'default'" dot processing />
              <span>{{ getChannelHealth('feishu').running ? t('settings.channels.running') : t('settings.channels.stopped') }}</span>
            </div>
          </div>
        </div>
        
        <div class="channel-form-card">
          <n-form :model="feishuForm" label-placement="top">
            <n-grid :cols="2" :x-gap="24">
              <n-gi>
                <n-form-item label="App ID">
                  <n-input v-model:value="feishuForm.app_id" placeholder="cli_xxx" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="App Secret">
                  <n-input v-model:value="feishuForm.app_secret" type="password" show-password-on="mousedown" placeholder="App Secret" />
                </n-form-item>
              </n-gi>
            </n-grid>
            <div class="form-actions">
              <n-button type="primary" secondary round :loading="saving === 'feishu'" @click="saveConfig('feishu', feishuForm)">
                {{ t('common.save') }}
              </n-button>
            </div>
          </n-form>
        </div>
      </div>

      <!-- 钉钉频道 -->
      <div class="channel-item">
        <div class="channel-brand">
          <div class="brand-icon dingtalk"><n-icon :component="RocketOutline" /></div>
          <div class="brand-info">
            <div class="brand-name">{{ t('settings.channels.dingtalk') }}</div>
            <div class="brand-status">
              <n-badge :type="getChannelHealth('dingtalk').running ? 'success' : 'default'" dot processing />
              <span>{{ getChannelHealth('dingtalk').running ? t('settings.channels.running') : t('settings.channels.stopped') }}</span>
            </div>
          </div>
        </div>
        
        <div class="channel-form-card">
          <n-form :model="dingtalkForm" label-placement="top">
            <n-grid :cols="2" :x-gap="24">
              <n-gi>
                <n-form-item label="Client ID">
                  <n-input v-model:value="dingtalkForm.client_id" placeholder="Suite Key" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="Client Secret">
                  <n-input v-model:value="dingtalkForm.client_secret" type="password" show-password-on="mousedown" placeholder="Secret" />
                </n-form-item>
              </n-gi>
            </n-grid>
            <div class="form-actions">
              <n-button type="primary" secondary round :loading="saving === 'dingtalk'" @click="saveConfig('dingtalk', dingtalkForm)">
                {{ t('common.save') }}
              </n-button>
            </div>
          </n-form>
        </div>
      </div>

      <!-- Webhook 频道 -->
      <div class="channel-item">
        <div class="channel-brand">
          <div class="brand-icon webhook"><n-icon :component="LinkOutline" /></div>
          <div class="brand-info">
            <div class="brand-name">{{ t('settings.channels.webhook') }}</div>
            <div class="brand-status">
              <n-badge :type="getChannelHealth('webhook').running ? 'success' : 'default'" dot processing />
              <span>{{ getChannelHealth('webhook').running ? t('settings.channels.configured') : t('settings.channels.notConfigured') }}</span>
            </div>
          </div>
        </div>
        
        <div class="channel-form-card">
          <n-form :model="webhookForm" label-placement="top">
            <n-form-item label="Auth Token">
              <n-input v-model:value="webhookForm.token" placeholder="Token" />
            </n-form-item>
            
            <div class="endpoint-tip">
              {{ t('settings.channels.endpoint') }} <span class="code">/webhook/{{ webhookForm.token || '{token}' }}</span>
            </div>

            <div class="form-actions">
              <n-button type="primary" secondary round :loading="saving === 'webhook'" @click="saveConfig('webhook', webhookForm)">
                {{ t('common.save') }}
              </n-button>
            </div>
          </n-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, reactive } from 'vue'
import { NH2, NText, NButton, NIcon, NBadge, NForm, NFormItem, NInput, NGrid, NGi, useMessage } from 'naive-ui'
import { BusinessOutline, RocketOutline, LinkOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getChannelConfig, saveChannelConfig, getChannelsHealth } from '@/api/settings'
import type { ChannelStatus } from '@/types'

const { t } = useI18n()
const message = useMessage()

const saving = ref<string | null>(null)
const healthData = ref<ChannelStatus[]>([])
let healthTimer: ReturnType<typeof setInterval>

const feishuForm = reactive({ app_id: '', app_secret: '' })
const dingtalkForm = reactive({ client_id: '', client_secret: '' })
const webhookForm = reactive({ token: '' })

async function loadConfigs() {
  try {
    const [fs, dt, wh] = await Promise.all([getChannelConfig('feishu'), getChannelConfig('dingtalk'), getChannelConfig('webhook')])
    Object.assign(feishuForm, fs)
    Object.assign(dingtalkForm, dt)
    Object.assign(webhookForm, wh)
  } catch (error) {
    console.error(error)
  }
}

async function loadHealth() {
  try { healthData.value = await getChannelsHealth() } catch (error) { console.error(error) }
}

function getChannelHealth(name: string): Partial<ChannelStatus> {
  return healthData.value.find(h => h.name === name) || { running: false }
}

async function saveConfig(name: string, data: Record<string, string>) {
  saving.value = name
  try {
    await saveChannelConfig(name, data)
    message.success(t('common.success'))
    loadHealth()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = null
  }
}

onMounted(() => {
  loadConfigs()
  loadHealth()
  healthTimer = setInterval(loadHealth, 10000)
})

onUnmounted(() => { if (healthTimer) clearInterval(healthTimer) })
</script>

<style scoped lang="scss">
.channels-view { display: flex; flex-direction: column; gap: 40px; }
.view-header { .title { margin: 0 0 8px; font-weight: 800; font-size: 32px; letter-spacing: -1px; } }
.channel-list { display: flex; flex-direction: column; gap: 48px; }
.channel-item { display: flex; gap: 48px; @media (max-width: 1000px) { flex-direction: column; gap: 24px; } }
.channel-brand { width: 200px; display: flex; flex-direction: column; gap: 16px; .brand-icon { width: 56px; height: 56px; border-radius: 16px; display: flex; align-items: center; justify-content: center; font-size: 28px; color: #fff; &.feishu { background: linear-gradient(135deg, #2ecc71, #18a058); } &.dingtalk { background: linear-gradient(135deg, #3498db, #2080f0); } &.webhook { background: linear-gradient(135deg, #f39c12, #f0a020); } } .brand-name { font-weight: 700; font-size: 18px; color: #1a1a1a; } .brand-status { display: flex; align-items: center; gap: 8px; font-size: 13px; color: #888; } }
.channel-form-card { flex: 1; background: #fdfdfd; padding: 32px; border-radius: 24px; border: 1px solid rgba(0, 0, 0, 0.04); transition: all 0.3s; &:hover { background: #fff; box-shadow: 0 8px 32px rgba(0, 0, 0, 0.03); border-color: rgba(0, 0, 0, 0.08); } }
.endpoint-tip { margin-top: -8px; margin-bottom: 24px; font-size: 12px; color: #999; .code { background: #f5f5f5; padding: 2px 6px; border-radius: 4px; font-family: monospace; color: #e67e22; } }
.form-actions { display: flex; justify-content: flex-end; }
</style>
