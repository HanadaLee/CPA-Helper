<script setup lang="ts">
import type { Component } from 'vue'
import { computed, reactive, ref } from 'vue'
import { toast } from 'vue-sonner'
import { useConfirmDialog } from '@/shared/ui/confirm-dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
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
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from '@/components/ui/input-group'
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
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
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
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import {
  Activity,
  Bot,
  Cloud,
  Clock3,
  Code2,
  Eye,
  KeyRound,
  MoreHorizontal,
  Network,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Sparkles,
  Trash2,
  X,
} from '@lucide/vue'

import {
  listUpstreamSections,
  replaceUpstreamSection,
  upstreamSectionNames,
  type UpstreamItem,
  type UpstreamSection,
} from '@/features/upstreams/api/upstreamApi'
import { useI18n } from '@/shared/i18n'
import { formatDateTime } from '@/shared/utils/format'

interface SectionDefinition {
  name: UpstreamSection
  label: string
  description: string
  icon: Component
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

const message = toast
const dialog = useConfirmDialog()
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
  { name: 'gemini-api-key', label: 'Gemini', description: t('Gemini API 密钥', 'Gemini API keys'), icon: Sparkles },
  { name: 'codex-api-key', label: 'Codex', description: t('Codex 上游', 'Codex upstreams'), icon: Code2 },
  { name: 'xai-api-key', label: 'xAI', description: t('xAI 上游', 'xAI upstreams'), icon: KeyRound },
  { name: 'claude-api-key', label: 'Claude', description: t('Claude API 密钥', 'Claude API keys'), icon: Bot },
  { name: 'vertex-api-key', label: 'Vertex', description: t('Vertex API 密钥', 'Vertex API keys'), icon: Cloud },
  {
    name: 'openai-compatibility',
    label: t('OpenAI 兼容', 'OpenAI Compatible'),
    description: t('OpenAI 兼容提供商', 'OpenAI-compatible providers'),
    icon: Network,
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

const drawerContentClass = computed(() => cn(
  'overflow-hidden data-[side=right]:w-full',
  drawerMode.value === 'detail'
    ? 'data-[side=right]:sm:max-w-[460px]'
    : 'data-[side=right]:sm:max-w-[720px]',
))

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

function selectSection(name: UpstreamSection) {
  activeSection.value = name
  searchText.value = ''
}

function handleSectionChange(value: unknown) {
  if (typeof value !== 'string' || !upstreamSectionNames.includes(value as UpstreamSection)) return
  selectSection(value as UpstreamSection)
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

function setPriority(value: string | number) {
  if (value === '') {
    form.priority = null
    return
  }
  const parsed = Number(value)
  form.priority = Number.isFinite(parsed) ? Math.trunc(parsed) : null
}

function setCloakMode(value: unknown) {
  form.cloakMode = value === 'auto' || value === 'always' || value === 'never' ? value : null
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

function handleToggleUpstream(row: UpstreamTableRow, value: unknown) {
  void toggleUpstream(row, Boolean(value))
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
  return formatDateTime(updatedAt.value)
}

void loadUpstreams()
</script>

<template>
  <section class="page upstream-page">
    <div class="page-toolbar upstream-header">
      <h1 data-page-title class="page-title">{{ t('上游管理', 'Upstream Management') }}</h1>
      <Button variant="outline" :disabled="isLoading" @click="loadUpstreams(true)">
        <Spinner v-if="isLoading" data-icon="inline-start" />
        <RefreshCw v-else data-icon="inline-start" />
        {{ t('刷新', 'Refresh') }}
      </Button>
    </div>

    <div class="metric-grid">
      <Card>
        <CardHeader class="flex flex-row items-start justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <CardDescription>{{ t('活跃资源', 'Active resources') }}</CardDescription>
            <CardTitle class="text-2xl tabular-nums">{{ activeResources }} / {{ totalResources }}</CardTitle>
          </div>
          <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Activity class="size-5" />
          </div>
        </CardHeader>
        <CardContent class="text-xs text-muted-foreground">
          {{ t('启用 / 已配置', 'Enabled / configured') }}
        </CardContent>
      </Card>
      <Card>
        <CardHeader class="flex flex-row items-start justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <CardDescription>{{ t('提供商类型', 'Provider families') }}</CardDescription>
            <CardTitle class="text-2xl tabular-nums">
              {{ configuredFamilies }} / {{ upstreamSectionNames.length }}
            </CardTitle>
          </div>
          <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Network class="size-5" />
          </div>
        </CardHeader>
        <CardContent class="text-xs text-muted-foreground">
          {{ t('已配置 / 支持', 'Configured / supported') }}
        </CardContent>
      </Card>
      <Card>
        <CardHeader class="flex flex-row items-start justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <CardDescription>{{ t('最近同步', 'Last synced') }}</CardDescription>
            <CardTitle class="text-lg tabular-nums xl:text-xl">{{ formatUpdatedAt() }}</CardTitle>
          </div>
          <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Clock3 class="size-5" />
          </div>
        </CardHeader>
        <CardContent class="text-xs text-muted-foreground">
          {{ t('CLIProxyAPI 配置', 'CLIProxyAPI configuration') }}
        </CardContent>
      </Card>
    </div>

    <Alert v-if="loadError" variant="destructive">
      <AlertTitle>{{ t('无法加载上游配置', 'Unable to load upstream configurations') }}</AlertTitle>
      <AlertDescription>{{ loadError }}</AlertDescription>
    </Alert>

    <Card class="upstream-workbench gap-0 py-0">
      <CardHeader class="provider-switcher">
        <Tabs
          :model-value="activeSection"
          class="provider-tabs"
          @update:model-value="handleSectionChange"
        >
          <TabsList
            class="provider-tabs__list"
            :aria-label="t('提供商', 'Providers')"
          >
            <TabsTrigger
              v-for="definition in sectionDefinitions"
              :key="definition.name"
              :value="definition.name"
              class="provider-tabs__item"
            >
              <component :is="definition.icon" data-icon="inline-start" />
              <span class="provider-tabs__label">{{ definition.label }}</span>
              <Badge variant="outline" class="provider-tabs__count tabular-nums">
                {{ sections[definition.name].length }}
              </Badge>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </CardHeader>

      <CardContent class="min-h-0 p-0">
        <div class="provider-panel">
          <div class="provider-panel__toolbar">
            <InputGroup class="provider-search">
              <InputGroupAddon>
                <Search aria-hidden="true" />
              </InputGroupAddon>
              <InputGroupInput
                v-model="searchText"
                :placeholder="t('搜索密钥、地址或前缀', 'Search keys, URLs, or prefixes')"
              />
              <InputGroupAddon v-if="searchText" align="inline-end">
                <InputGroupButton
                  :aria-label="t('清空搜索', 'Clear search')"
                  :title="t('清空搜索', 'Clear search')"
                  @click="searchText = ''"
                >
                  <X aria-hidden="true" />
                </InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
            <div class="provider-toolbar__actions">
              <Badge variant="secondary" class="provider-result-count tabular-nums">
                {{ filteredRows.length }} / {{ sections[activeSection].length }}
              </Badge>
              <Button class="provider-create-button" :disabled="isLoading" @click="openEditor()">
                <Plus data-icon="inline-start" />
                {{ t('新建', 'New') }}
              </Button>
            </div>
          </div>

          <div class="provider-table-shell">
            <Table class="min-w-[766px] table-fixed">
              <TableHeader>
                <TableRow>
                  <TableHead class="w-[160px]">
                    {{ isOpenAISection ? t('提供商', 'Provider') : t('密钥', 'Key') }}
                  </TableHead>
                  <TableHead class="w-[180px]">{{ t('服务地址', 'Base URL') }}</TableHead>
                  <TableHead class="w-[80px]">{{ t('前缀', 'Prefix') }}</TableHead>
                  <TableHead class="w-[140px]">{{ t('模型 / 请求头', 'Models / Headers') }}</TableHead>
                  <TableHead class="w-[70px] text-center">{{ t('优先级', 'Priority') }}</TableHead>
                  <TableHead class="w-[80px]">{{ t('状态', 'Status') }}</TableHead>
                  <TableHead class="w-[56px]">
                    <span class="sr-only">{{ t('操作', 'Actions') }}</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <template v-if="isLoading && filteredRows.length === 0">
                  <TableRow v-for="rowIndex in 6" :key="`upstream-skeleton-${rowIndex}`">
                    <TableCell v-for="columnIndex in 7" :key="columnIndex">
                      <Skeleton class="h-4 w-full" />
                    </TableCell>
                  </TableRow>
                </template>

                <TableEmpty v-else-if="filteredRows.length === 0" :colspan="7">
                  {{ t('暂无上游配置', 'No upstream configurations') }}
                </TableEmpty>

                <TableRow v-for="row in filteredRows" v-else :key="row.key">
                  <TableCell>
                    <div class="identity-cell">
                      <strong :title="upstreamTitle(activeSection, row.item, row.index)">
                        {{ upstreamTitle(activeSection, row.item, row.index) }}
                      </strong>
                      <code>{{ maskSecret(primaryAPIKey(activeSection, row.item)) }}</code>
                    </div>
                  </TableCell>
                  <TableCell>
                    <code class="url-cell" :title="readString(row.item, 'base-url') || '-'">
                      {{ readString(row.item, 'base-url') || '-' }}
                    </code>
                  </TableCell>
                  <TableCell>
                    <span class="block truncate" :title="readString(row.item, 'prefix') || '-'">
                      {{ readString(row.item, 'prefix') || '-' }}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div class="flex flex-wrap gap-1">
                      <Badge variant="secondary">
                        {{ t(`模型 ${modelCount(row.item)}`, `Models ${modelCount(row.item)}`) }}
                      </Badge>
                      <Badge variant="outline">
                        {{ t(`头 ${headerCount(row.item)}`, `Headers ${headerCount(row.item)}`) }}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell class="text-center tabular-nums">
                    {{ readNumber(row.item, 'priority') ?? '-' }}
                  </TableCell>
                  <TableCell>
                    <div class="status-control">
                      <Switch
                        :model-value="!upstreamDisabled(activeSection, row.item)"
                        :disabled="isSaving || isLoading || mutatingKey === row.key"
                        :aria-label="t('切换上游状态', 'Toggle upstream status')"
                        @update:model-value="handleToggleUpstream(row, $event)"
                      />
                      <Spinner v-if="mutatingKey === row.key" />
                    </div>
                  </TableCell>
                  <TableCell class="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger as-child>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          :aria-label="t(`打开 ${upstreamTitle(activeSection, row.item, row.index)} 的操作菜单`, `Open actions for ${upstreamTitle(activeSection, row.item, row.index)}`)"
                        >
                          <MoreHorizontal />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" class="w-40">
                        <DropdownMenuGroup>
                          <DropdownMenuItem @select="openDetail(row)">
                            <Eye />
                            <span>{{ t('详情', 'Details') }}</span>
                          </DropdownMenuItem>
                          <DropdownMenuItem @select="openEditor(row)">
                            <Pencil />
                            <span>{{ t('编辑', 'Edit') }}</span>
                          </DropdownMenuItem>
                        </DropdownMenuGroup>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem variant="destructive" @select="confirmDelete(row)">
                          <Trash2 />
                          <span>{{ t('删除', 'Delete') }}</span>
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </div>
      </CardContent>
    </Card>

    <Sheet v-model:open="drawerOpen">
      <SheetContent side="right" :class="drawerContentClass">
        <SheetHeader>
          <SheetTitle>{{ drawerTitle }}</SheetTitle>
          <SheetDescription>{{ activeDefinition?.description }}</SheetDescription>
        </SheetHeader>

        <div class="sheet-body">
          <template v-if="drawerMode === 'detail' && selectedItem">
            <dl class="detail-list">
              <div class="detail-row">
                <dt>{{ isOpenAISection ? t('提供商', 'Provider') : t('API 密钥', 'API key') }}</dt>
                <dd>
                  {{ isOpenAISection ? readString(selectedItem, 'name') || '-' : maskSecret(primaryAPIKey(activeSection, selectedItem)) }}
                </dd>
              </div>
              <div class="detail-row">
                <dt>{{ t('服务地址', 'Base URL') }}</dt>
                <dd><code>{{ readString(selectedItem, 'base-url') || '-' }}</code></dd>
              </div>
              <div class="detail-row">
                <dt>{{ t('代理 URL', 'Proxy URL') }}</dt>
                <dd><code>{{ readString(selectedItem, 'proxy-url') || '-' }}</code></dd>
              </div>
              <div class="detail-row">
                <dt>{{ t('前缀', 'Prefix') }}</dt>
                <dd>{{ readString(selectedItem, 'prefix') || '-' }}</dd>
              </div>
              <div class="detail-row">
                <dt>{{ t('优先级', 'Priority') }}</dt>
                <dd>{{ readNumber(selectedItem, 'priority') ?? '-' }}</dd>
              </div>
              <div class="detail-row">
                <dt>{{ t('模型 / 请求头 / 密钥', 'Models / Headers / Keys') }}</dt>
                <dd>
                  {{ modelCount(selectedItem) }} / {{ headerCount(selectedItem) }} /
                  {{ isOpenAISection ? readObjectArray(selectedItem, 'api-key-entries').length : 1 }}
                </dd>
              </div>
              <div class="detail-row">
                <dt>{{ t('状态', 'Status') }}</dt>
                <dd>
                  <Badge :variant="upstreamDisabled(activeSection, selectedItem) ? 'destructive' : 'secondary'">
                    {{ upstreamDisabled(activeSection, selectedItem) ? t('已停用', 'Disabled') : t('活跃', 'Active') }}
                  </Badge>
                </dd>
              </div>
            </dl>
            <Button
              class="w-full"
              @click="openEditor({ key: '', index: editingIndex, item: selectedItem })"
            >
              <Pencil data-icon="inline-start" />
              {{ t('编辑', 'Edit') }}
            </Button>
          </template>

          <form
            v-else
            id="upstream-editor-form"
            class="upstream-form"
            @submit.prevent="saveUpstream"
          >
            <FieldGroup>
              <Field v-if="isOpenAISection">
                <FieldLabel for="upstream-name">{{ t('名称', 'Name') }}</FieldLabel>
                <Input
                  id="upstream-name"
                  v-model="form.name"
                  required
                  :placeholder="t('例如 OpenRouter', 'e.g. OpenRouter')"
                />
              </Field>
              <Field v-else>
                <FieldLabel for="upstream-api-key">{{ t('API 密钥', 'API key') }}</FieldLabel>
                <Input id="upstream-api-key" v-model="form.apiKey" type="password" required />
              </Field>

              <FieldGroup class="form-grid">
                <Field>
                  <FieldLabel for="upstream-base-url">{{ t('服务地址', 'Base URL') }}</FieldLabel>
                  <Input
                    id="upstream-base-url"
                    v-model="form.baseUrl"
                    :required="isOpenAISection || activeSection === 'codex-api-key' || activeSection === 'xai-api-key'"
                    placeholder="https://api.example.com/v1"
                  />
                </Field>
                <Field v-if="!isOpenAISection">
                  <FieldLabel for="upstream-proxy-url">{{ t('代理 URL', 'Proxy URL') }}</FieldLabel>
                  <Input id="upstream-proxy-url" v-model="form.proxyUrl" placeholder="socks5://127.0.0.1:1080" />
                </Field>
                <Field>
                  <FieldLabel for="upstream-prefix">{{ t('前缀', 'Prefix') }}</FieldLabel>
                  <Input id="upstream-prefix" v-model="form.prefix" />
                </Field>
                <Field>
                  <FieldLabel for="upstream-priority">{{ t('优先级', 'Priority') }}</FieldLabel>
                  <Input
                    id="upstream-priority"
                    type="number"
                    step="1"
                    :model-value="form.priority ?? ''"
                    @update:model-value="setPriority"
                  />
                </Field>
              </FieldGroup>

              <FieldGroup class="switch-grid">
                <Field orientation="horizontal" class="switch-option">
                  <FieldContent>
                    <FieldTitle>{{ t('停用此上游', 'Disable this upstream') }}</FieldTitle>
                  </FieldContent>
                  <Switch v-model="form.disabled" />
                </Field>
                <Field
                  v-if="activeSection === 'codex-api-key' || activeSection === 'xai-api-key'"
                  orientation="horizontal"
                  class="switch-option"
                >
                  <FieldContent>
                    <FieldTitle>{{ t('启用 WebSockets', 'Enable WebSockets') }}</FieldTitle>
                  </FieldContent>
                  <Switch v-model="form.websockets" />
                </Field>
                <Field orientation="horizontal" class="switch-option">
                  <FieldContent>
                    <FieldTitle>{{ t('禁用冷却调度', 'Disable cooldown scheduling') }}</FieldTitle>
                  </FieldContent>
                  <Switch v-model="form.disableCooling" />
                </Field>
              </FieldGroup>

              <section v-if="isOpenAISection" class="form-section">
                <div class="form-section__header">
                  <div>
                    <h3>{{ t('API 密钥条目', 'API key entries') }}</h3>
                    <p>{{ t('一个提供商可以配置多个密钥和独立代理。', 'A provider can use multiple keys and per-key proxies.') }}</p>
                  </div>
                  <Button type="button" size="sm" variant="outline" @click="addAPIKeyEntry">
                    <Plus data-icon="inline-start" />
                    {{ t('添加密钥', 'Add key') }}
                  </Button>
                </div>
                <div
                  v-for="(entry, index) in form.apiKeyEntries"
                  :key="index"
                  class="repeat-row repeat-row--keys"
                >
                  <Input
                    v-model="entry.apiKey"
                    type="password"
                    :placeholder="t(`API 密钥 #${index + 1}`, `API key #${index + 1}`)"
                  />
                  <Input v-model="entry.proxyUrl" :placeholder="t('代理 URL（可选）', 'Proxy URL (optional)')" />
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    :disabled="form.apiKeyEntries.length <= 1"
                    :aria-label="t('删除密钥', 'Delete key')"
                    @click="form.apiKeyEntries.splice(index, 1)"
                  >
                    <Trash2 />
                  </Button>
                </div>
              </section>

              <section class="form-section">
                <div class="form-section__header">
                  <div>
                    <h3>{{ t('自定义请求头', 'Custom headers') }}</h3>
                    <p>{{ t('请求上游时附加的 Header。', 'Headers appended to upstream requests.') }}</p>
                  </div>
                  <Button type="button" size="sm" variant="outline" @click="addHeader">
                    <Plus data-icon="inline-start" />
                    {{ t('添加请求头', 'Add header') }}
                  </Button>
                </div>
                <div v-if="form.headers.length === 0" class="form-empty">
                  {{ t('未配置请求头', 'No custom headers') }}
                </div>
                <div v-for="(header, index) in form.headers" :key="index" class="repeat-row">
                  <Input v-model="header.key" placeholder="Header-Name" />
                  <Input v-model="header.value" :placeholder="t('值', 'Value')" />
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    :aria-label="t('删除请求头', 'Delete header')"
                    @click="form.headers.splice(index, 1)"
                  >
                    <Trash2 />
                  </Button>
                </div>
              </section>

              <section class="form-section">
                <div class="form-section__header">
                  <div>
                    <h3>{{ t('自定义模型', 'Custom models') }}</h3>
                    <p>{{ t('配置上游模型名称、路由别名和显示名称。', 'Configure upstream model names, routing aliases, and display names.') }}</p>
                  </div>
                  <Button type="button" size="sm" variant="outline" @click="addModel">
                    <Plus data-icon="inline-start" />
                    {{ t('添加模型', 'Add model') }}
                  </Button>
                </div>
                <div v-if="form.models.length === 0" class="form-empty">
                  {{ t('未限制自定义模型', 'No custom model list') }}
                </div>
                <div
                  v-for="(model, index) in form.models"
                  :key="index"
                  class="repeat-row repeat-row--models"
                >
                  <Input v-model="model.name" :placeholder="t('上游模型名称', 'Upstream model name')" />
                  <Input v-model="model.alias" :placeholder="t('路由别名（可选）', 'Routing alias (optional)')" />
                  <Input v-model="model.displayName" :placeholder="t('显示名称（可选）', 'Display name (optional)')" />
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    :aria-label="t('删除模型', 'Delete model')"
                    @click="form.models.splice(index, 1)"
                  >
                    <Trash2 />
                  </Button>
                </div>
              </section>

              <Field v-if="!isOpenAISection">
                <FieldLabel for="upstream-excluded-models">{{ t('排除模型', 'Excluded models') }}</FieldLabel>
                <Textarea
                  id="upstream-excluded-models"
                  v-model="form.excludedModels"
                  :placeholder="t('每行一个模型或通配规则', 'One model or wildcard rule per line')"
                />
              </Field>

              <section v-if="isClaudeSection" class="form-section">
                <div class="form-section__header">
                  <div>
                    <h3>Cloak</h3>
                    <p>{{ t('Claude 请求混淆与缓存设置。', 'Claude request cloaking and cache settings.') }}</p>
                  </div>
                </div>
                <FieldGroup class="form-grid">
                  <Field>
                    <FieldLabel for="cloak-mode">{{ t('模式', 'Mode') }}</FieldLabel>
                    <div class="flex items-center gap-2">
                      <Select :model-value="form.cloakMode ?? undefined" @update:model-value="setCloakMode">
                        <SelectTrigger id="cloak-mode" class="flex-1">
                          <SelectValue :placeholder="t('未设置', 'Not set')" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="auto">{{ t('自动', 'Auto') }}</SelectItem>
                            <SelectItem value="always">{{ t('始终', 'Always') }}</SelectItem>
                            <SelectItem value="never">{{ t('从不', 'Never') }}</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <Button
                        v-if="form.cloakMode"
                        type="button"
                        size="icon-sm"
                        variant="ghost"
                        :aria-label="t('清除模式', 'Clear mode')"
                        @click="form.cloakMode = null"
                      >
                        <X />
                      </Button>
                    </div>
                  </Field>
                  <Field>
                    <FieldLabel for="cloak-sensitive-words">{{ t('敏感词', 'Sensitive words') }}</FieldLabel>
                    <Textarea
                      id="cloak-sensitive-words"
                      v-model="form.cloakSensitiveWords"
                      :placeholder="t('每行一个敏感词', 'One sensitive word per line')"
                    />
                  </Field>
                </FieldGroup>
                <FieldGroup class="switch-grid">
                  <Field orientation="horizontal" class="switch-option">
                    <FieldTitle>{{ t('严格模式', 'Strict mode') }}</FieldTitle>
                    <Switch v-model="form.cloakStrict" />
                  </Field>
                  <Field orientation="horizontal" class="switch-option">
                    <FieldTitle>{{ t('缓存 user_id', 'Cache user_id') }}</FieldTitle>
                    <Switch v-model="form.cloakCacheUserId" />
                  </Field>
                  <Field orientation="horizontal" class="switch-option">
                    <FieldTitle>{{ t('实验性 CCH 签名', 'Experimental CCH signing') }}</FieldTitle>
                    <Switch v-model="form.experimentalCCHSigning" />
                  </Field>
                </FieldGroup>
              </section>
            </FieldGroup>
          </form>
        </div>

        <SheetFooter v-if="drawerMode !== 'detail'" class="flex-row justify-end border-0 shadow-none">
          <Button type="button" variant="outline" :disabled="isSaving" @click="drawerOpen = false">
            {{ t('取消', 'Cancel') }}
          </Button>
          <Button type="submit" form="upstream-editor-form" :disabled="isSaving">
            <Spinner v-if="isSaving" data-icon="inline-start" />
            {{ drawerMode === 'create' ? t('创建', 'Create') : t('保存', 'Save') }}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
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
  gap: 1rem;
}

.upstream-workbench {
  min-height: 0;
  overflow: hidden;
}

.provider-switcher {
  min-width: 0;
  padding: .75rem;
  border-bottom: 1px solid var(--border);
}

.provider-tabs {
  display: block;
  min-width: 0;
  width: 100%;
  overflow-x: auto;
}

.provider-tabs__list {
  display: grid;
  width: 100%;
  min-width: 750px;
  height: 2.5rem;
  grid-template-columns: repeat(6, minmax(7.5rem, 1fr));
  padding: .25rem;
}

.provider-tabs__item {
  min-width: 0;
  gap: .375rem;
  padding-inline: .625rem;
}

.provider-tabs__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-tabs__count {
  min-width: 1.5rem;
}

.provider-panel {
  display: grid;
  min-width: 0;
  grid-template-rows: auto minmax(0, 1fr);
}

.provider-panel__toolbar {
  display: flex;
  min-height: 3.5rem;
  min-width: 0;
  align-items: center;
  gap: .625rem;
  padding: .75rem 1rem;
  border-bottom: 1px solid var(--border);
}

.provider-toolbar__actions {
  display: flex;
  margin-left: auto;
  align-items: center;
  gap: .5rem;
}

.provider-result-count {
  min-width: 3.5rem;
  justify-content: center;
}

.provider-search {
  width: min(100%, 36rem);
  min-width: 13rem;
}

.provider-table-shell {
  min-width: 0;
  overflow-x: auto;
}

.identity-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: .1875rem;
}

.identity-cell strong,
.url-cell {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity-cell code,
.url-cell {
  color: var(--muted-foreground);
  font-size: .6875rem;
}

.status-control {
  display: flex;
  align-items: center;
  gap: .5rem;
}

.sheet-body {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 0 1rem 1rem;
}

.detail-list {
  display: grid;
  gap: 0;
  margin: 0 0 1rem;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.detail-row {
  display: grid;
  grid-template-columns: 8rem minmax(0, 1fr);
  gap: 1rem;
  padding: .75rem 1rem;
  border-bottom: 1px solid var(--border);
}

.detail-row:last-child {
  border-bottom: 0;
}

.detail-row dt {
  color: var(--muted-foreground);
  font-size: .75rem;
}

.detail-row dd {
  min-width: 0;
  overflow-wrap: anywhere;
}

.upstream-form {
  display: grid;
  gap: 1rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.switch-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: .75rem;
}

.switch-option {
  min-height: 3.25rem;
  padding: .75rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--muted) 22%, transparent);
}

.form-section {
  display: grid;
  gap: .75rem;
  padding: 1rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--muted) 18%, transparent);
}

.form-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: .75rem;
}

.form-section__header h3 {
  margin: 0;
  font-size: .875rem;
  font-weight: 600;
}

.form-section__header p {
  margin: .1875rem 0 0;
  color: var(--muted-foreground);
  font-size: .75rem;
}

.form-empty {
  padding: .75rem;
  color: var(--muted-foreground);
  font-size: .75rem;
  text-align: center;
}

.repeat-row {
  display: grid;
  grid-template-columns: minmax(0, .8fr) minmax(0, 1.2fr) 2rem;
  gap: .5rem;
  align-items: center;
}

.repeat-row--keys {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 2rem;
}

.repeat-row--models {
  grid-template-columns: repeat(3, minmax(0, 1fr)) 2rem;
}

@media (max-width: 980px) {
  .metric-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .provider-panel__toolbar {
    flex-wrap: wrap;
  }

  .provider-search {
    width: 100%;
    min-width: 0;
  }

  .provider-toolbar__actions {
    width: 100%;
    margin-left: 0;
  }

  .provider-create-button {
    margin-left: auto;
  }

  .detail-row,
  .form-grid,
  .switch-grid,
  .repeat-row,
  .repeat-row--keys,
  .repeat-row--models {
    grid-template-columns: 1fr;
  }

  .detail-row {
    gap: .25rem;
  }

  .repeat-row > :last-child {
    justify-self: end;
  }

  .form-section__header {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
