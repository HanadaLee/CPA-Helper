<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import {
  Check,
  ChevronDown,
  CircleCheck,
  Cpu,
  Database,
  KeyRound,
  Route,
  Server,
  UserRound,
} from '@lucide/vue'
import {
  Combobox,
  ComboboxAnchor,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxList,
  ComboboxTrigger,
} from '@/components/ui/combobox'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'
import DateTimeRangePicker from '@/shared/ui/DateTimeRangePicker.vue'
import TablePaginationFooter from '@/shared/ui/TablePaginationFooter.vue'

import { getUsageOptions, getUsageRecord, getUsageRecords } from '@/features/usage/api/usageApi'
import type {
  RankingItem,
  UsageFilters,
  UsageOptionsResponse,
  UsageRecordDetail,
  UsageRecordListItem,
} from '@/shared/types/api'
import {
  formatDateTime,
  formatInteger,
  formatLocalDateTimeParam,
  formatUsd,
  jsonPretty,
} from '@/shared/utils/format'
import { useI18n } from '@/shared/i18n'

type FailedFilter = 'all' | 'success' | 'failed'
type QuickRangeKey = 'today' | 'last24h' | 'last3d' | 'last7d' | 'all'
type UsageScope = 'admin' | 'account'

interface RefreshOptions {
  resetPage?: boolean
  silent?: boolean
}

interface Props {
  scope: UsageScope
}

interface UsageFilterOption {
  label: string
  value: string | number
}

interface UsageFilterCombobox {
  key: string
  icon: Component
  selected: UsageFilterOption | null
  options: UsageFilterOption[]
  placeholder: string
  searchPlaceholder: string
  emptyText: string
  onChange: (value: unknown) => void
}

const AUTO_REFRESH_INTERVAL_MS = 10_000
const TOTAL_REFRESH_INTERVAL_MS = 60_000
const HOUR_MS = 60 * 60 * 1000
const DAY_MS = 24 * HOUR_MS
const ALL_RECORDS_START_PARAM = '0001-01-01T00:00:00+08:00'
const ALL_RECORDS_END_PARAM = '9999-12-31T23:59:59+08:00'

const route = useRoute()
const router = useRouter()
const message = toast
const props = defineProps<Props>()
const { currentLanguage, errorText, t } = useI18n()
const isLoading = ref(false)
const isAutoRefreshing = ref(false)
const autoRefreshError = ref<string | null>(null)
const lastRefreshedAt = ref<Date | null>(null)
const drawerOpen = ref(false)
const selectedRecord = ref<UsageRecordDetail | null>(null)
const records = ref<UsageRecordListItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const options = ref<UsageOptionsResponse>({
  users: [],
  api_key_descriptions: [],
  providers: [],
  models: [],
  sources: [],
  endpoints: [],
})

function initialRange(): [number, number] | null {
  const startQuery = typeof route.query.start === 'string' ? route.query.start : ''
  const endQuery = typeof route.query.end === 'string' ? route.query.end : ''
  if (startQuery && endQuery) {
    return [new Date(startQuery).getTime(), new Date(endQuery).getTime()]
  }
  return null
}

function todayRange(): [number, number] {
  const now = new Date()
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const tomorrow = new Date(start)
  tomorrow.setDate(start.getDate() + 1)
  return [start.getTime(), tomorrow.getTime()]
}

function isTodayRange(range: [number, number] | null): boolean {
  if (!range) {
    return false
  }
  const [todayStart, tomorrowStart] = todayRange()
  return range[0] === todayStart && range[1] === tomorrowStart
}

function failedFromQuery(): FailedFilter {
  if (route.query.failed === 'true') {
    return 'failed'
  }
  if (route.query.failed === 'false') {
    return 'success'
  }
  return 'all'
}

function numberFromQuery(value: unknown): number | null {
  if (typeof value !== 'string' || !value) {
    return null
  }
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

const initialIsAllRange = route.query.range === 'all'
const initialDateRange = initialIsAllRange ? null : initialRange()
const dateRange = ref<[number, number] | null>(initialDateRange)
const activeQuickRange = ref<QuickRangeKey | null>(
  initialIsAllRange
    ? 'all'
    : initialDateRange === null || isTodayRange(initialDateRange)
      ? 'today'
      : null,
)
const filterForm = reactive({
  user_id: numberFromQuery(route.query.user_id),
  api_key_description:
    typeof route.query.api_key_description === 'string' ? route.query.api_key_description : null,
  provider: typeof route.query.provider === 'string' ? route.query.provider : null,
  model: typeof route.query.model === 'string' ? route.query.model : null,
  source_key: typeof route.query.source_key === 'string' ? route.query.source_key : null,
  endpoint: typeof route.query.endpoint === 'string' ? route.query.endpoint : null,
  failed: failedFromQuery(),
})

const quickRangeOptions = computed<Array<{ key: QuickRangeKey; label: string }>>(() => [
  { key: 'today', label: t('今日', 'Today') },
  { key: 'last24h', label: t('近24小时', 'Last 24 hours') },
  { key: 'last3d', label: t('近3日', 'Last 3 days') },
  { key: 'last7d', label: t('近7日', 'Last 7 days') },
  { key: 'all', label: t('全部', 'All') },
])

const failedFilterOptions = computed(() => [
  { label: t('全部', 'All'), value: 'all' },
  { label: t('成功', 'Success'), value: 'success' },
  { label: t('失败', 'Failed'), value: 'failed' },
])

function apiKeyFilterLabel(item: UsageOptionsResponse['api_key_descriptions'][number]): string {
  return item.label?.trim() || item.key
}

function emptyRankingItem(
  key: string,
  label: string,
  extra: Partial<Pick<RankingItem, 'api_key_description' | 'user_id'>> = {},
): RankingItem {
  return {
    key,
    label,
    records: 0,
    failed_records: 0,
    total_tokens: 0,
    estimated_cost_usd: 0,
    user_id: null,
    api_key_description: null,
    ...extra,
  }
}

function fallbackOptionsFromRecords(items: UsageRecordListItem[]): UsageOptionsResponse {
  const users = new Map<number, RankingItem>()
  const apiKeyDescriptions = new Map<string, RankingItem>()
  const providers = new Set<string>()
  const models = new Set<string>()
  const endpoints = new Set<string>()

  items.forEach((item) => {
    if (item.user_id !== null && !users.has(item.user_id)) {
      users.set(
        item.user_id,
        emptyRankingItem(String(item.user_id), userLabel(item.user_label), { user_id: item.user_id }),
      )
    }
    const description = item.api_key_description?.trim()
    if (description && !apiKeyDescriptions.has(description)) {
      apiKeyDescriptions.set(
        description,
        emptyRankingItem(description, description, { api_key_description: description }),
      )
    }
    if (item.provider) {
      providers.add(item.provider)
    }
    if (item.model) {
      models.add(item.model)
    }
    if (item.endpoint) {
      endpoints.add(item.endpoint)
    }
  })

  return {
    users: [...users.values()],
    api_key_descriptions: [...apiKeyDescriptions.values()],
    providers: [...providers].sort(),
    models: [...models].sort(),
    sources: [],
    endpoints: [...endpoints].sort(),
  }
}

function normalizeUsageOptions(
  nextOptions: Partial<UsageOptionsResponse> | null | undefined,
  fallbackRecords: UsageRecordListItem[],
): UsageOptionsResponse {
  const fallback = fallbackOptionsFromRecords(fallbackRecords)
  return {
    users: nextOptions?.users?.length ? nextOptions.users : fallback.users,
    api_key_descriptions: nextOptions?.api_key_descriptions?.length
      ? nextOptions.api_key_descriptions
      : fallback.api_key_descriptions,
    providers: nextOptions?.providers?.length ? nextOptions.providers : fallback.providers,
    models: nextOptions?.models?.length ? nextOptions.models : fallback.models,
    sources: nextOptions?.sources ?? fallback.sources,
    endpoints: nextOptions?.endpoints?.length ? nextOptions.endpoints : fallback.endpoints,
  }
}

const selectOptions = computed(() => ({
  users: options.value.users
    .filter((item) => item.user_id !== null)
    .map((item) => ({ label: item.label, value: item.user_id as number })),
  apiKeyDescriptions: options.value.api_key_descriptions.map((item) => ({
    label: apiKeyFilterLabel(item),
    value: item.key,
  })),
  providers: options.value.providers.map((item) => ({ label: item, value: item })),
  models: options.value.models.map((item) => ({ label: item, value: item })),
  sources: options.value.sources.map((item) => ({ label: item.label, value: item.key })),
  endpoints: options.value.endpoints.map((item) => ({ label: item, value: item })),
}))

const isAccountScope = computed(() => props.scope === 'account')
const pageTitle = computed(() =>
  isAccountScope.value ? t('我的明细', 'My records') : t('请求明细', 'Request records'),
)

function selectedFilterOption(
  filterOptions: UsageFilterOption[],
  value: string | number | null,
): UsageFilterOption | null {
  return filterOptions.find((option) => option.value === value) ?? null
}

function filterOptionValue(value: unknown): string | number | null {
  if (typeof value !== 'object' || value === null || !('value' in value)) {
    return null
  }
  const optionValue = value.value
  return typeof optionValue === 'string' || typeof optionValue === 'number' ? optionValue : null
}

const filterComboboxes = computed<UsageFilterCombobox[]>(() => {
  const items: UsageFilterCombobox[] = [
    {
      key: 'api-key',
      icon: KeyRound,
      selected: selectedFilterOption(selectOptions.value.apiKeyDescriptions, filterForm.api_key_description),
      options: selectOptions.value.apiKeyDescriptions,
      placeholder: t('KEY 描述', 'Key description'),
      searchPlaceholder: t('搜索 KEY 描述', 'Search key descriptions'),
      emptyText: t('没有匹配的 KEY 描述', 'No matching key descriptions'),
      onChange: (value) => handleApiKeyChange(filterOptionValue(value)),
    },
    {
      key: 'provider',
      icon: Server,
      selected: selectedFilterOption(selectOptions.value.providers, filterForm.provider),
      options: selectOptions.value.providers,
      placeholder: t('服务商', 'Provider'),
      searchPlaceholder: t('搜索服务商', 'Search providers'),
      emptyText: t('没有匹配的服务商', 'No matching providers'),
      onChange: (value) => handleProviderChange(filterOptionValue(value)),
    },
    {
      key: 'model',
      icon: Cpu,
      selected: selectedFilterOption(selectOptions.value.models, filterForm.model),
      options: selectOptions.value.models,
      placeholder: t('模型', 'Model'),
      searchPlaceholder: t('搜索模型', 'Search models'),
      emptyText: t('没有匹配的模型', 'No matching models'),
      onChange: (value) => handleModelChange(filterOptionValue(value)),
    },
  ]

  if (!isAccountScope.value) {
    items.unshift({
      key: 'user',
      icon: UserRound,
      selected: selectedFilterOption(selectOptions.value.users, filterForm.user_id),
      options: selectOptions.value.users,
      placeholder: t('用户昵称', 'User nickname'),
      searchPlaceholder: t('搜索用户', 'Search users'),
      emptyText: t('没有匹配的用户', 'No matching users'),
      onChange: (value) => handleUserChange(filterOptionValue(value)),
    })
    items.push({
      key: 'source',
      icon: Database,
      selected: selectedFilterOption(selectOptions.value.sources, filterForm.source_key),
      options: selectOptions.value.sources,
      placeholder: t('来源', 'Source'),
      searchPlaceholder: t('搜索来源', 'Search sources'),
      emptyText: t('没有匹配的来源', 'No matching sources'),
      onChange: (value) => handleSourceChange(filterOptionValue(value)),
    })
  }

  items.push({
    key: 'endpoint',
    icon: Route,
    selected: selectedFilterOption(selectOptions.value.endpoints, filterForm.endpoint),
    options: selectOptions.value.endpoints,
    placeholder: t('接口', 'Endpoint'),
    searchPlaceholder: t('搜索接口', 'Search endpoints'),
    emptyText: t('没有匹配的接口', 'No matching endpoints'),
    onChange: (value) => handleEndpointChange(filterOptionValue(value)),
  })

  return items
})

const recordColumnCount = computed(() => isAccountScope.value ? 13 : 14)
const recordsTableClass = computed(() => cn(
  'table-fixed',
  isAccountScope.value ? 'min-w-[1392px]' : 'min-w-[1510px]',
))

const refreshStatusText = computed(() => {
  const lastRefreshTime = lastRefreshedAt.value
  if (!lastRefreshTime) {
    return autoRefreshError.value
      ? t('自动刷新异常 · 尚无成功同步', 'Auto refresh error · no successful sync yet')
      : t('每 10 秒自动刷新 · 等待首次同步', 'Auto refresh every 10 seconds · waiting for first sync')
  }
  const lastRefreshText = new Intl.DateTimeFormat(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(lastRefreshTime)
  if (autoRefreshError.value) {
    return t(`自动刷新异常 · 最近成功 ${lastRefreshText}`, `Auto refresh error · last success ${lastRefreshText}`)
  }
  return t(`每 10 秒自动刷新 · 最近 ${lastRefreshText}`, `Auto refresh every 10 seconds · latest ${lastRefreshText}`)
})

function buildFilters(): UsageFilters {
  const failed =
    filterForm.failed === 'all' ? undefined : filterForm.failed === 'failed' ? true : false
  const start =
    activeQuickRange.value === 'all'
      ? ALL_RECORDS_START_PARAM
      : dateRange.value
        ? formatLocalDateTimeParam(dateRange.value[0])
        : undefined
  const end =
    activeQuickRange.value === 'all'
      ? ALL_RECORDS_END_PARAM
      : dateRange.value
        ? formatLocalDateTimeParam(dateRange.value[1])
        : undefined
  return {
    scope: props.scope,
    start,
    end,
    user_id: isAccountScope.value ? undefined : (filterForm.user_id ?? undefined),
    api_key_description: filterForm.api_key_description ?? undefined,
    provider: filterForm.provider ?? undefined,
    model: filterForm.model ?? undefined,
    source_key: isAccountScope.value ? undefined : (filterForm.source_key ?? undefined),
    endpoint: filterForm.endpoint ?? undefined,
    failed,
  }
}

function filtersToQuery(
  filters: UsageFilters,
  rangeKey: QuickRangeKey | null = null,
): Record<string, string> {
  const query: Record<string, string> = {}
  Object.entries(filters).forEach(([key, value]) => {
    if (key !== 'scope' && value !== undefined && value !== '') {
      query[key] = String(value)
    }
  })
  if (rangeKey === 'all') {
    delete query.start
    delete query.end
    query.range = 'all'
  }
  return query
}

function buildQuickRange(key: QuickRangeKey): [number, number] | null {
  switch (key) {
    case 'today':
      return todayRange()
    case 'last24h': {
      const end = Date.now()
      return [end - 24 * HOUR_MS, end]
    }
    case 'last3d': {
      const end = Date.now()
      return [end - 3 * DAY_MS, end]
    }
    case 'last7d': {
      const end = Date.now()
      return [end - 7 * DAY_MS, end]
    }
    case 'all':
      return null
  }
}

function normalizeRangeValue(value: unknown): [number, number] | null {
  if (
    Array.isArray(value) &&
    value.length === 2 &&
    typeof value[0] === 'number' &&
    typeof value[1] === 'number'
  ) {
    return [value[0], value[1]]
  }
  return null
}

function normalizeSelectValue(value: unknown): string | null {
  if (value === null || value === undefined || value === '') {
    return null
  }
  return String(value)
}

function refreshAfterFilterChange() {
  void refresh({ resetPage: true })
}

function handleCustomRangeChange(value: unknown) {
  dateRange.value = normalizeRangeValue(value)
  activeQuickRange.value = null
  refreshAfterFilterChange()
}

function handleApiKeyChange(value: unknown) {
  filterForm.api_key_description = normalizeSelectValue(value)
  refreshAfterFilterChange()
}

function handleUserChange(value: unknown) {
  filterForm.user_id = typeof value === 'number' ? value : null
  refreshAfterFilterChange()
}

function handleProviderChange(value: unknown) {
  filterForm.provider = normalizeSelectValue(value)
  refreshAfterFilterChange()
}

function handleModelChange(value: unknown) {
  filterForm.model = normalizeSelectValue(value)
  refreshAfterFilterChange()
}

function handleSourceChange(value: unknown) {
  filterForm.source_key = normalizeSelectValue(value)
  refreshAfterFilterChange()
}

function handleEndpointChange(value: unknown) {
  filterForm.endpoint = normalizeSelectValue(value)
  refreshAfterFilterChange()
}

function handleFailedChange(value: unknown) {
  filterForm.failed = value === 'success' || value === 'failed' ? value : 'all'
  refreshAfterFilterChange()
}

async function applyQuickRange(key: QuickRangeKey) {
  activeQuickRange.value = key
  dateRange.value = buildQuickRange(key)
  await refresh({ resetPage: true })
}

function handleQuickRangeChange(value: unknown) {
  if (isQuickRangeKey(value)) {
    void applyQuickRange(value)
  }
}

function isQuickRangeKey(value: unknown): value is QuickRangeKey {
  return typeof value === 'string' && quickRangeOptions.value.some((option) => option.key === value)
}

function handlePageChange(value: number) {
  page.value = value
  void refresh()
}

function handlePageSizeChange(value: unknown) {
  const nextPageSize = Number(value)
  if (!Number.isFinite(nextPageSize) || nextPageSize <= 0 || nextPageSize === pageSize.value) {
    return
  }
  pageSize.value = nextPageSize
  void refresh({ resetPage: true })
}

let queuedRefresh: RefreshOptions | null = null
let lastTotalRefreshAt = 0

function queueRefresh(options: RefreshOptions) {
  if (options.silent) {
    return
  }
  queuedRefresh = {
    resetPage: Boolean(queuedRefresh?.resetPage || options.resetPage),
    silent: false,
  }
}

async function refresh({ resetPage = false, silent = false }: RefreshOptions = {}) {
  if (silent && (document.hidden || page.value !== 1)) {
    return
  }
  if (isLoading.value || isAutoRefreshing.value) {
    queueRefresh({ resetPage, silent })
    return
  }
  if (resetPage) {
    page.value = 1
  }
  if (activeQuickRange.value) {
    dateRange.value = buildQuickRange(activeQuickRange.value)
  }
  if (silent) {
    isAutoRefreshing.value = true
  } else {
    isLoading.value = true
  }
  try {
    const filters = buildFilters()
    const usedServerDefaultRange = filters.start === undefined && filters.end === undefined
    const includeTotal = !silent || Date.now() - lastTotalRefreshAt >= TOTAL_REFRESH_INTERVAL_MS
    const [recordsResult, optionsResult] = await Promise.allSettled([
      getUsageRecords(filters, page.value, pageSize.value, includeTotal),
      silent ? Promise.resolve(null) : getUsageOptions(filters),
    ])
    if (recordsResult.status === 'rejected') {
      throw recordsResult.reason
    }
    const nextRecords = recordsResult.value
    const nextOptions = optionsResult.status === 'fulfilled' ? optionsResult.value : null
    const previousIDs = new Set(records.value.map((record) => record.id))
    records.value = nextRecords.items
    if (typeof nextRecords.total === 'number') {
      total.value = nextRecords.total
      lastTotalRefreshAt = Date.now()
    } else if (page.value === 1 && nextRecords.items.length < pageSize.value) {
      total.value = nextRecords.items.length
    } else {
      total.value += nextRecords.items.filter((record) => !previousIDs.has(record.id)).length
    }
    if (!silent) {
      options.value = normalizeUsageOptions(nextOptions, nextRecords.items)
    }
    if (usedServerDefaultRange) {
      dateRange.value = [new Date(nextRecords.start).getTime(), new Date(nextRecords.end).getTime()]
    }
    void router.replace({
      query: filtersToQuery(
        usedServerDefaultRange
          ? { ...filters, start: nextRecords.start, end: nextRecords.end }
          : filters,
        activeQuickRange.value,
      ),
    })
    autoRefreshError.value = null
    lastRefreshedAt.value = new Date()
  } catch (error) {
    const errorMessage = errorText(error, '加载明细失败', 'Failed to load records')
    if (silent) {
      autoRefreshError.value = errorMessage
    } else {
      message.error(errorMessage)
    }
  } finally {
    if (silent) {
      isAutoRefreshing.value = false
    } else {
      isLoading.value = false
    }
    const nextRefresh = queuedRefresh
    queuedRefresh = null
    if (nextRefresh) {
      void refresh(nextRefresh)
    }
  }
}

async function openRecord(record: UsageRecordListItem) {
  try {
    selectedRecord.value = await getUsageRecord(record.id, props.scope)
    drawerOpen.value = true
  } catch (error) {
    message.error(errorText(error, '加载原始数据失败', 'Failed to load raw data'))
  }
}

function textOrDash(value: string | null | undefined): string {
  const normalized = value?.trim()
  return normalized || '-'
}

function userLabel(value: string | null | undefined): string {
  const normalized = value?.trim()
  if (!normalized || normalized === '未绑定') {
    return t('未知', 'Unknown')
  }
  return normalized
}

function apiKeyDescriptionLabel(value: string | null | undefined): string {
  const normalized = value?.trim()
  return normalized || t('未知', 'Unknown')
}

function formatLatency(value: number | null): string {
  if (value === null) {
    return '-'
  }
  return `${formatInteger(Math.round(value))} ms`
}

function formatPositiveLatency(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value) || value <= 0) {
    return '-'
  }
  return formatLatency(value)
}

function formatModelWithReasoning(
  record: Pick<UsageRecordListItem, 'model' | 'reasoning_effort' | 'request_service_tier'>,
  includeFast = false,
): string {
  const model = textOrDash(record.model)
  if (model === '-') {
    return model
  }
  const parts = [model]
  const reasoningEffort = record.reasoning_effort?.trim()
  if (reasoningEffort) {
    parts.push(reasoningEffort)
  }
  if (includeFast && isFastServiceTier(record.request_service_tier)) {
    parts.push('fast')
  }
  return parts.join(' ')
}

function isFastServiceTier(serviceTier: string | null | undefined): boolean {
  const normalized = serviceTier?.trim().toLowerCase()
  return normalized === 'priority' || normalized === 'fast'
}

function formatOutputTps(row: Pick<UsageRecordListItem, 'latency_ms' | 'output_tokens'>): string {
  if (row.latency_ms === null || row.latency_ms <= 0) {
    return '-'
  }
  const tokensPerSecond = (row.output_tokens / row.latency_ms) * 1000
  return new Intl.NumberFormat(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    maximumFractionDigits: tokensPerSecond < 10 ? 2 : 1,
  }).format(tokensPerSecond)
}

function formatOutputWithTps(row: Pick<UsageRecordListItem, 'latency_ms' | 'output_tokens'>): string {
  const output = formatInteger(row.output_tokens)
  const outputTps = formatOutputTps(row)
  return outputTps === '-' ? output : `${output} (${outputTps} tps)`
}

function isClaudeProvider(provider: string | null | undefined): boolean {
  const normalized = provider?.trim().toLowerCase()
  return normalized === 'claude' || normalized === 'anthropic'
}

function formatCacheTokens(row: UsageRecordListItem): string {
  if (isClaudeProvider(row.provider)) {
    return t(
      `(读 ${formatInteger(row.cache_read_tokens)} / 写 ${formatInteger(row.cache_creation_tokens)})`,
      `(read ${formatInteger(row.cache_read_tokens)} / write ${formatInteger(row.cache_creation_tokens)})`,
    )
  }
  return formatInteger(row.cached_tokens)
}

function uncachedInputTokens(row: UsageRecordListItem): number {
  if (isClaudeProvider(row.provider)) {
    return Math.max(0, row.input_tokens)
  }
  return Math.max(0, row.input_tokens - row.cached_tokens)
}

const detailRows = computed(() => {
  const record = selectedRecord.value
  if (!record) {
    return []
  }
  const rows = [
    { label: t('时间', 'Time'), value: formatDateTime(record.timestamp) },
    { label: t('模型', 'Model'), value: formatModelWithReasoning(record, true) },
    { label: t('服务商', 'Provider'), value: textOrDash(record.provider) },
    { label: t('接口', 'Endpoint'), value: textOrDash(record.endpoint) },
    { label: t('API KEY 描述', 'API key description'), value: apiKeyDescriptionLabel(record.api_key_description) },
    { label: t('认证类型', 'Auth type'), value: textOrDash(record.auth) },
    { label: t('请求 ID', 'Request ID'), value: textOrDash(record.request_id) },
    { label: t('结果', 'Result'), value: record.failed ? t('失败', 'Failed') : t('成功', 'Success') },
    { label: t('首字耗时', 'TTFT'), value: formatPositiveLatency(record.ttft_ms) },
    { label: t('总耗时', 'Latency'), value: formatLatency(record.latency_ms) },
    { label: t('输入 Token', 'Input tokens'), value: formatInteger(uncachedInputTokens(record)) },
    { label: t('缓存 Token', 'Cached tokens'), value: formatInteger(record.cached_tokens) },
    { label: t('缓存读 Token', 'Cache read tokens'), value: formatInteger(record.cache_read_tokens) },
    { label: t('缓存写 Token', 'Cache write tokens'), value: formatInteger(record.cache_creation_tokens) },
    { label: t('输出 Token', 'Output tokens'), value: formatOutputWithTps(record) },
    { label: t('思考 Token', 'Reasoning tokens'), value: formatInteger(record.reasoning_tokens) },
    { label: t('总 Token', 'Total tokens'), value: formatInteger(record.total_tokens) },
    { label: t('费用', 'Cost'), value: formatUsd(record.estimated_cost_usd) },
  ]
  if (!isAccountScope.value) {
    rows.splice(
      4,
      0,
      { label: t('来源', 'Source'), value: textOrDash(record.source) },
      { label: t('用户昵称', 'User nickname'), value: userLabel(record.user_label) },
    )
  }
  return rows
})

let autoRefreshTimer: number | undefined

function handleVisibilityChange() {
  if (!document.hidden) {
    void refresh({ silent: true })
  }
}

onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  void refresh()
  autoRefreshTimer = window.setInterval(() => {
    void refresh({ silent: true })
  }, AUTO_REFRESH_INTERVAL_MS)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  if (autoRefreshTimer !== undefined) {
    window.clearInterval(autoRefreshTimer)
  }
})
</script>

<template>
  <section class="page records-page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ pageTitle }}</h1>
      <div class="header-actions">
        <span class="refresh-status" :class="{ 'is-error': autoRefreshError }">
          {{ refreshStatusText }}
        </span>
      </div>
    </div>

    <section class="panel">
      <div class="panel-inner filter-toolbar">
        <div class="time-row">
          <Tabs
            class="quick-ranges"
            :model-value="activeQuickRange ?? undefined"
            @update:model-value="handleQuickRangeChange"
          >
            <TabsList
              class="quick-range-options grid h-8 w-full grid-cols-5"
              :aria-label="t('快捷时间范围', 'Quick time ranges')"
            >
              <TabsTrigger
                v-for="option in quickRangeOptions"
                :key="option.key"
                :value="option.key"
                class="min-w-0 justify-center px-2 text-center"
                :aria-label="option.label"
              >
                {{ option.label }}
              </TabsTrigger>
            </TabsList>
          </Tabs>
          <DateTimeRangePicker
            :model-value="dateRange"
            class="range-picker"
            clearable
            @update:model-value="handleCustomRangeChange"
          />
        </div>
        <div class="field-row" :class="{ 'is-account-scope': isAccountScope }">
          <Combobox
            v-for="filter in filterComboboxes"
            :key="filter.key"
            class="filter-combobox"
            :model-value="filter.selected"
            by="value"
            @update:model-value="filter.onChange"
          >
            <ComboboxAnchor as-child>
              <ComboboxTrigger as-child>
                <Button variant="outline" class="filter-combobox-trigger">
                  <component :is="filter.icon" data-icon="inline-start" />
                  <span class="min-w-0 flex-1 truncate text-left">
                    {{ filter.selected?.label ?? filter.placeholder }}
                  </span>
                  <ChevronDown data-icon="inline-end" class="text-muted-foreground" />
                </Button>
              </ComboboxTrigger>
            </ComboboxAnchor>
            <ComboboxList align="start">
              <ComboboxInput :placeholder="filter.searchPlaceholder" />
              <ComboboxEmpty>{{ filter.emptyText }}</ComboboxEmpty>
              <ComboboxGroup>
                <ComboboxItem :value="null">
                  {{ t('清除筛选', 'Clear filter') }}
                </ComboboxItem>
                <ComboboxItem
                  v-for="option in filter.options"
                  :key="String(option.value)"
                  :value="option"
                >
                  <span class="truncate">{{ option.label }}</span>
                  <ComboboxItemIndicator>
                    <Check />
                  </ComboboxItemIndicator>
                </ComboboxItem>
              </ComboboxGroup>
            </ComboboxList>
          </Combobox>
          <div class="status-actions">
            <Select :model-value="filterForm.failed" @update:model-value="handleFailedChange">
              <SelectTrigger class="status-select">
                <CircleCheck />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem
                    v-for="option in failedFilterOptions"
                    :key="option.value"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Button variant="outline" :disabled="isLoading" @click="refresh({ resetPage: true })">
              <Spinner v-if="isLoading" data-icon="inline-start" />
              {{ t('筛选', 'Filter') }}
            </Button>
          </div>
        </div>
      </div>
    </section>

    <section class="panel table-panel records-table-panel">
      <div class="records-table">
        <Table :class="recordsTableClass">
          <TableHeader class="sticky top-0 bg-card">
            <TableRow>
              <TableHead class="w-[138px]">{{ t('时间', 'Time') }}</TableHead>
              <TableHead v-if="!isAccountScope" class="w-[118px]">{{ t('用户昵称', 'User nickname') }}</TableHead>
              <TableHead class="w-[110px]">{{ t('KEY 描述', 'Key description') }}</TableHead>
              <TableHead class="w-[152px]">{{ t('模型', 'Model') }}</TableHead>
              <TableHead class="w-[72px]">{{ t('结果', 'Result') }}</TableHead>
              <TableHead class="w-[92px] text-right">{{ t('首字耗时', 'TTFT') }}</TableHead>
              <TableHead class="w-[92px] text-right">{{ t('总耗时', 'Latency') }}</TableHead>
              <TableHead class="w-[86px] text-right">{{ t('输入', 'Input') }}</TableHead>
              <TableHead class="w-[132px] text-right">{{ t('输出', 'Output') }}</TableHead>
              <TableHead class="w-[106px] text-right">{{ t('缓存', 'Cache') }}</TableHead>
              <TableHead class="w-[108px] text-right">{{ t('总 Token', 'Total tokens') }}</TableHead>
              <TableHead class="w-[100px]">{{ t('服务商', 'Provider') }}</TableHead>
              <TableHead class="w-[132px]">{{ t('请求 ID', 'Request ID') }}</TableHead>
              <TableHead class="w-[72px]"><span class="sr-only">{{ t('操作', 'Actions') }}</span></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="isLoading && records.length === 0">
              <TableRow v-for="rowIndex in 8" :key="`record-skeleton-${rowIndex}`">
                <TableCell v-for="columnIndex in recordColumnCount" :key="columnIndex">
                  <Skeleton class="h-4 w-full" />
                </TableCell>
              </TableRow>
            </template>

            <TableEmpty v-else-if="records.length === 0" :colspan="recordColumnCount">
              {{ t('暂无请求明细', 'No request records') }}
            </TableEmpty>

            <TableRow v-for="record in records" v-else :key="record.id">
              <TableCell class="tabular-nums">{{ formatDateTime(record.timestamp) }}</TableCell>
              <TableCell v-if="!isAccountScope">
                <Badge variant="outline" class="max-w-full" :title="userLabel(record.user_label)">
                  <span class="truncate">{{ userLabel(record.user_label) }}</span>
                </Badge>
              </TableCell>
              <TableCell>
                <span class="block truncate" :title="apiKeyDescriptionLabel(record.api_key_description)">
                  {{ apiKeyDescriptionLabel(record.api_key_description) }}
                </span>
              </TableCell>
              <TableCell>
                <span class="block truncate" :title="formatModelWithReasoning(record)">
                  {{ formatModelWithReasoning(record) }}
                </span>
              </TableCell>
              <TableCell>
                <Badge :variant="record.failed ? 'destructive' : 'secondary'">
                  {{ record.failed ? t('失败', 'Failed') : t('成功', 'Success') }}
                </Badge>
              </TableCell>
              <TableCell class="text-right tabular-nums">{{ formatPositiveLatency(record.ttft_ms) }}</TableCell>
              <TableCell class="text-right tabular-nums">{{ formatLatency(record.latency_ms) }}</TableCell>
              <TableCell class="text-right tabular-nums">{{ formatInteger(uncachedInputTokens(record)) }}</TableCell>
              <TableCell class="text-right tabular-nums">
                <span class="output-with-tps">
                  {{ formatInteger(record.output_tokens) }}
                  <span v-if="formatOutputTps(record) !== '-'" class="output-tps-muted">
                    ({{ formatOutputTps(record) }} tps)
                  </span>
                </span>
              </TableCell>
              <TableCell class="text-right tabular-nums">{{ formatCacheTokens(record) }}</TableCell>
              <TableCell class="text-right tabular-nums">{{ formatInteger(record.total_tokens) }}</TableCell>
              <TableCell>
                <span class="block truncate" :title="textOrDash(record.provider)">{{ textOrDash(record.provider) }}</span>
              </TableCell>
              <TableCell class="font-mono text-xs">
                <span class="block truncate" :title="textOrDash(record.request_id)">{{ textOrDash(record.request_id) }}</span>
              </TableCell>
              <TableCell class="text-right">
                <Button size="xs" variant="ghost" @click="openRecord(record)">
                  {{ t('详情', 'Details') }}
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>

        <div v-if="isLoading && records.length > 0" class="table-loading-overlay">
          <Spinner />
        </div>
      </div>
      <TablePaginationFooter
        :page="page"
        :page-size="pageSize"
        :total="total"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </section>

    <Sheet v-model:open="drawerOpen">
      <SheetContent
        side="right"
        class="w-full overflow-y-auto data-[side=right]:sm:w-[88vw] data-[side=right]:sm:max-w-[1080px]"
      >
        <SheetHeader>
          <SheetTitle>{{ t('请求事件详情', 'Request event details') }}</SheetTitle>
          <SheetDescription>
            {{ t('查看结构化字段和原始请求记录。', 'Review structured fields and the raw request record.') }}
          </SheetDescription>
        </SheetHeader>
        <div class="sheet-body">
          <h3 class="drawer-section-title">{{ t('结构化信息', 'Structured information') }}</h3>
          <div class="detail-grid">
            <div v-for="row in detailRows" :key="row.label" class="detail-item">
              <div class="detail-label">{{ row.label }}</div>
              <div class="detail-value">{{ row.value }}</div>
            </div>
          </div>
          <h3 class="drawer-section-title">{{ t('原始数据', 'Raw data') }}</h3>
          <pre class="mono-json">{{ jsonPretty(selectedRecord?.raw_json ?? {}) }}</pre>
        </div>
      </SheetContent>
    </Sheet>
  </section>
</template>

<style scoped>
.records-table-panel,
.records-table {
  min-width: 0;
}

.records-table {
  position: relative;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius) var(--radius) 0 0;
  background: var(--card);
}

.records-table :deep([data-slot="table-container"]) {
  max-height: max(320px, calc(100dvh - 318px));
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

.output-with-tps {
  white-space: nowrap;
}

.output-tps-muted {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.status-select {
  min-width: 96px;
}

.filter-toolbar {
  display: grid;
  gap: 12px;
  padding-block: 14px;
}

.time-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(380px, 440px);
  gap: 12px;
  align-items: center;
  min-width: 0;
}

.field-row {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr)) minmax(184px, 1.2fr);
  gap: 10px;
  align-items: stretch;
  width: 100%;
  min-width: 0;
}

.field-row.is-account-scope {
  grid-template-columns: repeat(4, minmax(0, 1fr)) minmax(184px, 1.2fr);
}

.range-picker {
  min-width: 0;
  width: 100%;
}

.quick-ranges {
  min-width: 0;
  width: 100%;
}

.filter-combobox {
  min-width: 0;
  width: 100%;
}

.filter-combobox-trigger {
  width: 100%;
  justify-content: flex-start;
  border-color: var(--input);
  font-weight: 400;
  box-shadow: var(--cpa-shadow-card);
}

.status-actions {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  min-width: 0;
}

.status-actions :deep([data-slot="button"]) {
  min-width: 82px;
}

.header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.refresh-status {
  color: var(--cpa-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.refresh-status.is-error {
  color: var(--cpa-danger);
}

.drawer-section-title {
  margin: 0 0 8px;
  color: var(--cpa-text);
  font-size: 14px;
  font-weight: 700;
}

.sheet-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0 16px 16px;
}

.drawer-section-title:not(:first-child) {
  margin-top: 16px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.detail-item {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface-muted);
  box-shadow: var(--cpa-shadow-hairline);
}

.detail-label {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.detail-value {
  margin-top: 3px;
  color: var(--cpa-text);
  font-weight: 600;
  overflow-wrap: anywhere;
}

@media (min-width: 861px) {
  .records-page {
    grid-template-rows: auto auto minmax(0, 1fr);
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .records-table-panel {
    display: grid;
    grid-template-rows: minmax(0, 1fr) auto;
    min-height: 0;
  }

  .records-table {
    height: 100%;
    min-height: 0;
  }

  .records-table :deep([data-slot="table-container"]) {
    height: 100%;
    max-height: none;
  }
}

@media (max-width: 1320px) {
  .field-row,
  .field-row.is-account-scope {
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  }
}

@media (max-width: 720px) {
  .filter-toolbar {
    gap: 8px;
    padding-block: 10px;
  }

  .time-row {
    grid-template-columns: 1fr;
    align-items: stretch;
    gap: 8px;
  }

  .status-actions {
    grid-column: 1 / -1;
    align-items: stretch;
  }

  .status-actions :deep([data-slot="select-trigger"]) {
    min-width: 0;
  }

  .header-actions {
    width: 100%;
    align-items: flex-start;
    justify-content: space-between;
  }

  .refresh-status {
    white-space: normal;
  }

  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
