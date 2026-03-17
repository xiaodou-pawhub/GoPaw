<template>
  <div class="app-root">
    <TitleBar :title="pageTitle" />
    <div class="workspace">
      <IconSidebar />
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import TitleBar from './TitleBar.vue'
import IconSidebar from './IconSidebar.vue'

const route = useRoute()

const pageTitle = computed(() => {
  if (route.path.startsWith('/chat')) return 'GoPaw'
  if (route.path.startsWith('/market')) return '技能市场'
  if (route.path.startsWith('/traces')) return '执行轨迹'
  if (route.path.startsWith('/agents')) return 'Agents'
  if (route.path.startsWith('/mcp')) return 'MCP 服务器'
  if (route.path.startsWith('/triggers')) return 'Triggers'
  if (route.path.startsWith('/agent-messages')) return 'Agent 消息'
  if (route.path.startsWith('/workflows')) return '工作流'
  if (route.path.startsWith('/settings')) return '设置'
  return 'GoPaw'
})
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
