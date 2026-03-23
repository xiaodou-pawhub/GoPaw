<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { PlayIcon } from 'lucide-vue-next'
</script>

<template>
  <div class="flow-node start-node">
    <div class="node-icon">
      <PlayIcon :size="16" />
    </div>
    <span class="node-label">开始</span>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>

<style scoped>
.flow-node {
  padding: 8px 12px;
  border-radius: 8px;
  background: #fff;
  border: 2px solid #22c55e;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 80px;
}
.node-icon {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #22c55e;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.node-label {
  font-size: 12px;
  font-weight: 500;
  color: #333;
}
</style>