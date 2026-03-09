<template>
  <div class="titlebar">
    <!-- macOS 红绿灯占位 -->
    <div class="titlebar-traffic" />
    <!-- 标题（可选） -->
    <span v-if="title" class="titlebar-title">{{ title }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
}>()
</script>

<style scoped>
.titlebar {
  height: 38px;
  width: 100%;
  display: flex;
  align-items: center;
  padding: 0 12px;
  -webkit-app-region: drag;
  flex-shrink: 0;
  background: var(--bg-app);
  border-bottom: 1px solid var(--border-subtle);
  user-select: none;
}

.titlebar-traffic {
  width: 72px;
  height: 100%;
  -webkit-app-region: no-drag;
  flex-shrink: 0;
}

.titlebar-title {
  flex: 1;
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
