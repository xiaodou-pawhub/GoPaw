<template>
  <div class="app-root">
    <div class="workspace">
      <IconSidebar />
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import IconSidebar from './IconSidebar.vue'
</script>

<style scoped>
.app-root {
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-app);
  overflow: hidden;
}

.workspace {
  flex: 1;
  display: flex;
  overflow: hidden;
}
</style>
