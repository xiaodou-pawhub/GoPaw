<template>
  <div class="models-page">
    <ProvidersTab />
  </div>
</template>

<script setup lang="ts">
import ProvidersTab from '@/components/settings/ProvidersTab.vue'
</script>

<style scoped>
.models-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-app);
  padding: 24px;
}
</style>
