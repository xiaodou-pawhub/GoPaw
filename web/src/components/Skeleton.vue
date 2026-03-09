<template>
  <div class="skeleton-container" :style="{ width, height: height || 'auto' }">
    <div class="skeleton-item" :class="[animation, shape]" />
  </div>
</template>

<script setup lang="ts">
interface Props {
  width?: string
  height?: string
  animation?: 'pulse' | 'wave' | 'none'
  shape?: 'rect' | 'circle' | 'round'
}

withDefaults(defineProps<Props>(), {
  width: '100%',
  height: '16px',
  animation: 'pulse',
  shape: 'rect'
})
</script>

<style scoped>
.skeleton-container {
  display: inline-block;
  background: var(--bg-overlay);
  overflow: hidden;
}

.skeleton-item {
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    var(--bg-overlay) 0%,
    var(--bg-elevated) 50%,
    var(--bg-overlay) 100%
  );
  background-size: 200% 100%;
}

/* 形状 */
.skeleton-item.rect {
  border-radius: 4px;
}

.skeleton-item.circle {
  border-radius: 50%;
}

.skeleton-item.round {
  border-radius: 8px;
}

/* 动画 */
.skeleton-item.pulse {
  animation: skeleton-pulse 1.5s ease-in-out infinite;
}

.skeleton-item.wave {
  animation: skeleton-wave 1.5s ease-in-out infinite;
}

@keyframes skeleton-pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

@keyframes skeleton-wave {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}
</style>
