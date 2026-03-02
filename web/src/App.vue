<template>
  <n-config-provider :theme="theme">
    <n-message-provider>
      <router-view />
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { computed } from 'vue'
import { NConfigProvider, NMessageProvider, darkTheme } from 'naive-ui'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

// 中文：根据应用商店状态选择主题
// English: Select theme based on app store state
const theme = computed(() => (appStore.isDark ? darkTheme : null))
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
