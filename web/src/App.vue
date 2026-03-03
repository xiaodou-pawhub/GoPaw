<template>
  <n-config-provider>
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { NConfigProvider, NMessageProvider, NDialogProvider } from 'naive-ui'
</script>

<style>
#app {
  width: 100%;
  height: 100vh;
}

body {
  margin: 0;
  padding: 0;
}
</style>
