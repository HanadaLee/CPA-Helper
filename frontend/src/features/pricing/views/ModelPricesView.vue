<script setup lang="ts">
import type { Component, CSSProperties } from 'vue'
import { computed, h, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  AppAlert,
  AppButton,
  AppDataTable,
  AppForm,
  AppFormItem,
  AppIcon,
  AppInput,
  AppNumberInput,
  AppModal,
  AppSelect,
  AppStack,
  AppSwitch,
  AppBadge,
  useDialog,
  useMessage,
  type DataTableColumns,
} from '@/shared/ui/app-kit'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
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
  listModelPriceCatalog,
  listModelPrices,
  syncLitellmModelPrices,
  updateLiteLLMProxySettings,
  updateModelPrice,
} from '@/features/pricing/api/pricingApi'
import type {
  LiteLLMProxySettingsPayload,
  ModelPrice,
  ModelPriceCatalogResponse,
  ModelPricePayload,
} from '@/shared/types/api'
import { formatDateTime, formatInteger } from '@/shared/utils/format'
import { useI18n } from '@/shared/i18n'

type PriceTableLayoutProps =
  | { flexHeight: true }
  | { flexHeight: false; maxHeight: string }

type PriceRowStatus = 'missing' | 'litellm' | 'manual'
type PriceStatusFilter = 'cpa' | 'missing' | 'litellm' | 'manual' | 'library'
type BillingUnit = 'token' | 'request'
type PriceFieldName = keyof Pick<
  ModelPrice,
  | 'input_usd_per_million'
  | 'output_usd_per_million'
  | 'cache_read_usd_per_million'
  | 'cache_creation_usd_per_million'
>

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

const PRICE_TABLE_FALLBACK_MAX_HEIGHT = 'max(240px, calc(100dvh - 360px))'
const priceModalStyle: CSSProperties = { width: 'min(640px, calc(100vw - 32px))' }
const proxyModalStyle: CSSProperties = { width: 'min(460px, calc(100vw - 32px))' }
const proxyModalContentStyle: CSSProperties = { padding: '16px 22px 4px' }
const proxyModalFooterStyle: CSSProperties = { padding: '12px 22px 18px' }
const desktopPriceLayoutQuery = window.matchMedia('(min-width: 861px)')
const message = useMessage()
const dialog = useDialog()
const { errorText, serverText, t } = useI18n()
const isLoading = ref(false)
const isSyncing = ref(false)
const isPriceSaving = ref(false)
const modalOpen = ref(false)
const proxyModalOpen = ref(false)
const isProxyLoading = ref(false)
const isProxySaving = ref(false)
const editingId = ref<number | null>(null)
const prices = ref<ModelPrice[]>([])
const catalog = ref<ModelPriceCatalogResponse | null>(null)
const selectedProvider = ref<string | null>(null)
const selectedStatus = ref<PriceStatusFilter | null>(null)
const searchQuery = ref('')
const isDesktopPriceLayout = ref(desktopPriceLayoutQuery.matches)
const pagination = reactive({
  page: 1,
  pageSize: 20,
  onUpdatePage: updatePricePage,
})
const form = reactive<ModelPricePayload>({
  provider: '',
  model: '',
  input_usd_per_million: 0,
  output_usd_per_million: 0,
  cache_read_usd_per_million: 0,
  cache_creation_usd_per_million: 0,
  request_usd: null,
  fast_multiplier: 1,
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

watch([selectedProvider, selectedStatus, searchQuery], () => {
  pagination.page = 1
})

function renderSearchIcon() {
  return h(AppIcon, { component: Search })
}

function updatePricePage(page: number) {
  pagination.page = page
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
const priceTableLayoutProps = computed<PriceTableLayoutProps>(() =>
  isDesktopPriceLayout.value
    ? { flexHeight: true }
    : { flexHeight: false, maxHeight: PRICE_TABLE_FALLBACK_MAX_HEIGHT },
)
const isRequestPriceForm = computed(() => billingUnitForModel(form.model) === 'request')
const priceSaveHint = computed(() =>
  isRequestPriceForm.value
    ? t(
        'image 模型按每次成功调用固定金额计费；仅修改 FAST 倍率不会取消 LiteLLM 同步。',
        'Image models are charged a fixed amount per successful call. Changing only the FAST multiplier keeps LiteLLM sync enabled.',
      )
    : t(
        '基础价格修改后会转为手动价格；仅修改 FAST 倍率不会取消 LiteLLM 同步。',
        'Changing base prices makes them manual. Changing only the FAST multiplier keeps LiteLLM sync enabled.',
      ),
)

interface PriceMetricCard {
  key: string
  label: string
  value: string
  footnote: string
  tone: 'primary' | 'blue' | 'purple' | 'orange'
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
    tone: 'primary',
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
    tone: 'blue',
    icon: Server,
  },
  {
    key: 'synced',
    label: t('LiteLLM 同步', 'LiteLLM sync'),
    value: formatInteger(syncedPriceCount.value),
    footnote: t('自动维护', 'Auto maintained'),
    tone: 'purple',
    icon: RefreshCw,
  },
  {
    key: 'manual',
    label: t('手动价格', 'Manual prices'),
    value: formatInteger(manualPriceCount.value),
    footnote: t('优先保留', 'Preserved first'),
    tone: 'orange',
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
}

async function refresh() {
  isLoading.value = true
  try {
    const [nextPrices, nextCatalog] = await Promise.all([listModelPrices(), listModelPriceCatalog()])
    prices.value = nextPrices
    catalog.value = nextCatalog
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
  modalOpen.value = true
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

function handleDesktopPriceLayoutChange(event: MediaQueryListEvent) {
  isDesktopPriceLayout.value = event.matches
}

function rowKey(row: PriceDisplayRow) {
  return row.key
}

function formatPriceValue(value: number | null | undefined) {
  return typeof value === 'number' ? String(value) : '-'
}

function renderBillingUnitCell(row: PriceDisplayRow) {
  const isRequest = row.billing_unit === 'request'
  return h(
    AppBadge,
    { size: 'small', type: isRequest ? 'success' : 'info', bordered: false },
    { default: () => (isRequest ? t('按次', 'Per call') : t('按 Token', 'Per token')) },
  )
}

function renderTokenPriceValue(row: PriceDisplayRow, field: PriceFieldName) {
  if (row.billing_unit === 'request') {
    return h('span', { class: 'price-muted' }, '-')
  }
  return formatPriceValue(row.price?.[field])
}

function renderRequestPriceValue(row: PriceDisplayRow) {
  if (row.billing_unit !== 'request') {
    return h('span', { class: 'price-muted' }, '-')
  }
  if (row.price?.request_usd === null || row.price?.request_usd === undefined) {
    return h('span', { class: 'price-muted' }, t('未定价', 'Unpriced'))
  }
  return formatPriceValue(row.price.request_usd)
}

function renderFastMultiplier(row: PriceDisplayRow) {
  return row.price ? `×${row.price.fast_multiplier}` : '-'
}

function renderModelCell(row: PriceDisplayRow) {
  return h('div', { class: 'model-cell' }, [
    h('div', { class: 'model-title-row' }, [
      h('span', { class: 'model-name' }, row.id),
      row.in_cpa
        ? h(
            AppBadge,
            {
              class: 'model-availability-tag',
              size: 'small',
              type: 'success',
              bordered: false,
              style: { marginLeft: '16px' },
            },
            { default: () => t('CPA 可用模型', 'CPA available model') },
          )
        : null,
    ]),
    row.name && row.name !== row.id ? h('div', { class: 'model-sub' }, row.name) : null,
  ])
}

function renderProviderCell(row: PriceDisplayRow) {
  return h('div', { class: 'provider-cell' }, [
    h('div', { class: 'provider-main' }, row.provider || '-'),
    row.owner && row.owner !== row.provider
      ? h('div', { class: 'model-sub' }, t('所有者', 'Owner') + `: ${row.owner}`)
      : null,
  ])
}

function renderStatusCell(row: PriceDisplayRow) {
  const label = row.status === 'missing' ? t('未定价', 'Unpriced') : row.status === 'litellm' ? 'LiteLLM' : t('手动', 'Manual')
  const type = row.status === 'missing' ? 'warning' : row.status === 'litellm' ? 'info' : 'default'
  return h(
    AppBadge,
    { size: 'small', type, bordered: false },
    { default: () => label },
  )
}

const columns = computed<DataTableColumns<PriceDisplayRow>>(() => [
  {
    title: t('模型', 'Model'),
    key: 'id',
    width: 300,
    ellipsis: { tooltip: true },
    render: renderModelCell,
  },
  {
    title: t('服务商', 'Provider'),
    key: 'provider',
    width: 130,
    ellipsis: { tooltip: true },
    render: renderProviderCell,
  },
  {
    title: t('定价', 'Pricing'),
    key: 'status',
    width: 90,
    render: renderStatusCell,
  },
  {
    title: t('计费方式', 'Billing'),
    key: 'billing_unit',
    width: 100,
    render: renderBillingUnitCell,
  },
  {
    title: t('FAST 倍率', 'FAST multiplier'),
    key: 'fast_multiplier',
    width: 100,
    render: renderFastMultiplier,
  },
  {
    title: t('每次 ($)', 'Per call ($)'),
    key: 'request_usd',
    width: 100,
    render: renderRequestPriceValue,
  },
  {
    title: t('输入 ($/MTok)', 'Input ($/MTok)'),
    key: 'input_usd_per_million',
    width: 110,
    render: (row) => renderTokenPriceValue(row, 'input_usd_per_million'),
  },
  {
    title: t('输出 ($/MTok)', 'Output ($/MTok)'),
    key: 'output_usd_per_million',
    width: 110,
    render: (row) => renderTokenPriceValue(row, 'output_usd_per_million'),
  },
  {
    title: t('缓存读 ($/MTok)', 'Cache read ($/MTok)'),
    key: 'cache_read_usd_per_million',
    width: 120,
    render: (row) => renderTokenPriceValue(row, 'cache_read_usd_per_million'),
  },
  {
    title: t('缓存写 ($/MTok)', 'Cache write ($/MTok)'),
    key: 'cache_creation_usd_per_million',
    width: 120,
    render: (row) => renderTokenPriceValue(row, 'cache_creation_usd_per_million'),
  },
  {
    title: t('更新', 'Updated'),
    key: 'updated_at',
    width: 120,
    render: (row) => (row.price ? formatDateTime(row.price.updated_at) : '-'),
  },
  {
    title: '',
    key: 'actions',
    width: 56,
    align: 'right',
    fixed: 'right',
    render: (row) =>
      h(
        DropdownMenu,
        {},
        {
          default: () => [
            h(
              DropdownMenuTrigger,
              { asChild: true },
              {
                default: () =>
                  h(
                    Button,
                    {
                      variant: 'ghost',
                      size: 'icon-sm',
                      class: 'price-actions-trigger',
                      'aria-label': t(`打开 ${row.id} 的操作菜单`, `Open actions for ${row.id}`),
                    },
                    { default: () => h(MoreHorizontal) },
                  ),
              },
            ),
            h(
              DropdownMenuContent,
              { align: 'end', sideOffset: 4, class: 'w-40' },
              {
                default: () => [
                  h(
                    DropdownMenuGroup,
                    {},
                    {
                      default: () =>
                        h(
                          DropdownMenuItem,
                          { onSelect: () => (row.price ? openEdit(row.price) : openCreateForRow(row)) },
                          {
                            default: () => [
                              h(row.price ? Pencil : Plus),
                              h('span', row.price ? t('改价', 'Edit price') : t('设价', 'Set price')),
                            ],
                          },
                        ),
                    },
                  ),
                  row.price ? h(DropdownMenuSeparator) : null,
                  row.price
                    ? h(
                        DropdownMenuItem,
                        { variant: 'destructive', onSelect: () => confirmDelete(row.price as ModelPrice) },
                        { default: () => [h(Trash2), h('span', t('删除', 'Delete'))] },
                      )
                    : null,
                ],
              },
            ),
          ],
        },
      ),
  },
])

onMounted(() => {
  desktopPriceLayoutQuery.addEventListener('change', handleDesktopPriceLayoutChange)
  void refresh()
})

onBeforeUnmount(() => {
  desktopPriceLayoutQuery.removeEventListener('change', handleDesktopPriceLayoutChange)
})
</script>

<template>
  <section class="page price-page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('模型价格', 'Model prices') }}</h1>
      <AppStack>
        <AppButton secondary :loading="isSyncing" @click="syncPrices">
          <template #icon>
            <AppIcon :component="RefreshCw" />
          </template>
          {{ t('同步 LiteLLM', 'Sync LiteLLM') }}
        </AppButton>
        <AppButton secondary :disabled="isSyncing" @click="openProxySettings">
          <template #icon>
            <AppIcon :component="Settings2" />
          </template>
          {{ t('代理配置', 'Proxy settings') }}
        </AppButton>
        <AppButton type="primary" @click="() => openCreate()">{{ t('新增价格', 'Add price') }}</AppButton>
      </AppStack>
    </div>

    <div class="metric-grid price-metrics">
      <div v-for="metric in priceMetrics" :key="metric.key" class="metric-card" :class="`is-${metric.tone}`">
        <div class="metric-icon" aria-hidden="true">
          <component :is="metric.icon" :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ metric.label }}</div>
        <div class="metric-value">{{ metric.value }}</div>
        <div class="metric-footnote">{{ metric.footnote }}</div>
      </div>
    </div>

    <section class="panel table-panel price-table-panel">
      <div class="price-table-top">
        <AppAlert v-if="catalogNotice" class="price-alert" type="warning" :show-icon="false">
          {{ catalogNotice }}
        </AppAlert>
        <div class="table-toolbar">
          <AppStack class="price-toolbar-layout" justify="space-between" align="center">
            <AppStack class="price-filters" align="center" :size="8">
              <span class="filter-label">{{ t('服务商', 'Provider') }}</span>
              <AppSelect
                v-model:value="selectedProvider"
                class="provider-filter"
                :icon="Server"
                :options="providerOptions"
                clearable
                filterable
                :placeholder="t('全部服务商', 'All providers')"
              />
              <AppSelect
                v-model:value="selectedStatus"
                class="status-filter"
                :icon="ListFilter"
                :options="statusOptions"
                clearable
                :placeholder="t('全部状态', 'All statuses')"
              />
              <AppInput
                v-model:value="searchQuery"
                class="price-search"
                clearable
                :placeholder="t('搜索模型或服务商', 'Search models or providers')"
                :render-prefix="renderSearchIcon"
              />
            </AppStack>
            <span class="result-count">
              {{ t(`共 ${filteredPriceCount} / ${totalPriceCount} 条`, `${filteredPriceCount} / ${totalPriceCount} items`) }}
            </span>
          </AppStack>
        </div>
      </div>
      <AppDataTable
        class="price-table"
        v-bind="priceTableLayoutProps"
        size="small"
        :loading="isLoading"
        :columns="columns"
        :data="filteredPrices"
        :pagination="pagination"
        :row-key="rowKey"
        :scroll-x="1456"
        table-layout="fixed"
      />
    </section>

    <AppModal
      v-model:show="modalOpen"
      preset="card"
      :title="editingId === null ? t('新增价格', 'Add price') : t('编辑价格', 'Edit price')"
      :style="priceModalStyle"
      class="price-modal"
    >
      <AppForm :model="form" label-placement="top">
        <div class="form-grid">
          <AppFormItem :label="t('服务商', 'Provider')">
            <AppInput v-model:value="form.provider" />
          </AppFormItem>
          <AppFormItem :label="t('模型', 'Model')">
            <AppInput v-model:value="form.model" />
          </AppFormItem>
          <AppFormItem :label="t('FAST 倍率', 'FAST multiplier')" class="wide-form-item">
            <AppNumberInput v-model:value="form.fast_multiplier" :min="0.01" :step="0.1" />
          </AppFormItem>
          <AppFormItem v-if="isRequestPriceForm" :label="t('每次调用价格 USD', 'Per-call price USD')" class="wide-form-item">
            <AppNumberInput v-model:value="form.request_usd" :min="0" :placeholder="t('例如：0.04', 'Example: 0.04')" />
          </AppFormItem>
          <template v-else>
            <AppFormItem :label="t('输入价格', 'Input price')">
              <AppNumberInput v-model:value="form.input_usd_per_million" :min="0" />
            </AppFormItem>
            <AppFormItem :label="t('输出价格', 'Output price')">
              <AppNumberInput v-model:value="form.output_usd_per_million" :min="0" />
            </AppFormItem>
            <AppFormItem :label="t('缓存读价格', 'Cache read price')">
              <AppNumberInput v-model:value="form.cache_read_usd_per_million" :min="0" />
            </AppFormItem>
            <AppFormItem :label="t('缓存写价格', 'Cache write price')">
              <AppNumberInput v-model:value="form.cache_creation_usd_per_million" :min="0" />
            </AppFormItem>
          </template>
        </div>
      </AppForm>
      <p class="price-save-hint">{{ priceSaveHint }}</p>
      <template #footer>
        <AppStack justify="end">
          <AppButton secondary :disabled="isPriceSaving" @click="modalOpen = false">{{ t('取消', 'Cancel') }}</AppButton>
          <AppButton type="primary" :loading="isPriceSaving" @click="savePrice">{{ t('保存', 'Save') }}</AppButton>
        </AppStack>
      </template>
    </AppModal>

    <AppModal
      v-model:show="proxyModalOpen"
      preset="card"
      :title="t('LiteLLM 代理配置', 'LiteLLM proxy settings')"
      :style="proxyModalStyle"
      :content-style="proxyModalContentStyle"
      :footer-style="proxyModalFooterStyle"
      class="proxy-modal"
    >
      <AppForm :model="proxyForm" label-placement="top">
        <div class="proxy-form">
          <p class="proxy-hint">{{ liteLLMProxyHint }}</p>
          <div class="proxy-switch-row">
            <span class="proxy-switch-label">{{ t('使用代理', 'Use proxy') }}</span>
            <AppSwitch
              v-model:value="proxyForm.enabled"
              :disabled="isProxyLoading || isProxySaving"
              :aria-label="t('使用代理', 'Use proxy')"
            />
          </div>
          <AppFormItem :label="t('代理地址', 'Proxy URL')">
            <AppInput
              v-model:value="proxyForm.proxy_url"
              :disabled="!proxyForm.enabled || isProxyLoading || isProxySaving"
              :placeholder="t('http://127.0.0.1:7890 或 socks5://127.0.0.1:1080', 'http://127.0.0.1:7890 or socks5://127.0.0.1:1080')"
            />
          </AppFormItem>
        </div>
      </AppForm>
      <template #footer>
        <AppStack justify="end">
          <AppButton :disabled="isProxySaving" @click="proxyModalOpen = false">{{ t('取消', 'Cancel') }}</AppButton>
          <AppButton type="primary" :loading="isProxySaving" @click="saveProxySettings">{{ t('保存', 'Save') }}</AppButton>
        </AppStack>
      </template>
    </AppModal>
  </section>
</template>

<style scoped>
.price-modal {
  width: min(640px, calc(100vw - 24px));
}

.proxy-modal {
  width: min(520px, calc(100vw - 24px));
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 12px;
}

.wide-form-item {
  grid-column: 1 / -1;
}

.proxy-form {
  display: grid;
  gap: 14px;
}

.proxy-hint {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--cpa-primary) 22%, var(--cpa-border));
  border-radius: var(--cpa-radius);
  background: color-mix(in srgb, var(--cpa-primary-wash) 74%, var(--cpa-surface));
  color: var(--cpa-text-muted);
  font-size: 13px;
  line-height: 1.55;
}

.proxy-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 34px;
  padding: 8px 10px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface-raised);
}

.proxy-switch-label {
  color: var(--cpa-text);
  font-size: 14px;
  font-weight: 600;
}

.proxy-form :deep(.n-form-item) {
  margin-bottom: 0;
}

.price-metrics {
  grid-template-columns: repeat(4, minmax(150px, 1fr));
}

.price-alert {
  border-radius: var(--cpa-radius);
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
  padding: 14px 16px;
  border: 1px solid var(--cpa-border);
  border-bottom: 0;
  border-radius: var(--cpa-radius) var(--cpa-radius) 0 0;
  background: var(--cpa-surface-raised);
  box-shadow: var(--cpa-shadow-hairline);
}

.price-toolbar-layout {
  width: 100%;
  min-width: 0;
}

.price-table :deep(.n-data-table-wrapper) {
  border-radius: 0 0 var(--cpa-radius) var(--cpa-radius);
}

.filter-label,
.result-count {
  color: var(--cpa-text-muted);
  font-size: 13px;
  white-space: nowrap;
}

.provider-filter {
  width: 220px;
}

.status-filter {
  width: 150px;
}

.price-filters {
  min-width: 0;
  max-width: 100%;
  flex: 1 1 auto;
}

.price-search {
  width: 280px;
}

:global(.price-actions-trigger) {
  margin-left: auto;
  color: var(--cpa-text-muted);
}

.model-cell,
.provider-cell {
  min-width: 0;
}

.model-title-row {
  display: flex;
  align-items: center;
  gap: 0;
  min-width: 0;
}

.model-availability-tag {
  flex: 0 0 auto;
  margin-left: 2px;
}

.model-name,
.provider-main {
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-text);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-sub {
  margin-top: 2px;
  overflow: hidden;
  color: var(--cpa-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.price-muted {
  color: var(--cpa-text-muted);
}

.price-save-hint {
  margin: 4px 0 0;
  color: var(--cpa-text-muted);
  font-size: 13px;
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
    grid-template-rows: auto minmax(0, 1fr);
    min-height: 0;
  }

  .price-table {
    height: 100%;
    min-height: 0;
  }

  .price-table :deep(.n-data-table-wrapper),
  .price-table :deep(.n-data-table-base-table),
  .price-table :deep(.n-data-table-base-table-body) {
    min-height: 0;
  }
}

@media (max-width: 980px) {
  .table-toolbar {
    padding: 12px;
  }

  .provider-filter {
    width: min(200px, calc(100vw - 32px));
  }

  .status-filter {
    width: min(160px, calc(100vw - 32px));
  }

  .price-search {
    width: min(240px, calc(100vw - 32px));
  }
}

@media (max-width: 620px) {
  .price-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .price-toolbar-layout {
    display: grid !important;
    gap: 8px !important;
  }

  .price-filters {
    display: grid !important;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px !important;
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

  .result-count {
    justify-self: start;
  }
}
</style>
