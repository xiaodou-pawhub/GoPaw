<template>
  <div class="diamond-node" :class="{ selected }">
    <Handle v-if="showInputHandle" type="target" :position="Position.Top" class="node-handle" />

    <div class="diamond-shape" :style="{ background: `linear-gradient(135deg, ${iconColor}, ${iconColor}cc)` }">
      <div class="diamond-content">
        <span class="diamond-abbr">{{ title.charAt(0) }}</span>
        <span class="diamond-title">{{ title }}</span>
        <span v-if="typeLabel" class="type-label">{{ typeLabel }}</span>
      </div>
    </div>

    <Handle v-if="showLeftHandle" id="left" type="source" :position="Position.Left" class="node-handle" />
    <Handle v-if="showRightHandle" id="right" type="source" :position="Position.Right" class="node-handle" />
    <Handle v-if="showBottomHandle" id="bottom" type="source" :position="Position.Bottom" class="node-handle" />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'

withDefaults(defineProps<{
  id: string
  selected?: boolean
  icon?: string
  title: string
  typeLabel?: string
  iconColor?: string
  showInputHandle?: boolean
  showLeftHandle?: boolean
  showRightHandle?: boolean
  showBottomHandle?: boolean
}>(), {
  selected: false,
  iconColor: '#4facfe',
  showInputHandle: true,
  showLeftHandle: true,
  showRightHandle: true,
  showBottomHandle: true,
})
</script>

<style scoped>
.diamond-node {
  width: 100px; height: 100px;
  display: flex; align-items: center; justify-content: center;
  transition: all 0.2s ease;
}

.diamond-shape {
  width: 80px; height: 80px;
  transform: rotate(45deg);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  display: flex; align-items: center; justify-content: center;
  transition: all 0.2s ease;
}

.diamond-node:hover .diamond-shape { box-shadow: 0 4px 12px rgba(0,0,0,0.3); transform: rotate(45deg) scale(1.05); }
.diamond-node.selected .diamond-shape { box-shadow: 0 0 0 4px rgba(79,172,254,0.3); }

.diamond-content {
  transform: rotate(-45deg);
  text-align: center;
  display: flex; flex-direction: column; align-items: center;
  gap: 2px;
  width: 80px;
}

.diamond-abbr {
  font-size: 14px; font-weight: 700; color: #fff;
}

.diamond-title {
  font-weight: 600; font-size: 11px; color: rgba(255,255,255,0.9);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 70px;
}

.type-label {
  font-size: 10px; color: rgba(255,255,255,0.8);
  background: rgba(255,255,255,0.2);
  padding: 1px 5px; border-radius: 4px;
}

.node-handle {
  width: 10px; height: 10px;
  background: #4facfe; border: 2px solid white;
}

.node-handle:hover { transform: scale(1.2); }
</style>
