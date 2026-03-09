<template>
  <div class="empty-state" :class="{ 'empty-centered': centered }">
    <div class="empty-icon">
      <slot name="icon">
        <component :is="icon" :size="iconSize" />
      </slot>
    </div>
    
    <h3 v-if="title" class="empty-title">{{ title }}</h3>
    <p v-if="description" class="empty-description">{{ description }}</p>
    
    <div v-if="$slots.default" class="empty-actions">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

interface Props {
  icon?: Component
  title?: string
  description?: string
  iconSize?: number
  centered?: boolean
}

withDefaults(defineProps<Props>(), {
  icon: undefined,
  title: '',
  description: '',
  iconSize: 48,
  centered: false
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
}

.empty-centered {
  min-height: 400px;
}

.empty-icon {
  margin-bottom: 24px;
  color: var(--text-tertiary);
  opacity: 0.5;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.empty-description {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 24px 0;
  line-height: 1.6;
  max-width: 400px;
}

.empty-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}
</style>
