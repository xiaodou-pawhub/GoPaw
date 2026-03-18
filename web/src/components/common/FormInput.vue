<template>
  <v-text-field
    :label="label"
    :model-value="modelValue"
    :type="inputType"
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
    :append-icon="appendIcon"
    :clearable="clearable"
    :counter="counter"
    :maxlength="maxlength"
    @update:model-value="$emit('update:modelValue', $event)"
    @click:prepend="onPrependClick"
    @click:append="onAppendClick"
    @click:clear="$emit('clear')"
  >
    <template v-if="prependInner" #prepend-inner>
      <slot name="prepend-inner" />
    </template>
    <template v-if="appendInner" #append-inner>
      <slot name="append-inner" />
    </template>
  </v-text-field>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue?: string | number
  label?: string
  type?: 'text' | 'password' | 'email' | 'number' | 'tel' | 'url'
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
  appendIcon?: string
  clearable?: boolean
  counter?: boolean | number
  maxlength?: number
  prependInner?: boolean
  appendInner?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  variant: 'outlined',
  density: 'comfortable',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'clear': []
  'prependClick': []
  'appendClick': []
}>()

const inputType = computed(() => props.type)

const errorMessages = computed(() => {
  return props.error ? [props.error] : undefined
})

const onPrependClick = () => {
  emit('prependClick')
}

const onAppendClick = () => {
  emit('appendClick')
}
</script>
