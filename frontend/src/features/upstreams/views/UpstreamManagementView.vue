<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import {
  AppAlert,
  AppButton,
  AppCard,
  AppDataTable,
  AppDescriptions,
  AppDescriptionsItem,
  AppDrawer,
  AppDrawerContent,
  AppForm,
  AppFormItem,
  AppIcon,
  AppInput,
  AppNumberInput,
  AppSelect,
  AppStack,
  AppSpinner,
  AppSwitch,
  AppBadge,
  useDialog,
  useMessage,
  type DataTableColumns,
} from '@/shared/ui/app-kit'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from '@/components/ui/input-group'
import { Activity, Clock3, Eye, Network, Pencil, Plus, RefreshCw, Search, ServerCog, Trash2, X } from '@lucide/vue'

import {
  listUpstreamSections,
  replaceUpstreamSection,
  upstreamSectionNames,
  type UpstreamItem,
  type UpstreamSection,
} from '@/features/upstreams/api/upstreamApi'
import { useI18n } from '@/shared/i18n'

interface SectionDefinition {
  name: UpstreamSection
  label: string
  description: string
}

interface UpstreamTableRow {
  key: string
  index: number
  item: UpstreamItem
}

interface HeaderFormRow {
  key: string
  value: string
}

interface ModelFormRow {
  name: string
  alias: string
  displayName: string
  raw?: UpstreamItem
}

interface APIKeyFormRow {
  apiKey: string
  proxyUrl: string
  raw?: UpstreamItem
}

interface UpstreamForm {
  name: string
  apiKey: string
  baseUrl: string
  proxyUrl: string
  prefix: string
  priority: number | null
  disabled: boolean
  websockets: boolean
  disableCooling: boolean
  headers: HeaderFormRow[]
  models: ModelFormRow[]
  excludedModels: string
  apiKeyEntries: APIKeyFormRow[]
  cloakMode: string | null
  cloakStrict: boolean
  cloakCacheUserId: boolean
  cloakSensitiveWords: string
  experimentalCCHSigning: boolean
}

type DrawerMode = 'detail' | 'edit' | 'create'

const message = useMessage()
const dialog = useDialog()
const { errorText, t } = useI18n()
const isLoading = ref(false)
const isSaving = ref(false)
const mutatingKey = ref('')
const loadError = ref('')
const updatedAt = ref<Date | null>(null)
const activeSection = ref<UpstreamSection>('gemini-api-key')
const searchText = ref('')
const drawerOpen = ref(false)
const drawerMode = ref<DrawerMode>('detail')
const editingIndex = ref(-1)
const selectedItem = ref<UpstreamItem | null>(null)
const sections = reactive<Record<UpstreamSection, UpstreamItem[]>>({
  'gemini-api-key': [],
  'codex-api-key': [],
  'xai-api-key': [],
  'claude-api-key': [],
  'vertex-api-key': [],
  'openai-compatibility': [],
})
const form = reactive<UpstreamForm>(emptyForm())

const sectionDefinitions = computed<SectionDefinition[]>(() => [
  { name: 'gemini-api-key', label: 'Gemini', description: t('Gemini API 密钥', 'Gemini API keys') },
  { name: 'codex-api-key', label: 'Codex', description: t('Codex 上游', 'Codex upstreams') },
  { name: 'xai-api-key', label: 'xAI', description: t('xAI 上游', 'xAI upstreams') },
  { name: 'claude-api-key', label: 'Claude', description: t('Claude API 密钥', 'Claude API keys') },
  { name: 'vertex-api-key', label: 'Vertex', description: t('Vertex API 密钥', 'Vertex API keys') },
  {
    name: 'openai-compatibility',
    label: t('OpenAI 兼容', 'OpenAI Compatible'),
    description: t('OpenAI 兼容提供商', 'OpenAI-compatible providers'),
  },
])

const activeDefinition = computed(
  () => sectionDefinitions.value.find((definition) => definition.name === activeSection.value) ?? sectionDefinitions.value[0],
)
const isOpenAISection = computed(() => activeSection.value === 'openai-compatibility')
const isClaudeSection = computed(() => activeSection.value === 'claude-api-key')
const currentItems = computed(() => sections[activeSection.value])
const totalResources = computed(() => upstreamSectionNames.reduce((total, name) => total + sections[name].length, 0))
const activeResources = computed(() =>
  upstreamSectionNames.reduce(
    (total, name) => total + sections[name].filter((item) => !upstreamDisabled(name, item)).length,
    0,
  ),
)
const configuredFamilies = computed(() => upstreamSectionNames.filter((name) => sections[name].length > 0).length)
const normalizedSearch = computed(() => searchText.value.trim().toLowerCase())
const filteredRows = computed<UpstreamTableRow[]>(() =>
  currentItems.value
    .map((item, index) => ({ key: `${activeSection.value}:${index}`, index, item }))
    .filter((row) => upstreamSearchText(row.item).includes(normalizedSearch.value)),
)

const drawerTitle = computed(() => {
  const provider = activeDefinition.value?.label ?? ''
  if (drawerMode.value === 'create') return t(`新建 ${provider} 上游`, `New ${provider} upstream`)
  if (drawerMode.value === 'edit') return t(`编辑 ${provider} 上游`, `Edit ${provider} upstream`)
  return t(`${provider} 上游详情`, `${provider} upstream details`)
})

const tableColumns = computed<DataTableColumns<UpstreamTableRow>>(() => [
  {
    title: isOpenAISection.value ? t('提供商', 'Provider') : t('密钥', 'Key'),
    key: 'identity',
    width: 160,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('div', { class: 'identity-cell' }, [
        h('strong', upstreamTitle(activeSection.value, row.item, row.index)),
        h('code', maskSecret(primaryAPIKey(activeSection.value, row.item))),
      ]),
  },
  {
    title: t('服务地址', 'Base URL'),
    key: 'base_url',
    width: 180,
    ellipsis: { tooltip: true },
    render: (row) => h('code', { class: 'url-cell' }, readString(row.item, 'base-url') || '-'),
  },
  {
    title: t('前缀', 'Prefix'),
    key: 'prefix',
    width: 80,
    ellipsis: { tooltip: true },
    render: (row) => readString(row.item, 'prefix') || '-',
  },
  {
    title: t('模型 / 请求头', 'Models / Headers'),
    key: 'metrics',
    width: 140,
    render: (row) =>
      h(AppStack, { size: 6, wrap: false }, () => [
        h(AppBadge, { size: 'small', bordered: false }, () => t(`模型 ${modelCount(row.item)}`, `Models ${modelCount(row.item)}`)),
        h(AppBadge, { size: 'small', bordered: false }, () => t(`头 ${headerCount(row.item)}`, `Headers ${headerCount(row.item)}`)),
      ]),
  },
  {
    title: t('优先级', 'Priority'),
    key: 'priority',
    width: 60,
    align: 'center',
    render: (row) => readNumber(row.item, 'priority') ?? '-',
  },
  {
    title: t('状态', 'Status'),
    key: 'status',
    width: 80,
    render: (row) =>
      h(AppSwitch, {
        value: !upstreamDisabled(activeSection.value, row.item),
        loading: mutatingKey.value === row.key,
        disabled: isSaving.value || isLoading.value,
        'onUpdate:value': (enabled: boolean) => void toggleUpstream(row, enabled),
      }),
  },
  {
    title: t('操作', 'Actions'),
    key: 'actions',
    width: 120,
    align: 'right',
    render: (row) =>
      h(AppStack, { size: 2, wrap: false, justify: 'end', class: 'upstream-row-actions' }, () => [
        iconButton(Eye, t('详情', 'Details'), () => openDetail(row)),
        iconButton(Pencil, t('编辑', 'Edit'), () => openEditor(row)),
        iconButton(Trash2, t('删除', 'Delete'), () => confirmDelete(row), true),
      ]),
  },
])

function emptyForm(): UpstreamForm {
  return {
    name: '',
    apiKey: '',
    baseUrl: '',
    proxyUrl: '',
    prefix: '',
    priority: null,
    disabled: false,
    websockets: false,
    disableCooling: false,
    headers: [],
    models: [],
    excludedModels: '',
    apiKeyEntries: [{ apiKey: '', proxyUrl: '' }],
    cloakMode: null,
    cloakStrict: false,
    cloakCacheUserId: false,
    cloakSensitiveWords: '',
    experimentalCCHSigning: false,
  }
}

function replaceForm(next: UpstreamForm) {
  Object.assign(form, next)
}

function readString(item: UpstreamItem, key: string): string {
  const value = item[key]
  return typeof value === 'string' ? value.trim() : ''
}

function readNumber(item: UpstreamItem, key: string): number | null {
  const value = item[key]
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return null
}

function readObject(value: unknown): UpstreamItem | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value) ? (value as UpstreamItem) : null
}

function readObjectArray(item: UpstreamItem, key: string): UpstreamItem[] {
  const value = item[key]
  return Array.isArray(value) ? value.map(readObject).filter((entry): entry is UpstreamItem => entry !== null) : []
}

function stringArray(item: UpstreamItem, key: string): string[] {
  const value = item[key]
  return Array.isArray(value) ? value.map((entry) => String(entry).trim()).filter(Boolean) : []
}

function primaryAPIKey(section: UpstreamSection, item: UpstreamItem): string {
  if (section === 'openai-compatibility') {
    return readString(readObjectArray(item, 'api-key-entries')[0] ?? {}, 'api-key')
  }
  return readString(item, 'api-key')
}

function maskSecret(secret: string): string {
  if (!secret) return '-'
  if (secret.length <= 8) return '********'
  return `${secret.slice(0, 4)}********${secret.slice(-4)}`
}

function upstreamTitle(section: UpstreamSection, item: UpstreamItem, index: number): string {
  if (section === 'openai-compatibility') return readString(item, 'name') || t(`提供商 #${index + 1}`, `Provider #${index + 1}`)
  return readString(item, 'prefix') || t(`配置 #${index + 1}`, `Config #${index + 1}`)
}

function upstreamDisabled(section: UpstreamSection, item: UpstreamItem): boolean {
  if (section === 'openai-compatibility') return item.disabled === true
  return stringArray(item, 'excluded-models').includes('*')
}

function modelCount(item: UpstreamItem): number {
  return readObjectArray(item, 'models').length
}

function headerCount(item: UpstreamItem): number {
  return Object.keys(readObject(item.headers) ?? {}).length
}

function upstreamSearchText(item: UpstreamItem): string {
  const entries = readObjectArray(item, 'api-key-entries')
  return [
    readString(item, 'name'),
    readString(item, 'api-key'),
    readString(item, 'base-url'),
    readString(item, 'proxy-url'),
    readString(item, 'prefix'),
    ...entries.flatMap((entry) => [readString(entry, 'api-key'), readString(entry, 'proxy-url')]),
  ]
    .join('\n')
    .toLowerCase()
}

function iconButton(icon: typeof Eye, label: string, onClick: () => void, danger = false) {
  return h(
    AppButton,
    { text: true, type: danger ? 'error' : 'default', title: label, 'aria-label': label, onClick },
    { icon: () => h(AppIcon, { component: icon, size: 17 }) },
  )
}

function selectSection(name: UpstreamSection) {
  activeSection.value = name
  searchText.value = ''
}

async function loadUpstreams(showSuccess = false) {
  isLoading.value = true
  loadError.value = ''
  try {
    const next = await listUpstreamSections()
    for (const name of upstreamSectionNames) sections[name] = next[name]
    updatedAt.value = new Date()
    if (showSuccess) message.success(t('上游配置已刷新', 'Upstream configurations refreshed'))
  } catch (error) {
    loadError.value = errorText(error, '加载上游配置失败', 'Failed to load upstream configurations')
  } finally {
    isLoading.value = false
  }
}

function formFromItem(item: UpstreamItem): UpstreamForm {
  const cloak = readObject(item.cloak) ?? {}
  return {
    name: readString(item, 'name'),
    apiKey: readString(item, 'api-key'),
    baseUrl: readString(item, 'base-url'),
    proxyUrl: readString(item, 'proxy-url'),
    prefix: readString(item, 'prefix'),
    priority: readNumber(item, 'priority'),
    disabled: upstreamDisabled(activeSection.value, item),
    websockets: item.websockets === true,
    disableCooling: item['disable-cooling'] === true,
    headers: Object.entries(readObject(item.headers) ?? {}).map(([key, value]) => ({ key, value: String(value) })),
    models: readObjectArray(item, 'models').map((model) => ({
      name: readString(model, 'name'),
      alias: readString(model, 'alias'),
      displayName: readString(model, 'display-name'),
      raw: model,
    })),
    excludedModels: stringArray(item, 'excluded-models').filter((model) => model !== '*').join('\n'),
    apiKeyEntries: readObjectArray(item, 'api-key-entries').map((entry) => ({
      apiKey: readString(entry, 'api-key'),
      proxyUrl: readString(entry, 'proxy-url'),
      raw: entry,
    })),
    cloakMode: readString(cloak, 'mode') || null,
    cloakStrict: cloak['strict-mode'] === true,
    cloakCacheUserId: cloak['cache-user-id'] === true,
    cloakSensitiveWords: stringArray(cloak, 'sensitive-words').join('\n'),
    experimentalCCHSigning: item['experimental-cch-signing'] === true,
  }
}

function openDetail(row: UpstreamTableRow) {
  selectedItem.value = row.item
  editingIndex.value = row.index
  drawerMode.value = 'detail'
  drawerOpen.value = true
}

function openEditor(row?: UpstreamTableRow) {
  editingIndex.value = row?.index ?? -1
  selectedItem.value = row?.item ?? null
  replaceForm(row ? formFromItem(row.item) : emptyForm())
  if (isOpenAISection.value && form.apiKeyEntries.length === 0) addAPIKeyEntry()
  drawerMode.value = row ? 'edit' : 'create'
  drawerOpen.value = true
}

function addHeader() {
  form.headers.push({ key: '', value: '' })
}

function addModel() {
  form.models.push({ name: '', alias: '', displayName: '' })
}

function addAPIKeyEntry() {
  form.apiKeyEntries.push({ apiKey: '', proxyUrl: '' })
}

function setOptionalString(target: UpstreamItem, key: string, value: string) {
  const normalized = value.trim()
  if (normalized) target[key] = normalized
  else delete target[key]
}

function formHeaders(): Record<string, string> | undefined {
  const result: Record<string, string> = {}
  for (const row of form.headers) {
    const key = row.key.trim()
    if (key) result[key] = row.value.trim()
  }
  return Object.keys(result).length > 0 ? result : undefined
}

function formModels(): UpstreamItem[] | undefined {
  const result = form.models
    .map((row) => {
      const name = row.name.trim()
      if (!name) return null
      const model: UpstreamItem = { ...(row.raw ?? {}), name }
      setOptionalString(model, 'alias', row.alias)
      setOptionalString(model, 'display-name', row.displayName)
      return model
    })
    .filter((model): model is UpstreamItem => model !== null)
  return result.length > 0 ? result : undefined
}

function lineValues(value: string): string[] {
  return [...new Set(value.split(/\r?\n/).map((entry) => entry.trim()).filter(Boolean))]
}

function buildPayload(): UpstreamItem | null {
  const existing = selectedItem.value ? { ...selectedItem.value } : {}
  const models = formModels()
  const headers = formHeaders()
  if (models) existing.models = models
  else delete existing.models
  if (headers) existing.headers = headers
  else delete existing.headers
  setOptionalString(existing, 'base-url', form.baseUrl)
  setOptionalString(existing, 'prefix', form.prefix)
  if (form.priority === null) delete existing.priority
  else existing.priority = Math.trunc(form.priority)
  if (form.disableCooling) existing['disable-cooling'] = true
  else delete existing['disable-cooling']

  if (isOpenAISection.value) {
    const name = form.name.trim()
    const baseUrl = form.baseUrl.trim()
    const entries = form.apiKeyEntries
      .map((row) => {
        const apiKey = row.apiKey.trim()
        if (!apiKey) return null
        const entry: UpstreamItem = { ...(row.raw ?? {}), 'api-key': apiKey }
        setOptionalString(entry, 'proxy-url', row.proxyUrl)
        return entry
      })
      .filter((entry): entry is UpstreamItem => entry !== null)
    if (!name || !baseUrl || entries.length === 0) {
      message.error(t('请填写提供商名称、服务地址和至少一个 API 密钥', 'Enter a provider name, base URL, and at least one API key'))
      return null
    }
    existing.name = name
    existing['base-url'] = baseUrl
    existing['api-key-entries'] = entries
    existing.disabled = form.disabled
    return existing
  }

  const apiKey = form.apiKey.trim()
  if (!apiKey) {
    message.error(t('请填写 API 密钥', 'Enter an API key'))
    return null
  }
  if ((activeSection.value === 'codex-api-key' || activeSection.value === 'xai-api-key') && !form.baseUrl.trim()) {
    message.error(t('请填写服务地址', 'Enter a base URL'))
    return null
  }
  existing['api-key'] = apiKey
  setOptionalString(existing, 'proxy-url', form.proxyUrl)
  if (form.websockets) existing.websockets = true
  else delete existing.websockets
  const excluded = lineValues(form.excludedModels).filter((model) => model !== '*')
  if (form.disabled) excluded.unshift('*')
  if (excluded.length > 0) existing['excluded-models'] = excluded
  else delete existing['excluded-models']

  if (isClaudeSection.value) {
    const cloak: UpstreamItem = { ...(readObject(existing.cloak) ?? {}) }
    setOptionalString(cloak, 'mode', form.cloakMode ?? '')
    if (form.cloakStrict) cloak['strict-mode'] = true
    else delete cloak['strict-mode']
    if (form.cloakCacheUserId) cloak['cache-user-id'] = true
    else delete cloak['cache-user-id']
    const sensitiveWords = lineValues(form.cloakSensitiveWords)
    if (sensitiveWords.length > 0) cloak['sensitive-words'] = sensitiveWords
    else delete cloak['sensitive-words']
    if (Object.keys(cloak).length > 0) existing.cloak = cloak
    else delete existing.cloak
    if (form.experimentalCCHSigning) existing['experimental-cch-signing'] = true
    else delete existing['experimental-cch-signing']
  }
  return existing
}

async function saveUpstream() {
  const payload = buildPayload()
  if (!payload) return
  isSaving.value = true
  try {
    const current = [...sections[activeSection.value]]
    if (editingIndex.value >= 0) current.splice(editingIndex.value, 1, payload)
    else current.push(payload)
    await replaceUpstreamSection(activeSection.value, current)
    sections[activeSection.value] = current
    drawerOpen.value = false
    updatedAt.value = new Date()
    message.success(editingIndex.value >= 0 ? t('上游配置已更新', 'Upstream updated') : t('上游配置已创建', 'Upstream created'))
  } catch (error) {
    message.error(errorText(error, '保存上游配置失败', 'Failed to save upstream'))
  } finally {
    isSaving.value = false
  }
}

async function toggleUpstream(row: UpstreamTableRow, enabled: boolean) {
  mutatingKey.value = row.key
  try {
    const next = [...sections[activeSection.value]]
    const item = { ...row.item }
    if (activeSection.value === 'openai-compatibility') {
      item.disabled = !enabled
    } else {
      const excluded = stringArray(item, 'excluded-models').filter((model) => model !== '*')
      if (!enabled) excluded.unshift('*')
      if (excluded.length > 0) item['excluded-models'] = excluded
      else delete item['excluded-models']
    }
    next.splice(row.index, 1, item)
    await replaceUpstreamSection(activeSection.value, next)
    sections[activeSection.value] = next
    message.success(enabled ? t('上游已启用', 'Upstream enabled') : t('上游已停用', 'Upstream disabled'))
  } catch (error) {
    message.error(errorText(error, '更新上游状态失败', 'Failed to update upstream status'))
  } finally {
    mutatingKey.value = ''
  }
}

function confirmDelete(row: UpstreamTableRow) {
  dialog.warning({
    title: t('删除上游', 'Delete upstream'),
    content: t(
      `确定要删除“${upstreamTitle(activeSection.value, row.item, row.index)}”吗？此操作不可撤销。`,
      `Delete “${upstreamTitle(activeSection.value, row.item, row.index)}”? This cannot be undone.`,
    ),
    positiveText: t('删除', 'Delete'),
    negativeText: t('取消', 'Cancel'),
    onPositiveClick: async () => {
      mutatingKey.value = row.key
      try {
        const next = sections[activeSection.value].filter((_, index) => index !== row.index)
        await replaceUpstreamSection(activeSection.value, next)
        sections[activeSection.value] = next
        message.success(t('上游已删除', 'Upstream deleted'))
      } catch (error) {
        message.error(errorText(error, '删除上游失败', 'Failed to delete upstream'))
      } finally {
        mutatingKey.value = ''
      }
    },
  })
}

function formatUpdatedAt(): string {
  return updatedAt.value ? updatedAt.value.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '-'
}

void loadUpstreams()
</script>

<template>
  <section class="page upstream-page">
    <div class="page-toolbar upstream-header">
      <h1 data-page-title class="page-title">{{ t('上游管理', 'Upstream Management') }}</h1>
      <AppButton secondary :loading="isLoading" @click="loadUpstreams(true)">
        <template #icon><AppIcon :component="RefreshCw" /></template>
        {{ t('刷新', 'Refresh') }}
      </AppButton>
    </div>

    <div class="metric-grid">
      <div class="metric-card is-green">
        <div class="metric-icon" aria-hidden="true"><Activity /></div>
        <div class="metric-label">{{ t('活跃资源', 'Active resources') }}</div>
        <div class="metric-value">{{ activeResources }} / {{ totalResources }}</div>
        <div class="metric-footnote">{{ t('启用 / 已配置', 'Enabled / configured') }}</div>
      </div>
      <div class="metric-card is-blue">
        <div class="metric-icon" aria-hidden="true"><Network /></div>
        <div class="metric-label">{{ t('提供商类型', 'Provider families') }}</div>
        <div class="metric-value">{{ configuredFamilies }} / {{ upstreamSectionNames.length }}</div>
        <div class="metric-footnote">{{ t('已配置 / 支持', 'Configured / supported') }}</div>
      </div>
      <div class="metric-card is-purple">
        <div class="metric-icon" aria-hidden="true"><Clock3 /></div>
        <div class="metric-label">{{ t('最近同步', 'Last synced') }}</div>
        <div class="metric-value">{{ formatUpdatedAt() }}</div>
        <div class="metric-footnote">{{ t('CLIProxyAPI 配置', 'CLIProxyAPI configuration') }}</div>
      </div>
    </div>

    <AppAlert v-if="loadError" type="error" :title="t('无法加载上游配置', 'Unable to load upstream configurations')">
      {{ loadError }}
    </AppAlert>

    <AppCard class="upstream-workbench" content-style="padding: 0; min-height: 0;">
      <div class="workbench-layout">
        <aside class="provider-nav">
          <p class="provider-nav__title">{{ t('提供商', 'Providers') }}</p>
          <button
            v-for="definition in sectionDefinitions"
            :key="definition.name"
            type="button"
            class="provider-nav__item"
            :class="{ 'provider-nav__item--active': activeSection === definition.name }"
            @click="selectSection(definition.name)"
          >
            <span class="provider-nav__copy">
              <strong>{{ definition.label }}</strong>
              <small>{{ definition.description }}</small>
            </span>
            <AppBadge size="small" round :bordered="false">
              {{ sections[definition.name].filter((item) => !upstreamDisabled(definition.name, item)).length }}/{{ sections[definition.name].length }}
            </AppBadge>
          </button>
        </aside>

        <div class="provider-panel">
          <div class="provider-panel__header">
            <div class="provider-panel__title">
              <AppIcon :component="ServerCog" :size="22" />
              <div>
                <h2>{{ activeDefinition?.label }}</h2>
                <p>{{ activeDefinition?.description }}</p>
              </div>
            </div>
            <AppBadge size="small" round :bordered="false">
              {{ sections[activeSection].length }}
            </AppBadge>
          </div>

          <div class="provider-panel__toolbar">
            <InputGroup class="provider-search">
              <InputGroupAddon>
                <Search :size="16" aria-hidden="true" />
              </InputGroupAddon>
              <InputGroupInput
                v-model="searchText"
                class="h-full"
                :placeholder="t('搜索密钥、地址或前缀', 'Search keys, URLs, or prefixes')"
              />
              <InputGroupAddon v-if="searchText" align="inline-end">
                <InputGroupButton
                  :aria-label="t('清空搜索', 'Clear search')"
                  :title="t('清空搜索', 'Clear search')"
                  @click="searchText = ''"
                >
                  <X :size="14" aria-hidden="true" />
                </InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
            <span class="provider-result-count">
              {{ t(`显示 ${filteredRows.length} 项`, `${filteredRows.length} shown`) }}
            </span>
            <AppButton class="provider-create-button" type="primary" :disabled="isLoading" @click="openEditor()">
              <template #icon><AppIcon :component="Plus" /></template>
              {{ t('新建', 'New') }}
            </AppButton>
          </div>

          <div class="provider-table-shell">
            <AppSpinner :show="isLoading">
              <AppDataTable
                :columns="tableColumns"
                :data="filteredRows"
                :row-key="(row: UpstreamTableRow) => row.key"
                :pagination="{ pageSize: 50 }"
                :scroll-x="820"
                :bordered="false"
              />
            </AppSpinner>
          </div>
        </div>
      </div>
    </AppCard>

    <AppDrawer v-model:show="drawerOpen" placement="right" :width="drawerMode === 'detail' ? 460 : 720">
      <AppDrawerContent :title="drawerTitle" closable>
        <template v-if="drawerMode === 'detail' && selectedItem">
          <AppDescriptions label-placement="top" :column="1" bordered>
            <AppDescriptionsItem :label="isOpenAISection ? t('提供商', 'Provider') : t('API 密钥', 'API key')">
              {{ isOpenAISection ? readString(selectedItem, 'name') || '-' : maskSecret(primaryAPIKey(activeSection, selectedItem)) }}
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('服务地址', 'Base URL')">
              <code>{{ readString(selectedItem, 'base-url') || '-' }}</code>
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('代理 URL', 'Proxy URL')">
              <code>{{ readString(selectedItem, 'proxy-url') || '-' }}</code>
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('前缀', 'Prefix')">{{ readString(selectedItem, 'prefix') || '-' }}</AppDescriptionsItem>
            <AppDescriptionsItem :label="t('优先级', 'Priority')">{{ readNumber(selectedItem, 'priority') ?? '-' }}</AppDescriptionsItem>
            <AppDescriptionsItem :label="t('模型 / 请求头 / 密钥', 'Models / Headers / Keys')">
              {{ modelCount(selectedItem) }} / {{ headerCount(selectedItem) }} /
              {{ isOpenAISection ? readObjectArray(selectedItem, 'api-key-entries').length : 1 }}
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('状态', 'Status')">
              <AppBadge :type="upstreamDisabled(activeSection, selectedItem) ? 'warning' : 'success'" size="small">
                {{ upstreamDisabled(activeSection, selectedItem) ? t('已停用', 'Disabled') : t('活跃', 'Active') }}
              </AppBadge>
            </AppDescriptionsItem>
          </AppDescriptions>
          <AppButton type="primary" block class="detail-edit-button" @click="openEditor({ key: '', index: editingIndex, item: selectedItem })">
            <template #icon><AppIcon :component="Pencil" /></template>
            {{ t('编辑', 'Edit') }}
          </AppButton>
        </template>

        <AppForm v-else label-placement="top" class="upstream-form">
          <AppFormItem v-if="isOpenAISection" :label="t('名称', 'Name')" required>
            <AppInput v-model:value="form.name" :placeholder="t('例如 OpenRouter', 'e.g. OpenRouter')" />
          </AppFormItem>
          <AppFormItem v-else :label="t('API 密钥', 'API key')" required>
            <AppInput v-model:value="form.apiKey" type="password" show-password-on="click" />
          </AppFormItem>

          <div class="form-grid">
            <AppFormItem :label="t('服务地址', 'Base URL')" :required="isOpenAISection || activeSection === 'codex-api-key' || activeSection === 'xai-api-key'">
              <AppInput v-model:value="form.baseUrl" placeholder="https://api.example.com/v1" />
            </AppFormItem>
            <AppFormItem v-if="!isOpenAISection" :label="t('代理 URL', 'Proxy URL')">
              <AppInput v-model:value="form.proxyUrl" placeholder="socks5://127.0.0.1:1080" />
            </AppFormItem>
            <AppFormItem :label="t('前缀', 'Prefix')">
              <AppInput v-model:value="form.prefix" />
            </AppFormItem>
            <AppFormItem :label="t('优先级', 'Priority')">
              <AppNumberInput v-model:value="form.priority" clearable :precision="0" />
            </AppFormItem>
          </div>

          <div class="switch-grid">
            <label><AppSwitch v-model:value="form.disabled" /> <span>{{ t('停用此上游', 'Disable this upstream') }}</span></label>
            <label v-if="activeSection === 'codex-api-key' || activeSection === 'xai-api-key'">
              <AppSwitch v-model:value="form.websockets" /> <span>{{ t('启用 WebSockets', 'Enable WebSockets') }}</span>
            </label>
            <label><AppSwitch v-model:value="form.disableCooling" /> <span>{{ t('禁用冷却调度', 'Disable cooldown scheduling') }}</span></label>
          </div>

          <section v-if="isOpenAISection" class="form-section">
            <div class="form-section__header">
              <div><h3>{{ t('API 密钥条目', 'API key entries') }}</h3><p>{{ t('一个提供商可以配置多个密钥和独立代理。', 'A provider can use multiple keys and per-key proxies.') }}</p></div>
              <AppButton size="small" secondary @click="addAPIKeyEntry">{{ t('添加密钥', 'Add key') }}</AppButton>
            </div>
            <div v-for="(entry, index) in form.apiKeyEntries" :key="index" class="repeat-row repeat-row--keys">
              <AppInput v-model:value="entry.apiKey" type="password" show-password-on="click" :placeholder="t(`API 密钥 #${index + 1}`, `API key #${index + 1}`)" />
              <AppInput v-model:value="entry.proxyUrl" :placeholder="t('代理 URL（可选）', 'Proxy URL (optional)')" />
              <AppButton text type="error" :disabled="form.apiKeyEntries.length <= 1" @click="form.apiKeyEntries.splice(index, 1)">
                <AppIcon :component="Trash2" />
              </AppButton>
            </div>
          </section>

          <section class="form-section">
            <div class="form-section__header">
              <div><h3>{{ t('自定义请求头', 'Custom headers') }}</h3><p>{{ t('请求上游时附加的 Header。', 'Headers appended to upstream requests.') }}</p></div>
              <AppButton size="small" secondary @click="addHeader">{{ t('添加请求头', 'Add header') }}</AppButton>
            </div>
            <div v-if="form.headers.length === 0" class="form-empty">{{ t('未配置请求头', 'No custom headers') }}</div>
            <div v-for="(header, index) in form.headers" :key="index" class="repeat-row">
              <AppInput v-model:value="header.key" placeholder="Header-Name" />
              <AppInput v-model:value="header.value" :placeholder="t('值', 'Value')" />
              <AppButton text type="error" @click="form.headers.splice(index, 1)"><AppIcon :component="Trash2" /></AppButton>
            </div>
          </section>

          <section class="form-section">
            <div class="form-section__header">
              <div><h3>{{ t('自定义模型', 'Custom models') }}</h3><p>{{ t('配置上游模型名称、路由别名和显示名称。', 'Configure upstream model names, routing aliases, and display names.') }}</p></div>
              <AppButton size="small" secondary @click="addModel">{{ t('添加模型', 'Add model') }}</AppButton>
            </div>
            <div v-if="form.models.length === 0" class="form-empty">{{ t('未限制自定义模型', 'No custom model list') }}</div>
            <div v-for="(model, index) in form.models" :key="index" class="repeat-row repeat-row--models">
              <AppInput v-model:value="model.name" :placeholder="t('上游模型名称', 'Upstream model name')" />
              <AppInput v-model:value="model.alias" :placeholder="t('路由别名（可选）', 'Routing alias (optional)')" />
              <AppInput v-model:value="model.displayName" :placeholder="t('显示名称（可选）', 'Display name (optional)')" />
              <AppButton text type="error" @click="form.models.splice(index, 1)"><AppIcon :component="Trash2" /></AppButton>
            </div>
          </section>

          <AppFormItem v-if="!isOpenAISection" :label="t('排除模型', 'Excluded models')">
            <AppInput v-model:value="form.excludedModels" type="textarea" :rows="4" :placeholder="t('每行一个模型或通配规则', 'One model or wildcard rule per line')" />
          </AppFormItem>

          <section v-if="isClaudeSection" class="form-section">
            <div class="form-section__header">
              <div><h3>Cloak</h3><p>{{ t('Claude 请求混淆与缓存设置。', 'Claude request cloaking and cache settings.') }}</p></div>
            </div>
            <div class="form-grid">
              <AppFormItem :label="t('模式', 'Mode')">
                <AppSelect
                  v-model:value="form.cloakMode"
                  clearable
                  :options="[
                    { label: t('自动', 'Auto'), value: 'auto' },
                    { label: t('始终', 'Always'), value: 'always' },
                    { label: t('从不', 'Never'), value: 'never' },
                  ]"
                />
              </AppFormItem>
              <AppFormItem :label="t('敏感词', 'Sensitive words')">
                <AppInput v-model:value="form.cloakSensitiveWords" type="textarea" :rows="3" :placeholder="t('每行一个敏感词', 'One sensitive word per line')" />
              </AppFormItem>
            </div>
            <div class="switch-grid">
              <label><AppSwitch v-model:value="form.cloakStrict" /> <span>{{ t('严格模式', 'Strict mode') }}</span></label>
              <label><AppSwitch v-model:value="form.cloakCacheUserId" /> <span>{{ t('缓存 user_id', 'Cache user_id') }}</span></label>
              <label><AppSwitch v-model:value="form.experimentalCCHSigning" /> <span>{{ t('实验性 CCH 签名', 'Experimental CCH signing') }}</span></label>
            </div>
          </section>
        </AppForm>

        <template v-if="drawerMode !== 'detail'" #footer>
          <AppStack justify="end">
            <AppButton :disabled="isSaving" @click="drawerOpen = false">{{ t('取消', 'Cancel') }}</AppButton>
            <AppButton type="primary" :loading="isSaving" @click="saveUpstream">{{ drawerMode === 'create' ? t('创建', 'Create') : t('保存', 'Save') }}</AppButton>
          </AppStack>
        </template>
      </AppDrawerContent>
    </AppDrawer>
  </section>
</template>

<style scoped>
.upstream-page {
  grid-template-rows: auto auto auto minmax(0, 1fr);
  min-height: 0;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.upstream-workbench {
  min-height: 0;
  overflow: hidden;
}

.workbench-layout {
  display: grid;
  grid-template-columns: 236px minmax(0, 1fr);
  min-height: 520px;
}

.provider-nav {
  padding: 18px 12px;
  border-right: 1px solid var(--cpa-border);
  background: color-mix(in srgb, var(--cpa-surface) 88%, var(--cpa-primary-soft));
}

.provider-nav__title {
  margin: 0 10px 10px;
  color: var(--cpa-text-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}

.provider-nav__item {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin: 3px 0;
  padding: 10px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  color: var(--cpa-text);
  text-align: left;
  cursor: pointer;
}

.provider-nav__item:hover { background: var(--cpa-surface); }
.provider-nav__item--active { border-color: color-mix(in srgb, var(--cpa-primary) 34%, var(--cpa-border)); background: var(--cpa-surface); box-shadow: var(--cpa-shadow-hairline); }
.provider-nav__copy { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
.provider-nav__copy strong { font-size: 13px; }
.provider-nav__copy small { overflow: hidden; color: var(--cpa-text-muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }

.provider-panel { display: grid; min-width: 0; grid-template-rows: auto auto minmax(0, 1fr); }
.provider-panel__header { display: flex; min-height: 72px; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 18px; border-bottom: 1px solid var(--cpa-border); }
.provider-panel__title { display: flex; align-items: center; gap: 10px; }
.provider-panel__title h2 { margin: 0; font-size: 17px; }
.provider-panel__title p { margin: 2px 0 0; color: var(--cpa-text-muted); font-size: 11px; }
.provider-panel__toolbar { display: flex; min-width: 0; align-items: stretch; gap: 10px; padding: 12px 16px; border-bottom: 1px solid var(--cpa-border); background: color-mix(in srgb, var(--cpa-surface-muted) 72%, transparent); }
.provider-search { width: min(100%, 520px); height: 36px; min-width: 260px; }
.provider-result-count { display: inline-flex; align-items: center; color: var(--cpa-text-muted); font-size: 12px; white-space: nowrap; }
.provider-create-button { height: 36px; margin-left: auto; }
.provider-table-shell { min-width: 0; padding: 16px; }
:global(.upstream-row-actions) { width: 100%; justify-content: flex-end; }

:deep(.identity-cell) { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
:deep(.identity-cell strong) { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
:deep(.identity-cell code), :deep(.url-cell) { color: var(--cpa-text-muted); font-size: 11px; }
.detail-edit-button { margin-top: 16px; }

.upstream-form { display: grid; gap: 4px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 14px; }
.switch-grid { display: flex; flex-wrap: wrap; gap: 16px 24px; margin: 0 0 18px; }
.switch-grid label { display: inline-flex; align-items: center; gap: 8px; color: var(--cpa-text-secondary); font-size: 13px; }
.form-section { display: grid; gap: 10px; margin: 0 0 18px; padding: 14px; border: 1px solid var(--cpa-border); border-radius: var(--cpa-radius); background: color-mix(in srgb, var(--cpa-surface) 94%, var(--cpa-primary-soft)); }
.form-section__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.form-section__header h3 { margin: 0; font-size: 14px; }
.form-section__header p { margin: 3px 0 0; color: var(--cpa-text-muted); font-size: 11px; }
.form-empty { padding: 10px; color: var(--cpa-text-muted); font-size: 12px; text-align: center; }
.repeat-row { display: grid; grid-template-columns: minmax(0, .8fr) minmax(0, 1.2fr) 30px; gap: 8px; align-items: center; }
.repeat-row--keys { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 30px; }
.repeat-row--models { grid-template-columns: repeat(3, minmax(0, 1fr)) 30px; }

@media (max-width: 980px) {
  .workbench-layout { grid-template-columns: 1fr; }
  .provider-nav { display: flex; overflow-x: auto; padding: 10px; border-right: 0; border-bottom: 1px solid var(--cpa-border); }
  .provider-nav__title { display: none; }
  .provider-nav__item { min-width: 155px; }
  .metric-grid { grid-template-columns: 1fr; }
}

@media (max-width: 700px) {
  .provider-panel__toolbar { align-items: center; flex-wrap: wrap; }
  .provider-search { width: 100%; max-width: none; min-width: 0; }
  .provider-result-count { order: 2; }
  .provider-create-button { order: 3; }
  .form-grid, .repeat-row, .repeat-row--keys, .repeat-row--models { grid-template-columns: 1fr; }
  .repeat-row :deep(.n-button) { justify-self: end; }
}
</style>
