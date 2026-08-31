<script setup lang="ts">
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from '@lucide/vue'

import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationFirst,
  PaginationItem,
  PaginationLast,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useI18n } from '@/shared/i18n'
import { formatInteger } from '@/shared/utils/format'

const props = withDefaults(defineProps<{
  page: number
  pageSize: number
  total: number
  pageSizeOptions?: number[]
}>(), {
  pageSizeOptions: () => [20, 50, 100, 200],
})

const emit = defineEmits<{
  (event: 'update:page', value: number): void
  (event: 'update:pageSize', value: number): void
}>()

const { t } = useI18n()

function updatePageSize(value: unknown) {
  const parsed = Number(value)
  if (props.pageSizeOptions.includes(parsed)) {
    emit('update:pageSize', parsed)
  }
}
</script>

<template>
  <div data-slot="table-pagination-footer" class="table-pagination-footer">
    <div data-slot="table-pagination-summary" class="table-pagination-summary">
      <span>{{ t(`共 ${formatInteger(total)} 条`, `${formatInteger(total)} total`) }}</span>
      <span>{{ t('每页', 'Per page') }}</span>
      <Select :model-value="String(pageSize)" @update:model-value="updatePageSize">
        <SelectTrigger size="sm" class="w-20" :aria-label="t('每页数量', 'Rows per page')">
          <SelectValue />
        </SelectTrigger>
        <SelectContent align="end">
          <SelectGroup>
            <SelectItem
              v-for="size in pageSizeOptions"
              :key="size"
              :value="String(size)"
            >
              {{ size }}
            </SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>

    <Pagination
      :page="page"
      :items-per-page="pageSize"
      :total="total"
      :sibling-count="1"
      show-edges
      class="mx-0 w-auto"
      @update:page="emit('update:page', $event)"
    >
      <PaginationContent v-slot="{ items }">
        <PaginationFirst size="icon-sm" :aria-label="t('第一页', 'First page')">
          <ChevronsLeft />
        </PaginationFirst>
        <PaginationPrevious size="icon-sm" :aria-label="t('上一页', 'Previous page')">
          <ChevronLeft />
        </PaginationPrevious>
        <template v-for="(item, index) in items" :key="index">
          <PaginationItem
            v-if="item.type === 'page'"
            :value="item.value"
            :is-active="item.value === page"
            size="icon-sm"
          >
            {{ item.value }}
          </PaginationItem>
          <PaginationEllipsis v-else :index="index" />
        </template>
        <PaginationNext size="icon-sm" :aria-label="t('下一页', 'Next page')">
          <ChevronRight />
        </PaginationNext>
        <PaginationLast size="icon-sm" :aria-label="t('最后一页', 'Last page')">
          <ChevronsRight />
        </PaginationLast>
      </PaginationContent>
    </Pagination>
  </div>
</template>

<style scoped>
.table-pagination-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-top: 0;
  border-radius: 0 0 var(--radius) var(--radius);
  background: var(--card);
}

.table-pagination-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted-foreground);
  font-size: 12px;
  white-space: nowrap;
}

@media (max-width: 720px) {
  .table-pagination-footer {
    justify-content: flex-start;
    overflow-x: auto;
  }
}
</style>
