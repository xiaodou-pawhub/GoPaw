<template>
  <v-card class="card-list" :loading="loading" :height="height">
    <v-card-title class="d-flex align-center">
      <span class="text-h5">{{ title }}</span>
      <v-spacer />
      <!-- Search -->
      <v-text-field
        v-if="searchable"
        v-model="searchQuery"
        density="compact"
        label="搜索..."
        prepend-inner-icon="mdi-magnify"
        single-line
        hide-details
        style="max-width: 300px"
        @update:model-value="handleSearch"
      />
    </v-card-title>

    <v-card-text>
      <!-- Filters -->
      <div v-if="$slots.filters" class="filters mb-4">
        <slot name="filters" />
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="loading-state text-center py-8">
        <v-progress-circular indeterminate color="primary" />
        <p class="mt-4 text-medium-emphasis">加载中...</p>
      </div>

      <!-- Empty state -->
      <div v-else-if="items.length === 0" class="empty-state text-center py-8">
        <v-icon size="64" color="disabled" class="mb-4">
          {{ emptyIcon }}
        </v-icon>
        <p class="text-h6 text-medium-emphasis">{{ emptyText }}</p>
        <p v-if="emptySubtext" class="text-caption text-disabled mt-2">
          {{ emptySubtext }}
        </p>
        <slot name="empty-action" />
      </div>

      <!-- Grid layout -->
      <div v-else :class="`grid-container grid-${columns}`">
        <div v-for="item in items" :key="itemKey ? item[itemKey] : item.id" class="grid-item">
          <slot name="item" :item="item" />
        </div>
      </div>
    </v-card-text>

    <!-- Pagination -->
    <v-card-actions v-if="totalItems > itemsPerPage">
      <v-spacer />
      <v-pagination
        v-model="currentPage"
        :length="totalPages"
        :total-visible="visiblePages"
        @update:model-value="handlePageChange"
      />
    </v-card-actions>
  </v-card>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Props {
  title?: string
  items: any[]
  totalItems: number
  loading?: boolean
  page?: number
  itemsPerPage?: number
  columns?: 1 | 2 | 3 | 4 | 5 | 6
  itemKey?: string
  searchable?: boolean
  emptyIcon?: string
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
  emptyIcon: 'mdi-inbox',
  emptyText: '暂无数据',
})

const emit = defineEmits<{
  'update:page': [page: number]
  'search': [query: string]
  'load': [options: { page: number; itemsPerPage: number; search?: string }]
}>()

const currentPage = ref(props.page)
const searchQuery = ref('')

const totalPages = computed(() => {
  return Math.ceil(props.totalItems / props.itemsPerPage)
})

const visiblePages = computed(() => {
  if (totalPages.value <= 7) {
    return totalPages.value
  }
  return 7
})

const handlePageChange = (page: number) => {
  emit('update:page', page)
  emit('load', {
    page,
    itemsPerPage: props.itemsPerPage,
    search: searchQuery.value || undefined,
  })
}

const handleSearch = (query: string) => {
  emit('search', query)
  emit('load', {
    page: 1,
    itemsPerPage: props.itemsPerPage,
    search: query || undefined,
  })
}

// Watch for external page changes
watch(() => props.page, (newPage) => {
  if (newPage !== currentPage.value) {
    currentPage.value = newPage
  }
})

defineExpose({
  refresh: () => {
    emit('load', {
      page: currentPage.value,
      itemsPerPage: props.itemsPerPage,
      search: searchQuery.value || undefined,
    })
  },
})
</script>

<style scoped>
.grid-container {
  display: grid;
  gap: 16px;
}

.grid-1 {
  grid-template-columns: repeat(1, 1fr);
}

.grid-2 {
  grid-template-columns: repeat(2, 1fr);
}

.grid-3 {
  grid-template-columns: repeat(3, 1fr);
}

.grid-4 {
  grid-template-columns: repeat(4, 1fr);
}

.grid-5 {
  grid-template-columns: repeat(5, 1fr);
}

.grid-6 {
  grid-template-columns: repeat(6, 1fr);
}

@media (max-width: 960px) {
  .grid-3,
  .grid-4,
  .grid-5,
  .grid-6 {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .grid-2,
  .grid-3,
  .grid-4,
  .grid-5,
  .grid-6 {
    grid-template-columns: repeat(1, 1fr);
  }
}

.loading-state,
.empty-state {
  color: rgba(0, 0, 0, 0.6);
}
</style>
