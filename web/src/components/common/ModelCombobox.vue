<template>
  <div class="model-combobox" ref="comboboxRef">
    <div class="model-input-wrapper" :class="{ focused: isFocused }">
      <input
        ref="inputRef"
        v-model="inputValue"
        type="text"
        :placeholder="placeholder"
        class="model-input"
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
    <p v-if="showHint" class="model-hint">
      <InfoIcon :size="10" />
      支持输入自定义模型名称，已加载 {{ totalModels }} 个模型
    </p>

    <!-- 下拉弹框 - 使用 Teleport 渲染到 body -->
    <Teleport to="body">
      <transition name="dropdown">
        <div v-if="isOpen" class="model-dropdown" :style="dropdownStyle">
          <div class="dropdown-header" v-if="showHeader">
            <span class="header-text">选择模型</span>
            <span class="header-hint">支持搜索</span>
          </div>

          <div class="dropdown-content" ref="contentRef">
            <!-- 当前厂商分组 -->
            <div v-if="currentVendorGroup" class="model-group">
              <div class="model-group-label">{{ currentVendorGroup.label }}</div>
              <div
                v-for="model in currentVendorGroup.models"
                :key="model"
                class="model-item"
                :class="{ 
                  active: model === inputValue,
                  highlighted: model === highlightedIndex 
                }"
                @click="selectModel(model)"
                @mouseenter="highlightedIndex = model"
              >
                <span class="model-name">{{ model }}</span>
                <span v-if="isDefaultModel(model)" class="default-tag">默认</span>
              </div>
            </div>

            <!-- 其他热门模型分组 -->
            <div v-if="otherModels.length > 0" class="model-group">
              <div class="model-group-label">其他热门模型</div>
              <div
                v-for="model in filteredOtherModels"
                :key="model"
                class="model-item"
                :class="{ 
                  active: model === inputValue,
                  highlighted: model === highlightedIndex 
                }"
                @click="selectModel(model)"
                @mouseenter="highlightedIndex = model"
              >
                <span class="model-name">{{ model }}</span>
                <span v-if="searchQuery" class="match-hint">
                  匹配 "{{ searchQuery }}"
                </span>
              </div>
              <div v-if="filteredOtherModels.length === 0" class="no-results">
                未找到匹配的模型
              </div>
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
  selectedVendorId: string
  builtinProviders: BuiltinProvider[]
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '输入或选择模型'
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// 状态
const inputValue = ref(props.modelValue)
const isFocused = ref(false)
const isOpen = ref(false)
const searchQuery = ref('')
const highlightedIndex = ref('')
const comboboxRef = ref<HTMLElement>()
const inputRef = ref<HTMLInputElement>()
const contentRef = ref<HTMLElement>()
const dropdownStyle = ref({})

// 计算当前厂商
const currentVendor = computed(() => {
  return props.builtinProviders.find(v => v.id === props.selectedVendorId)
})

// 当前厂商模型分组
const currentVendorGroup = computed(() => {
  if (!currentVendor.value || currentVendor.value.models.length === 0) {
    return null
  }
  return {
    label: `当前厂商（${currentVendor.value.name}）`,
    models: currentVendor.value.models
  }
})

// 其他所有模型（去重）
const otherModels = computed(() => {
  const models = new Set<string>()
  props.builtinProviders.forEach(vendor => {
    if (vendor.id !== props.selectedVendorId) {
      vendor.models.forEach(m => models.add(m))
    }
  })
  return Array.from(models)
})

// 搜索过滤后的其他模型
const filteredOtherModels = computed(() => {
  if (!searchQuery.value) {
    return otherModels.value.slice(0, 20)
  }
  const query = searchQuery.value.toLowerCase()
  return otherModels.value
    .filter(m => m.toLowerCase().includes(query))
    .slice(0, 20)
})

// 总模型数
const totalModels = computed(() => {
  return (currentVendorGroup.value?.models.length || 0) + otherModels.value.length
})

// 是否显示提示
const showHint = computed(() => {
  return props.selectedVendorId && totalModels.value > 0
})

// 是否显示头部
const showHeader = computed(() => {
  return searchQuery.value.length === 0
})

// 默认模型判断
function isDefaultModel(model: string): boolean {
  return model.includes('默认') || model.includes('default') || model.includes('latest')
}

// 聚焦处理
function handleFocus() {
  isFocused.value = true
  isOpen.value = true
  searchQuery.value = ''
  nextTick(() => {
    updateDropdownPosition()
  })
}

// 失焦处理
function handleBlur() {
  setTimeout(() => {
    isFocused.value = false
    isOpen.value = false
  }, 200)
}

// 输入处理
function handleInput(e: Event) {
  const target = e.target as HTMLInputElement
  searchQuery.value = target.value
  isOpen.value = true
  highlightedIndex.value = ''
  emit('update:modelValue', target.value)
  nextTick(() => {
    updateDropdownPosition()
  })
}

// 键盘导航
function handleKeydown(e: KeyboardEvent) {
  if (!isOpen.value) {
    isOpen.value = true
    return
  }

  const allModels = [
    ...(currentVendorGroup.value?.models || []),
    ...filteredOtherModels.value
  ]

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      navigateHighlight(1, allModels)
      break
    case 'ArrowUp':
      e.preventDefault()
      navigateHighlight(-1, allModels)
      break
    case 'Enter':
      e.preventDefault()
      if (highlightedIndex.value) {
        selectModel(highlightedIndex.value)
      }
      break
    case 'Escape':
      isOpen.value = false
      inputRef.value?.blur()
      break
  }
}

// 导航高亮
function navigateHighlight(direction: number, models: string[]) {
  const currentIndex = models.indexOf(highlightedIndex.value)
  let nextIndex = currentIndex + direction
  if (nextIndex < 0) nextIndex = models.length - 1
  if (nextIndex >= models.length) nextIndex = 0
  highlightedIndex.value = models[nextIndex]
  scrollToHighlighted()
}

// 滚动到高亮项
function scrollToHighlighted() {
  if (!contentRef.value || !highlightedIndex.value) return
  const highlightedEl = contentRef.value.querySelector('.model-item.highlighted')
  if (highlightedEl) {
    highlightedEl.scrollIntoView({ block: 'nearest' })
  }
}

// 选择模型
function selectModel(model: string) {
  inputValue.value = model
  emit('update:modelValue', model)
  isOpen.value = false
  inputRef.value?.focus()
}

// 清空值
function clearValue() {
  inputValue.value = ''
  emit('update:modelValue', '')
  inputRef.value?.focus()
}

// 切换下拉
function toggleDropdown() {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    inputRef.value?.focus()
    nextTick(() => {
      updateDropdownPosition()
    })
  }
}

// 更新下拉框位置（智能定位 - fixed 模式）
function updateDropdownPosition() {
  if (!comboboxRef.value || !contentRef.value) return
  const rect = comboboxRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const dropdownHeight = 320 // max-height
  
  // 计算下方和上方可用空间
  const spaceBelow = viewportHeight - rect.bottom
  const spaceAbove = rect.top
  
  // 优先向下，空间不足时向上
  if (spaceBelow < dropdownHeight && spaceAbove > spaceBelow) {
    // 向上展开
    dropdownStyle.value = {
      width: `${rect.width}px`,
      left: `${rect.left}px`,
      top: `${rect.top - 4}px`,
      bottom: 'auto'
    }
  } else {
    // 向下展开（默认）
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
  }
}

// 监听同步
watch(() => props.modelValue, (newVal) => {
  inputValue.value = newVal
})

// 生命周期
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('resize', updateDropdownPosition)
  updateDropdownPosition()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('resize', updateDropdownPosition)
})
</script>

<style scoped>
.model-combobox {
  position: relative;
}

.model-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-app);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.model-input-wrapper.focused {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(124, 106, 247, 0.15);
}

.model-input {
  flex: 1;
  width: 100%;
  padding: 8px 72px 8px 12px;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
}

.model-input::placeholder {
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

/* 下拉弹框 - Teleport 到 body，完全脱离文档流 */
.model-dropdown {
  position: fixed; /* 使用 fixed 定位，相对于视口 */
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(0, 0, 0, 0.2);
  z-index: 9999; /* 最高层级 */
  max-height: 320px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  /* 确保不会被裁剪 */
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

/* 滚动条样式 */
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

/* 模型分组 */
.model-group {
  padding: 8px 0;
}

.model-group-label {
  padding: 6px 12px;
  font-size: 10px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.model-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  transition: background 0.1s;
}

.model-item:hover,
.model-item.highlighted {
  background: var(--bg-overlay);
}

.model-item.active {
  background: rgba(124, 106, 247, 0.15);
}

.model-item.active .model-name {
  color: var(--accent);
  font-weight: 500;
}

.model-name {
  font-size: 12px;
  color: var(--text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.default-tag {
  font-size: 9px;
  padding: 2px 6px;
  background: var(--accent);
  color: white;
  border-radius: 3px;
  margin-left: 8px;
}

.match-hint {
  font-size: 10px;
  color: var(--text-tertiary);
  margin-left: 8px;
}

.no-results {
  padding: 12px;
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
}

/* 提示信息 */
.model-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-tertiary);
  margin: 4px 0 0 0;
}
</style>
