<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('settings.channels.title') }}</h1>
        <p class="page-description">{{ t('settings.channels.description') }}</p>
      </div>
    </div>

    <div class="channel-list">
      <!-- 飞书频道 -->
      <div class="channel-item">
        <div class="channel-brand">
          <div class="brand-icon feishu"><n-icon :component="BusinessOutline" /></div>
          <div class="brand-info">
            <h3 class="brand-name">{{ t('settings.channels.feishu') }}</h3>
            <div class="brand-status">
              <n-badge :type="getChannelHealth('feishu').running ? 'success' : 'default'" dot processing />
              <span>{{ getChannelHealth('feishu').running ? t('settings.channels.running') : t('settings.channels.stopped') }}</span>
            </div>
          </div>
        </div>
        
        <div class="channel-form-card">
          <n-form :model="feishuForm" label-placement="top" label-width="auto">
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
              <n-button type="primary" round :loading="saving === 'feishu'" @click="saveConfig('feishu', feishuForm)">
                {{ t('common.save') }}
              </n-button>
              <n-button round :loading="testing === 'feishu'" @click="testChannelConn('feishu')">
                {{ t('settings.channels.test') }}
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
            <h3 class="brand-name">{{ t('settings.channels.dingtalk') }}</h3>
            <div class="brand-status">
              <n-badge :type="getChannelHealth('dingtalk').running ? 'success' : 'default'" dot processing />
              <span>{{ getChannelHealth('dingtalk').running ? t('settings.channels.running') : t('settings.channels.stopped') }}</span>
            </div>
          </div>
        </div>
        
        <div class="channel-form-card">
          <n-form :model="dingtalkForm" label-placement="top" label-width="auto">
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
              <n-button type="primary" round :loading="saving === 'dingtalk'" @click="saveConfig('dingtalk', dingtalkForm)">
                {{ t('common.save') }}
              </n-button>
              <n-button round :loading="testing === 'dingtalk'" @click="testChannelConn('dingtalk')">
                {{ t('settings.channels.test') }}
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
            <h3 class="brand-name">{{ t('settings.channels.webhook') }}</h3>
            <div class="brand-status">
              <n-badge :type="getChannelHealth('webhook').running ? 'success' : 'default'" dot processing />
              <span>{{ getChannelHealth('webhook').running ? t('settings.channels.running') : t('settings.channels.stopped') }}</span>
            </div>
          </div>
        </div>

        <div class="channel-form-card">
          <n-form :model="webhookForm" label-placement="top" label-width="auto">
            <n-form-item :label="t('settings.channels.webhookUrl')">
              <n-input v-model:value="webhookForm.url" :placeholder="t('settings.channels.webhookUrlPlaceholder')" />
            </n-form-item>

            <div class="endpoint-tip">
              {{ t('settings.channels.webhookTip') }}
            </div>

            <div class="form-actions">
              <n-button type="primary" round :loading="saving === 'webhook'" @click="saveConfig('webhook', webhookForm)">
                {{ t('common.save') }}
              </n-button>
              <n-button round :loading="testing === 'webhook'" @click="testChannelConn('webhook')">
                {{ t('settings.channels.test') }}
              </n-button>
            </div>
          </n-form>
        </div>
      </div>

      <!-- Email 邮件助手 -->
      <div class="channel-item">
        <div class="channel-brand">
          <div class="brand-icon email"><n-icon :component="MailOutline" /></div>
          <div class="brand-info">
            <h3 class="brand-name">Email 邮件助手</h3>
            <p class="brand-status">配置 SMTP 以发送通知邮件</p>
          </div>
        </div>

        <div class="channel-form-card">
          <n-form :model="emailForm" label-placement="top" label-width="auto">
            <n-grid :cols="3" :x-gap="12">
              <n-gi :span="2">
                <n-form-item label="SMTP Host">
                  <n-input v-model:value="emailForm.host" placeholder="smtp.gmail.com" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="Port">
                  <n-input-number v-model:value="emailForm.port" :show-button="false" placeholder="465" />
                </n-form-item>
              </n-gi>
            </n-grid>
            
            <n-grid :cols="2" :x-gap="12">
              <n-gi>
                <n-form-item label="Username">
                  <n-input v-model:value="emailForm.username" placeholder="your@email.com" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="Password / Token">
                  <n-input v-model:value="emailForm.password" type="password" show-password-on="mousedown" placeholder="App Password" />
                </n-form-item>
              </n-gi>
            </n-grid>

            <n-grid :cols="2" :x-gap="12">
              <n-gi>
                <n-form-item label="From Name">
                  <n-input v-model:value="emailForm.from" placeholder="GoPaw Assistant" />
                </n-form-item>
              </n-gi>
              <n-gi style="display: flex; align-items: center; justify-content: center;">
                <n-form-item label="Use SSL">
                  <n-switch v-model:value="emailForm.ssl" />
                </n-form-item>
              </n-gi>
            </n-grid>

            <div class="form-actions">
              <n-button type="primary" round :loading="saving === 'email'" @click="saveConfig('email', emailForm)">
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
import { NButton, NIcon, NBadge, NForm, NFormItem, NInput, NInputNumber, NSwitch, NGrid, NGi, useMessage } from 'naive-ui'
import { BusinessOutline, RocketOutline, LinkOutline, MailOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getChannelConfig, saveChannelConfig, getChannelsHealth, testChannel } from '@/api/settings'
import type { ChannelStatus } from '@/types'

const { t } = useI18n()
const message = useMessage()

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
      getChannelConfig('feishu'), 
      getChannelConfig('dingtalk'), 
      getChannelConfig('webhook'),
      getChannelConfig('email')
    ])
    Object.assign(feishuForm, fs)
    Object.assign(dingtalkForm, dt)
    Object.assign(webhookForm, wh)
    Object.assign(emailForm, em)
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

async function testChannelConn(name: string) {
  testing.value = name
  try {
    const result = await testChannel(name)
    if (result.success) {
      message.success(result.message)
    } else {
      message.error(result.message)
    }
  } catch (error: any) {
    message.error(error?.message || t('common.error'))
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

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;
@use '@/styles/page-layout' as *;

.channel-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-6;
}

.channel-item {
  display: flex;
  gap: $spacing-8;
  padding: $spacing-8;
  border: 1px solid $color-border-light;
  border-radius: $radius-xl;
  background: $color-bg-primary;
  transition: $transition-normal;
  animation: slideUp 0.5s ease-out;
  animation-fill-mode: both;

  @for $i from 1 through 6 {
    &:nth-child(#{$i}) {
      animation-delay: #{$i * 0.08}s;
    }
  }

  &:hover {
    transform: translateY(-3px);
    box-shadow: $shadow-lg;
    border-color: $color-border-medium;
  }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.channel-brand {
  width: 240px;
  display: flex;
  flex-direction: column;
  gap: $spacing-4;
  padding: $spacing-2;

  .brand-icon {
    width: 48px;
    height: 48px;
    border-radius: $radius-lg;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    color: $color-white;
    transition: all 0.3s ease;

    &:hover {
      transform: scale(1.1) rotate(5deg);
    }

    &.feishu {
      background: linear-gradient(135deg, $color-success, $color-success-dark);
    }

    &.dingtalk {
      background: linear-gradient(135deg, $color-info, $color-info-dark);
    }

    &.webhook {
      background: linear-gradient(135deg, $color-warning, $color-warning-dark);
    }

    &.email {
      background: linear-gradient(135deg, #f5222d, #cf1322); // Red for email
    }
  }

  .brand-name {
    font-weight: $font-weight-semibold;
    font-size: $font-size-h4;
    color: $color-text-primary;
    margin: $spacing-1 0 0 0;
  }

  .brand-status {
    display: flex;
    align-items: center;
    gap: $spacing-2;
    font-size: $font-size-sm;
    color: $color-text-secondary;
  }
}

.channel-form-card {
  flex: 1;
  background: $color-bg-secondary;
  padding: $spacing-6;
  border-radius: $radius-lg;
  border: 1px solid $color-border-light;
  transition: $transition-normal;

  &:hover {
    background: $color-bg-primary;
    box-shadow: $shadow-sm;
  }
}

.endpoint-tip {
  margin: $spacing-3 0 $spacing-5 0;
  font-size: $font-size-sm;
  color: $color-text-secondary;
  line-height: $line-height-normal;

  .code {
    background: $color-bg-tertiary;
    padding: $spacing-1 $spacing-2;
    border-radius: $radius-sm;
    font-family: $font-family-mono;
    color: $color-primary;
    font-weight: $font-weight-medium;
  }
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-3;
  margin-top: $spacing-4;

  :deep(.n-button) {
    transition: all 0.2s ease;

    &:hover {
      transform: translateY(-1px);
      box-shadow: $shadow-md;
    }

    &:active {
      transform: translateY(0);
    }
  }
}
</style>
