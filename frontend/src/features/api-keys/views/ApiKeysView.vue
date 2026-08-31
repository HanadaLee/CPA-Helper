<script setup lang="ts">
import type { Component } from 'vue'
import { computed, h, onMounted, ref, watch } from 'vue'
import {
  AppAlert,
  AppButton,
  AppDataTable,
  AppEllipsis,
  AppForm,
  AppFormItem,
  AppIcon,
  AppInput,
  AppModal,
  AppSelect,
  AppStack,
  useDialog,
  useMessage,
  type DataTableColumns,
} from '@/shared/ui/app-kit'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  Activity,
  Bot,
  Braces,
  CircleDollarSign,
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  Layers3,
  Link2,
  MessageSquare,
  Send,
} from '@lucide/vue'

import {
  createApiKey,
  deleteApiKey,
  getModelRequestGuide,
  listApiKeys,
  testModelRequest,
  updateApiKey,
} from '@/features/api-keys/api/apiKeysApi'
import { listAvailableModels } from '@/features/models/api/availableModelsApi'
import { getCurrentUserQuota } from '@/features/users/api/usersApi'
import { getUsageOverview } from '@/features/usage/api/usageApi'
import type {
  AvailableModel,
  AvailableModelsResponse,
  ModelRequestEndpoint,
  ModelRequestGuide,
  ModelRequestTestResponse,
  UsageSummary,
  UserApiKeySummary,
  UserQuotaStatus,
} from '@/shared/types/api'
import { useI18n } from '@/shared/i18n'
import { copyToClipboard } from '@/shared/utils/clipboard'
import { formatCompact, formatDateTime, formatInteger, formatUsd } from '@/shared/utils/format'

const message = useMessage()
const dialog = useDialog()
const { copiedText, currentLanguage, errorText, t } = useI18n()
const isLoading = ref(false)
const isSaving = ref(false)
const apiKeys = ref<UserApiKeySummary[]>([])
const usageSummary = ref<UsageSummary | null>(null)
const quotaStatus = ref<UserQuotaStatus | null>(null)
const modelRequestGuide = ref<ModelRequestGuide | null>(null)
const availableModels = ref<AvailableModelsResponse | null>(null)
const editorVisible = ref(false)
const requestTestVisible = ref(false)
const requestTestApiKey = ref<UserApiKeySummary | null>(null)
const requestEndpoint = ref<ModelRequestEndpoint>('chat_completions')
type PublicRequestURLType = 'base' | ModelRequestEndpoint
const publicRequestURLType = ref<PublicRequestURLType>('base')
const requestTestModel = ref<string | null>(null)
const requestTestMessageDefaults = {
  zh: '请用一句中文回复：连接测试成功。',
  en: 'Reply in one English sentence: connection test succeeded.',
} as const
const requestTestMessage = ref(requestTestMessageDefaults[currentLanguage.value])
const requestTestResult = ref<ModelRequestTestResponse | null>(null)
const requestTestError = ref<string | null>(null)
const isAvailableModelsLoading = ref(false)
const isRequestTesting = ref(false)
const editingApiKeyHash = ref<string | null>(null)
const apiKeyDescription = ref('VSCode')
const generatedApiKey = ref<string | null>(null)
const generatedApiKeyHash = ref<string | null>(null)
const visibleApiKeyHashes = ref<Set<string>>(new Set())

const requestLoadingText = computed(() => t('加载中', 'Loading'))

interface RequestEndpointOption {
  label: string
  value: ModelRequestEndpoint
  path: string
  urlLabel: string
}

interface PublicRequestEndpoint {
  key: string
  label: string
  baseURL: string
}

interface PublicRequestURLTypeOption {
  label: string
  ariaLabel: string
  value: PublicRequestURLType
  path: string
  icon: Component
}

const chatCompletionsEndpointOption = computed<RequestEndpointOption>(() => ({
  label: t('聊天补全', 'Chat Completions'),
  value: 'chat_completions',
  path: '/chat/completions',
  urlLabel: t('聊天补全 URL', 'Chat Completions URL'),
}))

const requestEndpointOptions = computed<RequestEndpointOption[]>(() => [
  chatCompletionsEndpointOption.value,
  {
    label: t('Responses 响应', 'Responses'),
    value: 'responses',
    path: '/responses',
    urlLabel: t('Responses 响应 URL', 'Responses URL'),
  },
  {
    label: t('Claude 消息', 'Claude Messages'),
    value: 'claude_messages',
    path: '/messages',
    urlLabel: t('Claude 消息 URL', 'Claude Messages URL'),
  },
])

interface ApiKeyMetricCard {
  key: string
  label: string
  value: string
  footnote: string
  tone: 'primary' | 'blue' | 'purple' | 'green'
  icon: Component
}

const apiKeyMetrics = computed<ApiKeyMetricCard[]>(() => {
  const summary = usageSummary.value
  const todayRequests = summary?.total_records ?? 0
  const failedToday = summary?.failed_records ?? 0
  const todayCost = summary?.estimated_cost_usd ?? 0
  const todayTokens = summary?.total_tokens ?? 0
  const quota = quotaStatus.value
  return [
    {
      key: 'keys',
      label: t('API 密钥', 'API keys'),
      value: formatInteger(apiKeys.value.length),
      footnote: t('当前账号', 'Current account'),
      tone: 'primary',
      icon: KeyRound,
    },
    {
      key: 'requests',
      label: t('今日请求', 'Requests today'),
      value: formatInteger(todayRequests),
      footnote: t(`失败 ${formatInteger(failedToday)}`, `${formatInteger(failedToday)} failed`),
      tone: 'blue',
      icon: Activity,
    },
    {
      key: 'tokens',
      label: t('今日 Token', 'Tokens today'),
      value: formatCompact(todayTokens),
      footnote: t('当前账号用量', 'Current account usage'),
      tone: 'purple',
      icon: Layers3,
    },
    {
      key: 'cost',
      label: t('今日费用', 'Cost today'),
      value: formatUsd(todayCost),
      footnote: t('按现价估算', 'Estimated at current prices'),
      tone: 'green',
      icon: CircleDollarSign,
    },
    {
      key: 'quota',
      label: t('可用余额', 'Available balance'),
      value: quotaValueText(quota),
      footnote: quotaFootnote(quota),
      tone: quota?.paused ? 'purple' : 'green',
      icon: CircleDollarSign,
    },
  ]
})

const canCreateApiKey = computed(() => quotaStatus.value?.can_create_keys ?? true)

const requestBaseURL = computed(() => modelRequestGuide.value?.openai_base_url ?? requestLoadingText.value)
const publicRequestURLTypeOptions = computed<PublicRequestURLTypeOption[]>(() => [
  {
    label: t('基础', 'Base'),
    ariaLabel: t('基础 URL', 'Base URL'),
    value: 'base',
    path: '',
    icon: Link2,
  },
  ...requestEndpointOptions.value.map((option) => ({
    label:
      option.value === 'chat_completions'
        ? t('聊天', 'Chat')
        : option.value === 'responses'
          ? 'Responses'
          : 'Claude',
    ariaLabel: option.label,
    value: option.value,
    path: option.path,
    icon:
      option.value === 'chat_completions'
        ? MessageSquare
        : option.value === 'responses'
          ? Braces
          : Bot,
  })),
])
const publicRequestURLTypeMeta = computed(
  () =>
    publicRequestURLTypeOptions.value.find((option) => option.value === publicRequestURLType.value) ??
    publicRequestURLTypeOptions.value[0],
)
const publicRequestEndpoints = computed<PublicRequestEndpoint[]>(() => {
  const defaultBaseURL = modelRequestGuide.value?.openai_base_url
  const extraEndpoints = modelRequestGuide.value?.extra_endpoints ?? []
  return [
    {
      key: 'default',
      label: t('默认 Endpoint', 'Default endpoint'),
      baseURL: defaultBaseURL || requestLoadingText.value,
    },
    ...extraEndpoints.map((endpoint, index) => ({
      key: `extra-${index}`,
      label: endpoint.description,
      baseURL: modelRequestOpenAIBaseURL(endpoint.url),
    })),
  ]
})
const publicRequestEndpointRows = computed(() => {
  const path = publicRequestURLTypeMeta.value?.path ?? ''
  return publicRequestEndpoints.value.map((endpoint) => ({
    ...endpoint,
    url:
      endpoint.baseURL === requestLoadingText.value
        ? endpoint.baseURL
        : `${endpoint.baseURL.replace(/\/$/, '')}${path}`,
  }))
})

function updatePublicRequestURLType(value: unknown) {
  if (publicRequestURLTypeOptions.value.some((option) => option.value === value)) {
    publicRequestURLType.value = value as PublicRequestURLType
  }
}

const requestEndpointMeta = computed(
  () =>
    requestEndpointOptions.value.find((option) => option.value === requestEndpoint.value) ??
    chatCompletionsEndpointOption.value,
)
const requestEndpointURL = computed(() => {
  const baseURL = modelRequestGuide.value?.openai_base_url
  if (!baseURL) {
    return requestLoadingText.value
  }
  return `${baseURL.replace(/\/$/, '')}${requestEndpointMeta.value.path}`
})
const requestEndpointURLLabel = computed(() => requestEndpointMeta.value.urlLabel)
const requestTestApiKeyText = computed(() => requestTestApiKey.value?.api_key || t('<你的 API KEY>', '<your API key>'))
const requestHeaderLines = computed(() => {
  if (requestEndpoint.value === 'claude_messages') {
    return [`x-api-key: ${requestTestApiKeyText.value}`, 'anthropic-version: 2023-06-01']
  }
  return [`Authorization: Bearer ${requestTestApiKeyText.value}`]
})
const requestHeadersText = computed(() => requestHeaderLines.value.join('\n'))
const sampleRequest = computed(() => {
  const targetURL =
    requestEndpointURL.value === requestLoadingText.value
      ? `<${requestEndpointURLLabel.value}>`
      : requestEndpointURL.value
  const model = requestTestModel.value || t('<模型名>', '<model name>')
  const content = requestTestMessage.value.trim() || t('你好', 'Hello')
  const body = requestBodyForEndpoint(requestEndpoint.value, model, content)
  return [
    `curl ${targetURL} \\`,
    ...requestHeaderLines.value.map((header) => `  -H "${header}" \\`),
    '  -H "Content-Type: application/json" \\',
    `  -d ${quoteForCurl(JSON.stringify(body))}`,
  ].join('\n')
})
const requestTestModelOptions = computed(() => {
  const selectedHash = requestTestApiKey.value?.api_key_hash
  const models = availableModels.value?.models ?? []
  const filtered = selectedHash
    ? models.filter((model) => model.sources.some((source) => source.api_key_hash === selectedHash))
    : models
  return filtered.map((model) => ({
    label: modelOptionLabel(model),
    value: model.id,
  }))
})
const requestTestReplyText = computed(() => {
  const reply = requestTestResult.value?.reply?.trim()
  return reply || t('模型返回成功，但没有可展示文本。', 'The model returned successfully, but there is no displayable text.')
})
const requestTestUsageText = computed(() => {
  const usage = requestTestResult.value?.usage
  if (!usage) {
    return ''
  }
  const input = numberFromUsage(usage.prompt_tokens ?? usage.input_tokens)
  const output = numberFromUsage(usage.completion_tokens ?? usage.output_tokens)
  const total = numberFromUsage(usage.total_tokens)
  const parts: string[] = []
  if (input !== null) {
    parts.push(t(`输入 ${formatInteger(input)}`, `Input ${formatInteger(input)}`))
  }
  if (output !== null) {
    parts.push(t(`输出 ${formatInteger(output)}`, `Output ${formatInteger(output)}`))
  }
  if (total !== null) {
    parts.push(t(`总计 ${formatInteger(total)}`, `Total ${formatInteger(total)}`))
  }
  return parts.join(' / ')
})

watch(requestEndpoint, () => {
  requestTestResult.value = null
  requestTestError.value = null
})

watch(currentLanguage, (language, previousLanguage) => {
  const previousDefault = requestTestMessageDefaults[previousLanguage]
  if (requestTestMessage.value === previousDefault) {
    requestTestMessage.value = requestTestMessageDefaults[language]
  }
})

function requestBodyForEndpoint(
  endpoint: ModelRequestEndpoint,
  model: string,
  content: string,
): Record<string, unknown> {
  if (endpoint === 'responses') {
    return {
      model,
      input: content,
      stream: false,
    }
  }
  if (endpoint === 'claude_messages') {
    return {
      model,
      max_tokens: 1024,
      messages: [{ role: 'user', content }],
    }
  }
  return {
    model,
    messages: [{ role: 'user', content }],
    stream: false,
  }
}

function quoteForCurl(value: string): string {
  return "'" + value.replace(/'/g, "'\"'\"'") + "'"
}

function modelRequestOpenAIBaseURL(value: string): string {
  const normalized = value.trim().replace(/\/+$/, '')
  if (!normalized) {
    return requestLoadingText.value
  }
  return /\/v1$/i.test(normalized) ? normalized : `${normalized}/v1`
}

function quotaValueText(quota: UserQuotaStatus | null): string {
  if (!quota) {
    return t('加载中', 'Loading')
  }
  if (quota.unlimited) {
    return t('无限制', 'Unlimited')
  }
  const total =
    (quota.daily_remaining_usd ?? 0) +
    (quota.weekly_remaining_usd ?? 0) +
    (quota.monthly_remaining_usd ?? 0) +
    (quota.lifetime_remaining_usd ?? 0)
  return formatUsd(total)
}

function quotaFootnote(quota: UserQuotaStatus | null): string {
  if (!quota) {
    return t('额度加载中', 'Quota loading')
  }
  if (quota.unlimited) {
    return t(
      '每日 无限制 / 每周 无限制 / 每月 无限制 / 不限时 无限制',
      'Daily unlimited / Weekly unlimited / Monthly unlimited / Lifetime unlimited',
    )
  }
  const balancesText = t(
    `每日 ${formatUsd(quota.daily_remaining_usd ?? 0)} / 每周 ${formatUsd(quota.weekly_remaining_usd ?? 0)} / 每月 ${formatUsd(quota.monthly_remaining_usd ?? 0)} / 不限时 ${formatUsd(quota.lifetime_remaining_usd ?? 0)}`,
    `Daily ${formatUsd(quota.daily_remaining_usd ?? 0)} / Weekly ${formatUsd(quota.weekly_remaining_usd ?? 0)} / Monthly ${formatUsd(quota.monthly_remaining_usd ?? 0)} / Lifetime ${formatUsd(quota.lifetime_remaining_usd ?? 0)}`,
  )
  const notes: string[] = []
  if (quota.sync_error) {
    notes.push(t('Key 同步异常', 'Key sync error'))
  }
  if (quota.unpriced_records > 0) {
    notes.push(t(`未定价 ${formatInteger(quota.unpriced_records)} 条`, `${formatInteger(quota.unpriced_records)} unpriced records`))
  }
  if (quota.paused) {
    notes.push(t('Key 已因余额暂停', 'Key paused due to balance'))
  }
  return notes.length > 0 ? `${balancesText} · ${notes.join(' · ')}` : balancesText
}

function modelOptionLabel(model: AvailableModel): string {
  return model.id
}

function numberFromUsage(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return null
  }
  return value
}

function ensureRequestTestModel() {
  const options = requestTestModelOptions.value
  if (options.length === 0) {
    requestTestModel.value = null
    return
  }
  if (!requestTestModel.value || !options.some((option) => option.value === requestTestModel.value)) {
    const firstOption = options[0]
    requestTestModel.value = firstOption ? firstOption.value : null
  }
}

function displayedApiKey(row: UserApiKeySummary): string {
  if (row.api_key && isApiKeyVisible(row)) {
    return row.api_key
  }
  return maskDisplayedApiKey(row.api_key)
}

function maskDisplayedApiKey(apiKey: string | null | undefined): string {
  if (!apiKey) {
    return t('未知', 'Unknown')
  }
  if (apiKey.length <= 12) {
    return `${apiKey.slice(0, 3)}${'*'.repeat(Math.max(apiKey.length - 3, 0))}`
  }
  const visiblePrefix = apiKey.startsWith('sk-') ? 4 : 6
  const visibleSuffix = 4
  const maskedLength = Math.max(apiKey.length - visiblePrefix - visibleSuffix, 8)
  return `${apiKey.slice(0, visiblePrefix)}${'*'.repeat(maskedLength)}${apiKey.slice(-visibleSuffix)}`
}

function renderMaskedKeyTitle() {
  return h('span', { class: 'api-key-title' }, [
    h(AppIcon, { class: 'api-key-mask-icon', component: EyeOff }),
    h('span', t('密钥（点击复制）', 'Key (click to copy)')),
  ])
}

function isApiKeyVisible(row: UserApiKeySummary): boolean {
  return visibleApiKeyHashes.value.has(row.api_key_hash)
}

function toggleApiKeyVisibility(row: UserApiKeySummary) {
  if (!row.api_key) {
    message.info(t('当前没有完整密钥可显示', 'No full key is available to show'))
    return
  }
  const nextVisible = new Set(visibleApiKeyHashes.value)
  if (nextVisible.has(row.api_key_hash)) {
    nextVisible.delete(row.api_key_hash)
  } else {
    nextVisible.add(row.api_key_hash)
  }
  visibleApiKeyHashes.value = nextVisible
}

async function copyApiKey(row: UserApiKeySummary) {
  try {
    if (!row.api_key) {
      message.info(t('当前没有完整密钥可复制', 'No full key is available to copy'))
      return
    }
    await copyToClipboard(row.api_key)
    message.success(t('API 密钥已复制', 'API key copied'))
  } catch (error) {
    message.error(errorText(error, '复制失败', 'Copy failed'))
  }
}

async function copyGeneratedApiKey() {
  if (!generatedApiKey.value) {
    return
  }
  try {
    await copyToClipboard(generatedApiKey.value)
    message.success(t('API 密钥已复制', 'API key copied'))
  } catch (error) {
    message.error(errorText(error, '复制失败', 'Copy failed'))
  }
}

async function loadModelRequestGuide() {
  try {
    modelRequestGuide.value = await getModelRequestGuide()
  } catch (error) {
    message.error(errorText(error, '加载请求地址失败', 'Failed to load request endpoints'))
  }
}

async function loadAvailableModelsForTest() {
  isAvailableModelsLoading.value = true
  try {
    availableModels.value = await listAvailableModels()
    ensureRequestTestModel()
  } catch (error) {
    message.error(errorText(error, '加载可用模型失败', 'Failed to load available models'))
  } finally {
    isAvailableModelsLoading.value = false
  }
}

function openRequestTest(row: UserApiKeySummary) {
  requestTestApiKey.value = row
  requestTestModel.value = row.last_model ?? row.models[0] ?? null
  requestTestMessage.value = requestTestMessageDefaults[currentLanguage.value]
  requestTestResult.value = null
  requestTestError.value = null
  requestTestVisible.value = true
  if (!modelRequestGuide.value) {
    void loadModelRequestGuide()
  }
  if (!availableModels.value) {
    void loadAvailableModelsForTest()
  } else {
    ensureRequestTestModel()
  }
}

async function copyRequestValue(label: string, value: string) {
  if (!value || value === requestLoadingText.value) {
    return
  }
  try {
    await copyToClipboard(value)
    message.success(copiedText(label))
  } catch (error) {
    message.error(errorText(error, '复制失败', 'Copy failed'))
  }
}

async function runRequestTest() {
  if (isRequestTesting.value) {
    return
  }
  const currentKey = requestTestApiKey.value
  const model = requestTestModel.value?.trim() ?? ''
  if (!currentKey) {
    message.error(t('请选择要测试的 API KEY', 'Select an API key to test'))
    return
  }
  if (!model) {
    message.error(t('请选择测试模型', 'Select a test model'))
    return
  }
  isRequestTesting.value = true
  requestTestResult.value = null
  requestTestError.value = null
  try {
    requestTestResult.value = await testModelRequest({
      api_key_hash: currentKey.api_key_hash,
      endpoint: requestEndpoint.value,
      model,
      message: requestTestMessage.value,
    })
    message.success(t('请求测试完成', 'Request test completed'))
  } catch (error) {
    requestTestError.value = errorText(error, '请求测试失败', 'Request test failed')
  } finally {
    isRequestTesting.value = false
  }
}

function openCreateDialog() {
  if (!canCreateApiKey.value) {
    message.error(t('当前账号额度已用尽，API KEY 已暂停', 'This account has exhausted its quota, so API keys are paused'))
    return
  }
  editingApiKeyHash.value = null
  apiKeyDescription.value = 'VSCode'
  generatedApiKey.value = null
  generatedApiKeyHash.value = null
  editorVisible.value = true
}

function closeGeneratedApiKey() {
  generatedApiKey.value = null
  generatedApiKeyHash.value = null
}

function editApiKey(row: UserApiKeySummary) {
  editingApiKeyHash.value = row.api_key_hash
  apiKeyDescription.value = row.description || 'VSCode'
  generatedApiKey.value = null
  generatedApiKeyHash.value = null
  editorVisible.value = true
}

async function refresh() {
  isLoading.value = true
  try {
    const [nextApiKeys, overview, quota, guide] = await Promise.all([
      listApiKeys(),
      getUsageOverview({ scope: 'account' }, false),
      getCurrentUserQuota(),
      getModelRequestGuide(),
    ])
    apiKeys.value = nextApiKeys
    usageSummary.value = overview.summary
    quotaStatus.value = quota
    modelRequestGuide.value = guide
    if (editingApiKeyHash.value) {
      const current = apiKeys.value.find((item) => item.api_key_hash === editingApiKeyHash.value)
      if (!current) {
        editorVisible.value = false
        editingApiKeyHash.value = null
      }
    }
  } catch (error) {
    message.error(errorText(error, '加载 API 密钥失败', 'Failed to load API keys'))
  } finally {
    isLoading.value = false
  }
}

async function saveApiKey() {
  if (isSaving.value) {
    return
  }
  const description = apiKeyDescription.value.trim()
  if (!description) {
    message.error(t('API KEY 描述不能为空', 'API key description is required'))
    return
  }
  isSaving.value = true
  try {
    if (editingApiKeyHash.value) {
      await updateApiKey(editingApiKeyHash.value, { description })
      message.success(t('API 密钥已更新', 'API key updated'))
    } else {
      if (!canCreateApiKey.value) {
        message.error(t('当前账号额度已用尽，API KEY 已暂停', 'This account has exhausted its quota, so API keys are paused'))
        return
      }
      const created = await createApiKey({ description })
      generatedApiKey.value = created.api_key ?? null
      generatedApiKeyHash.value = created.api_key_hash
      message.success(t('API 密钥已创建并同步到 CPA', 'API key created and synced to CPA'))
    }
    editorVisible.value = false
    editingApiKeyHash.value = null
    await refresh()
  } catch (error) {
    message.error(errorText(error, '保存 API 密钥失败', 'Failed to save API key'))
  } finally {
    isSaving.value = false
  }
}

function confirmDelete(row: UserApiKeySummary) {
  dialog.warning({
    title: t('删除 API 密钥', 'Delete API key'),
    content: t(
      `将删除 ${row.description || '未命名'} 对应的密钥，并从 CPA 中移除。`,
      `This deletes the key for ${row.description || 'Unnamed'} and removes it from CPA.`,
    ),
    positiveText: t('删除', 'Delete'),
    negativeText: t('取消', 'Cancel'),
    onPositiveClick: async () => {
      await deleteApiKey(row.api_key_hash)
      message.success(t('API 密钥已删除', 'API key deleted'))
      if (editingApiKeyHash.value === row.api_key_hash) {
        editorVisible.value = false
        editingApiKeyHash.value = null
      }
      if (generatedApiKeyHash.value === row.api_key_hash) {
        generatedApiKey.value = null
        generatedApiKeyHash.value = null
      }
      await refresh()
    },
  })
}

const columns = computed<DataTableColumns<UserApiKeySummary>>(() => [
  {
    title: renderMaskedKeyTitle,
    key: 'api_key',
    width: 540,
    render: (row) =>
      h(
        'div',
        { class: 'api-key-cell' },
        [
          h(
            'button',
            {
              class: 'api-key-visibility-button',
              disabled: !row.api_key,
              title: isApiKeyVisible(row) ? t('隐藏完整密钥', 'Hide full key') : t('显示完整密钥', 'Show full key'),
              type: 'button',
              onClick: () => toggleApiKeyVisibility(row),
            },
            [
              h(AppIcon, {
                class: 'api-key-mask-icon',
                component: isApiKeyVisible(row) ? Eye : EyeOff,
              }),
            ],
          ),
          h(
            'button',
            {
              class: 'api-key-copy-button',
              type: 'button',
              title: row.api_key ? t('点击复制完整密钥', 'Click to copy full key') : t('无完整密钥可复制', 'No full key available to copy'),
              onClick: () => copyApiKey(row),
            },
            h('span', { class: 'api-key-mask-text' }, displayedApiKey(row)),
          ),
        ],
      ),
  },
  {
    title: t('描述', 'Description'),
    key: 'description',
    width: 160,
    render: (row) =>
      row.description
        ? h(AppEllipsis, { tooltip: true, style: { maxWidth: '100%' } }, { default: () => row.description })
        : '-',
  },
  {
    title: t('创建时间', 'Created at'),
    key: 'created_at',
    width: 150,
    render: (row) => formatDateTime(row.created_at),
  },
  {
    title: '',
    key: 'actions',
    width: 230,
    fixed: 'right',
    render: (row) =>
      h(AppStack, { size: 4 }, {
        default: () => [
          h(
            AppButton,
            { size: 'small', quaternary: true, onClick: () => openRequestTest(row) },
            {
              icon: () => h(AppIcon, { component: Send }),
              default: () => t('请求测试', 'Request test'),
            },
          ),
          h(
            AppButton,
            { size: 'small', quaternary: true, onClick: () => editApiKey(row) },
            { default: () => t('编辑', 'Edit') },
          ),
          h(
            AppButton,
            { size: 'small', quaternary: true, type: 'error', onClick: () => confirmDelete(row) },
            { default: () => t('删除', 'Delete') },
          ),
        ],
      }),
  },
])

onMounted(refresh)
</script>

<template>
  <section class="page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('API 密钥', 'API keys') }}</h1>
      <AppStack>
        <AppButton secondary :loading="isLoading" @click="refresh">{{ t('刷新', 'Refresh') }}</AppButton>
        <AppButton type="primary" :disabled="!canCreateApiKey" @click="openCreateDialog">
          {{ t('新建 API 密钥', 'New API key') }}
        </AppButton>
      </AppStack>
    </div>

    <div class="metric-grid api-key-metrics">
      <div v-for="metric in apiKeyMetrics" :key="metric.key" class="metric-card" :class="`is-${metric.tone}`">
        <div class="metric-icon" aria-hidden="true">
          <component :is="metric.icon" :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ metric.label }}</div>
        <div class="metric-value">{{ metric.value }}</div>
        <div class="metric-footnote">{{ metric.footnote }}</div>
      </div>
    </div>

    <div class="grid-two api-key-content-grid">
      <section class="panel api-key-panel-shell">
        <div class="panel-inner api-key-panel">
          <AppAlert v-if="quotaStatus?.paused" type="error" :bordered="false" :title="t('额度已用尽', 'Quota exhausted')">
            {{ t('当前账号 API KEY 已从 CPA 暂停。补充额度或进入新的日、周、月周期后，系统会自动恢复可用 Key。', 'API keys for this account are paused in CPA. Available keys are restored automatically after quota is added or a new daily, weekly, or monthly period begins.') }}
          </AppAlert>
          <AppAlert v-else-if="quotaStatus?.unpriced_records" type="warning" :bordered="false">
            {{ t(`当前账号存在 ${formatInteger(quotaStatus.unpriced_records)} 条未定价用量，未计入额度扣减。`, `This account has ${formatInteger(quotaStatus.unpriced_records)} unpriced usage records that are not deducted from quota.`) }}
          </AppAlert>

          <div v-if="generatedApiKey" class="generated-key-box">
            <div class="generated-key-main">
              <div class="generated-key-title">{{ t('新创建的密钥', 'Newly created key') }}</div>
              <div class="generated-key-value">{{ generatedApiKey }}</div>
            </div>
            <AppStack>
              <AppButton secondary @click="copyGeneratedApiKey">{{ t('复制', 'Copy') }}</AppButton>
              <AppButton tertiary @click="closeGeneratedApiKey">{{ t('关闭', 'Close') }}</AppButton>
            </AppStack>
          </div>

          <AppDataTable
            class="api-key-table"
            size="small"
            :loading="isLoading"
            :columns="columns"
            :data="apiKeys"
            :pagination="{ pageSize: 12 }"
            table-layout="fixed"
            :scroll-x="1080"
          />
        </div>
      </section>

      <section class="panel api-endpoint-panel-shell">
        <div class="panel-inner api-endpoint-panel">
          <h2 class="section-title">API Endpoint</h2>
          <div class="api-endpoint-type-picker">
            <span id="api-endpoint-url-type-label" class="request-endpoint-label">{{ t('URL 类型', 'URL type') }}</span>
            <ToggleGroup
              :model-value="publicRequestURLType"
              class="api-endpoint-type-options"
              type="single"
              variant="outline"
              size="default"
              aria-labelledby="api-endpoint-url-type-label"
              @update:model-value="updatePublicRequestURLType"
            >
              <ToggleGroupItem
                v-for="option in publicRequestURLTypeOptions"
                :key="option.value"
                :value="option.value"
                :aria-label="option.ariaLabel"
                :title="option.ariaLabel"
              >
                <component :is="option.icon" data-icon="inline-start" aria-hidden="true" />
                <span>{{ option.label }}</span>
              </ToggleGroupItem>
            </ToggleGroup>
          </div>
          <div class="request-guide-list">
            <div v-for="endpoint in publicRequestEndpointRows" :key="endpoint.key" class="request-guide-row">
              <div>
                <div class="request-guide-label">{{ endpoint.label }}</div>
                <code class="request-guide-value">{{ endpoint.url }}</code>
              </div>
              <AppButton size="small" secondary @click="copyRequestValue(endpoint.label, endpoint.url)">
                <template #icon>
                  <AppIcon :component="Copy" />
                </template>
                {{ t('复制', 'Copy') }}
              </AppButton>
            </div>
          </div>
        </div>
      </section>
    </div>

    <AppModal
      v-model:show="editorVisible"
      preset="card"
      :mask-closable="false"
      :closable="false"
      :title="editingApiKeyHash ? t('编辑 API 密钥', 'Edit API key') : t('新建 API 密钥', 'New API key')"
      :style="{ width: 'min(520px, calc(100vw - 32px))' }"
    >
      <AppForm label-placement="top">
        <AppFormItem :label="t('API KEY 描述', 'API key description')">
          <AppInput
            v-model:value="apiKeyDescription"
            :disabled="isSaving"
            :placeholder="t('例如：VSCode', 'Example: VSCode')"
            @keyup.enter="saveApiKey"
          />
        </AppFormItem>
        <div class="modal-actions api-key-editor-actions">
          <AppButton secondary :disabled="isSaving" @click="editorVisible = false">{{ t('取消', 'Cancel') }}</AppButton>
          <AppButton
            type="primary"
            :loading="isSaving"
            :disabled="isSaving || (!editingApiKeyHash && !canCreateApiKey)"
            @click="saveApiKey"
          >
            {{ editingApiKeyHash ? t('保存', 'Save') : t('创建', 'Create') }}
          </AppButton>
        </div>
      </AppForm>
    </AppModal>

    <AppModal
      v-model:show="requestTestVisible"
      preset="card"
      :title="t('请求测试', 'Request test')"
      :style="{ width: 'min(760px, calc(100vw - 32px))' }"
    >
      <div class="request-test">
        <AppAlert type="info" :bordered="false">
          {{ t('这里提供当前 API KEY 的请求说明，也可以直接选择模型发起一次真实测试。', 'This shows request instructions for the current API key. You can also choose a model to run a real test request.') }}
        </AppAlert>

        <div class="request-endpoint-switch">
          <span class="request-endpoint-label">{{ t('请求格式', 'Request format') }}</span>
          <AppRadioGroup v-model:value="requestEndpoint" size="small">
            <AppRadioButton
              v-for="option in requestEndpointOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </AppRadioButton>
          </AppRadioGroup>
        </div>

        <div class="request-guide-list">
          <div class="request-guide-row">
            <div>
              <div class="request-guide-label">{{ t('基础 URL', 'Base URL') }}</div>
              <code class="request-guide-value">{{ requestBaseURL }}</code>
            </div>
            <AppButton size="small" secondary @click="copyRequestValue(t('基础 URL', 'Base URL'), requestBaseURL)">
              <template #icon>
                <AppIcon :component="Copy" />
              </template>
              {{ t('复制', 'Copy') }}
            </AppButton>
          </div>
          <div class="request-guide-row">
            <div>
              <div class="request-guide-label">{{ requestEndpointURLLabel }}</div>
              <code class="request-guide-value">{{ requestEndpointURL }}</code>
            </div>
            <AppButton size="small" secondary @click="copyRequestValue(t('请求 URL', 'Request URL'), requestEndpointURL)">
              <template #icon>
                <AppIcon :component="Copy" />
              </template>
              {{ t('复制', 'Copy') }}
            </AppButton>
          </div>
          <div class="request-guide-row">
            <div>
              <div class="request-guide-label">{{ t('请求 Header', 'Request headers') }}</div>
              <code class="request-guide-value request-guide-value-multiline">{{ requestHeadersText }}</code>
            </div>
            <AppButton size="small" secondary @click="copyRequestValue(t('请求 Header', 'Request headers'), requestHeadersText)">
              <template #icon>
                <AppIcon :component="Copy" />
              </template>
              {{ t('复制', 'Copy') }}
            </AppButton>
          </div>
        </div>

        <div class="request-example">
          <div class="request-example-head">
            <span>{{ t('curl 示例', 'curl example') }}</span>
            <AppButton size="small" secondary @click="copyRequestValue(t('curl 示例', 'curl example'), sampleRequest)">
              <template #icon>
                <AppIcon :component="Copy" />
              </template>
              {{ t('复制示例', 'Copy example') }}
            </AppButton>
          </div>
          <pre>{{ sampleRequest }}</pre>
        </div>

        <div class="request-test-section-title">{{ t('请求测试', 'Request test') }}</div>

        <AppForm label-placement="top" class="request-test-form">
          <AppFormItem :label="t('测试模型', 'Test model')">
            <AppSelect
              v-model:value="requestTestModel"
              filterable
              clearable
              :loading="isAvailableModelsLoading"
              :options="requestTestModelOptions"
              :placeholder="t('选择当前 Key 可用的模型', 'Select a model available to this key')"
            />
          </AppFormItem>
          <AppFormItem :label="t('测试消息', 'Test message')">
            <AppInput
              v-model:value="requestTestMessage"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 5 }"
              :placeholder="t('输入要发送给模型的测试消息', 'Enter the test message to send to the model')"
            />
          </AppFormItem>
        </AppForm>

        <AppAlert
          v-if="!isAvailableModelsLoading && requestTestModelOptions.length === 0"
          type="warning"
          :bordered="false"
        >
          {{ t('当前 Key 暂未查询到可选模型，可以先刷新模型列表，或到「可用模型」页面检查 Key 是否可用。', 'No selectable models were found for this key. Refresh the model list, or check whether the key is available on the Available models page.') }}
        </AppAlert>

        <div class="modal-actions request-test-actions">
          <AppButton secondary :loading="isAvailableModelsLoading" @click="loadAvailableModelsForTest">
            {{ t('刷新模型', 'Refresh models') }}
          </AppButton>
          <AppButton
            type="primary"
            :loading="isRequestTesting"
            :disabled="!requestTestModel || isAvailableModelsLoading"
            @click="runRequestTest"
          >
            <template #icon>
              <AppIcon :component="Send" />
            </template>
            {{ t('发送测试', 'Send test') }}
          </AppButton>
        </div>

        <AppAlert v-if="requestTestError" type="error" :bordered="false">
          {{ requestTestError }}
        </AppAlert>

        <div v-if="requestTestResult" class="request-test-result">
          <div class="request-test-result-head">
            <span>{{ t('模型回复', 'Model reply') }}</span>
            <span>
              HTTP {{ requestTestResult.status_code }} · {{ requestTestResult.duration_ms }}ms
              <template v-if="requestTestUsageText"> · {{ requestTestUsageText }}</template>
            </span>
          </div>
          <pre>{{ requestTestReplyText }}</pre>
        </div>
      </div>
    </AppModal>
  </section>
</template>

<style scoped>
.api-key-panel {
  display: grid;
  gap: 14px;
  min-width: 0;
}

.api-key-content-grid {
  align-items: start;
}

.api-endpoint-panel {
  display: grid;
  gap: 12px;
  min-width: 0;
}

.api-endpoint-panel .section-title {
  margin-bottom: 0;
}

.api-endpoint-type-options {
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.api-endpoint-type-options :deep([data-slot="toggle-group-item"]) {
  flex: 1 1 0;
  min-width: 0;
  min-height: 36px;
  padding-inline: 8px;
  border-color: var(--cpa-border-strong);
  color: var(--cpa-text-muted);
  cursor: pointer;
  box-shadow: none;
}

.api-endpoint-type-options :deep([data-slot="toggle-group-item"]:hover) {
  color: var(--cpa-text-strong);
}

.api-endpoint-type-options :deep([data-slot="toggle-group-item"][data-state="on"]) {
  z-index: 1;
  border-color: var(--cpa-primary);
  background: var(--cpa-primary);
  color: var(--cpa-primary-foreground);
}

.api-endpoint-type-picker {
  display: grid;
  justify-items: start;
  gap: 8px;
  min-width: 0;
}

.api-key-metrics {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.api-key-panel-shell,
.api-endpoint-panel-shell,
.api-key-table {
  min-width: 0;
  min-height: 0;
}

.api-key-table :deep(.n-data-table-wrapper) {
  overflow: hidden;
}

.generated-key-box {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-primary-wash);
  box-shadow: var(--cpa-shadow-hairline);
}

.generated-key-main {
  min-width: 0;
}

.generated-key-title {
  margin-bottom: 4px;
  font-weight: 700;
}

.generated-key-value {
  overflow-wrap: anywhere;
  font-family: Consolas, 'SFMono-Regular', 'Microsoft YaHei UI', monospace;
  font-size: 13px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.api-key-editor-actions {
  margin-top: 12px;
}

.request-test {
  display: grid;
  gap: 14px;
}

.request-endpoint-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px 12px;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface-muted);
}

.request-endpoint-label {
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.request-guide-list {
  display: grid;
  overflow: hidden;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface);
}

.request-guide-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px 14px;
  border-bottom: 1px solid var(--cpa-border);
}

.request-guide-row:last-child {
  border-bottom: 0;
}

.request-guide-label {
  margin-bottom: 4px;
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.request-guide-value {
  display: block;
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--cpa-text);
  font-family: Consolas, 'SFMono-Regular', 'Microsoft YaHei UI', monospace;
  font-size: 13px;
  line-height: 1.45;
}

.request-guide-value-multiline {
  white-space: pre-wrap;
}

.request-example,
.request-test-form {
  display: grid;
}

.request-example {
  overflow: hidden;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface);
}

.request-example-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--cpa-border);
  font-weight: 700;
}

.request-example pre {
  overflow: auto;
  margin: 0;
  padding: 14px;
  background: var(--cpa-surface-muted);
  color: var(--cpa-text);
  font-family: Consolas, 'SFMono-Regular', 'Microsoft YaHei UI', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}

.request-test-section-title {
  color: var(--cpa-text);
  font-size: 14px;
  font-weight: 700;
}

.request-test-form {
  gap: 2px;
}

.request-test-actions {
  align-items: center;
}

.request-test-result {
  overflow: hidden;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface);
}

.request-test-result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--cpa-border);
  font-weight: 700;
}

.request-test-result-head span:last-child {
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 500;
}

.request-test-result pre {
  overflow: auto;
  margin: 0;
  padding: 14px;
  background: var(--cpa-surface-muted);
  color: var(--cpa-text);
  font-family: Consolas, 'SFMono-Regular', 'Microsoft YaHei UI', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}

:global(.api-key-cell) {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-width: 0;
}

:global(.api-key-visibility-button),
:global(.api-key-copy-button) {
  border: 0;
  background: transparent;
  color: var(--cpa-text);
  font: inherit;
  cursor: pointer;
}

:global(.api-key-visibility-button) {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border-radius: 6px;
}

:global(.api-key-copy-button) {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  padding: 0;
  line-height: 1.35;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.api-key-title) {
  display: inline-flex;
  align-items: center;
  gap: 12px;
}

:global(.api-key-mask-icon) {
  flex: 0 0 auto;
  color: var(--cpa-text-muted);
}

:global(.api-key-mask-text) {
  display: block;
  min-width: 0;
  overflow: hidden;
  font-family: Consolas, 'SFMono-Regular', 'Microsoft YaHei UI', monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.api-key-visibility-button:hover),
:global(.api-key-visibility-button:focus-visible),
:global(.api-key-copy-button:hover),
:global(.api-key-copy-button:focus-visible) {
  color: var(--cpa-primary);
}

:global(.api-key-visibility-button:disabled) {
  color: var(--cpa-text-muted);
  cursor: not-allowed;
  opacity: 0.56;
}

:global(.api-key-visibility-button:focus-visible),
:global(.api-key-copy-button:focus-visible) {
  outline: 2px solid color-mix(in srgb, var(--cpa-primary) 32%, transparent);
  outline-offset: 2px;
}

@media (max-width: 900px) {
  .api-key-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .generated-key-box {
    flex-direction: column;
  }

  .request-guide-row {
    grid-template-columns: 1fr;
  }

  .request-example-head,
  .request-test-result-head {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 720px) {
  .api-key-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 430px) {
  .api-key-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
