<template>
  <div v-if="visible" class="modal-overlay" @click.self="handleReject">
    <div class="modal-content approval-dialog">
      <div class="modal-header">
        <div class="header-title">
          <ShieldAlertIcon class="icon-warning" />
          <h3>需要确认</h3>
        </div>
        <p class="header-desc">Agent 请求执行敏感操作，需要您的确认</p>
      </div>

      <div class="modal-body">
        <!-- Tool Info -->
        <div class="info-box">
          <div class="info-row">
            <span class="info-label">操作</span>
            <span class="info-value badge">{{ request?.tool_name }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">级别</span>
            <span class="info-value badge" :class="levelClass">{{ request?.level }}</span>
          </div>
          <div v-if="request?.agent_id" class="info-row">
            <span class="info-label">Agent</span>
            <span class="info-value">{{ request?.agent_id }}</span>
          </div>
        </div>

        <!-- Args Preview -->
        <div v-if="request?.args" class="args-section">
          <label class="section-label">参数</label>
          <div class="args-preview">
            <pre>{{ formattedArgs }}</pre>
          </div>
        </div>

        <!-- Reason Input -->
        <div class="reason-section">
          <label class="section-label">拒绝原因（可选）</label>
          <input
            v-model="reason"
            type="text"
            placeholder="如果拒绝，请说明原因..."
            class="reason-input"
          />
        </div>

        <!-- Warning -->
        <div class="warning-box">
          <AlertTriangleIcon class="icon-small" />
          <div>
            <strong>注意</strong>
            <p>此操作可能需要一定时间执行，批准后将立即开始。</p>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button
          class="btn btn-secondary"
          :disabled="loading"
          @click="handleReject"
        >
          <XCircleIcon class="icon-small" />
          拒绝
        </button>
        <button
          class="btn btn-primary"
          :disabled="loading"
          @click="handleApprove"
        >
          <CheckCircleIcon class="icon-small" />
          批准
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  ShieldAlert as ShieldAlertIcon,
  CheckCircle as CheckCircleIcon,
  XCircle as XCircleIcon,
  AlertTriangle as AlertTriangleIcon,
} from 'lucide-vue-next'

interface ApprovalRequest {
  id: string
  tool_name: string
  args: string
  level: string
  requested_at: string
  session_id: string
  agent_id?: string
}

const props = defineProps<{
  request: ApprovalRequest | null
}>()

const emit = defineEmits<{
  approve: [request: ApprovalRequest, reason?: string]
  reject: [request: ApprovalRequest, reason?: string]
}>()

const visible = computed(() => props.request !== null)
const reason = ref('')
const loading = ref(false)

const formattedArgs = computed(() => {
  if (!props.request?.args) return ''
  try {
    const parsed = JSON.parse(props.request.args)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return props.request.args
  }
})

const levelClass = computed(() => {
  const level = props.request?.level
  if (level === 'L3') return 'badge-danger'
  if (level === 'L2') return 'badge-warning'
  return 'badge-info'
})

watch(() => props.request, () => {
  reason.value = ''
  loading.value = false
})

function handleApprove() {
  if (!props.request) return
  loading.value = true
  emit('approve', props.request, reason.value)
}

function handleReject() {
  if (!props.request) return
  loading.value = true
  emit('reject', props.request, reason.value)
}
</script>

<style scoped>
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

.modal-content {
  width: 90%;
  max-width: 500px;
  background: var(--bg-card);
  border-radius: 8px;
  overflow: hidden;
}

.modal-header {
  padding: 20px;
  border-bottom: 1px solid var(--border);
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.header-title h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.icon-warning {
  width: 20px;
  height: 20px;
  color: var(--amber);
}

.header-desc {
  font-size: 13px;
  color: var(--text-secondary);
}

.modal-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-box {
  background: var(--bg-overlay);
  border-radius: 6px;
  padding: 12px;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 0;
}

.info-row:not(:last-child) {
  border-bottom: 1px solid var(--border);
}

.info-label {
  font-size: 12px;
  color: var(--text-tertiary);
}

.info-value {
  font-size: 13px;
  color: var(--text-primary);
  font-family: monospace;
}

.badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  background: var(--bg-input);
}

.badge-danger {
  background: var(--red-dim);
  color: var(--red);
}

.badge-warning {
  background: var(--amber-dim);
  color: var(--amber);
}

.badge-info {
  background: var(--accent-dim);
  color: var(--accent);
}

.args-section,
.reason-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.args-preview {
  background: var(--bg-overlay);
  border-radius: 6px;
  padding: 12px;
  max-height: 120px;
  overflow: auto;
}

.args-preview pre {
  font-size: 11px;
  font-family: monospace;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.reason-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 13px;
}

.reason-input:focus {
  outline: none;
  border-color: var(--accent);
}

.warning-box {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: var(--amber-dim);
  border: 1px solid var(--amber);
  border-radius: 6px;
}

.warning-box .icon-small {
  width: 16px;
  height: 16px;
  color: var(--amber);
  flex-shrink: 0;
}

.warning-box strong {
  font-size: 12px;
  color: var(--amber);
}

.warning-box p {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0;
}

.modal-footer {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border);
}

.btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-secondary);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-overlay);
}

.btn-primary {
  border: none;
  background: var(--accent);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.icon-small {
  width: 16px;
  height: 16px;
}
</style>
