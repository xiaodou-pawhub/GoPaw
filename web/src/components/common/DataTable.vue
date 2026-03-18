<template>
  <v-data-table-server
    :headers="headers"
    :items="items"
    :items-length="totalItems"
    :loading="loading"
    :page="page"
    :items-per-page="itemsPerPage"
    :sort-by="sortBy"
    :server-items-length="totalItems"
    :density="density"
    :fixed-header="fixedHeader"
    :height="height"
    :hover="hover"
    :show-select="showSelect"
    :v-model="selectedItems"
    class="elevation-1"
    @update:options="loadItems"
    @update:model-value="$emit('update:selectedItems', $event)"
  >
    <!-- Custom slot for item actions -->
    <template v-if="$slots.actions" #item.actions="{ item }">
      <slot name="actions" :item="item" />
    </template>

    <!-- Custom slot for status -->
    <template v-if="$slots.status" #item.status="{ item }">
      <slot name="status" :item="item" />
    </template>

    <!-- Custom slot for any column -->
    <template v-for="slotName in dynamicSlots" :key="slotName" #[slotName]="slotData">
      <slot :name="slotName" v-bind="slotData" />
    </template>

    <!-- Loading slot -->
    <template v-if="$slots.loading" #loading>
      <slot name="loading" />
    </template>

    <!-- No data slot -->
    <template v-if="$slots['no-data']" #no-data>
      <slot name="no-data" />
    </template>
  </v-data-table-server>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Header {
  title: string
  key: string
  align?: 'start' | 'center' | 'end'
  filterable?: boolean
  groupable?: boolean
  sortable?: boolean
  width?: string | number
  fixed?: boolean
  cellClass?: string | string[]
  headerClass?: string | string[]
}

interface Props {
  headers: Header[]
  items: any[]
  totalItems: number
  loading?: boolean
  page?: number
  itemsPerPage?: number
  sortBy?: Array<{ key: string; order?: 'asc' | 'desc' }>
  density?: 'default' | 'comfortable' | 'compact'
  fixedHeader?: boolean
  height?: string | number
  hover?: boolean
  showSelect?: boolean
  selectedItems?: any[]
  disableSort?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  page: 1,
  itemsPerPage: 20,
  sortBy: () => [],
  density: 'comfortable',
  hover: true,
  showSelect: false,
  selectedItems: () => [],
  disableSort: false,
})

const emit = defineEmits<{
  'update:options': [options: {
    page: number
    itemsPerPage: number
    sortBy: Array<{ key: string; order?: 'asc' | 'desc' }>
  }]
  'update:selectedItems': [items: any[]]
  'load': [options: {
    page: number
    itemsPerPage: number
    sortBy: Array<{ key: string; order?: 'asc' | 'desc' }>
  }]
}>()

const loadItems = (options: {
  page: number
  itemsPerPage: number
  sortBy: Array<{ key: string; order?: 'asc' | 'desc' }>
}) => {
  emit('update:options', options)
  emit('load', options)
}

// Dynamic slots detection
const dynamicSlots = ref<string[]>([])

// Watch for slot changes
watch(() => props.items, () => {
  // Can be extended for dynamic slot detection
}, { immediate: true })
</script>

<style scoped>
.elevation-1 {
  border-radius: 8px;
  overflow: hidden;
}
</style>
