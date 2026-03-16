import { ref, onMounted, onUnmounted } from 'vue'

interface WebSocketMessage {
  type: string
  payload: any
}

interface ApprovalRequest {
  id: string
  tool_name: string
  args: string
  level: string
  requested_at: string
  session_id: string
  agent_id?: string
}

interface Notification {
  message: string
  tool_name?: string
  args?: string
  result?: string
  timestamp: string
}

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const approvalRequest = ref<ApprovalRequest | null>(null)
  const notifications = ref<Notification[]>([])
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5

  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  const connect = () => {
    const wsUrl = `${import.meta.env.VITE_WS_URL || 'ws://localhost:8080'}/ws`
    
    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      console.log('WebSocket connected')
      connected.value = true
      reconnectAttempts.value = 0
    }

    ws.value.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        handleMessage(message)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    ws.value.onclose = () => {
      console.log('WebSocket disconnected')
      connected.value = false
      attemptReconnect()
    }

    ws.value.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  const handleMessage = (message: WebSocketMessage) => {
    switch (message.type) {
      case 'approval_request':
        approvalRequest.value = message.payload as ApprovalRequest
        break
      case 'notification':
        notifications.value.unshift(message.payload as Notification)
        // Keep only last 50 notifications
        if (notifications.value.length > 50) {
          notifications.value = notifications.value.slice(0, 50)
        }
        break
      default:
        console.log('Unknown message type:', message.type)
    }
  }

  const attemptReconnect = () => {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      console.error('Max reconnect attempts reached')
      return
    }

    reconnectAttempts.value++
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 30000)
    
    console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts.value})`)
    
    reconnectTimer = setTimeout(() => {
      connect()
    }, delay)
  }

  const sendApprovalResponse = (requestId: string, approved: boolean, reason?: string) => {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
      console.error('WebSocket not connected')
      return
    }

    const message = {
      type: 'approval_response',
      payload: {
        request_id: requestId,
        approved,
        reason: reason || '',
        responded_at: new Date().toISOString(),
      },
    }

    ws.value.send(JSON.stringify(message))
    approvalRequest.value = null
  }

  const approve = (requestId: string) => {
    sendApprovalResponse(requestId, true)
  }

  const reject = (requestId: string, reason: string) => {
    sendApprovalResponse(requestId, false, reason)
  }

  const clearNotifications = () => {
    notifications.value = []
  }

  onMounted(() => {
    connect()
  })

  onUnmounted(() => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }
    if (ws.value) {
      ws.value.close()
    }
  })

  return {
    connected,
    approvalRequest,
    notifications,
    approve,
    reject,
    clearNotifications,
  }
}
