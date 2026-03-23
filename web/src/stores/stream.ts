import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage } from '@/types'

/**
 * StreamStore tracks in-flight chat sessions so that switching away from a
 * session does NOT abort the ongoing stream.  Components can query whether a
 * session is still running (to show a spinner) or has just finished in the
 * background (to show a "new reply" indicator).
 */
export const useStreamStore = defineStore('stream', () => {
  /** Session IDs that are currently streaming. */
  const runningSessions = ref<Set<string>>(new Set())

  /** Session IDs that finished in the background – user hasn't seen the result yet. */
  const pendingReload = ref<Set<string>>(new Set())

  /**
   * Messages accumulated for background sessions.
   * Keyed by session ID so we can restore them when the user switches back.
   */
  const backgroundMessages = ref<Map<string, ChatMessage[]>>(new Map())

  function startSession(id: string) {
    runningSessions.value.add(id)
    pendingReload.value.delete(id)
  }

  function finishSession(id: string, isForeground: boolean) {
    runningSessions.value.delete(id)
    if (!isForeground) {
      pendingReload.value.add(id)
    }
  }

  function markViewed(id: string) {
    pendingReload.value.delete(id)
    backgroundMessages.value.delete(id)
  }

  function saveBackgroundMessages(id: string, msgs: ChatMessage[]) {
    backgroundMessages.value.set(id, [...msgs])
  }

  function getBackgroundMessages(id: string): ChatMessage[] | null {
    return backgroundMessages.value.get(id) ?? null
  }

  function isRunning(id: string): boolean {
    return runningSessions.value.has(id)
  }

  function needsReload(id: string): boolean {
    return pendingReload.value.has(id)
  }

  return {
    runningSessions,
    pendingReload,
    startSession,
    finishSession,
    markViewed,
    saveBackgroundMessages,
    getBackgroundMessages,
    isRunning,
    needsReload,
  }
})
