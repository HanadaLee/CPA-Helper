<script setup lang="ts">
import type { Component } from 'vue'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import { useConfirmDialog } from '@/shared/ui/confirm-dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  CircleAlert,
  CalendarDays,
  Database,
  Layers3,
  ListFilter,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Server,
  Settings2,
  Trash2,
} from '@lucide/vue'

import {
  createModelPrice,
  deleteModelPrice,
  getLiteLLMProxySettings,
  getPricingHolidayCalendar,
  listModelPriceCatalog,
  listModelPrices,
  syncLitellmModelPrices,
  updateLiteLLMProxySettings,
  updateModelPrice,
  updatePricingHolidayCalendar,
} from '@/features/pricing/api/pricingApi'
import type {
  LiteLLMProxySettingsPayload,
  ModelPrice,
  ModelPriceCatalogResponse,
  ModelPricePayload,
} from '@/shared/types/api'
import { formatDateTime, formatInteger } from '@/shared/utils/format'
import { useI18n } from '@/shared/i18n'
import FilterCombobox from '@/shared/ui/FilterCombobox.vue'
import TablePaginationFooter from '@/shared/ui/TablePaginationFooter.vue'

type PriceRowStatus = 'missing' | 'litellm' | 'manual'
type PriceStatusFilter = 'cpa' | 'missing' | 'litellm' | 'manual' | 'library'
type BillingUnit = 'token' | 'request'
type PriceFieldName = keyof Pick<
  ModelPrice,
  | 'input_usd_per_million'
  | 'output_usd_per_million'
  | 'cache_read_usd_per_million'
  | 'cache_creation_usd_per_million'
  | 'long_context_input_usd_per_million'
  | 'long_context_output_usd_per_million'
  | 'long_context_cache_read_usd_per_million'
  | 'long_context_cache_creation_usd_per_million'
  | 'off_peak_input_usd_per_million'
  | 'off_peak_output_usd_per_million'
  | 'off_peak_cache_read_usd_per_million'
  | 'off_peak_cache_creation_usd_per_million'
  | 'long_context_off_peak_input_usd_per_million'
  | 'long_context_off_peak_output_usd_per_million'
  | 'long_context_off_peak_cache_read_usd_per_million'
  | 'long_context_off_peak_cache_creation_usd_per_million'
>

type PriceTier = { key: string; label: string; prefix: '' | 'long_context_' | 'off_peak_' | 'long_context_off_peak_' }

const offPeakRateFields: Array<{ key: PriceFieldName; zh: string; en: string }> = [
  { key: 'off_peak_input_usd_per_million', zh: '空闲输入 ($/MTok)', en: 'Off-peak input ($/MTok)' },
  { key: 'off_peak_output_usd_per_million', zh: '空闲输出 ($/MTok)', en: 'Off-peak output ($/MTok)' },
  { key: 'off_peak_cache_read_usd_per_million', zh: '空闲缓存读 ($/MTok)', en: 'Off-peak cache read ($/MTok)' },
  { key: 'off_peak_cache_creation_usd_per_million', zh: '空闲缓存写 ($/MTok)', en: 'Off-peak cache write ($/MTok)' },
]
const longOffPeakRateFields: Array<{ key: PriceFieldName; zh: string; en: string }> = [
  { key: 'long_context_off_peak_input_usd_per_million', zh: '长上下文空闲输入 ($/MTok)', en: 'Long off-peak input ($/MTok)' },
  { key: 'long_context_off_peak_output_usd_per_million', zh: '长上下文空闲输出 ($/MTok)', en: 'Long off-peak output ($/MTok)' },
  { key: 'long_context_off_peak_cache_read_usd_per_million', zh: '长上下文空闲缓存读 ($/MTok)', en: 'Long off-peak cache read ($/MTok)' },
  { key: 'long_context_off_peak_cache_creation_usd_per_million', zh: '长上下文空闲缓存写 ($/MTok)', en: 'Long off-peak cache write ($/MTok)' },
]

interface PriceDisplayRow {
  key: string
  in_cpa: boolean
  id: string
  name: string
  owner: string | null
  suggested_provider: string
  price: ModelPrice | null
  provider: string
  model: string
  billing_unit: BillingUnit
  status: PriceRowStatus
}

const message = toast
const dialog = useConfirmDialog()
const { errorText, serverText, t } = useI18n()
const isLoading = ref(false)
const isSyncing = ref(false)
const isPriceSaving = ref(false)
const modalOpen = ref(false)
const proxyModalOpen = ref(false)
const isProxyLoading = ref(false)
const isProxySaving = ref(false)
const calendarModalOpen = ref(false)
const calendarYear = ref(Number(new Intl.DateTimeFormat('en-US', { timeZone: 'Asia/Shanghai', year: 'numeric' }).format(new Date())))
const calendarDatesText = ref('')
const calendarSourceURL = ref('')
const calendarConfigured = ref(false)
const isCalendarLoading = ref(false)
const isCalendarSaving = ref(false)
const editingId = ref<number | null>(null)
const prices = ref<ModelPrice[]>([])
const catalog = ref<ModelPriceCatalogResponse | null>(null)
const selectedProvider = ref<string | null>(null)
const selectedStatus = ref<PriceStatusFilter | null>(null)
const searchQuery = ref('')
const page = ref(1)
const pageSize = ref(20)
const form = reactive<ModelPricePayload>({
  provider: '',
  model: '',
  input_usd_per_million: 0,
  output_usd_per_million: 0,
  cache_read_usd_per_million: 0,
  cache_creation_usd_per_million: 0,
  request_usd: null,
  fast_multiplier: 1,
  long_context_enabled: false,
  long_context_threshold_tokens: 0,
  long_context_input_usd_per_million: 0,
  long_context_output_usd_per_million: 0,
  long_context_cache_read_usd_per_million: 0,
  long_context_cache_creation_usd_per_million: 0,
  long_context_fast_unsupported: false,
  off_peak_enabled: false,
  off_peak_input_usd_per_million: 0,
  off_peak_output_usd_per_million: 0,
  off_peak_cache_read_usd_per_million: 0,
  off_peak_cache_creation_usd_per_million: 0,
  off_peak_request_usd: null,
  long_context_off_peak_input_usd_per_million: 0,
  long_context_off_peak_output_usd_per_million: 0,
  long_context_off_peak_cache_read_usd_per_million: 0,
  long_context_off_peak_cache_creation_usd_per_million: 0,
})
const proxyForm = reactive<LiteLLMProxySettingsPayload>({
  enabled: false,
  proxy_url: '',
})

const priceRows = computed<PriceDisplayRow[]>(() => {
  const rows: PriceDisplayRow[] = []
  const catalogPriceIds = new Set<number>()
  for (const model of catalog.value?.models ?? []) {
    if (model.price) {
      catalogPriceIds.add(model.price.id)
    }
    const provider = model.price?.provider || model.suggested_provider || model.owner || providerFromModelId(model.id)
    const billingUnit = billingUnitForPrice(model.price, model.id)
    rows.push({
      key: `catalog:${model.id}`,
      in_cpa: true,
      id: model.id,
      name: model.name || model.id,
      owner: model.owner,
      suggested_provider: model.suggested_provider,
      price: model.price,
      provider,
      model: model.price?.model || model.id,
      billing_unit: billingUnit,
      status: model.price ? priceStatus(model.price, model.id) : 'missing',
    })
  }
  for (const price of prices.value) {
    if (catalogPriceIds.has(price.id)) {
      continue
    }
    rows.push({
      key: `price:${price.id}`,
      in_cpa: false,
      id: price.model,
      name: price.model,
      owner: null,
      suggested_provider: '',
      price,
      provider: price.provider,
      model: price.model,
      billing_unit: billingUnitForPrice(price, price.model),
      status: priceStatus(price, price.model),
    })
  }
  return rows
})

const providerOptions = computed(() =>
  [...new Set(priceRows.value.map((row) => row.provider).filter(Boolean))]
    .sort((a, b) => a.localeCompare(b))
    .map((provider) => ({ label: provider, value: provider })),
)

const liteLLMProxyHint = computed(() =>
  t(
    'LiteLLM 价格数据从 GitHub 下载；如果当前网络无法访问 GitHub，可以启用代理后再同步。',
    'LiteLLM price data is downloaded from GitHub. If GitHub is not reachable from this network, enable a proxy and sync again.',
  ),
)

const statusOptions = computed<Array<{ label: string; value: PriceStatusFilter }>>(() => [
  { label: t('CPA 可用模型', 'CPA available models'), value: 'cpa' },
  { label: t('未定价', 'Unpriced'), value: 'missing' },
  { label: 'LiteLLM', value: 'litellm' },
  { label: t('手动', 'Manual'), value: 'manual' },
  { label: t('仅有价格', 'Prices only'), value: 'library' },
])

const filteredPrices = computed(() => {
  return priceRows.value.filter((row) => {
    if (selectedProvider.value && row.provider !== selectedProvider.value) {
      return false
    }
    if (selectedStatus.value && !rowMatchesStatus(row, selectedStatus.value)) {
      return false
    }
    return priceMatchesSearch(row)
  })
})

const pagedPrices = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredPrices.value.slice(start, start + pageSize.value)
})

watch([selectedProvider, selectedStatus, searchQuery], () => {
  page.value = 1
})

function updatePricePage(value: number) {
  page.value = value
}

function updatePricePageSize(value: number) {
  pageSize.value = value
  page.value = 1
}

function handleProviderFilterChange(value: unknown) {
  selectedProvider.value = typeof value === 'string' ? value : null
}

function handleStatusFilterChange(value: unknown) {
  selectedStatus.value = statusOptions.value.some((option) => option.value === value)
    ? value as PriceStatusFilter
    : null
}

function normalizePriceSearch(value: string) {
  return value.trim().toLowerCase()
}

function providerFromModelId(modelId: string) {
  const separator = modelId.indexOf('/')
  return separator > 0 ? modelId.slice(0, separator) : ''
}

function billingUnitForModel(model: string): BillingUnit {
  return model.trim().toLowerCase().includes('image') ? 'request' : 'token'
}

function billingUnitForPrice(price: ModelPrice | null, fallbackModel: string): BillingUnit {
  if (price?.billing_unit === 'request') {
    return 'request'
  }
  if (price?.billing_unit === 'token') {
    return 'token'
  }
  return billingUnitForModel(price?.model || fallbackModel)
}

function priceReadyForBilling(price: ModelPrice, fallbackModel: string): boolean {
  return billingUnitForPrice(price, fallbackModel) === 'request' ? typeof price.request_usd === 'number' : true
}

function priceStatus(price: ModelPrice, fallbackModel: string): PriceRowStatus {
  if (!priceReadyForBilling(price, fallbackModel)) {
    return 'missing'
  }
  return price.auto_synced ? 'litellm' : 'manual'
}

function rowMatchesStatus(row: PriceDisplayRow, status: PriceStatusFilter) {
  switch (status) {
    case 'cpa':
      return row.in_cpa
    case 'library':
      return !row.in_cpa
    default:
      return row.status === status
  }
}

const normalizedSearchQuery = computed(() => normalizePriceSearch(searchQuery.value))

const filteredPriceCount = computed(() => filteredPrices.value.length)

const totalPriceCount = computed(() => priceRows.value.length)
const cpaModelCount = computed(() => catalog.value?.models.length ?? 0)
const unpricedModelCount = computed(
  () => catalog.value?.unpriced_models ?? priceRows.value.filter((row) => row.in_cpa && row.status === 'missing').length,
)
const syncedPriceCount = computed(() => prices.value.filter((price) => price.auto_synced).length)
const manualPriceCount = computed(() => prices.value.filter((price) => !price.auto_synced).length)
const catalogNotice = computed(() => {
  const current = catalog.value
  if (!current) {
    return ''
  }
  if (!current.has_api_keys) {
    return t('还没有本地绑定的 API Key，当前只显示已有价格库条目。', 'No local API keys are bound yet. Only existing price library entries are shown.')
  }
  if (current.queryable_api_key_count === 0) {
    return t(
      '本地 API Key 没有保存明文 Key，暂时无法查询 CPA 当前模型，只显示已有价格库条目。',
      'Local API keys do not store plaintext keys, so CPA models cannot be queried for now. Only existing price library entries are shown.',
    )
  }
  if (current.errors.length > 0) {
    const details = current.errors
      .slice(0, 3)
      .map((item) =>
        t(
          `${item.description}：${serverText(item.message, '查询失败', 'Query failed')}`,
          `${item.description}: ${serverText(item.message, '查询失败', 'Query failed')}`,
        ),
      )
      .join(t('；', '; '))
    return t(`部分 Key 查询 CPA 模型失败：${details}`, `Some keys failed to query CPA models: ${details}`)
  }
  return ''
})
const isRequestPriceForm = computed(() => billingUnitForModel(form.model) === 'request')
const showLongContextFastUnsupported = computed(
  () => form.long_context_enabled && form.fast_multiplier !== 1,
)
const priceSaveHint = computed(() =>
  isRequestPriceForm.value
    ? t(
        'image 模型按每次成功调用固定金额计费；仅修改 FAST 倍率或峰谷价格不会取消 LiteLLM 同步。',
        'Image models are charged per successful call. Changing only the FAST multiplier or peak/off-peak settings keeps LiteLLM sync enabled.',
      )
    : t(
        '基础价格修改后会转为手动价格；仅修改 FAST 倍率、长上下文或峰谷设置不会取消 LiteLLM 同步。',
        'Changing base prices makes them manual. Changing only FAST, long-context or peak/off-peak settings keeps LiteLLM sync enabled.',
      ),
)

interface PriceMetricCard {
  key: string
  label: string
  value: string
  footnote: string
  icon: Component
}

const priceMetrics = computed<PriceMetricCard[]>(() => [
  {
    key: 'models',
    label: t('CPA 模型', 'CPA models'),
    value: formatInteger(cpaModelCount.value),
    footnote: catalog.value
      ? t(
          `可查询 Key ${formatInteger(catalog.value.queryable_api_key_count)} / ${formatInteger(catalog.value.api_key_count)}`,
          `Queryable keys ${formatInteger(catalog.value.queryable_api_key_count)} / ${formatInteger(catalog.value.api_key_count)}`,
        )
      : t('等待刷新', 'Waiting for refresh'),
    icon: Layers3,
  },
  {
    key: 'unpriced',
    label: t('未定价', 'Unpriced'),
    value: formatInteger(unpricedModelCount.value),
    footnote: t(
      `筛选后 ${formatInteger(filteredPriceCount.value)} / ${formatInteger(totalPriceCount.value)}`,
      `Filtered ${formatInteger(filteredPriceCount.value)} / ${formatInteger(totalPriceCount.value)}`,
    ),
    icon: Server,
  },
  {
    key: 'synced',
    label: t('LiteLLM 同步', 'LiteLLM sync'),
    value: formatInteger(syncedPriceCount.value),
    footnote: t('自动维护', 'Auto maintained'),
    icon: RefreshCw,
  },
  {
    key: 'manual',
    label: t('手动价格', 'Manual prices'),
    value: formatInteger(manualPriceCount.value),
    footnote: t('优先保留', 'Preserved first'),
    icon: Database,
  },
])

function priceMatchesSearch(row: PriceDisplayRow) {
  if (!normalizedSearchQuery.value) {
    return true
  }
  return (
    row.provider.toLowerCase().includes(normalizedSearchQuery.value) ||
    row.model.toLowerCase().includes(normalizedSearchQuery.value) ||
    row.id.toLowerCase().includes(normalizedSearchQuery.value) ||
    row.name.toLowerCase().includes(normalizedSearchQuery.value) ||
    (row.owner ?? '').toLowerCase().includes(normalizedSearchQuery.value) ||
    row.suggested_provider.toLowerCase().includes(normalizedSearchQuery.value)
  )
}

function resetForm() {
  editingId.value = null
  form.provider = ''
  form.model = ''
  form.input_usd_per_million = 0
  form.output_usd_per_million = 0
  form.cache_read_usd_per_million = 0
  form.cache_creation_usd_per_million = 0
  form.request_usd = null
  form.fast_multiplier = 1
  form.long_context_enabled = false
  form.long_context_threshold_tokens = 0
  form.long_context_input_usd_per_million = 0
  form.long_context_output_usd_per_million = 0
  form.long_context_cache_read_usd_per_million = 0
  form.long_context_cache_creation_usd_per_million = 0
  form.long_context_fast_unsupported = false
  form.off_peak_enabled = false
  form.off_peak_input_usd_per_million = 0
  form.off_peak_output_usd_per_million = 0
  form.off_peak_cache_read_usd_per_million = 0
  form.off_peak_cache_creation_usd_per_million = 0
  form.off_peak_request_usd = null
  form.long_context_off_peak_input_usd_per_million = 0
  form.long_context_off_peak_output_usd_per_million = 0
  form.long_context_off_peak_cache_read_usd_per_million = 0
  form.long_context_off_peak_cache_creation_usd_per_million = 0
}

async function refresh() {
  isLoading.value = true
  try {
    const [nextPrices, nextCatalog] = await Promise.all([listModelPrices(), listModelPriceCatalog()])
    prices.value = nextPrices
    catalog.value = nextCatalog
    const lastPage = Math.max(1, Math.ceil(filteredPrices.value.length / pageSize.value))
    page.value = Math.min(page.value, lastPage)
  } catch (error) {
    message.error(errorText(error, '加载模型价格失败', 'Failed to load model prices'))
  } finally {
    isLoading.value = false
  }
}

function openCreate(prefill: Partial<ModelPricePayload> = {}) {
  resetForm()
  form.provider = prefill.provider ?? ''
  form.model = prefill.model ?? ''
  form.input_usd_per_million = prefill.input_usd_per_million ?? 0
  form.output_usd_per_million = prefill.output_usd_per_million ?? 0
  form.cache_read_usd_per_million = prefill.cache_read_usd_per_million ?? 0
  form.cache_creation_usd_per_million = prefill.cache_creation_usd_per_million ?? 0
  form.request_usd = prefill.request_usd ?? null
  form.fast_multiplier = prefill.fast_multiplier ?? 1
  form.long_context_enabled = prefill.long_context_enabled ?? false
  form.long_context_threshold_tokens = prefill.long_context_threshold_tokens ?? 0
  form.long_context_input_usd_per_million = prefill.long_context_input_usd_per_million ?? 0
  form.long_context_output_usd_per_million = prefill.long_context_output_usd_per_million ?? 0
  form.long_context_cache_read_usd_per_million = prefill.long_context_cache_read_usd_per_million ?? 0
  form.long_context_cache_creation_usd_per_million = prefill.long_context_cache_creation_usd_per_million ?? 0
  form.long_context_fast_unsupported = prefill.long_context_fast_unsupported ?? false
  fillOffPeakForm(prefill)
  modalOpen.value = true
}

function openCreateForRow(row: PriceDisplayRow) {
  openCreate({
    provider: row.provider || row.suggested_provider || row.owner || '',
    model: row.id,
  })
}

function openEdit(row: ModelPrice) {
  editingId.value = row.id
  form.provider = row.provider
  form.model = row.model
  form.input_usd_per_million = row.input_usd_per_million
  form.output_usd_per_million = row.output_usd_per_million
  form.cache_read_usd_per_million = row.cache_read_usd_per_million
  form.cache_creation_usd_per_million = row.cache_creation_usd_per_million
  form.request_usd = row.request_usd
  form.fast_multiplier = row.fast_multiplier
  form.long_context_enabled = row.long_context_enabled
  form.long_context_threshold_tokens = row.long_context_threshold_tokens
  form.long_context_input_usd_per_million = row.long_context_input_usd_per_million
  form.long_context_output_usd_per_million = row.long_context_output_usd_per_million
  form.long_context_cache_read_usd_per_million = row.long_context_cache_read_usd_per_million
  form.long_context_cache_creation_usd_per_million = row.long_context_cache_creation_usd_per_million
  form.long_context_fast_unsupported = row.long_context_fast_unsupported
  fillOffPeakForm(row)
  modalOpen.value = true
}

function fillOffPeakForm(price: Partial<ModelPricePayload>) {
  form.off_peak_enabled = price.off_peak_enabled ?? false
  form.off_peak_input_usd_per_million = price.off_peak_input_usd_per_million ?? 0
  form.off_peak_output_usd_per_million = price.off_peak_output_usd_per_million ?? 0
  form.off_peak_cache_read_usd_per_million = price.off_peak_cache_read_usd_per_million ?? 0
  form.off_peak_cache_creation_usd_per_million = price.off_peak_cache_creation_usd_per_million ?? 0
  form.off_peak_request_usd = price.off_peak_request_usd ?? null
  form.long_context_off_peak_input_usd_per_million = price.long_context_off_peak_input_usd_per_million ?? 0
  form.long_context_off_peak_output_usd_per_million = price.long_context_off_peak_output_usd_per_million ?? 0
  form.long_context_off_peak_cache_read_usd_per_million = price.long_context_off_peak_cache_read_usd_per_million ?? 0
  form.long_context_off_peak_cache_creation_usd_per_million = price.long_context_off_peak_cache_creation_usd_per_million ?? 0
}

function normalizeNumberInput(value: string | number, fallback = 0): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function setPriceNumber(
  field: PriceFieldName | 'fast_multiplier',
  value: string | number,
) {
  form[field] = normalizeNumberInput(value)
}

function setRequestPrice(value: string | number) {
  form.request_usd = value === '' ? null : normalizeNumberInput(value)
}

function setOffPeakRequestPrice(value: string | number) {
  form.off_peak_request_usd = value === '' ? null : normalizeNumberInput(value)
}

function toggleOffPeak(enabled: boolean) {
  form.off_peak_enabled = enabled
  if (!enabled) return
  if (form.off_peak_request_usd === null) form.off_peak_request_usd = form.request_usd
  const fields = ['input', 'output', 'cache_read', 'cache_creation'] as const
  for (const field of fields) {
    const shortKey = `off_peak_${field}_usd_per_million` as PriceFieldName
    const longKey = `long_context_off_peak_${field}_usd_per_million` as PriceFieldName
    const baseKey = `${field}_usd_per_million` as PriceFieldName
    const longBaseKey = `long_context_${field}_usd_per_million` as PriceFieldName
    if (form[shortKey] === 0) form[shortKey] = form[baseKey]
    if (form[longKey] === 0) form[longKey] = form[longBaseKey]
  }
}

function setLongContextThreshold(value: string | number) {
  form.long_context_threshold_tokens = normalizeNumberInput(value)
}

function toggleLongContext(enabled: boolean) {
  form.long_context_enabled = enabled
  if (!enabled) {
    return
  }
  const longContextRates = [
    form.long_context_input_usd_per_million,
    form.long_context_output_usd_per_million,
    form.long_context_cache_read_usd_per_million,
    form.long_context_cache_creation_usd_per_million,
  ]
  if (longContextRates.every((value) => value === 0)) {
    form.long_context_input_usd_per_million = form.input_usd_per_million
    form.long_context_output_usd_per_million = form.output_usd_per_million
    form.long_context_cache_read_usd_per_million = form.cache_read_usd_per_million
    form.long_context_cache_creation_usd_per_million = form.cache_creation_usd_per_million
  }
}

async function savePrice() {
  const requestPriceMode = isRequestPriceForm.value
  const requestUSD = requestPriceMode && typeof form.request_usd === 'number' ? form.request_usd : null
  const payload: ModelPricePayload = {
    provider: form.provider.trim(),
    model: form.model.trim(),
    input_usd_per_million: form.input_usd_per_million,
    output_usd_per_million: form.output_usd_per_million,
    cache_read_usd_per_million: form.cache_read_usd_per_million,
    cache_creation_usd_per_million: form.cache_creation_usd_per_million,
    request_usd: requestUSD,
    fast_multiplier: form.fast_multiplier,
    long_context_enabled: !requestPriceMode && form.long_context_enabled,
    long_context_threshold_tokens: requestPriceMode ? 0 : form.long_context_threshold_tokens,
    long_context_input_usd_per_million: requestPriceMode ? 0 : form.long_context_input_usd_per_million,
    long_context_output_usd_per_million: requestPriceMode ? 0 : form.long_context_output_usd_per_million,
    long_context_cache_read_usd_per_million: requestPriceMode ? 0 : form.long_context_cache_read_usd_per_million,
    long_context_cache_creation_usd_per_million: requestPriceMode ? 0 : form.long_context_cache_creation_usd_per_million,
    long_context_fast_unsupported: !requestPriceMode && form.long_context_fast_unsupported,
    off_peak_enabled: form.off_peak_enabled,
    off_peak_input_usd_per_million: form.off_peak_input_usd_per_million,
    off_peak_output_usd_per_million: form.off_peak_output_usd_per_million,
    off_peak_cache_read_usd_per_million: form.off_peak_cache_read_usd_per_million,
    off_peak_cache_creation_usd_per_million: form.off_peak_cache_creation_usd_per_million,
    off_peak_request_usd: requestPriceMode ? form.off_peak_request_usd : null,
    long_context_off_peak_input_usd_per_million: form.long_context_off_peak_input_usd_per_million,
    long_context_off_peak_output_usd_per_million: form.long_context_off_peak_output_usd_per_million,
    long_context_off_peak_cache_read_usd_per_million: form.long_context_off_peak_cache_read_usd_per_million,
    long_context_off_peak_cache_creation_usd_per_million: form.long_context_off_peak_cache_creation_usd_per_million,
  }
  if (!payload.provider || !payload.model) {
    message.error(t('服务商和模型不能为空', 'Provider and model are required'))
    return
  }
  if (!Number.isFinite(payload.fast_multiplier) || payload.fast_multiplier <= 0) {
    message.error(t('FAST 倍率必须大于 0', 'FAST multiplier must be greater than 0'))
    return
  }
  if (requestPriceMode && requestUSD === null) {
    message.error(t('image 模型需要填写每次调用价格', 'Image models require a per-call price'))
    return
  }
  if (requestPriceMode && payload.off_peak_enabled && payload.off_peak_request_usd === null) {
    message.error(t('请填写空闲时段每次调用价格', 'Enter the off-peak per-call price'))
    return
  }
  if (payload.long_context_enabled && (!Number.isInteger(payload.long_context_threshold_tokens) || payload.long_context_threshold_tokens <= 0)) {
    message.error(t('长上下文 Token 阈值必须是大于 0 的整数', 'Long-context token threshold must be a positive integer'))
    return
  }
  isPriceSaving.value = true
  try {
    if (editingId.value === null) {
      await createModelPrice(payload)
      message.success(t('模型价格已创建', 'Model price created'))
    } else {
      await updateModelPrice(editingId.value, payload)
      message.success(t('模型价格已更新', 'Model price updated'))
    }
    modalOpen.value = false
    await refresh()
  } catch (error) {
    message.error(errorText(error, '保存模型价格失败', 'Failed to save model price'))
  } finally {
    isPriceSaving.value = false
  }
}

async function syncPrices() {
  isSyncing.value = true
  try {
    const result = await syncLitellmModelPrices()
    message.success(
      t(
        `同步完成：LiteLLM 价格 ${result.imported} 条，手动价格保留 ${result.skipped_manual} 条`,
        `Sync complete: ${result.imported} LiteLLM prices imported, ${result.skipped_manual} manual prices preserved`,
      ),
    )
    await refresh()
  } catch (error) {
    const detail = errorText(error, '同步模型价格失败', 'Failed to sync model prices')
    message.error(t(`${detail}。${liteLLMProxyHint.value}`, `${detail}. ${liteLLMProxyHint.value}`))
  } finally {
    isSyncing.value = false
  }
}

async function openProxySettings() {
  proxyModalOpen.value = true
  isProxyLoading.value = true
  try {
    const settings = await getLiteLLMProxySettings()
    proxyForm.enabled = settings.enabled
    proxyForm.proxy_url = settings.proxy_url
  } catch (error) {
    message.error(errorText(error, '加载代理配置失败', 'Failed to load proxy settings'))
  } finally {
    isProxyLoading.value = false
  }
}

async function saveProxySettings() {
  const payload: LiteLLMProxySettingsPayload = {
    enabled: proxyForm.enabled,
    proxy_url: proxyForm.proxy_url.trim(),
  }
  if (payload.enabled && !payload.proxy_url) {
    message.error(t('启用代理时必须填写代理地址', 'Proxy URL is required when proxy is enabled'))
    return
  }
  isProxySaving.value = true
  try {
    const saved = await updateLiteLLMProxySettings(payload)
    proxyForm.enabled = saved.enabled
    proxyForm.proxy_url = saved.proxy_url
    proxyModalOpen.value = false
    message.success(t('代理配置已保存', 'Proxy settings saved'))
  } catch (error) {
    message.error(errorText(error, '保存代理配置失败', 'Failed to save proxy settings'))
  } finally {
    isProxySaving.value = false
  }
}

function confirmDelete(row: ModelPrice) {
  dialog.warning({
    title: t('删除价格', 'Delete price'),
    content: `${row.provider} / ${row.model}`,
    positiveText: t('删除', 'Delete'),
    negativeText: t('取消', 'Cancel'),
    onPositiveClick: async () => {
      await deleteModelPrice(row.id)
      message.success(t('模型价格已删除', 'Model price deleted'))
      await refresh()
    },
  })
}

function formatPriceValue(value: number | null | undefined): string {
  return typeof value === 'number' ? String(value) : '-'
}

function priceTiers(row: PriceDisplayRow): PriceTier[] {
  const price = row.price
  if (!price) return [{ key: 'base', label: '-', prefix: '' }]
  const tiers: PriceTier[] = [
    { key: 'base', label: price.off_peak_enabled ? t('常规 · 高峰', 'Standard · peak') : t('常规', 'Standard'), prefix: '' },
  ]
  if (price.off_peak_enabled) tiers.push({ key: 'off-peak', label: t('常规 · 空闲', 'Standard · off-peak'), prefix: 'off_peak_' })
  if (price.long_context_enabled) {
    tiers.push({ key: 'long', label: price.off_peak_enabled ? t('长上下文 · 高峰', 'Long context · peak') : t('长上下文', 'Long context'), prefix: 'long_context_' })
    if (price.off_peak_enabled) tiers.push({ key: 'long-off-peak', label: t('长上下文 · 空闲', 'Long context · off-peak'), prefix: 'long_context_off_peak_' })
  }
  return tiers
}

function tierPriceValue(row: PriceDisplayRow, tier: PriceTier, field: 'input' | 'output' | 'cache_read' | 'cache_creation' | 'request'): string {
  if (!row.price) return '-'
  if (field === 'request') {
    if (row.billing_unit !== 'request') return '-'
    return formatPriceValue(tier.prefix === 'off_peak_' ? row.price.off_peak_request_usd : row.price.request_usd)
  }
  if (row.billing_unit === 'request') return '-'
  const key = `${tier.prefix}${field}_usd_per_million` as PriceFieldName
  return formatPriceValue(row.price[key])
}

async function loadCalendarYear() {
  isCalendarLoading.value = true
  try {
    const calendar = await getPricingHolidayCalendar(calendarYear.value)
    calendarDatesText.value = calendar.dates.join('\n')
    calendarSourceURL.value = calendar.source_url
    calendarConfigured.value = calendar.configured
  } catch (error) {
    message.error(errorText(error, '加载节假日失败', 'Failed to load holidays'))
  } finally {
    isCalendarLoading.value = false
  }
}

async function openCalendar() {
  calendarModalOpen.value = true
  await loadCalendarYear()
}

async function saveCalendar() {
  const dates = calendarDatesText.value.split(/[\s,，]+/).map((date) => date.trim()).filter(Boolean)
  isCalendarSaving.value = true
  try {
    const calendar = await updatePricingHolidayCalendar(calendarYear.value, dates)
    calendarDatesText.value = calendar.dates.join('\n')
    calendarConfigured.value = calendar.configured
    message.success(t('节假日日历已保存', 'Holiday calendar saved'))
    calendarModalOpen.value = false
  } catch (error) {
    message.error(errorText(error, '保存节假日失败', 'Failed to save holidays'))
  } finally {
    isCalendarSaving.value = false
  }
}

function fastMultiplierValue(row: PriceDisplayRow): string {
  return row.price ? `×${row.price.fast_multiplier}` : '-'
}

function longContextBadgeTitle(price: ModelPrice): string {
  const thresholdDescription = t(
    `输入上下文超过 ${formatInteger(price.long_context_threshold_tokens)} Token 后使用长上下文费率`,
    `Uses long-context rates above ${formatInteger(price.long_context_threshold_tokens)} input tokens`,
  )
  if (!price.long_context_fast_unsupported) {
    return thresholdDescription
  }
  return `${thresholdDescription}${t('；长上下文不支持 FAST 模式', '; long context does not support FAST mode')}`
}

function billingUnitLabel(row: PriceDisplayRow): string {
  return row.billing_unit === 'request' ? t('按次', 'Per call') : t('按 Token', 'Per token')
}

function billingUnitVariant(row: PriceDisplayRow): 'secondary' | 'outline' {
  return row.billing_unit === 'request' ? 'secondary' : 'outline'
}

function priceStatusLabel(row: PriceDisplayRow): string {
  if (row.status === 'missing') {
    return t('未定价', 'Unpriced')
  }
  return row.status === 'litellm' ? 'LiteLLM' : t('手动', 'Manual')
}

function priceStatusVariant(row: PriceDisplayRow): 'destructive' | 'secondary' | 'outline' {
  if (row.status === 'missing') {
    return 'destructive'
  }
  return row.status === 'litellm' ? 'secondary' : 'outline'
}

onMounted(() => {
  void refresh()
})

</script>

<template>
  <section class="page price-page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('模型价格', 'Model prices') }}</h1>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <Button variant="outline" :disabled="isSyncing" @click="syncPrices">
          <Spinner v-if="isSyncing" data-icon="inline-start" />
          <RefreshCw v-else data-icon="inline-start" />
          {{ t('同步 LiteLLM', 'Sync LiteLLM') }}
        </Button>
        <Button variant="outline" :disabled="isSyncing" @click="openProxySettings">
          <Settings2 data-icon="inline-start" />
          {{ t('代理配置', 'Proxy settings') }}
        </Button>
        <Button variant="outline" @click="openCalendar">
          <CalendarDays data-icon="inline-start" />
          {{ t('节假日日历', 'Holiday calendar') }}
        </Button>
        <Button @click="openCreate()">
          <Plus data-icon="inline-start" />
          {{ t('新增价格', 'Add price') }}
        </Button>
      </div>
    </div>

    <div class="metric-grid price-metrics">
      <Card v-for="metric in priceMetrics" :key="metric.key" class="price-metric-card border border-border ring-0">
        <CardHeader class="flex flex-row items-start justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <CardDescription>{{ metric.label }}</CardDescription>
            <CardTitle class="text-2xl tabular-nums">{{ metric.value }}</CardTitle>
          </div>
          <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary" aria-hidden="true">
            <component :is="metric.icon" class="size-5" />
          </div>
        </CardHeader>
        <CardContent class="text-xs text-muted-foreground">
          {{ metric.footnote }}
        </CardContent>
      </Card>
    </div>

    <section class="panel table-panel price-table-panel">
      <div class="price-table-top">
        <Alert v-if="catalogNotice" class="price-alert">
          <CircleAlert />
          <AlertDescription>{{ catalogNotice }}</AlertDescription>
        </Alert>

        <div class="table-toolbar">
          <div class="price-toolbar-layout">
            <div class="price-filters">
              <span class="filter-label">{{ t('筛选', 'Filters') }}</span>

              <FilterCombobox
                class="provider-filter"
                :model-value="selectedProvider"
                :options="providerOptions"
                :placeholder="t('服务商', 'Provider')"
                :search-placeholder="t('搜索服务商', 'Search providers')"
                :empty-text="t('没有匹配的服务商', 'No matching providers')"
                :icon="Server"
                @update:model-value="handleProviderFilterChange"
              />

              <FilterCombobox
                class="status-filter"
                :model-value="selectedStatus"
                :options="statusOptions"
                :placeholder="t('状态', 'Status')"
                :icon="ListFilter"
                :searchable="false"
                @update:model-value="handleStatusFilterChange"
              />

              <InputGroup class="price-search">
                <InputGroupAddon>
                  <Search />
                </InputGroupAddon>
                <InputGroupInput
                  v-model="searchQuery"
                  :placeholder="t('搜索模型或服务商', 'Search models or providers')"
                />
              </InputGroup>
            </div>

            <span class="result-count">
              {{ t(`共 ${filteredPriceCount} / ${totalPriceCount} 条`, `${filteredPriceCount} / ${totalPriceCount} items`) }}
            </span>
          </div>
        </div>
      </div>

      <div class="price-table">
        <Table class="min-w-[1600px] table-fixed">
          <TableHeader class="sticky top-0 bg-card">
            <TableRow>
              <TableHead class="w-[300px]">{{ t('模型', 'Model') }}</TableHead>
              <TableHead class="w-[130px]">{{ t('服务商', 'Provider') }}</TableHead>
              <TableHead class="w-[90px]">{{ t('定价', 'Pricing') }}</TableHead>
              <TableHead class="w-[100px]">{{ t('计费方式', 'Billing') }}</TableHead>
              <TableHead class="w-[150px]">{{ t('费率档位', 'Rate tiers') }}</TableHead>
              <TableHead class="w-[100px] text-right">{{ t('FAST 倍率', 'FAST multiplier') }}</TableHead>
              <TableHead class="w-[100px] text-right">{{ t('每次 ($)', 'Per call ($)') }}</TableHead>
              <TableHead class="w-[110px] text-right">{{ t('输入 ($/MTok)', 'Input ($/MTok)') }}</TableHead>
              <TableHead class="w-[110px] text-right">{{ t('输出 ($/MTok)', 'Output ($/MTok)') }}</TableHead>
              <TableHead class="w-[120px] text-right">{{ t('缓存读 ($/MTok)', 'Cache read ($/MTok)') }}</TableHead>
              <TableHead class="w-[120px] text-right">{{ t('缓存写 ($/MTok)', 'Cache write ($/MTok)') }}</TableHead>
              <TableHead class="w-[120px]">{{ t('更新', 'Updated') }}</TableHead>
              <TableHead class="w-[56px]">
                <span class="sr-only">{{ t('操作', 'Actions') }}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="isLoading && filteredPrices.length === 0">
              <TableRow v-for="rowIndex in 8" :key="`price-skeleton-${rowIndex}`">
                <TableCell v-for="columnIndex in 13" :key="columnIndex">
                  <Skeleton class="h-4 w-full" />
                </TableCell>
              </TableRow>
            </template>

            <TableEmpty v-else-if="filteredPrices.length === 0" :colspan="13">
              {{ t('暂无模型价格', 'No model prices') }}
            </TableEmpty>

            <TableRow v-for="row in pagedPrices" v-else :key="row.key">
              <TableCell>
                <div class="model-cell">
                  <div class="model-title-row">
                    <span class="model-name" :title="row.id">{{ row.id }}</span>
                    <Badge v-if="row.in_cpa" variant="secondary" class="model-availability-tag">
                      {{ t('CPA 可用模型', 'CPA available model') }}
                    </Badge>
                    <Badge
                      v-if="row.price?.long_context_enabled"
                      variant="outline"
                      class="model-availability-tag"
                      :title="longContextBadgeTitle(row.price)"
                    >
                      {{
                        row.price.long_context_fast_unsupported
                          ? t('长上下文 · 无 FAST', 'Long context · no FAST')
                          : t('长上下文', 'Long context')
                      }}
                    </Badge>
                  </div>
                  <div v-if="row.name && row.name !== row.id" class="model-sub" :title="row.name">
                    {{ row.name }}
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <div class="provider-cell">
                  <div class="provider-main" :title="row.provider || '-'">{{ row.provider || '-' }}</div>
                  <div v-if="row.owner && row.owner !== row.provider" class="model-sub">
                    {{ t('所有者', 'Owner') }}: {{ row.owner }}
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <Badge :variant="priceStatusVariant(row)">{{ priceStatusLabel(row) }}</Badge>
              </TableCell>
              <TableCell>
                <Badge :variant="billingUnitVariant(row)">{{ billingUnitLabel(row) }}</Badge>
              </TableCell>
              <TableCell>
                <div class="rate-tier-stack">
                  <div v-for="tier in priceTiers(row)" :key="tier.key" class="rate-tier-line">{{ tier.label }}</div>
                </div>
              </TableCell>
              <TableCell class="text-right tabular-nums">{{ fastMultiplierValue(row) }}</TableCell>
              <TableCell class="text-right tabular-nums"><div class="rate-tier-stack"><div v-for="tier in priceTiers(row)" :key="tier.key" class="rate-tier-line">{{ tierPriceValue(row, tier, 'request') }}</div></div></TableCell>
              <TableCell class="text-right tabular-nums">
                <div class="rate-tier-stack"><div v-for="tier in priceTiers(row)" :key="tier.key" class="rate-tier-line">{{ tierPriceValue(row, tier, 'input') }}</div></div>
              </TableCell>
              <TableCell class="text-right tabular-nums">
                <div class="rate-tier-stack"><div v-for="tier in priceTiers(row)" :key="tier.key" class="rate-tier-line">{{ tierPriceValue(row, tier, 'output') }}</div></div>
              </TableCell>
              <TableCell class="text-right tabular-nums">
                <div class="rate-tier-stack"><div v-for="tier in priceTiers(row)" :key="tier.key" class="rate-tier-line">{{ tierPriceValue(row, tier, 'cache_read') }}</div></div>
              </TableCell>
              <TableCell class="text-right tabular-nums">
                <div class="rate-tier-stack"><div v-for="tier in priceTiers(row)" :key="tier.key" class="rate-tier-line">{{ tierPriceValue(row, tier, 'cache_creation') }}</div></div>
              </TableCell>
              <TableCell class="tabular-nums">
                {{ row.price ? formatDateTime(row.price.updated_at) : '-' }}
              </TableCell>
              <TableCell class="text-right">
                <DropdownMenu>
                  <DropdownMenuTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      class="price-actions-trigger"
                      :aria-label="t(`打开 ${row.id} 的操作菜单`, `Open actions for ${row.id}`)"
                    >
                      <MoreHorizontal />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" :side-offset="4" class="w-40">
                    <DropdownMenuGroup>
                      <DropdownMenuItem @select="row.price ? openEdit(row.price) : openCreateForRow(row)">
                        <Pencil v-if="row.price" />
                        <Plus v-else />
                        <span>{{ row.price ? t('改价', 'Edit price') : t('设价', 'Set price') }}</span>
                      </DropdownMenuItem>
                    </DropdownMenuGroup>
                    <template v-if="row.price">
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        variant="destructive"
                        @select="confirmDelete(row.price)"
                      >
                        <Trash2 />
                        <span>{{ t('删除', 'Delete') }}</span>
                      </DropdownMenuItem>
                    </template>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>

        <div v-if="isLoading && filteredPrices.length > 0" class="table-loading-overlay">
          <Spinner />
        </div>
      </div>

      <TablePaginationFooter
        :page="page"
        :page-size="pageSize"
        :total="filteredPriceCount"
        @update:page="updatePricePage"
        @update:page-size="updatePricePageSize"
      />
    </section>

    <Dialog v-model:open="modalOpen">
      <DialogContent class="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-2xl">
        <form class="flex flex-col gap-5" @submit.prevent="savePrice">
          <DialogHeader>
            <DialogTitle>
              {{ editingId === null ? t('新增价格', 'Add price') : t('编辑价格', 'Edit price') }}
            </DialogTitle>
            <DialogDescription>{{ priceSaveHint }}</DialogDescription>
          </DialogHeader>

          <FieldGroup class="form-grid">
            <Field>
              <FieldLabel for="price-provider">{{ t('服务商', 'Provider') }}</FieldLabel>
              <Input id="price-provider" v-model="form.provider" required />
            </Field>
            <Field>
              <FieldLabel for="price-model">{{ t('模型', 'Model') }}</FieldLabel>
              <Input id="price-model" v-model="form.model" required />
            </Field>
            <Field class="wide-form-item">
              <FieldLabel for="price-fast-multiplier">{{ t('FAST 倍率', 'FAST multiplier') }}</FieldLabel>
              <Input
                id="price-fast-multiplier"
                type="number"
                min="0.01"
                step="0.01"
                :model-value="form.fast_multiplier"
                @update:model-value="setPriceNumber('fast_multiplier', $event)"
              />
            </Field>
            <Field v-if="isRequestPriceForm" class="wide-form-item">
              <FieldLabel for="price-per-request">{{ t('每次调用价格 USD', 'Per-call price USD') }}</FieldLabel>
              <Input
                id="price-per-request"
                type="number"
                min="0"
                step="any"
                :model-value="form.request_usd ?? ''"
                :placeholder="t('例如：0.04', 'Example: 0.04')"
                @update:model-value="setRequestPrice"
              />
            </Field>
            <template v-else>
              <Field>
                <FieldLabel for="price-input">{{ t('输入价格', 'Input price') }}</FieldLabel>
                <Input
                  id="price-input"
                  type="number"
                  min="0"
                  step="any"
                  :model-value="form.input_usd_per_million"
                  @update:model-value="setPriceNumber('input_usd_per_million', $event)"
                />
              </Field>
              <Field>
                <FieldLabel for="price-output">{{ t('输出价格', 'Output price') }}</FieldLabel>
                <Input
                  id="price-output"
                  type="number"
                  min="0"
                  step="any"
                  :model-value="form.output_usd_per_million"
                  @update:model-value="setPriceNumber('output_usd_per_million', $event)"
                />
              </Field>
              <Field>
                <FieldLabel for="price-cache-read">{{ t('缓存读价格', 'Cache read price') }}</FieldLabel>
                <Input
                  id="price-cache-read"
                  type="number"
                  min="0"
                  step="any"
                  :model-value="form.cache_read_usd_per_million"
                  @update:model-value="setPriceNumber('cache_read_usd_per_million', $event)"
                />
              </Field>
              <Field>
                <FieldLabel for="price-cache-write">{{ t('缓存写价格', 'Cache write price') }}</FieldLabel>
                <Input
                  id="price-cache-write"
                  type="number"
                  min="0"
                  step="any"
                  :model-value="form.cache_creation_usd_per_million"
                  @update:model-value="setPriceNumber('cache_creation_usd_per_million', $event)"
                />
              </Field>
              <div class="wide-form-item long-context-panel">
                <Field orientation="horizontal" class="long-context-switch-row">
                  <FieldContent>
                    <FieldLabel for="price-long-context-enabled">
                      {{ t('启用长上下文费率', 'Enable long-context rates') }}
                    </FieldLabel>
                    <FieldDescription>
                      {{
                        t(
                          '输入上下文 Token 超出阈值后，整条请求使用长上下文费率；FAST 请求默认继续应用倍率，不支持时可在下方标记。',
                          'When input context exceeds the threshold, the entire request uses long-context rates. FAST requests apply the multiplier by default and can be marked unsupported below.',
                        )
                      }}
                    </FieldDescription>
                  </FieldContent>
                  <Switch
                    id="price-long-context-enabled"
                    :model-value="form.long_context_enabled"
                    @update:model-value="toggleLongContext"
                  />
                </Field>

                <FieldGroup v-if="form.long_context_enabled" class="long-context-rate-grid">
                  <Field class="wide-form-item">
                    <FieldLabel for="price-long-context-threshold">
                      {{ t('Token 阈值', 'Token threshold') }}
                    </FieldLabel>
                    <Input
                      id="price-long-context-threshold"
                      type="number"
                      min="1"
                      step="1"
                      :model-value="form.long_context_threshold_tokens"
                      :placeholder="t('例如：272000', 'Example: 272000')"
                      @update:model-value="setLongContextThreshold"
                    />
                    <FieldDescription>
                      {{ t('仅统计完整输入上下文（含缓存 Token），输出 Token 不参与阈值判断。', 'Counts the full input context including cached tokens; output tokens do not affect the threshold.') }}
                    </FieldDescription>
                  </Field>
                  <Field>
                    <FieldLabel for="price-long-context-input">
                      {{ t('长上下文输入价格 ($/MTok)', 'Long-context input ($/MTok)') }}
                    </FieldLabel>
                    <Input
                      id="price-long-context-input"
                      type="number"
                      min="0"
                      step="any"
                      :model-value="form.long_context_input_usd_per_million"
                      @update:model-value="setPriceNumber('long_context_input_usd_per_million', $event)"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="price-long-context-output">
                      {{ t('长上下文输出价格 ($/MTok)', 'Long-context output ($/MTok)') }}
                    </FieldLabel>
                    <Input
                      id="price-long-context-output"
                      type="number"
                      min="0"
                      step="any"
                      :model-value="form.long_context_output_usd_per_million"
                      @update:model-value="setPriceNumber('long_context_output_usd_per_million', $event)"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="price-long-context-cache-read">
                      {{ t('长上下文缓存读价格 ($/MTok)', 'Long-context cache read ($/MTok)') }}
                    </FieldLabel>
                    <Input
                      id="price-long-context-cache-read"
                      type="number"
                      min="0"
                      step="any"
                      :model-value="form.long_context_cache_read_usd_per_million"
                      @update:model-value="setPriceNumber('long_context_cache_read_usd_per_million', $event)"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="price-long-context-cache-write">
                      {{ t('长上下文缓存写价格 ($/MTok)', 'Long-context cache write ($/MTok)') }}
                    </FieldLabel>
                    <Input
                      id="price-long-context-cache-write"
                      type="number"
                      min="0"
                      step="any"
                      :model-value="form.long_context_cache_creation_usd_per_million"
                      @update:model-value="setPriceNumber('long_context_cache_creation_usd_per_million', $event)"
                    />
                  </Field>
                  <Field
                    v-if="showLongContextFastUnsupported"
                    orientation="horizontal"
                    class="wide-form-item long-context-fast-row"
                  >
                    <FieldContent>
                      <FieldLabel for="price-long-context-fast-unsupported">
                        {{ t('长上下文不支持 FAST 模式', 'FAST mode unavailable for long context') }}
                      </FieldLabel>
                      <FieldDescription>
                        {{
                          t(
                            '开启后，超过长上下文阈值的请求不叠加 FAST 倍率；短上下文计费不受影响。',
                            'When enabled, requests above the long-context threshold do not apply the FAST multiplier. Short-context billing is unchanged.',
                          )
                        }}
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      id="price-long-context-fast-unsupported"
                      v-model="form.long_context_fast_unsupported"
                    />
                  </Field>
                </FieldGroup>
              </div>
            </template>
            <div class="wide-form-item long-context-panel">
              <Field orientation="horizontal" class="long-context-switch-row">
                <FieldContent>
                  <FieldLabel for="price-off-peak-enabled">{{ t('启用峰谷分段计费', 'Enable peak/off-peak pricing') }}</FieldLabel>
                  <FieldDescription>
                    {{ t('现有费率作为高峰价；北京时间工作日 09:00–12:00、14:00–18:00 为高峰，周末、法定节假日和其余时间为空闲。未配置节假日的年份全天使用空闲价。', 'Existing rates become peak rates. Beijing weekdays 09:00–12:00 and 14:00–18:00 are peak; weekends, holidays and other hours are off-peak. Years without a holiday calendar use off-peak all day.') }}
                  </FieldDescription>
                </FieldContent>
                <Switch id="price-off-peak-enabled" :model-value="form.off_peak_enabled" @update:model-value="toggleOffPeak" />
              </Field>
              <FieldGroup v-if="form.off_peak_enabled" class="long-context-rate-grid">
                <Field v-if="isRequestPriceForm" class="wide-form-item">
                  <FieldLabel for="price-off-peak-request">{{ t('空闲时段每次调用价格 USD', 'Off-peak per-call price USD') }}</FieldLabel>
                  <Input id="price-off-peak-request" type="number" min="0" step="any" :model-value="form.off_peak_request_usd ?? ''" @update:model-value="setOffPeakRequestPrice" />
                </Field>
                <template v-else>
                  <Field v-for="field in offPeakRateFields" :key="field.key">
                    <FieldLabel :for="`price-${field.key}`">{{ t(field.zh, field.en) }}</FieldLabel>
                    <Input :id="`price-${field.key}`" type="number" min="0" step="any" :model-value="form[field.key]" @update:model-value="setPriceNumber(field.key, $event)" />
                  </Field>
                  <template v-if="form.long_context_enabled">
                    <Field v-for="field in longOffPeakRateFields" :key="field.key">
                      <FieldLabel :for="`price-${field.key}`">{{ t(field.zh, field.en) }}</FieldLabel>
                      <Input :id="`price-${field.key}`" type="number" min="0" step="any" :model-value="form[field.key]" @update:model-value="setPriceNumber(field.key, $event)" />
                    </Field>
                  </template>
                </template>
              </FieldGroup>
            </div>
          </FieldGroup>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isPriceSaving" @click="modalOpen = false">
              {{ t('取消', 'Cancel') }}
            </Button>
            <Button type="submit" :disabled="isPriceSaving">
              <Spinner v-if="isPriceSaving" data-icon="inline-start" />
              {{ t('保存', 'Save') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="proxyModalOpen">
      <DialogContent class="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg">
        <form class="flex flex-col gap-5" @submit.prevent="saveProxySettings">
          <DialogHeader>
            <DialogTitle>{{ t('LiteLLM 代理配置', 'LiteLLM proxy settings') }}</DialogTitle>
            <DialogDescription>
              {{ t('为 LiteLLM 价格同步配置网络代理。', 'Configure a network proxy for LiteLLM price synchronization.') }}
            </DialogDescription>
          </DialogHeader>

          <Alert>
            <CircleAlert />
            <AlertDescription>{{ liteLLMProxyHint }}</AlertDescription>
          </Alert>

          <FieldGroup>
            <Field orientation="horizontal" class="proxy-switch-row">
              <FieldContent>
                <FieldLabel for="litellm-proxy-enabled">{{ t('使用代理', 'Use proxy') }}</FieldLabel>
                <FieldDescription>
                  {{ t('同步时通过下方代理地址访问 GitHub。', 'Use the proxy URL below when synchronizing from GitHub.') }}
                </FieldDescription>
              </FieldContent>
              <Switch
                id="litellm-proxy-enabled"
                v-model="proxyForm.enabled"
                :disabled="isProxyLoading || isProxySaving"
              />
            </Field>
            <Field :data-disabled="!proxyForm.enabled || isProxyLoading || isProxySaving || undefined">
              <FieldLabel for="litellm-proxy-url">{{ t('代理地址', 'Proxy URL') }}</FieldLabel>
              <Input
                id="litellm-proxy-url"
                v-model="proxyForm.proxy_url"
                :disabled="!proxyForm.enabled || isProxyLoading || isProxySaving"
                :placeholder="t('http://127.0.0.1:7890 或 socks5://127.0.0.1:1080', 'http://127.0.0.1:7890 or socks5://127.0.0.1:1080')"
              />
            </Field>
          </FieldGroup>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isProxySaving" @click="proxyModalOpen = false">
              {{ t('取消', 'Cancel') }}
            </Button>
            <Button type="submit" :disabled="isProxySaving">
              <Spinner v-if="isProxySaving" data-icon="inline-start" />
              {{ t('保存', 'Save') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="calendarModalOpen">
      <DialogContent class="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-xl">
        <form class="flex flex-col gap-5" @submit.prevent="saveCalendar">
          <DialogHeader>
            <DialogTitle>{{ t('峰谷计费节假日日历', 'Peak pricing holiday calendar') }}</DialogTitle>
            <DialogDescription>{{ t('按国务院每年公布的放假安排维护日期；周末始终为空闲时段。未配置的年份全天使用空闲费率。', 'Maintain dates from the annual State Council holiday notice. Weekends are always off-peak. Unconfigured years use off-peak rates all day.') }}</DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel for="pricing-calendar-year">{{ t('年份', 'Year') }}</FieldLabel>
              <div class="flex items-center gap-2">
                <Input id="pricing-calendar-year" v-model.number="calendarYear" type="number" min="2000" max="2100" step="1" :disabled="isCalendarLoading || isCalendarSaving" />
                <Button type="button" variant="outline" :disabled="isCalendarLoading || isCalendarSaving" @click="loadCalendarYear">{{ t('查询', 'Load') }}</Button>
              </div>
            </Field>
            <Field>
              <FieldLabel for="pricing-calendar-dates">{{ t('法定节假日日期（每行一个）', 'Holiday dates (one per line)') }}</FieldLabel>
              <Textarea id="pricing-calendar-dates" v-model="calendarDatesText" rows="10" placeholder="2026-01-01" :disabled="isCalendarLoading || isCalendarSaving" class="font-mono text-sm" />
              <FieldDescription>{{ calendarConfigured ? t('该年份已配置，可编辑后保存。', 'This year is configured; edit and save.') : t('该年份尚未配置，保存后工作日才会启用高峰时段。', 'This year is not configured; peak hours become active on weekdays after saving.') }}</FieldDescription>
            </Field>
            <a v-if="calendarSourceURL" :href="calendarSourceURL" target="_blank" rel="noopener noreferrer" class="text-sm text-primary underline-offset-4 hover:underline">{{ t('查看国务院放假通知', 'View State Council holiday notice') }}</a>
          </FieldGroup>
          <DialogFooter>
            <Button type="button" variant="outline" @click="calendarModalOpen = false">{{ t('取消', 'Cancel') }}</Button>
            <Button type="submit" :disabled="isCalendarLoading || isCalendarSaving">{{ t('保存', 'Save') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </section>
</template>
<style scoped>
.price-metrics {
  grid-template-columns: repeat(4, minmax(150px, 1fr));
}

.price-metric-card {
  min-width: 0;
}

.rate-tier-stack {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.rate-tier-line {
  min-height: 1.5rem;
  line-height: 1.5rem;
  white-space: nowrap;
}

.price-metric-card :deep([data-slot="card-header"]) {
  padding-bottom: 10px;
}

.price-metric-card :deep([data-slot="card-content"]) {
  padding-top: 0;
}

.price-table-panel,
.price-table {
  max-width: 100%;
  min-width: 0;
  min-height: 0;
}

.price-table-panel {
  overflow: hidden;
}

.price-table-top {
  display: grid;
  gap: 8px;
}

.table-toolbar {
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-bottom: 0;
  border-radius: var(--radius) var(--radius) 0 0;
  background: var(--card);
}

.price-toolbar-layout {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-width: 0;
}

.price-filters {
  display: flex;
  flex: 1 1 auto;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
}

.filter-label,
.result-count {
  color: var(--muted-foreground);
  font-size: 12px;
  white-space: nowrap;
}

.provider-filter {
  width: 210px;
}

.status-filter {
  width: 160px;
}

.price-search {
  width: min(300px, 100%);
}

.price-table {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--border);
  border-top: 0;
  background: var(--card);
}

.price-table :deep([data-slot="table-container"]) {
  max-height: max(240px, calc(100dvh - 420px));
  overflow: auto;
  overscroll-behavior: contain;
}

.table-loading-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--background) 62%, transparent);
}

.price-actions-trigger {
  margin-left: auto;
  color: var(--muted-foreground);
}

.model-cell,
.provider-cell {
  min-width: 0;
}

.model-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.model-availability-tag {
  flex: 0 0 auto;
}

.model-name,
.provider-main {
  min-width: 0;
  overflow: hidden;
  color: var(--foreground);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-sub {
  margin-top: 2px;
  overflow: hidden;
  color: var(--muted-foreground);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.wide-form-item {
  grid-column: 1 / -1;
}

.long-context-panel {
  display: grid;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--card);
}

.long-context-switch-row {
  gap: 20px;
}

.long-context-rate-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.long-context-fast-row {
  gap: 20px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.proxy-switch-row {
  gap: 20px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: color-mix(in oklch, var(--muted) 45%, transparent);
}

@media (min-width: 861px) {
  .price-page {
    grid-template-rows: auto auto minmax(0, 1fr);
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .price-table-panel {
    display: grid;
    grid-template-rows: auto minmax(0, 1fr) auto;
    min-height: 0;
  }

  .price-table {
    height: 100%;
  }

  .price-table :deep([data-slot="table-container"]) {
    height: 100%;
    max-height: none;
  }
}

@media (max-width: 980px) {
  .price-toolbar-layout {
    align-items: flex-start;
    flex-direction: column;
  }

  .provider-filter {
    width: min(200px, calc(100vw - 32px));
  }

  .status-filter {
    width: min(170px, calc(100vw - 32px));
  }

  .price-search {
    width: min(260px, calc(100vw - 32px));
  }
}

@media (max-width: 620px) {
  .price-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .wide-form-item {
    grid-column: auto;
  }

  .long-context-rate-grid {
    grid-template-columns: 1fr;
  }

  .price-filters {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .filter-label {
    display: none;
  }

  .provider-filter,
  .status-filter,
  .price-search {
    width: 100%;
  }

  .price-search {
    grid-column: 1 / -1;
  }

}
</style>
