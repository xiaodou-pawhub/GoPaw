<template>
  <aside v-if="!hidden" class="nav-panel">
    <slot />
  </aside>
</template>

<script setup lang="ts">
defineProps<{
  hidden?: boolean
}>()
</script>

<style scoped>
.nav-panel {
  width: 220px;
  height: 100%;
  background: var(--bg-panel);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}
</style>
