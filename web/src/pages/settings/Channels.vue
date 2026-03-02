<template>
  <div class="channels-page">
    <n-card :title="t('settings.channels.title')">
      <n-space vertical>
        <n-alert type="info">
          频道配置功能开发中... Channel configuration coming soon...
        </n-alert>
        
        <n-list>
          <n-list-item>
            <template #prefix>
              <n-icon :component="FeishuIcon" :size="24" />
            </template>
            <n-space vertical>
              <div class="channel-name">{{ t('settings.channels.feishu') }}</div>
              <n-text depth="3">飞书机器人配置 / Feishu bot configuration</n-text>
            </n-space>
            <template #suffix>
              <n-button disabled>{{ t('edit') }}</n-button>
            </template>
          </n-list-item>
          
          <n-list-item>
            <template #prefix>
              <n-icon :component="DingTalkIcon" :size="24" />
            </template>
            <n-space vertical>
              <div class="channel-name">{{ t('settings.channels.dingtalk') }}</div>
              <n-text depth="3">钉钉机器人配置 / DingTalk bot configuration</n-text>
            </n-space>
            <template #suffix>
              <n-button disabled>{{ t('edit') }}</n-button>
            </template>
          </n-list-item>
        </n-list>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { NCard, NSpace, NList, NListItem, NButton, NText, NIcon, NAlert } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { LogoFeishu } from '@vicons/ionicons5'

const { t } = useI18n()

const FeishuIcon = LogoFeishu
const DingTalkIcon = LogoFeishu // 临时使用相同图标
</script>

<style scoped>
.channels-page {
  padding: 24px;
}

.channel-name {
  font-weight: 600;
  font-size: 16px;
}
</style>
