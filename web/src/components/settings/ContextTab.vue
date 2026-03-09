<template>
  <div class="tab-root">
    <div class="tab-header">
      <div>
        <h2 class="tab-title">工作区背景</h2>
        <p class="tab-desc">设置当前工作区的背景信息、项目信息和用户偏好</p>
      </div>
      <button class="btn-primary" :disabled="saving" @click="handleSave">
        {{ saving ? '保存中...' : '保存' }}
      </button>
    </div>

    <div class="editor-card">
      <div class="editor-status" :class="{ modified: isModified }">
        <span class="status-dot" />
        <span>{{ isModified ? '未保存的修改' : '已同步' }}</span>
      </div>
      <textarea
        v-model="content"
        class="md-editor"
        placeholder="在此输入工作区背景描述..."
        @input="isModified = true"
      />
      <div class="editor-tip">支持 Markdown 语法，设定将作为 System Prompt 注入对话上下文</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import { getWorkspaceContext, saveWorkspaceContext } from '@/api/settings'

const content = ref('')
const saving = ref(false)
const isModified = ref(false)

async function loadData() {
  try {
    const res = await getWorkspaceContext()
    content.value = res.content || ''
    isModified.value = false
  } catch {}
}

async function handleSave() {
  saving.value = true
  try {
    await saveWorkspaceContext(content.value)
    toast.success('保存成功')
    isModified.value = false
  } catch {
    toast.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.tab-root { display: flex; flex-direction: column; gap: 20px; }

.tab-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.tab-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin: 0 0 4px; }
.tab-desc { font-size: 12px; color: var(--text-secondary); margin: 0; }

.editor-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}

.editor-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border-subtle);
  font-size: 11px;
  color: var(--text-tertiary);
}

.status-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); }
.editor-status.modified .status-dot { background: var(--yellow); }

.md-editor {
  width: 100%;
  min-height: 360px;
  padding: 16px;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-family: "SF Mono", "JetBrains Mono", Menlo, monospace;
  font-size: 13px;
  line-height: 1.7;
  resize: vertical;
  box-sizing: border-box;
}

.md-editor::placeholder { color: var(--text-disabled); }

.editor-tip {
  padding: 8px 14px;
  border-top: 1px solid var(--border-subtle);
  font-size: 11px;
  color: var(--text-tertiary);
  background: var(--bg-panel);
}

.btn-primary {
  padding: 7px 14px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-primary:hover { background: var(--accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
