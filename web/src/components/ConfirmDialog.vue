<template>
  <div class="modal-overlay" @click.self="$emit('cancel')">
    <div class="modal-content">
      <div class="modal-header">
        <h3>{{ title }}</h3>
      </div>
      <div class="modal-body">
        <p>{{ message }}</p>
      </div>
      <div class="modal-footer">
        <button class="btn secondary" @click="$emit('cancel')">取消</button>
        <button class="btn danger" @click="$emit('confirm')">确认</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// ---- Props ----
defineProps<{
  title: string
  message: string
}>()

// ---- Emits ----
defineEmits<{
  confirm: []
  cancel: []
}>()
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
  max-width: 400px;
  background: var(--bg-card);
  border-radius: 8px;
  overflow: hidden;
}

.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-body {
  padding: 20px;
}

.modal-body p {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border);
}

.btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn.secondary {
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-secondary);
}

.btn.secondary:hover {
  background: var(--bg-overlay);
}

.btn.danger {
  border: none;
  background: var(--red);
  color: white;
}

.btn.danger:hover {
  opacity: 0.9;
}
</style>
