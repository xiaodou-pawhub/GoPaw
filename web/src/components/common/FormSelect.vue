<template>
  <v-select
    :label="label"
    :model-value="modelValue"
    :items="items"
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
    :prepend-icon="prependIcon"
    :clearable="clearable"
    :multiple="multiple"
    :chips="chips"
    :item-title="itemTitle"
    :item-value="itemValue"
    @update:model-value="$emit('update:modelValue', $event)"
    @click:clear="$emit('clear')"
  >
    <template v-if="prependInner" #prepend-inner>
      <slot name="prepend-inner" />
    </template>
    <template v-if="appendInner" #append-inner>
      <slot name="append-inner" />
    </template>
  </v-select>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface SelectItem {
  title: string
  value: string | number
  disabled?: boolean
}

interface Props {
  modelValue?: string | number | Array<string | number>
  label?: string
  items?: SelectItem[] | string[]
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
  prependIcon?: string
  clearable?: boolean
  multiple?: boolean
  chips?: boolean
  itemTitle?: string
  itemValue?: string
  prependInner?: boolean
  appendInner?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'outlined',
  density: 'comfortable',
  itemTitle: 'title',
  itemValue: 'value',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number | Array<string | number>]
  'clear': []
}>()

const errorMessages = computed(() => {
  return props.error ? [props.error] : undefined
})
</script>
