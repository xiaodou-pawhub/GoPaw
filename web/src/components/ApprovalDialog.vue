<template>
  <Dialog v-model:open="visible" class="approval-dialog">
    <DialogContent class="sm:max-w-[500px]">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <ShieldAlert class="w-5 h-5 text-amber-500" />
          需要确认
        </DialogTitle>
        <DialogDescription>
          Agent 请求执行敏感操作，需要您的确认
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-4">
        <!-- Tool Info -->
        <div class="bg-muted rounded-lg p-4 space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-muted-foreground">操作</span>
            <Badge variant="secondary" class="font-mono">{{ request?.tool_name }}</Badge>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-muted-foreground">级别</span>
            <Badge :variant="levelVariant" class="font-mono">{{ request?.level }}</Badge>
          </div>
          <div v-if="request?.agent_id" class="flex items-center justify-between">
            <span class="text-sm font-medium text-muted-foreground">Agent</span>
            <span class="text-sm font-mono">{{ request?.agent_id }}</span>
          </div>
        </div>

        <!-- Args Preview -->
        <div v-if="request?.args" class="space-y-2">
          <Label class="text-sm font-medium">参数</Label>
          <div class="bg-muted rounded-lg p-3 max-h-32 overflow-auto">
            <pre class="text-xs font-mono whitespace-pre-wrap">{{ formattedArgs }}</pre>
          </div>
        </div>

        <!-- Reason Input -->
        <div class="space-y-2">
          <Label for="reason" class="text-sm font-medium">
            拒绝原因（可选）
          </Label>
          <Input
            id="reason"
            v-model="reason"
            placeholder="如果拒绝，请说明原因..."
            class="h-10"
          />
        </div>

        <!-- Warning -->
        <Alert variant="warning" class="border-amber-500/50 bg-amber-50">
          <AlertTriangle class="h-4 w-4 text-amber-600" />
          <AlertTitle class="text-amber-800">注意</AlertTitle>
          <AlertDescription class="text-amber-700">
            此操作可能需要一定时间执行，批准后将立即开始。
          </AlertDescription>
        </Alert>
      </div>

      <DialogFooter class="gap-2">
        <Button
          variant="outline"
          @click="handleReject"
          :disabled="loading"
          class="flex-1"
        >
          <XCircle class="w-4 h-4 mr-2" />
          拒绝
        </Button>
        <Button
          @click="handleApprove"
          :disabled="loading"
          class="flex-1"
        >
          <CheckCircle class="w-4 h-4 mr-2" />
          批准
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  ShieldAlert,
  CheckCircle,
  XCircle,
  AlertTriangle,
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
  approve: [requestId: string]
  reject: [requestId: string, reason: string]
}>()

const visible = ref(false)
const loading = ref(false)
const reason = ref('')

// Show dialog when request arrives
watch(() => props.request, (newRequest) => {
  if (newRequest) {
    visible.value = true
    reason.value = ''
  }
}, { immediate: true })

const formattedArgs = computed(() => {
  if (!props.request?.args) return ''
  try {
    const parsed = JSON.parse(props.request.args)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return props.request.args
  }
})

const levelVariant = computed(() => {
  switch (props.request?.level) {
    case 'L1':
      return 'default'
    case 'L2':
      return 'secondary'
    case 'L3':
      return 'destructive'
    default:
      return 'secondary'
  }
})

const handleApprove = async () => {
  if (!props.request) return
  
  loading.value = true
  try {
    emit('approve', props.request.id)
    visible.value = false
  } finally {
    loading.value = false
  }
}

const handleReject = async () => {
  if (!props.request) return
  
  loading.value = true
  try {
    emit('reject', props.request.id, reason.value)
    visible.value = false
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.approval-dialog :deep(.dialog-content) {
  max-height: 90vh;
  overflow-y: auto;
}
</style>
