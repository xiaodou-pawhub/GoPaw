<template>
  <div class="timeline-root">
    <div v-if="loading" class="timeline-loading">
      <div class="loading-spinner" />
      <span>加载步骤详情...</span>
    </div>

    <div v-else-if="error" class="timeline-error">
      <AlertCircleIcon :size="20" />
      <span>{{ error }}</span>
    </div>

    <div v-else-if="trace" class="timeline-content">
      <!-- 步骤时间线 -->
      <div class="timeline-steps">
        <div
          v-for="(step, index) in trace.steps"
          :key="index"
          class="timeline-step"
          :class="{ expanded: expandedStep === index }"
        >
          <!-- 步骤头部 -->
          <div class="step-header" @click="toggleStep(index)">
            <div class="step-connector">
              <div class="step-dot" :style="{ background: getStepTypeColor(step.step_type) }">
                <component :is="getStepTypeIcon(step.step_type)" :size="10" />
              </div>
              <div v-if="index < trace.steps.length - 1" class="step-line" />
            </div>

            <div class="step-info">
              <div class="step-title">
                <span class="step-type">{{ getStepTypeLabel(step.step_type) }}</span>
                <span class="step-number">#{{ step.step_number }}</span>
              </div>
              <div class="step-meta">
                <span class="meta-item">
                  <TimerIcon :size="11" />
                  {{ formatDuration(step.duration_ms) }}
                </span>
                <span class="meta-item">
                  <ClockIcon :size="11" />
                  {{ formatTimestamp(step.started_at) }}
                </span>
              </div>
            </div>

            <div class="step-toggle">
              <ChevronDownIcon :size="14" :class="{ rotated: expandedStep === index }" />
            </div>
          </div>

          <!-- 步骤详情 -->
          <div v-if="expandedStep === index" class="step-detail">
            <div class="detail-sections">
              <!-- Input -->
              <div v-if="step.input" class="detail-section">
                <div class="section-header" @click="toggleSection(index, 'input')">
                  <span class="section-title">输入</span>
                  <ChevronDownIcon :size="12" :class="{ rotated: isSectionExpanded(index, 'input') }" />
                </div>
                <pre v-if="isSectionExpanded(index, 'input')" class="section-content">{{ formatJSON(step.input) }}</pre>
              </div>

              <!-- Output -->
              <div v-if="step.output" class="detail-section">
                <div class="section-header" @click="toggleSection(index, 'output')">
                  <span class="section-title">输出</span>
                  <ChevronDownIcon :size="12" :class="{ rotated: isSectionExpanded(index, 'output') }" />
                </div>
                <pre v-if="isSectionExpanded(index, 'output')" class="section-content">{{ formatJSON(step.output) }}</pre>
              </div>

              <!-- Metadata -->
              <div v-if="step.metadata" class="detail-section">
                <div class="section-header" @click="toggleSection(index, 'metadata')">
                  <span class="section-title">元数据</span>
                  <ChevronDownIcon :size="12" :class="{ rotated: isSectionExpanded(index, 'metadata') }" />
                </div>
                <pre v-if="isSectionExpanded(index, 'metadata')" class="section-content">{{ formatJSON(step.metadata) }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部操作 -->
      <div class="timeline-footer">
        <button class="close-btn" @click="$emit('close')">
          <XIcon :size="14" />
          <span>关闭</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  ChevronDownIcon,
  ClockIcon,
  TimerIcon,
  AlertCircleIcon,
  XIcon,
  SettingsIcon,
  MessageSquareIcon,
  WrenchIcon,
  ZapIcon,
  CheckCircleIcon
} from 'lucide-vue-next'
import { getTrace, type TraceDetail } from '@/api/trace'
import { formatDuration, formatTimestamp, getStepTypeLabel, getStepTypeColor } from '@/api/trace'

// ---- Props ----
const props = defineProps<{
  traceId: string
}>()

// ---- Emits ----
defineEmits<{
  close: []
}>()

// ---- State ----
const trace = ref<TraceDetail | null>(null)
const loading = ref(false)
const error = ref('')
const expandedStep = ref<number | null>(null)
const expandedSections = ref<Map<string, boolean>>(new Map())

// ---- Methods ----
async function loadTrace() {
  loading.value = true
  error.value = ''
  try {
    trace.value = await getTrace(props.traceId)
  } catch (err) {
    error.value = '加载轨迹详情失败'
    console.error('Failed to load trace:', err)
  } finally {
    loading.value = false
  }
}

function toggleStep(index: number) {
  expandedStep.value = expandedStep.value === index ? null : index
}

function toggleSection(stepIndex: number, section: string) {
  const key = `${stepIndex}-${section}`
  expandedSections.value.set(key, !expandedSections.value.get(key))
}

function isSectionExpanded(stepIndex: number, section: string): boolean {
  const key = `${stepIndex}-${section}`
  return expandedSections.value.get(key) ?? false
}

function formatJSON(data: unknown): string {
  try {
    return JSON.stringify(data, null, 2)
  } catch {
    return String(data)
  }
}

function getStepTypeIcon(type: string) {
  switch (type) {
    case 'context_build': return SettingsIcon
    case 'llm_call': return MessageSquareIcon
    case 'tool_execution': return WrenchIcon
    case 'hook_execution': return ZapIcon
    case 'final_answer': return CheckCircleIcon
    default: return SettingsIcon
  }
}

// ---- Lifecycle ----
onMounted(() => {
  loadTrace()
})
</script>

<style scoped>
.timeline-root {
  width: 100%;
}

.timeline-loading,
.timeline-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  color: var(--text-tertiary);
}

.timeline-error {
  color: var(--red);
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Timeline Steps */
.timeline-steps {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.timeline-step {
  position: relative;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.step-header:hover {
  background: var(--bg-overlay);
}

.step-connector {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 20px;
}

.step-dot {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: white;
  flex-shrink: 0;
}

.step-line {
  width: 2px;
  flex: 1;
  min-height: 20px;
  background: var(--border);
  margin: 4px 0;
}

.step-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.step-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.step-type {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.step-number {
  font-size: 11px;
  color: var(--text-tertiary);
  background: var(--bg-overlay);
  padding: 1px 6px;
  border-radius: 4px;
}

.step-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-tertiary);
}

.step-toggle {
  color: var(--text-tertiary);
  transition: transform 0.15s;
}

.rotated {
  transform: rotate(180deg);
}

/* Step Detail */
.step-detail {
  padding: 0 0 12px 32px;
}

.detail-sections {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--bg-overlay);
  cursor: pointer;
  transition: background 0.15s;
}

.section-header:hover {
  background: var(--border);
}

.section-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.section-content {
  padding: 12px;
  margin: 0;
  font-size: 11px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  color: var(--text-secondary);
  background: var(--bg-app);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
}

/* Timeline Footer */
.timeline-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
  margin-top: 16px;
  border-top: 1px solid var(--border);
}

.close-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}
</style>
