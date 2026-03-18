<template>
  <v-textarea
    :label="label"
    :model-value="modelValue"
    :placeholder="placeholder"
    :hint="hint"
    :persistent-hint="persistentHint"
    :error="!!error"
    :error-messages="errorMessages"
    :disabled="disabled"
    :readonly="readonly"
    :required="required"
    :rules="rules"
    :variant="variant"
    :density="density"
    :clearable="clearable"
    :counter="counter"
    :maxlength="maxlength"
    :rows="rows"
    :auto-grow="autoGrow"
    :no-resize="noResize"
    @update:model-value="$emit('update:modelValue', $event)"
    @click:clear="$emit('clear')"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue?: string
  label?: string
  placeholder?: string
  hint?: string
  persistentHint?: boolean
  error?: string
  disabled?: boolean
  readonly?: boolean
  required?: boolean
  rules?: Array<(v: string) => boolean | string>
  variant?: 'outlined' | 'underlined' | 'filled' | 'plain' | 'solo'
  density?: 'default' | 'comfortable' | 'compact'
  clearable?: boolean
  counter?: boolean | number
  maxlength?: number
  rows?: number
  autoGrow?: boolean
  noResize?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'outlined',
  density: 'comfortable',
  rows: 3,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'clear': []
}>()

const errorMessages = computed(() => {
  return props.error ? [props.error] : undefined
})
</script>
