<template>
  <div class="combobox" ref="comboboxRef">
    <div class="combobox-input-wrapper" :class="{ focused: isFocused, disabled }">
      <button
        type="button"
        class="combobox-trigger"
        :disabled="disabled"
        @click="toggleDropdown"
      >
        <span class="combobox-value" :class="{ placeholder: !displayValue }">
          {{ displayValue || placeholder }}
        </span>
        <ChevronDownIcon :size="14" class="chevron" :class="{ rotated: isOpen }" />
      </button>
    </div>

    <!-- 下拉弹框 -->
    <Teleport to="body">
      <transition name="dropdown">
        <div v-if="isOpen" class="combobox-dropdown" :style="dropdownStyle">
          <!-- 搜索框 -->
          <div v-if="searchable" class="dropdown-search">
            <SearchIcon :size="14" />
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              class="search-input"
              placeholder="搜索..."
              @keydown="handleKeydown"
            />
          </div>

          <!-- 选项列表 -->
          <div class="dropdown-content" ref="contentRef">
            <div
              v-for="option in filteredOptions"
              :key="getOptionValue(option)"
              class="combobox-option"
              :class="{
                active: getOptionValue(option) === modelValue,
                highlighted: getOptionValue(option) === highlightedValue
              }"
              @click="selectOption(option)"
              @mouseenter="highlightedValue = getOptionValue(option)"
            >
              <span class="option-label">{{ getOptionLabel(option) }}</span>
              <CheckIcon v-if="getOptionValue(option) === modelValue" :size="14" class="check-icon" />
            </div>
            <div v-if="filteredOptions.length === 0" class="no-results">
              {{ emptyText || '无匹配选项' }}
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ChevronDownIcon, SearchIcon, CheckIcon } from 'lucide-vue-next'

interface Option {
  value: string | number
  label: string
  [key: string]: any
}

const props = withDefaults(defineProps<{
  modelValue: string | number | null
  options: (Option | string)[]
  placeholder?: string
  disabled?: boolean
  searchable?: boolean
  emptyText?: string
  valueKey?: string
  labelKey?: string
}>(), {
  placeholder: '请选择...',
  disabled: false,
  searchable: true,
  emptyText: '无匹配选项',
  valueKey: 'value',
  labelKey: 'label'
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number | null]
  'change': [value: string | number | null, option: Option | null]
}>()

// 状态
const isOpen = ref(false)
const isFocused = ref(false)
const searchQuery = ref('')
const highlightedValue = ref<string | number | null>(null)
const dropdownStyle = ref<Record<string, string>>({})
const comboboxRef = ref<HTMLElement>()
const contentRef = ref<HTMLElement>()
const searchInputRef = ref<HTMLInputElement>()

// 标准化选项
const normalizedOptions = computed<Option[]>(() => {
  return props.options.map(opt => {
    if (typeof opt === 'string') {
      return { value: opt, label: opt }
    }
    return {
      value: opt[props.valueKey as keyof typeof opt] as string | number,
      label: String(opt[props.labelKey as keyof typeof opt] || opt[props.valueKey as keyof typeof opt] || '')
    }
  })
})

// 过滤选项
const filteredOptions = computed(() => {
  if (!searchQuery.value) return normalizedOptions.value
  const query = searchQuery.value.toLowerCase()
  return normalizedOptions.value.filter(opt =>
    opt.label.toLowerCase().includes(query)
  )
})

// 显示值
const displayValue = computed(() => {
  const opt = normalizedOptions.value.find(o => o.value === props.modelValue)
  return opt?.label || ''
})

// 获取选项值
function getOptionValue(option: Option): string | number {
  return option.value
}

// 获取选项标签
function getOptionLabel(option: Option): string {
  return option.label
}

// 切换下拉
function toggleDropdown() {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    isFocused.value = true
    searchQuery.value = ''
    highlightedValue.value = props.modelValue
    nextTick(() => {
      updateDropdownPosition()
      if (props.searchable) {
        searchInputRef.value?.focus()
      }
    })
  }
}

// 选择选项
function selectOption(option: Option) {
  emit('update:modelValue', option.value)
  emit('change', option.value, option)
  isOpen.value = false
  isFocused.value = false
}

// 键盘导航
function handleKeydown(e: KeyboardEvent) {
  const options = filteredOptions.value
  const currentIndex = options.findIndex(o => o.value === highlightedValue.value)

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      if (currentIndex < options.length - 1) {
        highlightedValue.value = options[currentIndex + 1].value
      } else if (currentIndex === -1 && options.length > 0) {
        highlightedValue.value = options[0].value
      }
      scrollToHighlighted()
      break
    case 'ArrowUp':
      e.preventDefault()
      if (currentIndex > 0) {
        highlightedValue.value = options[currentIndex - 1].value
      }
      scrollToHighlighted()
      break
    case 'Enter':
      e.preventDefault()
      if (highlightedValue.value !== null) {
        const opt = options.find(o => o.value === highlightedValue.value)
        if (opt) selectOption(opt)
      }
      break
    case 'Escape':
      isOpen.value = false
      isFocused.value = false
      break
  }
}

// 滚动到高亮项
function scrollToHighlighted() {
  if (!contentRef.value || highlightedValue.value === null) return
  const highlightedEl = contentRef.value.querySelector('.combobox-option.highlighted')
  if (highlightedEl) {
    highlightedEl.scrollIntoView({ block: 'nearest' })
  }
}

// 更新下拉框位置
function updateDropdownPosition() {
  if (!comboboxRef.value) return
  const rect = comboboxRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const dropdownHeight = 280

  const spaceBelow = viewportHeight - rect.bottom
  const spaceAbove = rect.top

  if (spaceBelow < dropdownHeight && spaceAbove > spaceBelow) {
    dropdownStyle.value = {
      width: `${rect.width}px`,
      left: `${rect.left}px`,
      top: 'auto',
      bottom: `${viewportHeight - rect.top + 4}px`
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

// 点击外部关闭
function handleClickOutside(e: MouseEvent) {
  if (comboboxRef.value && !comboboxRef.value.contains(e.target as Node)) {
    isOpen.value = false
    isFocused.value = false
  }
}

// 生命周期
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('resize', updateDropdownPosition)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('resize', updateDropdownPosition)
})

// 监听打开状态更新位置
watch(isOpen, (open) => {
  if (open) {
    nextTick(updateDropdownPosition)
  }
})
</script>

<style scoped>
.combobox {
  position: relative;
  width: 100%;
}

.combobox-input-wrapper {
  position: relative;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-app);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.combobox-input-wrapper.focused {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(124, 106, 247, 0.15);
}

.combobox-input-wrapper.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.combobox-trigger {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background: transparent;
  border: none;
  cursor: pointer;
  text-align: left;
}

.combobox-trigger:disabled {
  cursor: not-allowed;
}

.combobox-value {
  flex: 1;
  font-size: 12px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.combobox-value.placeholder {
  color: var(--text-tertiary);
}

.chevron {
  color: var(--text-tertiary);
  transition: transform 0.2s;
  flex-shrink: 0;
  margin-left: 8px;
}

.chevron.rotated {
  transform: rotate(180deg);
}

/* 下拉弹框 */
.combobox-dropdown {
  position: fixed;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  z-index: 9999;
  max-height: 280px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.dropdown-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
  color: var(--text-tertiary);
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
}

.search-input::placeholder {
  color: var(--text-tertiary);
}

.dropdown-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.dropdown-content::-webkit-scrollbar {
  width: 6px;
}

.dropdown-content::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 3px;
}

.combobox-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  cursor: pointer;
  transition: background 0.1s;
}

.combobox-option:hover,
.combobox-option.highlighted {
  background: var(--bg-overlay);
}

.combobox-option.active {
  background: rgba(124, 106, 247, 0.15);
}

.combobox-option.active .option-label {
  color: var(--accent);
  font-weight: 500;
}

.option-label {
  font-size: 12px;
  color: var(--text-primary);
}

.check-icon {
  color: var(--accent);
  flex-shrink: 0;
}

.no-results {
  padding: 16px 12px;
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>