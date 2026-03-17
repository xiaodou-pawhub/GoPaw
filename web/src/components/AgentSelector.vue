<template>
  <div class="agent-selector">
    <button
      class="selector-trigger"
      :class="{ active: isOpen }"
      @click="toggleDropdown"
    >
      <span class="agent-avatar">{{ currentAgent?.avatar || '🤖' }}</span>
      <span class="agent-name">{{ currentAgent?.name || '选择 Agent' }}</span>
      <ChevronDownIcon :size="14" class="dropdown-icon" />
    </button>

    <div v-if="isOpen" class="dropdown-menu" v-click-outside="closeDropdown">
      <div class="dropdown-header">
        <span>选择 Agent</span>
        <button class="manage-btn" @click="goToAgents">
          <SettingsIcon :size="12" />
          管理
        </button>
      </div>

      <div class="agent-list">
        <div
          v-for="agent in agents"
          :key="agent.id"
          class="agent-option"
          :class="{ active: agent.id === currentAgentId }"
          @click="selectAgent(agent)"
        >
          <span class="option-avatar">{{ agent.avatar || '🤖' }}</span>
          <div class="option-info">
            <span class="option-name">{{ agent.name }}</span>
            <span class="option-desc">{{ agent.description || agent.id }}</span>
          </div>
          <CheckIcon
            v-if="agent.id === currentAgentId"
            :size="14"
            class="check-icon"
          />
        </div>
      </div>

      <div v-if="loading" class="dropdown-loading">
        <LoaderIcon :size="16" class="spinning" />
        <span>加载中...</span>
      </div>

      <div v-if="!loading && agents.length === 0" class="dropdown-empty">
        <span>暂无 Agents</span>
        <button class="create-btn" @click="goToAgents">创建 Agent</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronDownIcon, CheckIcon, SettingsIcon, LoaderIcon } from 'lucide-vue-next'
import { listAgents, type Agent } from '@/api/agents'

// ---- Props ----
const props = defineProps<{
  modelValue: string | null
}>()

// ---- Emits ----
const emit = defineEmits<{
  'update:modelValue': [value: string]
  'change': [agent: Agent]
}>()

// ---- State ----
const router = useRouter()
const isOpen = ref(false)
const agents = ref<Agent[]>([])
const loading = ref(false)
const currentAgentId = computed(() => props.modelValue)

const currentAgent = computed(() => {
  return agents.value.find(a => a.id === currentAgentId.value) || null
})

// ---- Methods ----
function toggleDropdown() {
  isOpen.value = !isOpen.value
  if (isOpen.value && agents.value.length === 0) {
    loadAgents()
  }
}

function closeDropdown() {
  isOpen.value = false
}

async function loadAgents() {
  loading.value = true
  try {
    const res = await listAgents()
    agents.value = res.agents

    // If no current agent, set default
    if (!currentAgentId.value && agents.value.length > 0) {
      const defaultAgent = agents.value.find(a => a.is_default) || agents.value[0]
      emit('update:modelValue', defaultAgent.id)
    }
  } catch (err) {
    console.error('Failed to load agents:', err)
  } finally {
    loading.value = false
  }
}

function selectAgent(agent: Agent) {
  emit('update:modelValue', agent.id)
  emit('change', agent)
  closeDropdown()
}

function goToAgents() {
  closeDropdown()
  router.push('/agents')
}

// ---- Lifecycle ----
onMounted(() => {
  loadAgents()
})

// ---- Directives ----
interface ClickOutsideElement extends HTMLElement {
  _clickOutside?: (event: Event) => void
}

const vClickOutside = {
  mounted(el: ClickOutsideElement, binding: any) {
    el._clickOutside = (event: Event) => {
      if (!(el === event.target || el.contains(event.target as Node))) {
        binding.value()
      }
    }
    document.addEventListener('click', el._clickOutside, true)
  },
  unmounted(el: ClickOutsideElement) {
    if (el._clickOutside) {
      document.removeEventListener('click', el._clickOutside, true)
    }
  }
}
</script>

<style scoped>
.agent-selector {
  position: relative;
}

.selector-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-input);
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.selector-trigger:hover {
  border-color: var(--border-hover);
  background: var(--bg-overlay);
}

.selector-trigger.active {
  border-color: var(--accent);
}

.agent-avatar {
  font-size: 16px;
}

.agent-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-icon {
  color: var(--text-tertiary);
  transition: transform 0.15s;
}

.selector-trigger.active .dropdown-icon {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  min-width: 240px;
  max-height: 320px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 100;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.dropdown-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
}

.manage-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--accent);
  font-size: 11px;
  cursor: pointer;
}

.manage-btn:hover {
  background: var(--accent-dim);
}

.agent-list {
  overflow-y: auto;
  padding: 4px;
}

.agent-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.agent-option:hover {
  background: var(--bg-overlay);
}

.agent-option.active {
  background: var(--accent-dim);
}

.option-avatar {
  font-size: 18px;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-input);
  border-radius: 6px;
}

.option-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.option-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.option-desc {
  font-size: 11px;
  color: var(--text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.check-icon {
  color: var(--accent);
}

.dropdown-loading,
.dropdown-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.create-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  background: var(--accent);
  color: white;
  font-size: 12px;
  cursor: pointer;
}

.create-btn:hover {
  opacity: 0.9;
}
</style>
