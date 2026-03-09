<template>
  <div class="loading-spinner" :class="{ 'loading-fullscreen': fullscreen }">
    <div class="spinner">
      <div class="spinner-circle" />
      <div class="spinner-circle" />
      <div class="spinner-circle" />
    </div>
    <p v-if="text" class="loading-text">{{ text }}</p>
  </div>
</template>

<script setup lang="ts">
interface Props {
  text?: string
  fullscreen?: boolean
}

withDefaults(defineProps<Props>(), {
  text: '',
  fullscreen: false
})
</script>

<style scoped>
.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.loading-fullscreen {
  position: fixed;
  inset: 0;
  background: rgba(247, 247, 245, 0.8);
  backdrop-filter: blur(4px);
  z-index: 9999;
}

.spinner {
  display: flex;
  gap: 8px;
  align-items: center;
}

.spinner-circle {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--accent);
  animation: bounce 1.4s ease-in-out infinite;
}

.spinner-circle:nth-child(1) {
  animation-delay: 0s;
}

.spinner-circle:nth-child(2) {
  animation-delay: 0.2s;
}

.spinner-circle:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes bounce {
  0%, 80%, 100% {
    transform: scale(0.6);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

.loading-text {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}
</style>
