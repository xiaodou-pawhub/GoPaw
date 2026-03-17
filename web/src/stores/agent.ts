import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listAgents, getDefaultAgent, type Agent } from '@/api/agents'

export const useAgentStore = defineStore('agent', () => {
  // ---- State ----
  const agents = ref<Agent[]>([])
  const currentAgentId = ref<string | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ---- Getters ----
  const currentAgent = computed(() => {
    return agents.value.find(a => a.id === currentAgentId.value) || null
  })

  const defaultAgent = computed(() => {
    return agents.value.find(a => a.is_default) || agents.value[0] || null
  })

  const agentOptions = computed(() => {
    return agents.value.map(agent => ({
      value: agent.id,
      label: `${agent.avatar || '🤖'} ${agent.name}`,
      agent
    }))
  })

  // ---- Actions ----
  async function loadAgents() {
    loading.value = true
    error.value = null
    try {
      const res = await listAgents()
      agents.value = res.agents

      // If no current agent, set to default
      if (!currentAgentId.value && defaultAgent.value) {
        currentAgentId.value = defaultAgent.value.id
      }
    } catch (err) {
      error.value = (err as Error).message
      console.error('Failed to load agents:', err)
    } finally {
      loading.value = false
    }
  }

  async function loadDefaultAgent() {
    try {
      const agent = await getDefaultAgent()
      currentAgentId.value = agent.id
    } catch (err) {
      console.error('Failed to load default agent:', err)
    }
  }

  function setCurrentAgent(agentId: string) {
    currentAgentId.value = agentId
  }

  function getAgentById(agentId: string): Agent | null {
    return agents.value.find(a => a.id === agentId) || null
  }

  // Initialize on store creation
  loadAgents()

  return {
    // State
    agents,
    currentAgentId,
    loading,
    error,
    // Getters
    currentAgent,
    defaultAgent,
    agentOptions,
    // Actions
    loadAgents,
    loadDefaultAgent,
    setCurrentAgent,
    getAgentById
  }
})
