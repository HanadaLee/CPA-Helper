<script setup lang="ts">
import { ChevronLeftIcon, ChevronRightIcon, Cpu, KeyRound, RefreshCw, ShieldCheck, TriangleAlertIcon } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { listAvailableModels } from '@/features/models/api/availableModelsApi'
import { useI18n } from '@/shared/i18n'
import type { AvailableModel, AvailableModelPrice, AvailableModelsResponse } from '@/shared/types/api'

type PriceField = keyof Pick<
  AvailableModelPrice,
  | 'input_usd_per_million'
  | 'output_usd_per_million'
  | 'cache_read_usd_per_million'
  | 'cache_creation_usd_per_million'
>
type BillingUnit = 'token' | 'request'

const router = useRouter()
const { currentLanguage, errorText, serverText, t } = useI18n()
const isLoading = ref(false)
const errorMessage = ref<string | null>(null)
const response = ref<AvailableModelsResponse | null>(null)
const page = ref(1)
const pageSize = 20

const modelCount = computed(() => response.value?.models.length ?? 0)
const pagedModels = computed(() => {
  const start = (page.value - 1) * pageSize
  return response.value?.models.slice(start, start + pageSize) ?? []
})
const keySummary = computed(() => {
  if (!response.value) {
    return '-'
  }
  return `${response.value.queryable_api_key_count} / ${response.value.api_key_count}`
})
const queryStatus = computed(() => {
  if (!response.value) {
    return '-'
  }
  if (response.value.errors.length > 0) {
    return t(`部分失败 ${response.value.errors.length}`, `${response.value.errors.length} failed`)
  }
  if (response.value.queryable_api_key_count === 0) {
    return response.value.has_api_keys ? t('不可查询', 'Unavailable') : t('无 Key', 'No keys')
  }
  return response.value.has_api_keys ? t('正常', 'Normal') : t('无 Key', 'No keys')
})

function displayText(value: string | null | undefined): string {
  return value?.trim() || '-'
}

function formatUsdPerMtok(value: number): string {
  return value.toLocaleString(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    maximumFractionDigits: 6,
  })
}

function billingUnitForModel(model: string): BillingUnit {
  return model.trim().toLowerCase().includes('image') ? 'request' : 'token'
}

function modelBillingUnit(row: AvailableModel): BillingUnit {
  if (row.price?.billing_unit === 'request') {
    return 'request'
  }
  if (row.price?.billing_unit === 'token') {
    return 'token'
  }
  return billingUnitForModel(row.price?.model || row.id)
}

function billingLabel(row: AvailableModel): string {
  const unit = modelBillingUnit(row)
  return unit === 'request' ? t('按次', 'Per request') : t('按 Token', 'Per token')
}

function requestPrice(row: AvailableModel): string {
  if (modelBillingUnit(row) !== 'request') {
    return '-'
  }
  if (row.price?.request_usd === null || row.price?.request_usd === undefined) {
    return t('未定价', 'Unpriced')
  }
  return formatUsdPerMtok(row.price.request_usd)
}

function priceValue(row: AvailableModel, field: PriceField): string {
  if (modelBillingUnit(row) === 'request') {
    return '-'
  }
  if (!row.price) {
    return '-'
  }
  return formatUsdPerMtok(row.price[field])
}

function fastMultiplier(row: AvailableModel): string {
  const multiplier = row.price?.fast_multiplier
  if (multiplier === null || multiplier === undefined) {
    return '-'
  }
  return `×${multiplier.toLocaleString(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    maximumFractionDigits: 4,
  })}`
}

function goToApiKeys() {
  void router.push('/account/keys')
}

async function refresh() {
  isLoading.value = true
  errorMessage.value = null
  try {
    response.value = await listAvailableModels()
    page.value = 1
  } catch (error) {
    response.value = null
    errorMessage.value = errorText(error, '加载可用模型失败', 'Failed to load available models')
  } finally {
    isLoading.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <section class="page models-page" :aria-busy="isLoading">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('可用模型', 'Available Models') }}</h1>
      <Button variant="outline" :disabled="isLoading" @click="refresh">
        <Spinner v-if="isLoading" data-icon="inline-start" />
        <RefreshCw v-else data-icon="inline-start" />
        {{ t('刷新', 'Refresh') }}
      </Button>
    </div>

    <section class="panel model-table-panel">
      <div class="panel-inner model-panel">
        <Alert v-if="errorMessage" variant="destructive">
          <AlertTitle>{{ t('加载可用模型失败', 'Failed to load available models') }}</AlertTitle>
          <AlertDescription>{{ errorMessage }}</AlertDescription>
          <AlertAction>
            <Button size="sm" variant="outline" :disabled="isLoading" @click="refresh">
              <Spinner v-if="isLoading" data-icon="inline-start" />
              <RefreshCw v-else data-icon="inline-start" />
              {{ t('重试', 'Retry') }}
            </Button>
          </AlertAction>
        </Alert>

        <template v-else>
          <Alert v-if="response?.errors.length">
            <TriangleAlertIcon />
            <AlertTitle>{{ t('部分 API Key 查询失败', 'Some API key queries failed') }}</AlertTitle>
            <AlertDescription>
              <div class="key-errors">
                <div v-for="error in response.errors" :key="error.api_key_hash">
                  {{
                    t(
                      `${error.description}（${error.api_key_preview}）：${serverText(error.message, '查询失败', 'Query failed')}`,
                      `${error.description} (${error.api_key_preview}): ${serverText(error.message, '查询失败', 'Query failed')}`,
                    )
                  }}
                </div>
              </div>
            </AlertDescription>
          </Alert>

          <div v-if="response" class="metric-grid model-metrics">
            <div class="metric-card">
              <div class="metric-icon" aria-hidden="true">
                <Cpu :size="20" :stroke-width="2.2" />
              </div>
              <div class="metric-label">{{ t('可用模型', 'Available models') }}</div>
              <div class="metric-value">{{ modelCount }}</div>
              <div class="metric-footnote">{{ t('CPA 返回', 'Returned by CPA') }}</div>
            </div>
            <div class="metric-card is-blue">
              <div class="metric-icon" aria-hidden="true">
                <KeyRound :size="20" :stroke-width="2.2" />
              </div>
              <div class="metric-label">{{ t('可查询 Key', 'Queryable keys') }}</div>
              <div class="metric-value">{{ keySummary }}</div>
              <div class="metric-footnote">{{ t('完整密钥', 'Complete keys') }}</div>
            </div>
            <div class="metric-card is-green">
              <div class="metric-icon" aria-hidden="true">
                <ShieldCheck :size="20" :stroke-width="2.2" />
              </div>
              <div class="metric-label">{{ t('查询状态', 'Query status') }}</div>
              <div class="metric-value">{{ queryStatus }}</div>
              <div class="metric-footnote">{{ t('当前账号', 'Current account') }}</div>
            </div>
          </div>

          <div v-if="isLoading && !response" class="loading-state">
            <Spinner class="size-5" />
            <span>{{ t('正在向 CPA 查询模型', 'Querying CPA for models') }}</span>
          </div>

          <Empty v-else-if="response && !response.has_api_keys" class="min-h-[220px]">
            <EmptyHeader>
              <EmptyMedia variant="icon"><KeyRound /></EmptyMedia>
              <EmptyTitle>{{ t('暂无可查询的 API 密钥', 'No queryable API keys') }}</EmptyTitle>
              <EmptyDescription>{{ t('还没有可用于查询模型的 API 密钥', 'No API keys are available for model queries yet') }}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button @click="goToApiKeys">{{ t('去创建 API 密钥', 'Create API key') }}</Button>
            </EmptyContent>
          </Empty>

          <Empty
            v-else-if="response && response.has_api_keys && response.queryable_api_key_count === 0"
            class="min-h-[220px]"
          >
            <EmptyHeader>
              <EmptyMedia variant="icon"><KeyRound /></EmptyMedia>
              <EmptyTitle>{{ t('API 密钥不可查询', 'API keys unavailable') }}</EmptyTitle>
              <EmptyDescription>{{ t('绑定的 API 密钥缺少完整密钥，无法查询模型', 'Bound API keys are missing complete keys and cannot query models') }}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button @click="goToApiKeys">{{ t('去 API 密钥页检查', 'Check API keys') }}</Button>
            </EmptyContent>
          </Empty>

          <Empty v-else-if="response && response.models.length === 0" class="min-h-[220px]">
            <EmptyHeader>
              <EmptyMedia variant="icon"><Cpu /></EmptyMedia>
              <EmptyTitle>{{ t('暂无可用模型', 'No available models') }}</EmptyTitle>
              <EmptyDescription>{{ t('CPA 未返回可用模型', 'CPA returned no available models') }}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button variant="outline" :disabled="isLoading" @click="refresh">
                <Spinner v-if="isLoading" data-icon="inline-start" />
                <RefreshCw v-else data-icon="inline-start" />
                {{ t('重新查询', 'Query again') }}
              </Button>
            </EmptyContent>
          </Empty>

          <div v-else-if="response" class="available-models-table">
            <Table class="min-w-[1240px] table-fixed">
              <TableHeader class="sticky top-0 bg-card">
                <TableRow>
                  <TableHead class="w-[200px]">{{ t('模型 ID', 'Model ID') }}</TableHead>
                  <TableHead class="w-[150px]">{{ t('名称', 'Name') }}</TableHead>
                  <TableHead class="w-[90px]">{{ t('所有者', 'Owner') }}</TableHead>
                  <TableHead class="w-[160px]">{{ t('来源 Key', 'Source Key') }}</TableHead>
                  <TableHead class="w-[84px] text-center">{{ t('计费方式', 'Billing') }}</TableHead>
                  <TableHead class="w-[90px] text-right">{{ t('每次 ($)', 'Per request ($)') }}</TableHead>
                  <TableHead class="w-[100px] text-right">{{ t('输入 $/MTok', 'Input $/MTok') }}</TableHead>
                  <TableHead class="w-[100px] text-right">{{ t('输出 $/MTok', 'Output $/MTok') }}</TableHead>
                  <TableHead class="w-[100px] text-right">{{ t('缓存读 $/MTok', 'Cache read $/MTok') }}</TableHead>
                  <TableHead class="w-[100px] text-right">{{ t('缓存写 $/MTok', 'Cache write $/MTok') }}</TableHead>
                  <TableHead class="w-[80px] text-right">{{ t('FAST 倍率', 'FAST multiplier') }}</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                <template v-if="isLoading && pagedModels.length === 0">
                  <TableRow v-for="rowIndex in 5" :key="`model-skeleton-${rowIndex}`">
                    <TableCell v-for="columnIndex in 11" :key="columnIndex">
                      <Skeleton class="h-4 w-full" />
                    </TableCell>
                  </TableRow>
                </template>

                <TableRow v-for="model in pagedModels" v-else :key="model.id">
                  <TableCell class="font-mono text-xs">
                    <span class="block truncate" :title="model.id">{{ model.id }}</span>
                  </TableCell>
                  <TableCell><span class="block truncate" :title="displayText(model.name)">{{ displayText(model.name) }}</span></TableCell>
                  <TableCell><span class="block truncate" :title="displayText(model.owner)">{{ displayText(model.owner) }}</span></TableCell>
                  <TableCell>
                    <div class="flex flex-wrap gap-1">
                      <Badge v-for="source in model.sources" :key="source.api_key_hash" variant="secondary">
                        {{ source.description }} · {{ source.api_key_preview }}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell class="text-center">
                    <Badge :variant="modelBillingUnit(model) === 'request' ? 'secondary' : 'outline'">
                      {{ billingLabel(model) }}
                    </Badge>
                  </TableCell>
                  <TableCell class="text-right tabular-nums">{{ requestPrice(model) }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ priceValue(model, 'input_usd_per_million') }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ priceValue(model, 'output_usd_per_million') }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ priceValue(model, 'cache_read_usd_per_million') }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ priceValue(model, 'cache_creation_usd_per_million') }}</TableCell>
                  <TableCell class="text-right tabular-nums">{{ fastMultiplier(model) }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>

            <div v-if="isLoading && pagedModels.length > 0" class="table-loading-overlay">
              <Spinner class="size-5" />
            </div>
          </div>

          <Pagination
            v-if="modelCount > pageSize"
            v-model:page="page"
            class="justify-end"
            :items-per-page="pageSize"
            :total="modelCount"
            :sibling-count="1"
            show-edges
          >
            <PaginationContent v-slot="{ items }">
              <PaginationPrevious size="icon" :aria-label="t('上一页', 'Previous page')">
                <ChevronLeftIcon data-icon="inline-start" />
              </PaginationPrevious>
              <template v-for="(item, index) in items" :key="index">
                <PaginationItem
                  v-if="item.type === 'page'"
                  :value="item.value"
                  :is-active="item.value === page"
                >
                  {{ item.value }}
                </PaginationItem>
                <PaginationEllipsis v-else :index="index" />
              </template>
              <PaginationNext size="icon" :aria-label="t('下一页', 'Next page')">
                <ChevronRightIcon data-icon="inline-end" />
              </PaginationNext>
            </PaginationContent>
          </Pagination>
        </template>
      </div>
    </section>
  </section>
</template>

<style scoped>
.model-panel {
  display: grid;
  gap: 14px;
  min-width: 0;
  min-height: 0;
}

.model-metrics {
  grid-template-columns: repeat(3, minmax(128px, 1fr));
}

.models-page,
.model-table-panel,
.available-models-table {
  min-width: 0;
}

.model-table-panel {
  overflow: hidden;
}

.available-models-table {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--card);
  box-shadow: var(--cpa-shadow-card);
}

.available-models-table :deep([data-slot="table-container"]) {
  max-height: max(240px, calc(100dvh - 360px));
  overflow: auto;
}

.table-loading-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--background) 62%, transparent);
}

.key-errors {
  display: grid;
  gap: 4px;
}

.loading-state {
  display: grid;
  min-height: 220px;
  place-items: center;
  color: var(--cpa-text-muted);
}

.loading-state {
  grid-auto-flow: column;
  justify-content: center;
  gap: 8px;
}

@media (min-width: 861px) {
  .models-page {
    grid-template-rows: auto minmax(0, 1fr);
    min-height: 0;
  }
}

@media (max-width: 720px) {
  .model-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .model-metrics .metric-card:last-child {
    grid-column: 1 / -1;
  }

}
</style>
