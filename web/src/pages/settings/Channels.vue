<template>
  <div class="channels-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <n-h2>{{ t('settings.channels.title') }}</n-h2>
        <n-text depth="3">配置各个接入频道的 API 密钥与回调参数 / Configure API keys and callback parameters for each channel</n-text>
      </div>

      <n-grid :cols="1" :x-gap="12" :y-gap="24" responsive="screen" item-responsive>
        <!-- ── 飞书配置 / Feishu Config ── -->
        <n-gi>
          <n-card bordered class="channel-card">
            <template #header>
              <div class="card-header">
                <div class="header-main">
                  <n-icon :component="BusinessOutline" :size="28" color="#18a058" />
                  <span class="channel-title">{{ t('settings.channels.feishu') }}</span>
                </div>
                <n-tag :type="getChannelHealth('feishu').running ? 'success' : 'default'" round size="small">
                  {{ getChannelHealth('feishu').running ? t('settings.channels.running') : t('settings.channels.stopped') }}
                </n-tag>
              </div>
            </template>

            <n-form label-placement="left" label-width="120" :model="feishuForm">
              <n-form-item label="App ID">
                <n-input v-model:value="feishuForm.app_id" placeholder="cli_xxx" />
              </n-form-item>
              <n-form-item label="App Secret">
                <n-input
                  v-model:value="feishuForm.app_secret"
                  type="password"
                  show-password-on="click"
                  placeholder="请输入 App Secret / Enter App Secret"
                />
              </n-form-item>
              
              <div class="form-actions">
                <n-button type="primary" :loading="saving === 'feishu'" @click="saveConfig('feishu', feishuForm)">
                  {{ t('common.save') }}
                </n-button>
              </div>
            </n-form>
          </n-card>
        </n-gi>

        <!-- ── 钉钉配置 / DingTalk Config ── -->
        <n-gi>
          <n-card bordered class="channel-card">
            <template #header>
              <div class="card-header">
                <div class="header-main">
                  <n-icon :component="RocketOutline" :size="28" color="#2080f0" />
                  <span class="channel-title">{{ t('settings.channels.dingtalk') }}</span>
                </div>
                <n-tag :type="getChannelHealth('dingtalk').running ? 'success' : 'default'" round size="small">
                  {{ getChannelHealth('dingtalk').running ? t('settings.channels.running') : t('settings.channels.stopped') }}
                </n-tag>
              </div>
            </template>

            <n-form label-placement="left" label-width="120" :model="dingtalkForm">
              <n-form-item label="Client ID">
                <n-input v-model:value="dingtalkForm.client_id" placeholder="Suite Key / App ID" />
              </n-form-item>
              <n-form-item label="Client Secret">
                <n-input
                  v-model:value="dingtalkForm.client_secret"
                  type="password"
                  show-password-on="click"
                  placeholder="请输入 Secret / Enter Secret"
                />
              </n-form-item>
              
              <div class="form-actions">
                <n-button type="primary" :loading="saving === 'dingtalk'" @click="saveConfig('dingtalk', dingtalkForm)">
                  {{ t('common.save') }}
                </n-button>
              </div>
            </n-form>
          </n-card>
        </n-gi>

        <!-- ── Webhook 配置 / Webhook Config ── -->
        <n-gi>
          <n-card bordered class="channel-card">
            <template #header>
              <div class="card-header">
                <div class="header-main">
                  <n-icon :component="LinkOutline" :size="28" color="#f0a020" />
                  <span class="channel-title">{{ t('settings.channels.webhook') }}</span>
                </div>
                <n-tag :type="getChannelHealth('webhook').running ? 'success' : 'default'" round size="small">
                  {{ getChannelHealth('webhook').running ? t('settings.channels.running') : t('settings.channels.stopped') }}
                </n-tag>
              </div>
            </template>

            <n-form label-placement="left" label-width="120" :model="webhookForm">
              <n-form-item label="Auth Token">
                <n-input v-model:value="webhookForm.token" placeholder="用于请求鉴权 / For auth" />
              </n-form-item>
              
              <n-alert title="Webhook 地址 / Endpoint" type="info" :show-icon="false" class="webhook-alert">
                POST <n-code>http://your-server:8088/webhook/{{ webhookForm.token || '{token}' }}</n-code>
              </n-alert>

              <div class="form-actions">
                <n-button type="primary" :loading="saving === 'webhook'" @click="saveConfig('webhook', webhookForm)">
                  {{ t('common.save') }}
                </n-button>
              </div>
            </n-form>
          </n-card>
        </n-gi>
      </n-grid>
    </n-space>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, onMounted, onUnmounted, reactive } from 'vue'
import {
  NCard, NSpace, NGrid, NGi, NForm, NFormItem, NInput,
  NButton, NIcon, NTag, NH2, NText, NAlert, NCode, useMessage
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { BusinessOutline, RocketOutline, LinkOutline } from '@vicons/ionicons5'
import { getChannelConfig, saveChannelConfig, getChannelsHealth } from '@/api/settings'
import type { ChannelStatus } from '@/types'

const { t } = useI18n()
const message = useMessage()

const saving = ref<string | null>(null)
const healthData = ref<ChannelStatus[]>([])
let healthTimer: ReturnType<typeof setInterval>

const feishuForm = reactive({
  app_id: '',
  app_secret: ''
})

const dingtalkForm = reactive({
  client_id: '',
  client_secret: ''
})

const webhookForm = reactive({
  token: ''
})

// 中文：加载所有频道配置
// English: Load all channel configs
async function loadConfigs() {
  try {
    const [fs, dt, wh] = await Promise.all([
      getChannelConfig('feishu'),
      getChannelConfig('dingtalk'),
      getChannelConfig('webhook')
    ])
    Object.assign(feishuForm, fs)
    Object.assign(dingtalkForm, dt)
    Object.assign(webhookForm, wh)
  } catch (error) {
    console.error('Failed to load channel configs:', error)
  }
}

// 中文：加载健康状态
// English: Load health status
async function loadHealth() {
  try {
    healthData.value = await getChannelsHealth()
  } catch (error) {
    console.error('Failed to load health status:', error)
  }
}

// 中文：获取指定频道的健康状态
// English: Get health status for a specific channel
function getChannelHealth(name: string): Partial<ChannelStatus> {
  return healthData.value.find(h => h.name === name) || { running: false }
}

// 中文：保存配置
// English: Save configuration
async function saveConfig(name: string, data: any) {
  try {
    saving.value = name
    await saveChannelConfig(name, data)
    message.success(t('common.success'))
    loadHealth() // 中文：保存后刷新状态 / Refresh health after save
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    saving.value = null
  }
}

onMounted(() => {
  loadConfigs()
  loadHealth()
  // 中文：每 10 秒刷新一次健康状态 / Poll health every 10s
  healthTimer = setInterval(loadHealth, 10000)
})

onUnmounted(() => {
  // 中文：销毁组件时清除定时器 / Clear timer when unmounting
  if (healthTimer) {
    clearInterval(healthTimer)
  }
})
</script>

<style scoped lang="scss">
.channels-page {
  padding: 12px;
}

.page-header {
  margin-bottom: 8px;
}

.channel-card {
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
  transition: transform 0.2s;

  &:hover {
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.06);
  }
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.header-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.channel-title {
  font-size: 18px;
  font-weight: 700;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.webhook-alert {
  margin: 16px 0;
  
  :deep(.n-code) {
    background: rgba(0, 0, 0, 0.05);
    padding: 2px 6px;
    border-radius: 4px;
  }
}
</style>
