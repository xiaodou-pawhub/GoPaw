<template>
  <div class="card-list" :style="height ? { height: typeof height === 'number' ? height + 'px' : height } : {}">
    <div class="card-list-header">
      <span class="card-list-title">{{ title }}</span>
      <input
        v-if="searchable"
        v-model="searchQuery"
        type="text"
        placeholder="搜索..."
        class="search-input"
        @input="handleSearch(searchQuery)"
      />
    </div>

    <div v-if="$slots.filters" class="filters">
      <slot name="filters" />
    </div>

    <div v-if="loading" class="state-center">
      <div class="spinner" />
      <p class="state-text">加载中...</p>
    </div>

    <div v-else-if="items.length === 0" class="state-center">
      <p class="state-text">{{ emptyText }}</p>
      <p v-if="emptySubtext" class="state-subtext">{{ emptySubtext }}</p>
      <slot name="empty-action" />
    </div>

    <div v-else :class="`grid-container grid-${columns}`">
      <div v-for="item in items" :key="itemKey ? (item as Record<string, any>)[itemKey] : (item as Record<string, any>).id" class="grid-item">
        <slot name="item" :item="item" />
      </div>
    </div>

    <div v-if="totalItems > itemsPerPage" class="pagination">
      <button class="page-btn" :disabled="currentPage <= 1" @click="handlePageChange(currentPage - 1)">
        上一页
      </button>
      <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
      <button class="page-btn" :disabled="currentPage >= totalPages" @click="handlePageChange(currentPage + 1)">
        下一页
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Props {
  title?: string
  items: unknown[]
  totalItems: number
  loading?: boolean
  page?: number
  itemsPerPage?: number
  columns?: 1 | 2 | 3 | 4 | 5 | 6
  itemKey?: string
  searchable?: boolean
  emptyText?: string
  emptySubtext?: string
  height?: string | number
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  page: 1,
  itemsPerPage: 20,
  columns: 3,
  itemKey: 'id',
  searchable: true,
  emptyText: '暂无数据',
})

const emit = defineEmits<{
  'update:page': [page: number]
  'search': [query: string]
  'load': [options: { page: number; itemsPerPage: number; search?: string }]
}>()

const currentPage = ref(props.page)
const searchQuery = ref('')

const totalPages = computed(() => Math.ceil(props.totalItems / props.itemsPerPage))

const handlePageChange = (page: number) => {
  currentPage.value = page
  emit('update:page', page)
  emit('load', { page, itemsPerPage: props.itemsPerPage, search: searchQuery.value || undefined })
}

const handleSearch = (query: string) => {
  emit('search', query)
  emit('load', { page: 1, itemsPerPage: props.itemsPerPage, search: query || undefined })
}

watch(() => props.page, (newPage) => {
  if (newPage !== currentPage.value) currentPage.value = newPage
})

defineExpose({
  refresh: () => {
    emit('load', { page: currentPage.value, itemsPerPage: props.itemsPerPage, search: searchQuery.value || undefined })
  },
})
</script>

<style scoped>
.card-list {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.card-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-overlay);
}

.card-list-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.search-input {
  padding: 6px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  width: 220px;
}

.filters {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.state-center {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  gap: 12px;
}

@keyframes spin { to { transform: rotate(360deg); } }

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.state-text {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.state-subtext {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 0;
}

.grid-container {
  flex: 1;
  display: grid;
  gap: 16px;
  padding: 16px;
  overflow-y: auto;
}

.grid-1 { grid-template-columns: repeat(1, 1fr); }
.grid-2 { grid-template-columns: repeat(2, 1fr); }
.grid-3 { grid-template-columns: repeat(3, 1fr); }
.grid-4 { grid-template-columns: repeat(4, 1fr); }
.grid-5 { grid-template-columns: repeat(5, 1fr); }
.grid-6 { grid-template-columns: repeat(6, 1fr); }

@media (max-width: 960px) {
  .grid-3, .grid-4, .grid-5, .grid-6 { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 600px) {
  .grid-2, .grid-3, .grid-4, .grid-5, .grid-6 { grid-template-columns: repeat(1, 1fr); }
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
}

.page-btn {
  padding: 6px 14px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: background 0.1s;
}

.page-btn:hover:not(:disabled) { background: var(--bg-overlay); }
.page-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.page-info {
  font-size: 13px;
  color: var(--text-secondary);
}
</style>
