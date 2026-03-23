<template>
  <div class="page-root">
    <!-- 顶栏 -->
    <div class="page-header">
      <div>
        <h1 class="page-title">告警管理</h1>
        <p class="page-desc">配置告警规则和通知渠道，实时监控系统状态</p>
      </div>
      <div class="header-actions">
        <button class="btn-secondary" @click="activeTab = 'channels'">
          <BellIcon :size="13" /> 通知渠道
        </button>
        <button class="btn-primary" @click="openRuleModal()">
          <PlusIcon :size="13" /> 新建规则
        </button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-value">{{ rules.length }}</div>
        <div class="stat-label">告警规则</div>
      </div>
      <div class="stat-card">
        <div class="stat-value text-success">{{ rules.filter(r => r.enabled).length }}</div>
        <div class="stat-label">已启用</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ channels.length }}</div>
        <div class="stat-label">通知渠道</div>
      </div>
      <div class="stat-card">
        <div class="stat-value text-warning">{{ history.filter(h => h.status === 'triggered').length }}</div>
        <div class="stat-label">未恢复</div>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="tab-bar">
      <button :class="['tab-btn', { active: activeTab === 'rules' }]" @click="activeTab = 'rules'">
        <AlertTriangleIcon :size="14" /> 告警规则
      </button>
      <button :class="['tab-btn', { active: activeTab === 'channels' }]" @click="activeTab = 'channels'">
        <BellIcon :size="14" /> 通知渠道
      </button>
      <button :class="['tab-btn', { active: activeTab === 'history' }]" @click="activeTab = 'history'">
        <HistoryIcon :size="14" /> 告警历史
      </button>
    </div>

    <!-- 告警规则列表 -->
    <div v-if="activeTab === 'rules'" class="content-panel">
      <div v-if="loadingRules" class="list-empty">加载中...</div>
      <div v-else-if="rules.length === 0" class="list-empty">
        <AlertTriangleIcon :size="32" class="empty-icon" />
        <p>暂无告警规则</p>
        <button class="btn-primary" @click="openRuleModal()">创建第一个规则</button>
      </div>
      <div v-else class="rule-list">
        <div v-for="rule in rules" :key="rule.id" class="rule-card">
          <div class="rule-header">
            <div class="rule-title-row">
              <span class="rule-name">{{ rule.name }}</span>
              <label class="toggle" :title="rule.enabled ? '点击禁用' : '点击启用'">
                <input type="checkbox" :checked="rule.enabled" @change="toggleRule(rule)" />
                <span class="toggle-slider" />
              </label>
            </div>
            <div class="rule-type" :class="rule.type">{{ getTypeLabel(rule.type) }}</div>
          </div>
          <div class="rule-condition">
            <span class="metric">{{ rule.condition.metric }}</span>
            <span class="operator">{{ rule.condition.operator }}</span>
            <span class="threshold">{{ rule.condition.threshold }}</span>
            <span v-if="rule.condition.duration" class="duration">持续 {{ rule.condition.duration }}s</span>
          </div>
          <div class="rule-channels">
            <BellIcon :size="12" />
            <span>{{ getChannelNames(rule.channels) }}</span>
          </div>
          <div v-if="rule.last_triggered" class="rule-last-trigger">
            上次触发：{{ formatTime(rule.last_triggered) }}
          </div>
          <div class="rule-actions">
            <button class="action-btn" @click="openRuleModal(rule)">
              <PencilIcon :size="12" /> 编辑
            </button>
            <button class="action-btn danger" @click="deleteRule(rule)">
              <TrashIcon :size="12" /> 删除
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 通知渠道列表 -->
    <div v-if="activeTab === 'channels'" class="content-panel">
      <div class="panel-header">
        <span>通知渠道配置</span>
        <button class="btn-primary btn-sm" @click="openChannelModal()">
          <PlusIcon :size="12" /> 新建渠道
        </button>
      </div>
      <div v-if="loadingChannels" class="list-empty">加载中...</div>
      <div v-else-if="channels.length === 0" class="list-empty">
        <BellIcon :size="32" class="empty-icon" />
        <p>暂无通知渠道</p>
        <button class="btn-primary" @click="openChannelModal()">创建第一个渠道</button>
      </div>
      <div v-else class="channel-list">
        <div v-for="channel in channels" :key="channel.id" class="channel-card">
          <div class="channel-header">
            <div class="channel-icon" :class="channel.type">
              <component :is="getChannelIcon(channel.type)" :size="16" />
            </div>
            <div class="channel-info">
              <span class="channel-name">{{ channel.name }}</span>
              <span class="channel-type">{{ getChannelTypeLabel(channel.type) }}</span>
            </div>
            <label class="toggle">
              <input type="checkbox" :checked="channel.enabled" @change="toggleChannel(channel)" />
              <span class="toggle-slider" />
            </label>
          </div>
          <div class="channel-config">
            <template v-if="channel.type === 'email'">
              <span>{{ channel.config.smtp_host }}:{{ channel.config.smtp_port }}</span>
            </template>
            <template v-else-if="channel.type === 'dingtalk'">
              <span>钉钉机器人</span>
            </template>
            <template v-else-if="channel.type === 'wecom'">
              <span>企业微信机器人</span>
            </template>
            <template v-else-if="channel.type === 'webhook'">
              <span class="webhook-url">{{ channel.config.webhook_url }}</span>
            </template>
          </div>
          <div class="channel-actions">
            <button class="action-btn" @click="testChannel(channel)">
              <FlaskConicalIcon :size="12" /> 测试
            </button>
            <button class="action-btn" @click="openChannelModal(channel)">
              <PencilIcon :size="12" /> 编辑
            </button>
            <button class="action-btn danger" @click="deleteChannel(channel)">
              <TrashIcon :size="12" /> 删除
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 告警历史 -->
    <div v-if="activeTab === 'history'" class="content-panel">
      <div v-if="loadingHistory" class="list-empty">加载中...</div>
      <div v-else-if="history.length === 0" class="list-empty">
        <HistoryIcon :size="32" class="empty-icon" />
        <p>暂无告警历史</p>
      </div>
      <div v-else class="history-list">
        <div v-for="item in history" :key="item.id" class="history-item" :class="item.status">
          <div class="history-icon">
            <AlertTriangleIcon v-if="item.status === 'triggered'" :size="16" />
            <CheckCircleIcon v-else :size="16" />
          </div>
          <div class="history-content">
            <div class="history-title">{{ item.rule_name }}</div>
            <div class="history-message">{{ item.message }}</div>
            <div class="history-meta">
              <span>值: {{ item.value }} / 阈值: {{ item.threshold }}</span>
              <span>{{ formatTime(item.created_at) }}</span>
            </div>
          </div>
          <div class="history-status" :class="item.status">
            {{ item.status === 'triggered' ? '已触发' : '已恢复' }}
          </div>
        </div>
      </div>
    </div>

    <!-- 告警规则弹窗 -->
    <div v-if="showRuleModal" class="modal-overlay" @click.self="showRuleModal = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">{{ ruleForm.id ? '编辑告警规则' : '新建告警规则' }}</h3>
          <button class="icon-close" @click="showRuleModal = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-field">
            <label>规则名称 <span class="required">*</span></label>
            <input v-model="ruleForm.name" placeholder="例如：API 延迟告警" class="form-input" />
          </div>
          <div class="form-field">
            <label>描述</label>
            <textarea v-model="ruleForm.description" placeholder="规则描述" class="form-textarea" />
          </div>
          <div class="form-row">
            <div class="form-field">
              <label>告警类型</label>
              <select v-model="ruleForm.type" class="form-select">
                <option value="metric">指标告警</option>
                <option value="error">错误告警</option>
                <option value="custom">自定义告警</option>
              </select>
            </div>
          </div>
          <div class="form-section-title">告警条件</div>
          <div class="form-row">
            <div class="form-field">
              <label>监控指标</label>
              <select v-model="ruleForm.condition.metric" class="form-select">
                <option value="latency">延迟 (ms)</option>
                <option value="error_rate">错误率 (%)</option>
                <option value="token_usage">Token 使用量</option>
                <option value="request_count">请求数</option>
              </select>
            </div>
            <div class="form-field">
              <label>操作符</label>
              <select v-model="ruleForm.condition.operator" class="form-select">
                <option value=">">大于 (&gt;)</option>
                <option value=">=">大于等于 (&gt;=)</option>
                <option value="<">小于 (&lt;)</option>
                <option value="<=">小于等于 (&lt;=)</option>
                <option value="==">等于 (==)</option>
                <option value="!=">不等于 (!=)</option>
              </select>
            </div>
          </div>
          <div class="form-row">
            <div class="form-field">
              <label>阈值</label>
              <input v-model.number="ruleForm.condition.threshold" type="number" class="form-input" />
            </div>
            <div class="form-field">
              <label>持续时间 (秒)</label>
              <input v-model.number="ruleForm.condition.duration" type="number" class="form-input" />
            </div>
          </div>
          <div class="form-field">
            <label>通知渠道</label>
            <div class="channel-checkboxes">
              <label v-for="ch in channels" :key="ch.id" class="checkbox-label">
                <input type="checkbox" :value="ch.id" v-model="ruleForm.channels" />
                <span>{{ ch.name }}</span>
              </label>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="showRuleModal = false">取消</button>
          <button class="btn-primary" @click="saveRule">保存</button>
        </div>
      </div>
    </div>

    <!-- 通知渠道弹窗 -->
    <div v-if="showChannelModal" class="modal-overlay" @click.self="showChannelModal = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">{{ channelForm.id ? '编辑通知渠道' : '新建通知渠道' }}</h3>
          <button class="icon-close" @click="showChannelModal = false"><XIcon :size="16" /></button>
        </div>
        <div class="modal-body">
          <div class="form-field">
            <label>渠道名称 <span class="required">*</span></label>
            <input v-model="channelForm.name" placeholder="例如：运维钉钉群" class="form-input" />
          </div>
          <div class="form-field">
            <label>渠道类型</label>
            <select v-model="channelForm.type" class="form-select" :disabled="!!channelForm.id">
              <option value="dingtalk">钉钉</option>
              <option value="wecom">企业微信</option>
              <option value="webhook">Webhook</option>
              <option value="email">邮件</option>
            </select>
          </div>

          <!-- 钉钉配置 -->
          <template v-if="channelForm.type === 'dingtalk'">
            <div class="form-field">
              <label>Webhook URL <span class="required">*</span></label>
              <input v-model="channelForm.config.dingtalk_webhook" placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." class="form-input" />
            </div>
            <div class="form-field">
              <label>加签密钥</label>
              <input v-model="channelForm.config.dingtalk_secret" placeholder="SEC... (可选)" class="form-input" />
            </div>
          </template>

          <!-- 企业微信配置 -->
          <template v-if="channelForm.type === 'wecom'">
            <div class="form-field">
              <label>Webhook URL <span class="required">*</span></label>
              <input v-model="channelForm.config.wecom_webhook" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." class="form-input" />
            </div>
          </template>

          <!-- Webhook 配置 -->
          <template v-if="channelForm.type === 'webhook'">
            <div class="form-field">
              <label>URL <span class="required">*</span></label>
              <input v-model="channelForm.config.webhook_url" placeholder="https://example.com/webhook" class="form-input" />
            </div>
            <div class="form-field">
              <label>HTTP 方法</label>
              <select v-model="channelForm.config.webhook_method" class="form-select">
                <option value="POST">POST</option>
                <option value="GET">GET</option>
              </select>
            </div>
          </template>

          <!-- 邮件配置 -->
          <template v-if="channelForm.type === 'email'">
            <div class="form-row">
              <div class="form-field">
                <label>SMTP 服务器</label>
                <input v-model="channelForm.config.smtp_host" placeholder="smtp.example.com" class="form-input" />
              </div>
              <div class="form-field">
                <label>端口</label>
                <input v-model.number="channelForm.config.smtp_port" type="number" placeholder="465" class="form-input" />
              </div>
            </div>
            <div class="form-row">
              <div class="form-field">
                <label>用户名</label>
                <input v-model="channelForm.config.smtp_user" class="form-input" />
              </div>
              <div class="form-field">
                <label>密码</label>
                <input v-model="channelForm.config.smtp_password" type="password" class="form-input" />
              </div>
            </div>
            <div class="form-field">
              <label>发件人</label>
              <input v-model="channelForm.config.from" placeholder="noreply@example.com" class="form-input" />
            </div>
          </template>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="showChannelModal = false">取消</button>
          <button class="btn-primary" @click="saveChannel">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  PlusIcon, XIcon, PencilIcon, TrashIcon, BellIcon, AlertTriangleIcon,
  HistoryIcon, CheckCircleIcon, FlaskConicalIcon, MailIcon, LinkIcon,
  MessageSquareIcon
} from 'lucide-vue-next'
import { alertApi, type AlertRule, type NotificationChannel, type AlertHistory, type AlertCondition } from '@/api/alert'

// 状态
const activeTab = ref<'rules' | 'channels' | 'history'>('rules')
const loadingRules = ref(false)
const loadingChannels = ref(false)
const loadingHistory = ref(false)
const rules = ref<AlertRule[]>([])
const channels = ref<NotificationChannel[]>([])
const history = ref<AlertHistory[]>([])

// 弹窗状态
const showRuleModal = ref(false)
const showChannelModal = ref(false)
const ruleForm = reactive<{
  id: string
  name: string
  description: string
  type: 'metric' | 'error' | 'custom'
  condition: AlertCondition
  channels: string[]
}>({
  id: '',
  name: '',
  description: '',
  type: 'metric',
  condition: { metric: 'latency', operator: '>', threshold: 1000, duration: 60, aggregation: 'avg' },
  channels: []
})
const channelForm = reactive<{
  id: string
  name: string
  type: 'email' | 'dingtalk' | 'wecom' | 'webhook'
  config: Record<string, any>
}>({
  id: '',
  name: '',
  type: 'dingtalk',
  config: {}
})

// 加载数据
async function loadRules() {
  loadingRules.value = true
  try {
    rules.value = await alertApi.listRules()
  } finally {
    loadingRules.value = false
  }
}

async function loadChannels() {
  loadingChannels.value = true
  try {
    channels.value = await alertApi.listChannels()
  } finally {
    loadingChannels.value = false
  }
}

async function loadHistory() {
  loadingHistory.value = true
  try {
    history.value = await alertApi.listHistory()
  } finally {
    loadingHistory.value = false
  }
}

// 规则操作
function openRuleModal(rule?: AlertRule) {
  if (rule) {
    ruleForm.id = rule.id
    ruleForm.name = rule.name
    ruleForm.description = rule.description
    ruleForm.type = rule.type
    ruleForm.condition = { ...rule.condition }
    ruleForm.channels = [...rule.channels]
  } else {
    ruleForm.id = ''
    ruleForm.name = ''
    ruleForm.description = ''
    ruleForm.type = 'metric'
    ruleForm.condition = { metric: 'latency', operator: '>', threshold: 1000, duration: 60, aggregation: 'avg' }
    ruleForm.channels = []
  }
  showRuleModal.value = true
}

async function saveRule() {
  if (!ruleForm.name) return alert('请输入规则名称')
  try {
    if (ruleForm.id) {
      await alertApi.updateRule(ruleForm.id, {
        name: ruleForm.name,
        description: ruleForm.description,
        type: ruleForm.type,
        condition: ruleForm.condition,
        channels: ruleForm.channels
      })
    } else {
      await alertApi.createRule({
        name: ruleForm.name,
        description: ruleForm.description,
        type: ruleForm.type,
        condition: ruleForm.condition,
        channels: ruleForm.channels
      })
    }
    showRuleModal.value = false
    loadRules()
  } catch (e: any) {
    alert(e.message || '保存失败')
  }
}

async function toggleRule(rule: AlertRule) {
  try {
    await alertApi.updateRule(rule.id, { enabled: !rule.enabled })
    loadRules()
  } catch (e) {
    loadRules()
  }
}

async function deleteRule(rule: AlertRule) {
  if (!confirm(`确定删除规则 "${rule.name}"？`)) return
  try {
    await alertApi.deleteRule(rule.id)
    loadRules()
  } catch (e: any) {
    alert(e.message || '删除失败')
  }
}

// 渠道操作
function openChannelModal(channel?: NotificationChannel) {
  if (channel) {
    channelForm.id = channel.id
    channelForm.name = channel.name
    channelForm.type = channel.type
    channelForm.config = { ...channel.config }
  } else {
    channelForm.id = ''
    channelForm.name = ''
    channelForm.type = 'dingtalk'
    channelForm.config = {}
  }
  showChannelModal.value = true
}

async function saveChannel() {
  if (!channelForm.name) return alert('请输入渠道名称')
  try {
    if (channelForm.id) {
      await alertApi.updateChannel(channelForm.id, {
        name: channelForm.name,
        type: channelForm.type,
        config: channelForm.config
      })
    } else {
      await alertApi.createChannel({
        name: channelForm.name,
        type: channelForm.type,
        config: channelForm.config
      })
    }
    showChannelModal.value = false
    loadChannels()
  } catch (e: any) {
    alert(e.message || '保存失败')
  }
}

async function toggleChannel(channel: NotificationChannel) {
  try {
    await alertApi.updateChannel(channel.id, { enabled: !channel.enabled })
    loadChannels()
  } catch (e) {
    loadChannels()
  }
}

async function deleteChannel(channel: NotificationChannel) {
  if (!confirm(`确定删除渠道 "${channel.name}"？`)) return
  try {
    await alertApi.deleteChannel(channel.id)
    loadChannels()
  } catch (e: any) {
    alert(e.message || '删除失败')
  }
}

async function testChannel(channel: NotificationChannel) {
  try {
    const result = await alertApi.testChannel(channel.id)
    alert(result.success ? '测试成功' : `测试失败: ${result.message}`)
  } catch (e: any) {
    alert(e.message || '测试失败')
  }
}

// 辅助函数
function getTypeLabel(type: string) {
  const labels: Record<string, string> = {
    metric: '指标告警',
    error: '错误告警',
    custom: '自定义'
  }
  return labels[type] || type
}

function getChannelNames(ids: string[]) {
  return ids.map(id => channels.value.find(c => c.id === id)?.name || id).join(', ') || '无'
}

function getChannelTypeLabel(type: string) {
  const labels: Record<string, string> = {
    email: '邮件',
    dingtalk: '钉钉',
    wecom: '企业微信',
    webhook: 'Webhook'
  }
  return labels[type] || type
}

function getChannelIcon(type: string) {
  const icons: Record<string, any> = {
    email: MailIcon,
    dingtalk: MessageSquareIcon,
    wecom: MessageSquareIcon,
    webhook: LinkIcon
  }
  return icons[type] || BellIcon
}

function formatTime(time: string) {
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadRules()
  loadChannels()
  loadHistory()
})
</script>

<style scoped>
.page-root {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 4px 0;
}

.page-desc {
  color: var(--text-secondary);
  font-size: 14px;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 16px;
  text-align: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--text-primary);
}

.stat-value.text-success { color: var(--success); }
.stat-value.text-warning { color: var(--warning); }

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 4px;
}

.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 8px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  font-size: 14px;
}

.tab-btn:hover {
  background: var(--bg-secondary);
}

.tab-btn.active {
  background: var(--primary);
  color: white;
}

.content-panel {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  font-weight: 500;
}

.list-empty {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
}

.empty-icon {
  opacity: 0.3;
  margin-bottom: 8px;
}

/* 规则卡片 */
.rule-list {
  display: grid;
  gap: 12px;
}

.rule-card {
  background: var(--bg-primary);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid var(--border);
}

.rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.rule-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.rule-name {
  font-weight: 500;
}

.rule-type {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--bg-tertiary);
}

.rule-type.metric { background: #dbeafe; color: #1e40af; }
.rule-type.error { background: #fee2e2; color: #991b1b; }
.rule-type.custom { background: #f3e8ff; color: #6b21a8; }

.rule-condition {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  margin-bottom: 8px;
}

.rule-condition .metric { font-weight: 500; }
.rule-condition .operator { color: var(--text-secondary); }
.rule-condition .threshold { color: var(--primary); font-weight: 500; }
.rule-condition .duration { color: var(--text-secondary); font-size: 12px; }

.rule-channels {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.rule-last-trigger {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 8px;
}

.rule-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

/* 渠道卡片 */
.channel-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}

.channel-card {
  background: var(--bg-primary);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid var(--border);
}

.channel-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.channel-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
}

.channel-icon.dingtalk { background: #e6f7ff; color: #1890ff; }
.channel-icon.wecom { background: #e8f5e9; color: #4caf50; }
.channel-icon.email { background: #fff3e0; color: #ff9800; }
.channel-icon.webhook { background: #f3e5f5; color: #9c27b0; }

.channel-info {
  flex: 1;
}

.channel-name {
  font-weight: 500;
  display: block;
}

.channel-type {
  font-size: 12px;
  color: var(--text-secondary);
}

.channel-config {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.webhook-url {
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}

.channel-actions {
  display: flex;
  gap: 8px;
}

/* 历史列表 */
.history-list {
  display: grid;
  gap: 8px;
}

.history-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  background: var(--bg-primary);
  border-radius: 8px;
  border: 1px solid var(--border);
}

.history-item.triggered { border-left: 3px solid var(--error); }
.history-item.resolved { border-left: 3px solid var(--success); }

.history-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
}

.history-item.triggered .history-icon { color: var(--error); }
.history-item.resolved .history-icon { color: var(--success); }

.history-content {
  flex: 1;
}

.history-title {
  font-weight: 500;
  margin-bottom: 4px;
}

.history-message {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.history-meta {
  font-size: 12px;
  color: var(--text-tertiary);
  display: flex;
  gap: 16px;
}

.history-status {
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 4px;
}

.history-status.triggered { background: #fee2e2; color: #991b1b; }
.history-status.resolved { background: #dcfce7; color: #166534; }

/* 开关 */
.toggle {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--bg-tertiary);
  transition: 0.2s;
  border-radius: 20px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 2px;
  bottom: 2px;
  background-color: white;
  transition: 0.2s;
  border-radius: 50%;
}

.toggle input:checked + .toggle-slider {
  background-color: var(--primary);
}

.toggle input:checked + .toggle-slider:before {
  transform: translateX(16px);
}

/* 按钮 */
.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--primary);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-secondary:hover {
  background: var(--border);
}

.btn-sm {
  padding: 6px 12px;
  font-size: 13px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.action-btn:hover {
  background: var(--border);
}

.action-btn.danger {
  color: var(--error);
}

/* 弹窗 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-card {
  background: var(--bg-primary);
  border-radius: 12px;
  width: 90%;
  max-width: 560px;
  max-height: 90vh;
  overflow: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}

.icon-close {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 4px;
}

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 20px;
  border-top: 1px solid var(--border);
}

/* 表单 */
.form-field {
  margin-bottom: 16px;
}

.form-field label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 6px;
}

.form-field .required {
  color: var(--error);
}

.form-input, .form-select, .form-textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 14px;
}

.form-textarea {
  min-height: 60px;
  resize: vertical;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.form-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin: 16px 0 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.channel-checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 13px;
}
</style>