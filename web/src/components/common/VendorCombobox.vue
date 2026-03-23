<template>
  <div class="vendor-combobox" ref="comboboxRef">
    <div class="vendor-input-wrapper" :class="{ focused: isFocused }">
      <input
        ref="inputRef"
        v-model="inputValue"
        type="text"
        :placeholder="placeholder"
        class="vendor-input"
        @focus="handleFocus"
        @blur="handleBlur"
        @input="handleInput"
        @keydown="handleKeydown"
      />
      <button
        v-if="inputValue"
        type="button"
        class="btn-clear"
        @click="clearValue"
        tabindex="-1"
      >
        <XIcon :size="12" />
      </button>
      <button
        type="button"
        class="btn-toggle"
        @click="toggleDropdown"
        tabindex="-1"
      >
        <ChevronDownIcon :size="14" :class="{ rotated: isOpen }" />
      </button>
    </div>

    <!-- 提示信息 -->
    <p v-if="showHint" class="vendor-hint">
      <InfoIcon :size="10" />
      选择厂商可快速填充配置
    </p>

    <!-- 下拉弹框 - 使用 Teleport 渲染到 body -->
    <Teleport to="body">
      <transition name="dropdown">
        <div v-if="isOpen" class="vendor-dropdown" :style="dropdownStyle">
          <div class="dropdown-header">
            <span class="header-text">选择厂商</span>
            <span class="header-hint">{{ filteredVendors.length }} 个厂商</span>
          </div>

          <div class="dropdown-content" ref="contentRef">
            <div
              v-for="vendor in filteredVendors"
              :key="vendor.id"
              class="vendor-item"
              :class="{ 
                active: vendor.id === selectedVendorId,
                highlighted: vendor.id === highlightedId 
              }"
              @click="selectVendor(vendor)"
              @mouseenter="highlightedId = vendor.id"
            >
              <div class="vendor-info">
                <span class="vendor-name">{{ vendor.name }}</span>
                <span class="vendor-models">{{ vendor.models.length }} 个模型</span>
              </div>
            </div>
            <div v-if="filteredVendors.length === 0" class="no-results">
              未找到匹配的厂商
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { XIcon, ChevronDownIcon, InfoIcon } from 'lucide-vue-next'
import type { BuiltinProvider } from '@/types'

interface Props {
  modelValue: string
  builtinProviders: BuiltinProvider[]
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '选择厂商以快速填充'
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'change': [vendorId: string]
}>()

// 状态
const inputValue = ref('')
const selectedVendorId = ref(props.modelValue)
const isFocused = ref(false)
const isOpen = ref(false)
const searchQuery = ref('')
const highlightedId = ref('')
const comboboxRef = ref<HTMLElement>()
const inputRef = ref<HTMLInputElement>()
const contentRef = ref<HTMLElement>()
const dropdownStyle = ref({})

// 计算属性
const filteredVendors = computed(() => {
  if (!searchQuery.value) {
    return props.builtinProviders
  }
  const query = searchQuery.value.toLowerCase()
  return props.builtinProviders.filter(v => 
    v.name.toLowerCase().includes(query) ||
    v.id.toLowerCase().includes(query)
  )
})

const showHint = computed(() => {
  return !selectedVendorId.value && props.builtinProviders.length > 0
})

// 方法
function handleFocus() {
  isFocused.value = true
  isOpen.value = true
  searchQuery.value = inputValue.value
  nextTick(() => {
    updateDropdownPosition()
  })
}

function handleBlur() {
  setTimeout(() => {
    isFocused.value = false
    isOpen.value = false
    // 恢复显示选中的厂商名称
    if (selectedVendorId.value) {
      const vendor = props.builtinProviders.find(v => v.id === selectedVendorId.value)
      inputValue.value = vendor ? vendor.name : ''
    } else {
      inputValue.value = ''
    }
  }, 200)
}

function handleInput(e: Event) {
  const target = e.target as HTMLInputElement
  searchQuery.value = target.value
  inputValue.value = target.value
  isOpen.value = true
  highlightedId.value = ''
  nextTick(() => {
    updateDropdownPosition()
  })
}

function handleKeydown(e: KeyboardEvent) {
  if (!isOpen.value) {
    isOpen.value = true
    return
  }

  const vendors = filteredVendors.value
  const currentIndex = vendors.findIndex(v => v.id === highlightedId.value)

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      const nextIndex = currentIndex < vendors.length - 1 ? currentIndex + 1 : 0
      highlightedId.value = vendors[nextIndex]?.id || ''
      scrollToHighlighted()
      break
    case 'ArrowUp':
      e.preventDefault()
      const prevIndex = currentIndex > 0 ? currentIndex - 1 : vendors.length - 1
      highlightedId.value = vendors[prevIndex]?.id || ''
      scrollToHighlighted()
      break
    case 'Enter':
      e.preventDefault()
      if (highlightedId.value) {
        const vendor = vendors.find(v => v.id === highlightedId.value)
        if (vendor) selectVendor(vendor)
      }
      break
    case 'Escape':
      isOpen.value = false
      inputRef.value?.blur()
      break
  }
}

function scrollToHighlighted() {
  if (!contentRef.value || !highlightedId.value) return
  const highlightedEl = contentRef.value.querySelector('.vendor-item.highlighted')
  if (highlightedEl) {
    highlightedEl.scrollIntoView({ block: 'nearest' })
  }
}

function selectVendor(vendor: BuiltinProvider) {
  selectedVendorId.value = vendor.id
  inputValue.value = vendor.name
  emit('update:modelValue', vendor.id)
  emit('change', vendor.id)
  isOpen.value = false
  inputRef.value?.focus()
}

function clearValue() {
  selectedVendorId.value = ''
  inputValue.value = ''
  emit('update:modelValue', '')
  emit('change', '')
  inputRef.value?.focus()
}

function toggleDropdown() {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    inputRef.value?.focus()
    nextTick(() => {
      updateDropdownPosition()
    })
  }
}

function updateDropdownPosition() {
  if (!comboboxRef.value || !contentRef.value) return
  const rect = comboboxRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const dropdownHeight = 320
  
  const spaceBelow = viewportHeight - rect.bottom
  const spaceAbove = rect.top
  
  if (spaceBelow < dropdownHeight && spaceAbove > spaceBelow) {
    dropdownStyle.value = {
      width: `${rect.width}px`,
      left: `${rect.left}px`,
      top: `${rect.top - 4}px`,
      bottom: 'auto'
    }
  } else {
    dropdownStyle.value = {
      width: `${rect.width}px`,
      left: `${rect.left}px`,
      top: `${rect.bottom + 4}px`,
      bottom: 'auto'
    }
  }
}

function handleClickOutside(e: MouseEvent) {
  if (comboboxRef.value && !comboboxRef.value.contains(e.target as Node)) {
    isOpen.value = false
  }
}

// 监听
watch(() => props.modelValue, (newVal) => {
  selectedVendorId.value = newVal
  if (newVal) {
    const vendor = props.builtinProviders.find(v => v.id === newVal)
    inputValue.value = vendor ? vendor.name : ''
  } else {
    inputValue.value = ''
  }
})

// 生命周期
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('resize', updateDropdownPosition)
  // 初始化显示
  if (props.modelValue) {
    const vendor = props.builtinProviders.find(v => v.id === props.modelValue)
    inputValue.value = vendor ? vendor.name : ''
  }
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('resize', updateDropdownPosition)
})
</script>

<style scoped>
.vendor-combobox {
  position: relative;
}

.vendor-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-app);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.vendor-input-wrapper.focused {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(124, 106, 247, 0.15);
}

.vendor-input {
  flex: 1;
  width: 100%;
  padding: 8px 72px 8px 12px;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
}

.vendor-input::placeholder {
  color: var(--text-tertiary);
}

.btn-clear,
.btn-toggle {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.15s;
}

.btn-clear {
  right: 36px;
}

.btn-clear:hover {
  color: var(--text-primary);
  background: var(--bg-overlay);
}

.btn-toggle:hover {
  color: var(--text-primary);
  background: var(--bg-overlay);
}

.btn-toggle .rotated {
  transform: rotate(180deg);
}

/* 提示信息 */
.vendor-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-tertiary);
  margin: 4px 0 0 0;
}

/* 下拉弹框 - Teleport 到 body，完全脱离文档流 */
.vendor-dropdown {
  position: fixed;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(0, 0, 0, 0.2);
  z-index: 9999;
  max-height: 320px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  clip: auto;
  width: auto;
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.dropdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-panel);
}

.header-text {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
}

.header-hint {
  font-size: 10px;
  color: var(--text-tertiary);
}

.dropdown-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.dropdown-content::-webkit-scrollbar {
  width: 6px;
}

.dropdown-content::-webkit-scrollbar-track {
  background: transparent;
}

.dropdown-content::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 3px;
}

.dropdown-content::-webkit-scrollbar-thumb:hover {
  background: var(--text-tertiary);
}

.vendor-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.1s;
}

.vendor-item:hover,
.vendor-item.highlighted {
  background: var(--bg-overlay);
}

.vendor-item.active {
  background: rgba(124, 106, 247, 0.15);
}

.vendor-item.active .vendor-name {
  color: var(--accent);
  font-weight: 500;
}

.vendor-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.vendor-name {
  font-size: 12px;
  color: var(--text-primary);
}

.vendor-models {
  font-size: 10px;
  color: var(--text-tertiary);
}

.no-results {
  padding: 12px;
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>