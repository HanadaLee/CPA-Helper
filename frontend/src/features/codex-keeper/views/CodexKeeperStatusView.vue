<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  AppButton,
  AppDataTable,
  AppDescriptions,
  AppDescriptionsItem,
  AppDrawer,
  AppDrawerContent,
  AppDropdown,
  AppIcon,
  AppInput,
  AppNumberInput,
  AppModal,
  AppPagination,
  AppSelect,
  AppStack,
  AppSwitch,
  AppBadge,
  useMessage,
  type DataTableColumns,
  type DataTableRowKey,
} from '@/shared/ui/app-kit'
import {
  Activity,
  ArrowLeft,
  Copy,
  ExternalLink,
  Gauge,
  LogIn,
  PauseCircle,
  Pencil,
  RefreshCw,
  ShieldAlert,
  ShieldCheck,
  Trash2,
  Upload,
  Users,
} from '@lucide/vue'

import {
  bulkDeleteCodexKeeperAccounts,
  deleteCodexKeeperAccount,
  disableCodexKeeperAccount,
  enableCodexKeeperAccount,
  getCodexKeeperAuthFile,
  getCodexKeeperOAuthStatus,
  getCodexKeeperStatus,
  listCodexKeeperAccounts,
  refreshCodexKeeperAccounts,
  runCodexKeeperOnce,
  startCodexKeeperOAuth,
  submitCodexKeeperOAuthCallback,
  syncCodexKeeperAccountList,
  updateCodexKeeperAuthFile,
  updateCodexKeeperPriority,
  uploadCodexKeeperAuthFiles,
} from '@/features/codex-keeper/api/codexKeeperApi'
import CodexKeeperLogsPanel from '@/features/codex-keeper/components/CodexKeeperLogsPanel.vue'
import { useCurrentUser } from '@/features/auth/state/currentUser'
import type {
  CodexKeeperAccount,
  CodexKeeperAuthFileFields,
  CodexKeeperPriorityRule,
  CodexKeeperQuotaWindowUsage,
  CodexKeeperStatus,
} from '@/shared/types/api'
import { useI18n } from '@/shared/i18n'
import { copyToClipboard } from '@/shared/utils/clipboard'
import {
  BEIJING_TIME_ZONE,
  formatCompact,
  formatDateTime,
  formatInteger,
  formatRelativeTime,
  formatUsd,
} from '@/shared/utils/format'

type FixedPriorityFilter = 'all' | 'high' | 'minusOne' | 'low'
type PriorityTypeFilter = `type:${string}`
type PriorityFilter = FixedPriorityFilter | PriorityTypeFilter
type AccountStatusFilter = 'all' | 'enabled' | 'disabled' | 'unauthorized' | 'quotaExhausted'
type AccountDisplaySize = 50 | 100 | 150 | 200
type AccountSortKey = 'quotaDay' | 'quotaWeek' | 'accountType' | 'status' | 'priority' | 'lastCheckedAt'
type SortDirection = 'asc' | 'desc'
type PriorityMode = 'low' | 'high' | 'default'
type AccountAction = 'toggle' | 'priority' | 'delete' | 'refresh'
type AccountConfirmType = 'default' | 'warning' | 'error' | 'primary'
type OAuthDialogStatus = 'idle' | 'waiting' | 'success' | 'error'
type KeeperPollMode = 'once' | 'accounts'
type QuotaWindowItem = {
  label: string
  usedPercent: number
  remainingPercent: number
  resetAt: string | null
  usage: CodexKeeperQuotaWindowUsage | null
}
type QuotaUsageTag = { label: string; value: string; tone?: 'stale' }
type AccountStatusPreferences = {
  displaySize?: unknown
  sort?: {
    key?: unknown
    direction?: unknown
  }
}
type AuthFileEditorState = {
  fileName: string
  fileInfoText: string
  loading: boolean
  saving: boolean
  error: string | null
  rawText: string
  invalidContentPreview: string
  json: Record<string, unknown> | null
  prefix: string
  proxyUrl: string
  priority: string
  websockets: boolean
  websocketsTouched: boolean
  note: string
  noteTouched: boolean
  headersText: string
  headersTouched: boolean
  headersError: string | null
}

const ACCOUNT_STATUS_PREFERENCE_STORAGE_KEY = 'cpa-helper-codex-keeper-status-preferences'
const AUTH_FILE_MAX_SIZE = 10 * 1024 * 1024
const CODEX_FIVE_HOUR_WINDOW_SECONDS = 5 * 60 * 60
const CODEX_WEEK_WINDOW_SECONDS = 7 * 24 * 60 * 60
const CODEX_MONTH_WINDOW_SECONDS = 30 * 24 * 60 * 60
const accountManageTableScrollX = 1622
const accountReadOnlyTableScrollX = 1426
const KEEPER_STATUS_POLL_INTERVAL_MS = 3000
const REFRESH_STATUS_POLL_INTERVAL_MS = 1500
const OAUTH_STATUS_POLL_INTERVAL_MS = 3000
const message = useMessage()
const { currentLanguage, errorText, keeperStatusText, serverText, t } = useI18n()
const { currentUser } = useCurrentUser()
const canManageAccounts = computed(() => currentUser.value?.is_admin === true)
const accountTableScrollX = computed(() =>
  canManageAccounts.value ? accountManageTableScrollX : accountReadOnlyTableScrollX,
)
const isLoading = ref(false)
const isBulkDeleting = ref(false)
const isBulkRefreshing = ref(false)
const bulkToggleAction = ref<'enable' | 'disable' | null>(null)
const isBulkOperationRunning = computed(
  () => isBulkDeleting.value || isBulkRefreshing.value || bulkToggleAction.value !== null,
)
const actingActions = ref<Set<string>>(new Set())
const accounts = ref<CodexKeeperAccount[]>([])
const priorityRules = ref<CodexKeeperPriorityRule[]>([])
const keeperStatus = ref<CodexKeeperStatus | null>(null)
const selectedAccount = ref<CodexKeeperAccount | null>(null)
const selectedAccountNote = ref<string | null>(null)
const isSelectedAccountNoteLoading = ref(false)
let selectedAccountNoteRequestID = 0
const selectedAccountKeys = ref<DataTableRowKey[]>([])
const detailOpen = ref(false)
const authFileInput = ref<HTMLInputElement | null>(null)
const isUploadingAuthFiles = ref(false)
const isStartingAccountInspection = ref(false)
const authFileEditor = ref<AuthFileEditorState | null>(null)
const oauthDialogOpen = ref(false)
const oauthDialogStatus = ref<OAuthDialogStatus>('idle')
const oauthAuthURL = ref('')
const oauthCallbackURL = ref('')
const oauthError = ref('')
const isStartingOAuth = ref(false)
const isSubmittingOAuthCallback = ref(false)
const accountDisplaySize = ref<AccountDisplaySize>(50)
const accountListPage = ref(1)
const relativeTimeNow = ref(Date.now())
const filters = reactive({
  keyword: '',
  accountType: null as string | null,
  priority: 'all' as PriorityFilter,
  status: 'all' as AccountStatusFilter,
})
const accountSort = reactive({
  key: null as AccountSortKey | null,
  direction: 'asc' as SortDirection,
})
const bulkDeleteDialog = reactive({
  show: false,
})
const accountConfirmDialog = reactive({
  show: false,
  title: '',
  content: '',
  positiveText: '',
  type: 'warning' as AccountConfirmType,
  action: null as (() => Promise<void>) | null,
})
const isAccountConfirmSubmitting = ref(false)
const priorityDialog = reactive({
  show: false,
  mode: 'low' as PriorityMode,
  account: null as CodexKeeperAccount | null,
  value: null as number | null,
})
const keeperPollTokens: Record<KeeperPollMode, number> = { once: 0, accounts: 0 }
let keeperStatusTimer: number | undefined
let oauthPollToken = 0

const oauthStatusType = computed(() => {
  switch (oauthDialogStatus.value) {
    case 'waiting':
      return 'warning'
    case 'success':
      return 'success'
    case 'error':
      return 'error'
    default:
      return 'default'
  }
})
const oauthStatusText = computed(() => {
  switch (oauthDialogStatus.value) {
    case 'waiting':
      return t('等待认证中', 'Waiting for authentication')
    case 'success':
      return t('认证成功', 'Authentication successful')
    case 'error':
      return t('认证失败', 'Authentication failed')
    default:
      return t('尚未开始', 'Not started')
  }
})

const priorityRuleMap = computed(() =>
  Object.fromEntries(priorityRules.value.map((rule) => [rule.account_type, rule.priority])),
)
const priorityFilterOptions = computed<Array<{ label: string; value: PriorityFilter }>>(() => [
  { label: t('全部优先级', 'All Priorities'), value: 'all' },
  { label: t('手动优先 >20', 'Manual Priority >20'), value: 'high' },
  ...[...priorityRules.value]
    .filter((rule) => rule.priority >= 0 && rule.priority <= 20)
    .sort((left, right) => {
      const priorityDiff = right.priority - left.priority
      return priorityDiff === 0
        ? left.account_type.localeCompare(right.account_type)
        : priorityDiff
    })
    .map((rule) => ({
      label: `${formatInteger(rule.priority)} (${rule.account_type})`,
      value: priorityTypeFilter(rule.account_type),
    })),
  { label: t('临时降级', 'Temporary Downgrade'), value: 'minusOne' },
  { label: t('手动低优先 <-1', 'Manual Low Priority <-1'), value: 'low' },
])
const accountDisplaySizeOptions = computed<Array<{ label: string; value: AccountDisplaySize }>>(() => [
  { label: '50', value: 50 },
  { label: '100', value: 100 },
  { label: '150', value: 150 },
  { label: '200', value: 200 },
])
const quotaSortOptions = computed(() => [
  { label: t('天', 'Day'), key: 'quotaDay' },
  { label: t('月/周', 'Month/Week'), key: 'quotaWeek' },
])

const accountTypeOptions = computed(() =>
  [...new Set(accounts.value.map((item) => item.account_type).filter(Boolean))]
    .sort((a, b) => String(a).localeCompare(String(b)))
    .map((value) => ({ label: accountTypeLabel(String(value)), value: String(value) })),
)

const filteredAccounts = computed(() =>
  accounts.value.filter((account) => {
    const keyword = filters.keyword.trim().toLowerCase()
    if (
      keyword &&
      ![account.name, account.email ?? ''].some((value) => value.toLowerCase().includes(keyword))
    ) {
      return false
    }
    if (filters.accountType && account.account_type !== filters.accountType) {
      return false
    }
    return matchesPriorityFilter(account, filters.priority) && matchesStatusFilter(account, filters.status)
  }),
)
const filteredDisabledAccounts = computed(() =>
  sortAccountsForDisplay(filteredAccounts.value.filter((account) => account.disabled)),
)
const filteredNormalAccounts = computed(() =>
  sortAccountsForDisplay(
    filteredAccounts.value.filter((account) => !account.disabled),
    compareNormalAccounts,
  ),
)
const sortedListAccounts = computed(() =>
  accountSort.key === null
    ? [...filteredDisabledAccounts.value, ...filteredNormalAccounts.value]
    : sortAccountsForDisplay(filteredAccounts.value),
)
const tableLoading = computed(() => isLoading.value)
const enabledAccountCount = computed(() => accounts.value.filter((account) => !account.disabled).length)
const disabledAccountCount = computed(() => accounts.value.filter((account) => account.disabled).length)
const hasDisabledAccounts = computed(() => disabledAccountCount.value > 0)
const showListLoadingState = computed(() => isLoading.value && accounts.value.length === 0)
const isKeeperRunning = computed(() => keeperStatus.value?.running === true)
const isKeeperDaemonRunning = computed(() => keeperStatus.value?.daemon_running === true)
const keeperRunningModes = computed(() => new Set(keeperStatus.value?.running_modes ?? []))
const isAccountInspectionRunning = computed(() => keeperRunningModes.value.has('once'))
const isAccountInspectionBlocked = computed(
  () => isAccountInspectionRunning.value || keeperRunningModes.value.has('daemon'),
)
const keeperStateType = computed(() => {
  if (isKeeperRunning.value || isKeeperDaemonRunning.value) {
    return 'success'
  }
  if (keeperStatus.value?.state === 'error' || keeperStatus.value?.state === 'failed') {
    return 'error'
  }
  if (keeperStatus.value?.state === 'stopping') {
    return 'warning'
  }
  return 'default'
})
const keeperStatusDetailText = computed(() => {
  const detail = keeperStatus.value?.detail
  if (isKeeperDaemonRunning.value && !isKeeperRunning.value) {
    return t('自动巡检已开启', 'Automatic inspection is enabled')
  }
  if (!detail) {
    return t('未运行', 'Not running')
  }
  return keeperStatusText(detail)
})
const keeperStatusFootnoteText = computed(() =>
  isKeeperDaemonRunning.value ? t('等待 Cron 调度', 'Waiting for Cron schedule') : t('后台自动巡检', 'Background automatic inspection'),
)
const unauthorizedErrorAccountCount = computed(
  () => accounts.value.filter((account) => account.last_status_code === 401).length,
)
const quotaExhaustedAccountCount = computed(
  () => accounts.value.filter(isQuotaExhaustedAccount).length,
)
const activeFilterCount = computed(
  () =>
    Number(filters.keyword.trim() !== '') +
    Number(filters.accountType !== null) +
    Number(filters.priority !== 'all') +
    Number(filters.status !== 'all'),
)
const accountListPageCount = computed(() => accountPageCount(sortedListAccounts.value.length))
const showAccountPagination = computed(() =>
  shouldShowAccountPagination(sortedListAccounts.value.length),
)
const visibleListAccounts = computed(() =>
  pagedAccounts(sortedListAccounts.value, accountListPage.value),
)
const accountRangeStart = computed(() => {
  if (sortedListAccounts.value.length === 0) {
    return 0
  }
  const page = clampPage(accountListPage.value, accountListPageCount.value)
  return (page - 1) * accountDisplaySize.value + 1
})
const accountRangeEnd = computed(() =>
  Math.min(
    clampPage(accountListPage.value, accountListPageCount.value) * accountDisplaySize.value,
    sortedListAccounts.value.length,
  ),
)
const activeQuotaSortLabel = computed(() => {
  if (accountSort.key === 'quotaDay') {
    return t('天', 'Day')
  }
  if (accountSort.key === 'quotaWeek') {
    return t('月/周', 'Month/Week')
  }
  return ''
})
const sortDirectionMark = computed(() => (accountSort.direction === 'asc' ? '↑' : '↓'))

function accountPageCount(rowCount: number): number {
  return Math.max(1, Math.ceil(rowCount / accountDisplaySize.value))
}

function shouldShowAccountPagination(rowCount: number): boolean {
  return rowCount > accountDisplaySize.value
}

function pagedAccounts(source: CodexKeeperAccount[], page: number): CodexKeeperAccount[] {
  const safePage = clampPage(page, accountPageCount(source.length))
  const start = (safePage - 1) * accountDisplaySize.value
  return source.slice(start, start + accountDisplaySize.value)
}

function clampPage(page: number, pageCount: number): number {
  return Math.min(Math.max(1, page), pageCount)
}

function resetAccountPages() {
  accountListPage.value = 1
}

function clampAccountPages() {
  accountListPage.value = clampPage(accountListPage.value, accountListPageCount.value)
}

function isAccountDisplaySize(value: unknown): value is AccountDisplaySize {
  return value === 50 || value === 100 || value === 150 || value === 200
}

function isAccountSortKey(value: unknown): value is AccountSortKey {
  return (
    value === 'quotaDay' ||
    value === 'quotaWeek' ||
    value === 'accountType' ||
    value === 'status' ||
    value === 'priority' ||
    value === 'lastCheckedAt'
  )
}

function isSortDirection(value: unknown): value is SortDirection {
  return value === 'asc' || value === 'desc'
}

function readAccountStatusPreferences(): AccountStatusPreferences | null {
  if (typeof localStorage === 'undefined') {
    return null
  }
  const raw = localStorage.getItem(ACCOUNT_STATUS_PREFERENCE_STORAGE_KEY)
  if (!raw) {
    return null
  }
  try {
    const value: unknown = JSON.parse(raw)
    return value && typeof value === 'object' ? (value as AccountStatusPreferences) : null
  } catch {
    return null
  }
}

function restoreAccountStatusPreferences() {
  const preferences = readAccountStatusPreferences()
  if (!preferences) {
    return
  }
  if (isAccountDisplaySize(preferences.displaySize)) {
    accountDisplaySize.value = preferences.displaySize
  }
  const sort = preferences.sort
  if (!sort || typeof sort !== 'object') {
    return
  }
  if (sort.key === null) {
    accountSort.key = null
    accountSort.direction = 'asc'
    return
  }
  if (isAccountSortKey(sort.key) && isSortDirection(sort.direction)) {
    accountSort.key = sort.key
    accountSort.direction = sort.direction
  }
}

function saveAccountStatusPreferences() {
  if (typeof localStorage === 'undefined') {
    return
  }
  try {
    localStorage.setItem(
      ACCOUNT_STATUS_PREFERENCE_STORAGE_KEY,
      JSON.stringify({
        displaySize: accountDisplaySize.value,
        sort: {
          key: accountSort.key,
          direction: accountSort.direction,
        },
      }),
    )
  } catch {
    // Keep the page usable when local storage is unavailable.
  }
}
const selectedAccountNames = computed(() =>
  selectedAccountKeys.value.map((key) => String(key)),
)
const selectedAccounts = computed(() => {
  const selectedNames = new Set(selectedAccountNames.value)
  return accounts.value.filter((account) => selectedNames.has(account.name))
})
const selectedDisabledAccounts = computed(() =>
  selectedAccounts.value.filter((account) => account.disabled),
)
const selectedEnabledAccounts = computed(() =>
  selectedAccounts.value.filter((account) => !account.disabled),
)
const selectedAccountCount = computed(() => selectedAccountNames.value.length)
const canBulkDelete = computed(() =>
  selectedAccountCount.value > 0 && !isBulkOperationRunning.value,
)
const canRefreshSelected = computed(
  () => selectedAccountCount.value > 0 && !isBulkOperationRunning.value && !isLoading.value,
)
const canBulkEnable = computed(
  () => selectedDisabledAccounts.value.length > 0 && !isBulkOperationRunning.value,
)
const canBulkDisable = computed(
  () => selectedEnabledAccounts.value.length > 0 && !isBulkOperationRunning.value,
)
const bulkDeletePreviewNames = computed(() => selectedAccountNames.value.slice(0, 5))
const bulkDeletePreviewOverflow = computed(() =>
  Math.max(0, selectedAccountCount.value - bulkDeletePreviewNames.value.length),
)
const bulkDeleteDialogTitle = computed(() => t('批量删除账号', 'Bulk Delete Accounts'))
const bulkDeleteWarningText = computed(() =>
  t(
    `将删除已选 ${selectedAccountCount.value} 个账号，并从 CPA 删除认证文件。此操作不可恢复。`,
    `This will delete ${selectedAccountCount.value} selected accounts and remove their auth files from CPA. This cannot be undone.`,
  ),
)
const canSubmitPriority = computed(() => {
  if (priorityDialog.mode === 'default') {
    return priorityDialog.account !== null && defaultPriority(priorityDialog.account) !== null
  }
  const value = priorityDialog.value
  if (value === null || !Number.isInteger(value)) {
    return false
  }
  return priorityDialog.mode === 'low' ? value < -1 : value > 20
})
const priorityDialogTitle = computed(() => t('修改优先级', 'Change Priority'))
const priorityDialogHint = computed(() => {
  if (priorityDialog.mode === 'low') {
    return t('手动低优先级必须小于 -1，巡检永远不会自动调整。', 'Manual low priority must be less than -1. Inspection will never adjust it automatically.')
  }
  if (priorityDialog.mode === 'high') {
    return t('手动优先必须大于 20，额度耗尽时会临时降为 -1，恢复后回到该值。', 'Manual priority must be greater than 20. When quota is exhausted it is temporarily lowered to -1 and restored to this value after recovery.')
  }
  const account = priorityDialog.account
  const value = account ? defaultPriority(account) : null
  return value === null
    ? t('该账号类型没有配置默认优先级，不能使用类型默认值。', 'This account type has no default priority, so the type default cannot be used.')
    : t(`将优先级设置为当前账号类型默认值 ${value}。`, `Set the priority to the current account type default: ${value}.`)
})
const priorityDialogBounds = computed(() => {
  if (priorityDialog.mode === 'low') {
    return { max: -2 }
  }
  if (priorityDialog.mode === 'high') {
    return { min: 21 }
  }
  return {}
})
const priorityModeOptions = computed(() => {
  const defaultValue = priorityDialog.account ? defaultPriority(priorityDialog.account) : null
  return [
    { label: t('手动低优先 (< -1)', 'Manual Low Priority (< -1)'), value: 'low' },
    { label: t('手动优先 (> 20)', 'Manual Priority (> 20)'), value: 'high' },
    {
      label: defaultValue === null
        ? t('类型默认优先级（不可用）', 'Type Default Priority (unavailable)')
        : t(`类型默认优先级 (${defaultValue})`, `Type Default Priority (${defaultValue})`),
      value: 'default',
      disabled: defaultValue === null,
    },
  ]
})

function matchesPriorityFilter(account: CodexKeeperAccount, value: PriorityFilter): boolean {
  const priority = accountPriority(account)
  if (value === 'high') {
    return priority > 20
  }
  if (value === 'minusOne') {
    return priority === -1
  }
  if (value === 'low') {
    return priority < -1
  }
  const accountType = priorityTypeFromFilter(value)
  if (accountType !== null) {
    return (
      account.account_type === accountType &&
      priority >= 0 &&
      priority <= 20
    )
  }
  return true
}

function matchesStatusFilter(account: CodexKeeperAccount, value: AccountStatusFilter): boolean {
  if (value === 'enabled') {
    return !account.disabled
  }
  if (value === 'disabled') {
    return account.disabled
  }
  if (value === 'unauthorized') {
    return account.last_status_code === 401
  }
  if (value === 'quotaExhausted') {
    return isQuotaExhaustedAccount(account)
  }
  return true
}

function toggleStatusFilter(value: Exclude<AccountStatusFilter, 'all'>) {
  filters.status = filters.status === value ? 'all' : value
}

function isStatusFilterActive(value: Exclude<AccountStatusFilter, 'all'>): boolean {
  return filters.status === value
}

function defaultSortDirection(key: AccountSortKey): SortDirection {
  return key === 'priority' || key === 'lastCheckedAt' ? 'desc' : 'asc'
}

function handleQuotaSortSelect(key: string | number) {
  if (key === 'quotaDay' || key === 'quotaWeek') {
    toggleAccountSort(key)
  }
}

function toggleAccountSort(key: AccountSortKey) {
  if (accountSort.key === key) {
    accountSort.direction = accountSort.direction === 'asc' ? 'desc' : 'asc'
    return
  }
  accountSort.key = key
  accountSort.direction = defaultSortDirection(key)
}

function isAccountSortActive(key: AccountSortKey): boolean {
  return accountSort.key === key
}

function accountSortMark(key: AccountSortKey): string {
  return isAccountSortActive(key) ? sortDirectionMark.value : ''
}

function accountPriority(account: CodexKeeperAccount): number {
  return account.priority ?? 0
}

function isQuotaExhaustedAccount(account: CodexKeeperAccount): boolean {
  return !account.disabled && accountPriority(account) === -1
}

function priorityTypeFilter(accountType: string): PriorityTypeFilter {
  return `type:${accountType}`
}

function priorityTypeFromFilter(value: PriorityFilter): string | null {
  return value.startsWith('type:') ? value.slice('type:'.length) : null
}

function normalAccountTypePriority(account: CodexKeeperAccount): number {
  if (!account.account_type) {
    return Number.NEGATIVE_INFINITY
  }
  return priorityRuleMap.value[account.account_type] ?? Number.NEGATIVE_INFINITY
}

function compareNormalAccounts(left: CodexKeeperAccount, right: CodexKeeperAccount): number {
  const priorityDiff = normalAccountTypePriority(right) - normalAccountTypePriority(left)
  if (priorityDiff !== 0) {
    return priorityDiff
  }
  return compareAccountFileName(left, right)
}

function sortAccountsForDisplay(
  source: CodexKeeperAccount[],
  defaultCompare?: (left: CodexKeeperAccount, right: CodexKeeperAccount) => number,
): CodexKeeperAccount[] {
  const rows = [...source]
  if (accountSort.key === null) {
    return defaultCompare ? rows.sort(defaultCompare) : rows
  }
  return rows.sort(compareAccountsByActiveSort)
}

function compareAccountsByActiveSort(left: CodexKeeperAccount, right: CodexKeeperAccount): number {
  const direction = accountSort.direction
  let result = 0
  switch (accountSort.key) {
    case 'quotaDay':
    case 'quotaWeek':
      result = compareNullableNumber(
        quotaSortRemainingPercent(left, accountSort.key),
        quotaSortRemainingPercent(right, accountSort.key),
        direction,
      )
      break
    case 'accountType':
      result = compareNullableString(left.account_type, right.account_type, direction)
      break
    case 'status':
      result = compareNullableNumber(left.disabled ? 1 : 0, right.disabled ? 1 : 0, direction)
      break
    case 'priority':
      result = compareNullableNumber(accountPriority(left), accountPriority(right), direction)
      break
    case 'lastCheckedAt':
      result = compareNullableNumber(
        timestampValue(left.last_checked_at),
        timestampValue(right.last_checked_at),
        direction,
      )
      break
    default:
      result = 0
  }
  return result === 0 ? compareAccountFileName(left, right) : result
}

function compareNullableNumber(
  left: number | null,
  right: number | null,
  direction: SortDirection,
): number {
  if (left === null && right === null) {
    return 0
  }
  if (left === null) {
    return 1
  }
  if (right === null) {
    return -1
  }
  const result = left - right
  return direction === 'asc' ? result : -result
}

function compareNullableString(
  left: string | null,
  right: string | null,
  direction: SortDirection,
): number {
  if (left === null && right === null) {
    return 0
  }
  if (left === null) {
    return 1
  }
  if (right === null) {
    return -1
  }
  const result = left.localeCompare(right)
  return direction === 'asc' ? result : -result
}

function timestampValue(value: string | null): number | null {
  if (!value) {
    return null
  }
  const timestamp = new Date(value).getTime()
  return Number.isNaN(timestamp) ? null : timestamp
}

function compareAccountFileName(left: CodexKeeperAccount, right: CodexKeeperAccount): number {
  return left.name.localeCompare(right.name)
}

function defaultPriority(account: CodexKeeperAccount): number | null {
  if (!account.account_type) {
    return null
  }
  return priorityRuleMap.value[account.account_type] ?? null
}

function accountTypeLabel(accountType: string | null): string {
  const normalized = accountType?.trim().toLowerCase()
  if (!normalized || normalized === 'unknown') {
    return t('未知', 'Unknown')
  }
  if (normalized === 'k12') {
    return 'K12'
  }
  return accountType ?? normalized
}

function isPaidQuotaWindowAccount(accountType: string | null): boolean {
  const normalized = accountType?.trim().toLowerCase()
  return normalized === 'plus' || normalized === 'team' || normalized === 'k12' || normalized?.startsWith('pro') === true
}

function isFreeQuotaWindowAccount(accountType: string | null): boolean {
  return accountType?.trim().toLowerCase() === 'free'
}

function quotaWindowSecondsFor(account: CodexKeeperAccount, window: 'primary' | 'secondary'): number | null {
  if (window === 'primary') {
    return account.primary_window_seconds ?? account.primary_window_usage?.window_seconds ?? null
  }
  return account.secondary_window_seconds ?? account.secondary_window_usage?.window_seconds ?? null
}

function isPaidQuotaWindow(account: CodexKeeperAccount): boolean {
  return (
    isPaidQuotaWindowAccount(account.account_type) ||
    (quotaWindowSecondsFor(account, 'primary') === CODEX_FIVE_HOUR_WINDOW_SECONDS &&
      quotaWindowSecondsFor(account, 'secondary') === CODEX_WEEK_WINDOW_SECONDS)
  )
}

function isFreeQuotaWindow(account: CodexKeeperAccount): boolean {
  return (
    isFreeQuotaWindowAccount(account.account_type) ||
    (quotaWindowSecondsFor(account, 'primary') === CODEX_MONTH_WINDOW_SECONDS &&
      quotaWindowSecondsFor(account, 'secondary') === null)
  )
}

function quotaWindowLabels(account: CodexKeeperAccount): { primary: string; secondary: string } {
  if (isFreeQuotaWindow(account)) {
    return { primary: t('月限额', 'Monthly Limit'), secondary: t('次限额', 'Secondary Limit') }
  }
  if (isPaidQuotaWindow(account)) {
    // The upstream usage payload decides the real window length; the plan
    // type alone does not. Pro accounts, for example, can report a weekly
    // primary window, so label by actual seconds and only fall back to the
    // conventional 5-hour/weekly pair when the seconds are unknown.
    const primaryLabel = quotaWindowLabelForSeconds(quotaWindowSecondsFor(account, 'primary'))
    const secondaryLabel = quotaWindowLabelForSeconds(quotaWindowSecondsFor(account, 'secondary'))
    return {
      primary: primaryLabel ?? t('5小时限额', '5-Hour Limit'),
      secondary: secondaryLabel ?? t('周限额', 'Weekly Limit'),
    }
  }
  return { primary: t('主', 'Primary'), secondary: t('次', 'Secondary') }
}

function quotaWindowLabelForSeconds(seconds: number | null): string | null {
  if (seconds === CODEX_FIVE_HOUR_WINDOW_SECONDS) {
    return t('5小时限额', '5-Hour Limit')
  }
  if (seconds === CODEX_WEEK_WINDOW_SECONDS) {
    return t('周限额', 'Weekly Limit')
  }
  if (seconds === CODEX_MONTH_WINDOW_SECONDS) {
    return t('月限额', 'Monthly Limit')
  }
  return null
}

function shouldShowQuotaWindow(account: CodexKeeperAccount): boolean {
  return !account.disabled
}

function quotaWindowItems(account: CodexKeeperAccount): QuotaWindowItem[] {
  if (!shouldShowQuotaWindow(account) || account.primary_used_percent === null) {
    return []
  }
  const labels = quotaWindowLabels(account)
  const items = [
    {
      label: labels.primary,
      usedPercent: Math.max(0, Math.min(100, account.primary_used_percent)),
      remainingPercent: remainingQuotaPercent(account.primary_used_percent),
      resetAt: account.primary_reset_at,
      usage: account.primary_window_usage,
    },
  ]
  if (account.secondary_used_percent !== null && !isFreeQuotaWindow(account)) {
    items.push({
      label: labels.secondary,
      usedPercent: Math.max(0, Math.min(100, account.secondary_used_percent)),
      remainingPercent: remainingQuotaPercent(account.secondary_used_percent),
      resetAt: account.secondary_reset_at,
      usage: account.secondary_window_usage,
    })
  }
  return items
}

function quotaSortRemainingPercent(account: CodexKeeperAccount, key: AccountSortKey): number | null {
  if (!shouldShowQuotaWindow(account)) {
    return null
  }
  if (key === 'quotaDay') {
    if (isFreeQuotaWindow(account)) {
      return null
    }
    return nullableRemainingQuotaPercent(account.primary_used_percent)
  }
  if (key === 'quotaWeek') {
    if (isFreeQuotaWindow(account)) {
      return nullableRemainingQuotaPercent(account.primary_used_percent)
    }
    if (isPaidQuotaWindow(account)) {
      return nullableRemainingQuotaPercent(account.secondary_used_percent)
    }
  }
  return null
}

function nullableRemainingQuotaPercent(usedPercent: number | null): number | null {
  return usedPercent === null ? null : remainingQuotaPercent(usedPercent)
}

function remainingQuotaPercent(usedPercent: number): number {
  return Math.max(0, Math.min(100, 100 - usedPercent))
}

function quotaBarTone(percent: number): string {
  if (percent < 30) {
    return 'is-danger'
  }
  if (percent < 70) {
    return 'is-warning'
  }
  return 'is-healthy'
}

function formatQuotaResetTime(value: string | null): string | null {
  if (!value) {
    return null
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return null
  }
  return new Intl.DateTimeFormat(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    timeZone: BEIJING_TIME_ZONE,
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function quotaText(account: CodexKeeperAccount): string {
  const items = quotaWindowItems(account)
  if (items.length === 0) {
    return '-'
  }
  return items
    .map((item) => {
      const resetTime = formatQuotaResetTime(item.resetAt)
      const usageText = quotaWindowUsageText(item)
      return resetTime
        ? t(
            `${item.label}剩余 ${item.remainingPercent}%，刷新 ${resetTime}，${usageText}`,
            `${item.label} ${item.remainingPercent}% remaining, refreshes ${resetTime}, ${usageText}`,
          )
        : t(
            `${item.label}剩余 ${item.remainingPercent}%，${usageText}`,
            `${item.label} ${item.remainingPercent}% remaining, ${usageText}`,
          )
    })
    .join(' / ')
}

function quotaWindowUsageText(item: QuotaWindowItem): string {
  if (!item.resetAt || item.usage?.stale === true) {
    return t('额度数据需刷新', 'Quota data needs refresh')
  }
  if (!item.usage) {
    return t('本窗口暂无用量', 'No usage in this window')
  }
  return t(
    `${formatInteger(item.usage.records)} 次 / ${formatCompact(item.usage.total_tokens)} Tokens / ${formatUsd(item.usage.estimated_cost_usd)}`,
    `${formatInteger(item.usage.records)} requests / ${formatCompact(item.usage.total_tokens)} Tokens / ${formatUsd(item.usage.estimated_cost_usd)}`,
  )
}

function quotaWindowUsageTags(item: QuotaWindowItem): QuotaUsageTag[] {
  if (!item.resetAt || item.usage?.stale === true) {
    return [{ label: t('状态', 'Status'), value: t('需刷新', 'Needs refresh'), tone: 'stale' }]
  }
  const usage = item.usage
  return [
    { label: t('请求', 'Requests'), value: formatInteger(usage?.records ?? 0) },
    { label: 'Tokens', value: formatCompact(usage?.total_tokens ?? 0) },
    { label: t('费用', 'Cost'), value: formatUsd(usage?.estimated_cost_usd ?? 0) },
  ]
}

function quotaWindowUsageTitle(item: QuotaWindowItem): string {
  const usage = item.usage
  if (!item.resetAt || usage?.stale === true) {
    return t(`${item.label} 额度数据需刷新`, `${item.label} quota data needs refresh`)
  }
  if (!usage) {
    return t(`${item.label} 本窗口暂无用量`, `${item.label} has no usage in this window`)
  }
  const unpriced =
    usage.unpriced_records > 0
      ? t(`，未计价 ${formatInteger(usage.unpriced_records)} 条`, `, ${formatInteger(usage.unpriced_records)} unpriced records`)
      : ''
  return t(
    `${item.label} 当前窗口：${formatInteger(usage.records)} 次请求，${formatCompact(usage.total_tokens)} Tokens，${formatUsd(usage.estimated_cost_usd)}${unpriced}`,
    `${item.label} current window: ${formatInteger(usage.records)} requests, ${formatCompact(usage.total_tokens)} Tokens, ${formatUsd(usage.estimated_cost_usd)}${unpriced}`,
  )
}

function quotaWindowUsageTone(item: QuotaWindowItem): string {
  return !item.resetAt || item.usage?.stale === true ? 'is-stale' : ''
}

function quotaWindowProjectedCost(item: QuotaWindowItem): number | null {
  if (!item.resetAt || item.usage?.stale === true || item.usedPercent <= 0) {
    return null
  }
  const cost = item.usage?.estimated_cost_usd ?? 0
  if (!Number.isFinite(cost) || cost < 0) {
    return null
  }
  return cost / (item.usedPercent / 100)
}

function quotaWindowPredictionTitle(item: QuotaWindowItem): string {
  if (!item.resetAt || item.usage?.stale === true) {
    return t(`${item.label} 窗口预测需刷新额度数据`, `${item.label} projection needs refreshed quota data`)
  }
  const projectedCost = quotaWindowProjectedCost(item)
  if (projectedCost === null) {
    return t(`${item.label} 已用限额为 0%，暂无窗口预测`, `${item.label} usage is 0%; no projection yet`)
  }
  const currentCost = item.usage?.estimated_cost_usd ?? 0
  return t(
    `${item.label} 窗口预测：${formatUsd(currentCost)} ÷ ${item.usedPercent}% = ${formatUsd(projectedCost)}`,
    `${item.label} projection: ${formatUsd(currentCost)} ÷ ${item.usedPercent}% = ${formatUsd(projectedCost)}`,
  )
}

function latestActionText(account: CodexKeeperAccount): string {
  const text = account.last_error?.trim() || account.latest_action?.trim()
  return text ? serverText(text, '账号状态', 'Account status') : '-'
}

function renderQuotaCell(account: CodexKeeperAccount) {
  const items = quotaWindowItems(account)
  if (items.length === 0) {
    return '-'
  }
  return h(
    'div',
    { class: 'quota-window-cell' },
    items.map((item) => {
      const resetTime = formatQuotaResetTime(item.resetAt)
      return h(
        'div',
        {
          class: 'quota-window-item',
          title: resetTime
            ? t(
                `${item.label}剩余 ${item.remainingPercent}%，刷新 ${resetTime}；${quotaWindowUsageTitle(item)}`,
                `${item.label} ${item.remainingPercent}% remaining, refreshes ${resetTime}; ${quotaWindowUsageTitle(item)}`,
              )
            : t(
                `${item.label}剩余 ${item.remainingPercent}%；${quotaWindowUsageTitle(item)}`,
                `${item.label} ${item.remainingPercent}% remaining; ${quotaWindowUsageTitle(item)}`,
              ),
        },
        [
          h('div', { class: 'quota-window-head' }, [
            h('span', { class: 'quota-window-label' }, item.label),
            h('span', { class: 'quota-window-meta' }, [
              h('span', { class: 'quota-window-percent' }, `${item.remainingPercent}%`),
              resetTime ? h('span', { class: 'quota-window-reset' }, resetTime) : null,
            ]),
          ]),
          h('div', { class: 'quota-window-track' }, [
            h('div', {
              class: ['quota-window-fill', quotaBarTone(item.remainingPercent)],
              style: { width: `${item.remainingPercent}%` },
            }),
          ]),
        ],
      )
    }),
  )
}

function renderQuotaUsageCell(account: CodexKeeperAccount) {
  const items = quotaWindowItems(account)
  if (items.length === 0) {
    return '-'
  }
  return h(
    'div',
    { class: 'quota-usage-cell' },
    items.map((item) =>
      h(
        'div',
        {
          class: ['quota-usage-item', quotaWindowUsageTone(item)],
          title: quotaWindowUsageTitle(item),
        },
        quotaWindowUsageTags(item).map((tag) =>
          h(
            'span',
            { class: ['quota-usage-chip', tag.tone ? `is-${tag.tone}` : undefined] },
            [
              h('span', { class: 'quota-usage-chip-label' }, tag.label),
              h('strong', { class: 'quota-usage-chip-value' }, tag.value),
            ],
          ),
        ),
      ),
    ),
  )
}

function renderQuotaPredictionCell(account: CodexKeeperAccount) {
  const items = quotaWindowItems(account)
  if (items.length === 0) {
    return '-'
  }
  return h(
    'div',
    { class: 'quota-usage-cell' },
    items.map((item) => {
      const projectedCost = quotaWindowProjectedCost(item)
      const needsRefresh = !item.resetAt || item.usage?.stale === true
      return h(
        'div',
        {
          class: 'quota-usage-item is-projection',
          title: quotaWindowPredictionTitle(item),
        },
        [
          h(
            'span',
            { class: ['quota-usage-chip', 'is-projection', needsRefresh ? 'is-stale' : undefined] },
            [
              h('span', { class: 'quota-usage-chip-label' }, t('额度', 'Quota')),
              h(
                'strong',
                { class: 'quota-usage-chip-value' },
                needsRefresh
                  ? t('需刷新', 'Refresh needed')
                  : projectedCost === null
                    ? '-'
                    : formatUsd(projectedCost),
              ),
            ],
          ),
        ],
      )
    }),
  )
}

function renderAccountIdentityCell(account: CodexKeeperAccount) {
  const primary = account.email ?? account.name
  const statusTags = [
    {
      label: account.disabled ? t('已禁用', 'Disabled') : t('启用中', 'Enabled'),
      tone: account.disabled ? 'is-warning' : 'is-success',
    },
    account.last_status_code === 401
      ? { label: t('401报错', '401 Error'), tone: 'is-danger' }
      : null,
    isQuotaExhaustedAccount(account)
      ? { label: t('额度耗尽', 'Quota Exhausted'), tone: 'is-purple' }
      : null,
  ].filter((item): item is { label: string; tone: string } => item !== null)
  const statusLabel = statusTags.map((item) => item.label).join(' / ')
  return h(
    'div',
    {
      class: 'account-table-identity',
      title: `${primary}\n${account.name}\n${t('状态', 'Status')} ${statusLabel}`,
    },
    [
      h('span', { class: 'account-table-email' }, primary),
      h('span', { class: 'account-table-name' }, account.name),
      h(
        'span',
        { class: 'account-table-meta' },
        statusTags.map((item) =>
          h('span', { class: ['account-table-chip', item.tone] }, item.label),
        ),
      ),
    ],
  )
}

function renderAccountTypeCell(account: CodexKeeperAccount) {
  const typeLabel = accountTypeLabel(account.account_type)
  return h(
    'span',
    { class: ['account-table-chip', 'is-type'], title: typeLabel },
    typeLabel,
  )
}

function renderAccountPriorityCell(account: CodexKeeperAccount) {
  const priorityLabel = formatInteger(accountPriority(account))
  return h(
    'span',
    { class: ['account-table-chip', 'is-priority'], title: t(`优先级 ${priorityLabel}`, `Priority ${priorityLabel}`) },
    priorityLabel,
  )
}

function renderLastCheckedCell(account: CodexKeeperAccount) {
  const text = formatRelativeTime(account.last_checked_at, relativeTimeNow.value)
  const fullText = formatDateTime(account.last_checked_at)
  return h(
    'span',
    {
      class: ['account-table-value-pill', 'is-time', text === '-' ? 'is-empty' : ''],
      title: fullText,
    },
    text,
  )
}

function renderLatestActionCell(account: CodexKeeperAccount) {
  const text = latestActionText(account)
  return h(
    'span',
    {
      class: ['account-table-value-pill', 'is-action', text === '-' ? 'is-empty' : ''],
      title: text === '-' ? undefined : text,
    },
    text,
  )
}

async function loadAccounts() {
  isLoading.value = true
  try {
    const [accountsResponse, nextStatus] = await Promise.all([
      listCodexKeeperAccounts(),
      getCodexKeeperStatus(),
    ])
    accounts.value = accountsResponse.items
    priorityRules.value = accountsResponse.priority_rules
    keeperStatus.value = nextStatus
  } catch (error) {
    message.error(errorText(error, '加载账号状态失败', 'Failed to load account status'))
  } finally {
    isLoading.value = false
  }
}

async function loadKeeperStatus() {
  try {
    keeperStatus.value = await getCodexKeeperStatus()
  } catch {
    return
  }
}

function accountRowKey(account: CodexKeeperAccount): string {
  return account.name
}

function handleAccountSelectionUpdate(keys: DataTableRowKey[]) {
  selectedAccountKeys.value = keys
}

function pruneSelectedAccountKeys() {
  const availableNames = new Set(visibleListAccounts.value.map((account) => account.name))
  selectedAccountKeys.value = selectedAccountKeys.value.filter((key) =>
    availableNames.has(String(key)),
  )
}

function authFileStringField(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function authFileBooleanField(value: unknown): boolean {
  if (typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'number') {
    return value !== 0
  }
  if (typeof value === 'string') {
    return ['1', 'true', 'yes', 'on'].includes(value.trim().toLowerCase())
  }
  return false
}

function authFilePriorityField(value: unknown): string {
  if (typeof value === 'number' && Number.isFinite(value) && Number.isInteger(value)) {
    return String(value)
  }
  if (typeof value === 'string' && /^[-+]?\d+$/.test(value.trim())) {
    return value.trim()
  }
  return ''
}

function authFileHeadersField(value: unknown): Record<string, string> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {}
  }
  return Object.fromEntries(
    Object.entries(value).filter(([, headerValue]) => typeof headerValue === 'string'),
  )
}

function parseAuthFileHeaders(value: string): Record<string, string> {
  if (!value.trim()) {
    return {}
  }
  let parsed: unknown
  try {
    parsed = JSON.parse(value)
  } catch {
    throw new Error(t('自定义请求头必须是有效的 JSON。', 'Custom headers must be valid JSON.'))
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(t('自定义请求头必须是 JSON 对象。', 'Custom headers must be a JSON object.'))
  }
  const entries = Object.entries(parsed)
  if (entries.some(([, headerValue]) => typeof headerValue !== 'string')) {
    throw new Error(t('每个自定义请求头的值都必须是字符串。', 'Every custom header value must be a string.'))
  }
  return Object.fromEntries(entries as Array<[string, string]>)
}

function buildAuthFileEditorFields(editor: AuthFileEditorState): CodexKeeperAuthFileFields {
  if (!editor.json) {
    return {}
  }
  const original = editor.json
  const fields: CodexKeeperAuthFileFields = {}
  if (editor.prefix.trim() !== authFileStringField(original.prefix).trim()) {
    fields.prefix = editor.prefix.trim()
  }
  if (editor.proxyUrl.trim() !== authFileStringField(original.proxy_url).trim()) {
    fields.proxy_url = editor.proxyUrl.trim()
  }
  const originalPriority = authFilePriorityField(original.priority)
  const nextPriority = editor.priority.trim()
  if (nextPriority !== originalPriority) {
    if (nextPriority === '') {
      fields.priority = 0
    } else if (!/^[-+]?\d+$/.test(nextPriority)) {
      throw new Error(t('认证文件优先级必须是整数。', 'Auth file priority must be an integer.'))
    } else {
      const priority = Number(nextPriority)
      if (!Number.isSafeInteger(priority)) {
        throw new Error(t('认证文件优先级必须是安全整数。', 'Auth file priority must be a safe integer.'))
      }
      fields.priority = priority
    }
  }
  if (editor.websocketsTouched && editor.websockets !== authFileBooleanField(original.websockets ?? original.websocket)) {
    fields.websockets = editor.websockets
  }
  if (editor.noteTouched && editor.note.trim() !== authFileStringField(original.note).trim()) {
    fields.note = editor.note.trim()
  }
  if (editor.headersTouched) {
    const nextHeaders = parseAuthFileHeaders(editor.headersText)
    const originalHeaders = authFileHeadersField(original.headers)
    const headerPatch: Record<string, string> = {}
    Object.entries(nextHeaders).forEach(([name, value]) => {
      if (originalHeaders[name] !== value) {
        headerPatch[name] = value
      }
    })
    Object.keys(originalHeaders).forEach((name) => {
      if (!(name in nextHeaders)) {
        headerPatch[name] = ''
      }
    })
    if (Object.keys(headerPatch).length > 0) {
      fields.headers = headerPatch
    }
  }
  return fields
}

function buildAuthFileUpdatedText(editor: AuthFileEditorState): string {
  if (!editor.json) {
    return editor.rawText
  }
  const fields = buildAuthFileEditorFields(editor)
  const updated: Record<string, unknown> = { ...editor.json }
  if (fields.prefix !== undefined) {
    if (fields.prefix) {
      updated.prefix = fields.prefix
    } else {
      delete updated.prefix
    }
  }
  if (fields.proxy_url !== undefined) {
    if (fields.proxy_url) {
      updated.proxy_url = fields.proxy_url
    } else {
      delete updated.proxy_url
    }
  }
  if (fields.priority !== undefined) {
    if (fields.priority === 0) {
      delete updated.priority
    } else {
      updated.priority = fields.priority
    }
  }
  if (fields.websockets !== undefined) {
    delete updated.websocket
    updated.websockets = fields.websockets
  }
  if (fields.note !== undefined) {
    if (fields.note) {
      updated.note = fields.note
    } else {
      delete updated.note
    }
  }
  if (fields.headers !== undefined) {
    const headers = authFileHeadersField(updated.headers)
    Object.entries(fields.headers).forEach(([name, value]) => {
      if (value) {
        headers[name] = value
      } else {
        delete headers[name]
      }
    })
    if (Object.keys(headers).length > 0) {
      updated.headers = headers
    } else {
      delete updated.headers
    }
  }
  return JSON.stringify(updated, null, 2)
}

const authFileEditorUpdatedText = computed(() => {
  const editor = authFileEditor.value
  if (!editor || editor.headersError) {
    return ''
  }
  try {
    return buildAuthFileUpdatedText(editor)
  } catch {
    return ''
  }
})

const authFileEditorDirty = computed(() => {
  const editor = authFileEditor.value
  if (!editor?.json) {
    return false
  }
  try {
    return Object.keys(buildAuthFileEditorFields(editor)).length > 0
  } catch {
    return true
  }
})

function handleAuthFileHeadersChange(value: string) {
  const editor = authFileEditor.value
  if (!editor) {
    return
  }
  editor.headersText = value
  editor.headersTouched = true
  try {
    parseAuthFileHeaders(value)
    editor.headersError = null
  } catch (error) {
    editor.headersError = error instanceof Error ? error.message : String(error)
  }
}

async function reloadAccounts() {
  isLoading.value = true
  if (canManageAccounts.value) {
    try {
      await syncCodexKeeperAccountList()
    } catch (error) {
      message.error(errorText(error, '同步 CPA 账号列表失败', 'Failed to sync the CPA account list'))
    }
  }
  try {
    const [accountsResponse, nextStatus] = await Promise.all([
      listCodexKeeperAccounts(),
      getCodexKeeperStatus(),
    ])
    accounts.value = accountsResponse.items
    priorityRules.value = accountsResponse.priority_rules
    keeperStatus.value = nextStatus
  } catch (error) {
    message.error(errorText(error, '加载账号状态失败', 'Failed to load account status'))
  } finally {
    isLoading.value = false
  }
}

async function openAuthFileEditor(account: CodexKeeperAccount) {
  if (!canManageAccounts.value) {
    return
  }
  const fileName = account.name
  authFileEditor.value = {
    fileName,
    fileInfoText: JSON.stringify(account, null, 2),
    loading: true,
    saving: false,
    error: null,
    rawText: '',
    invalidContentPreview: '',
    json: null,
    prefix: '',
    proxyUrl: '',
    priority: '',
    websockets: false,
    websocketsTouched: false,
    note: '',
    noteTouched: false,
    headersText: '',
    headersTouched: false,
    headersError: null,
  }
  try {
    const detail = await getCodexKeeperAuthFile(fileName)
    const current = authFileEditor.value
    if (!current || current.fileName !== fileName) {
      return
    }
    current.loading = false
    current.json = detail.json
    if (!detail.json) {
      current.rawText = detail.raw_text ?? ''
      const preview = current.rawText.trim()
      current.invalidContentPreview = preview.length > 1000 ? `${preview.slice(0, 1000)}\n...` : preview
      current.error = detail.invalid_reason === 'html_challenge'
        ? t(
            '下载到的是 HTML 验证页面，不是认证 JSON 对象。请重新认证或替换该认证文件后再编辑字段。',
            'Downloaded content is an HTML challenge page, not an auth JSON object. Re-authenticate or replace the auth file before editing fields.',
          )
        : t(
            '该认证文件不是 JSON 对象，无法编辑字段。',
            'This auth file is not a JSON object, so its fields cannot be edited.',
          )
      return
    }
    const headers = authFileHeadersField(detail.json.headers)
    current.prefix = authFileStringField(detail.json.prefix)
    current.proxyUrl = authFileStringField(detail.json.proxy_url)
    current.priority = authFilePriorityField(detail.json.priority)
    current.websockets = authFileBooleanField(detail.json.websockets ?? detail.json.websocket)
    current.note = authFileStringField(detail.json.note)
    current.headersText = Object.keys(headers).length > 0 ? JSON.stringify(headers, null, 2) : ''
  } catch (error) {
    const current = authFileEditor.value
    if (!current || current.fileName !== fileName) {
      return
    }
    current.loading = false
    current.error = errorText(error, '加载认证文件失败', 'Failed to load auth file')
  }
}

function closeAuthFileEditor() {
  if (!authFileEditor.value?.saving) {
    authFileEditor.value = null
  }
}

async function saveAuthFileEditor() {
  const editor = authFileEditor.value
  if (!editor?.json || editor.loading || editor.saving || editor.headersError) {
    return
  }
  let fields: CodexKeeperAuthFileFields
  try {
    fields = buildAuthFileEditorFields(editor)
  } catch (error) {
    message.error(error instanceof Error ? error.message : String(error))
    return
  }
  if (Object.keys(fields).length === 0) {
    closeAuthFileEditor()
    return
  }
  editor.saving = true
  try {
    await updateCodexKeeperAuthFile(editor.fileName, fields)
    message.success(t(`已更新认证文件“${editor.fileName}”`, `Auth file “${editor.fileName}” updated`))
    await loadAccounts()
    if (selectedAccount.value?.name === editor.fileName) {
      selectedAccountNoteRequestID += 1
      isSelectedAccountNoteLoading.value = true
      await loadSelectedAccountNote(editor.fileName, selectedAccountNoteRequestID)
    }
    authFileEditor.value = null
    await refreshAccounts([editor.fileName], {
      successMessage: t(
        `已开始巡检“${editor.fileName}”以同步账号信息`,
        `Started inspecting “${editor.fileName}” to sync account information`,
      ),
    })
  } catch (error) {
    message.error(errorText(error, '更新认证文件失败', 'Failed to update auth file'))
  } finally {
    if (authFileEditor.value?.fileName === editor.fileName) {
      editor.saving = false
    }
  }
}

async function copyAuthFileEditorText() {
  const text = authFileEditorUpdatedText.value
  if (!text) {
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    message.success(t('认证文件内容已复制', 'Auth file content copied'))
  } catch (error) {
    message.error(errorText(error, '复制认证文件内容失败', 'Failed to copy auth file content'))
  }
}

function resetCodexOAuthDialog() {
  oauthPollToken += 1
  oauthDialogStatus.value = 'idle'
  oauthAuthURL.value = ''
  oauthCallbackURL.value = ''
  oauthError.value = ''
  isStartingOAuth.value = false
  isSubmittingOAuthCallback.value = false
}

function openCodexOAuthDialog() {
  if (!canManageAccounts.value) {
    return
  }
  resetCodexOAuthDialog()
  oauthDialogOpen.value = true
}

function closeCodexOAuthDialog() {
  resetCodexOAuthDialog()
  oauthDialogOpen.value = false
}

async function reloadAccountsAfterOAuth() {
  try {
    await runCodexKeeperOnce()
    void pollKeeperModeUntilIdle('once')
  } catch {
    await loadAccounts()
  }
}

async function pollCodexOAuthStatus(state: string, token: number) {
  for (;;) {
    await sleep(OAUTH_STATUS_POLL_INTERVAL_MS)
    if (token !== oauthPollToken || !oauthDialogOpen.value) {
      return
    }
    try {
      const response = await getCodexKeeperOAuthStatus(state)
      if (token !== oauthPollToken || !oauthDialogOpen.value) {
        return
      }
      const status = response.status.trim().toLowerCase()
      if (status === 'ok') {
        oauthDialogStatus.value = 'success'
        oauthError.value = ''
        oauthPollToken += 1
        message.success(t('Codex OAuth 认证成功', 'Codex OAuth authentication successful'))
        void reloadAccountsAfterOAuth()
        return
      }
      if (status === 'error') {
        oauthDialogStatus.value = 'error'
        oauthError.value = response.error?.trim() || t('Codex OAuth 认证失败', 'Codex OAuth authentication failed')
        oauthPollToken += 1
        return
      }
      oauthError.value = ''
    } catch (error) {
      if (token !== oauthPollToken || !oauthDialogOpen.value) {
        return
      }
      oauthError.value = errorText(
        error,
        '暂时无法查询认证状态，将继续重试',
        'Unable to check authentication status; retrying',
      )
    }
  }
}

async function startCodexOAuth() {
  if (isStartingOAuth.value) {
    return
  }
  const token = ++oauthPollToken
  oauthDialogStatus.value = 'idle'
  oauthAuthURL.value = ''
  oauthCallbackURL.value = ''
  oauthError.value = ''
  isStartingOAuth.value = true
  try {
    const response = await startCodexKeeperOAuth()
    if (token !== oauthPollToken || !oauthDialogOpen.value) {
      return
    }
    oauthAuthURL.value = response.url
    oauthDialogStatus.value = 'waiting'
    void pollCodexOAuthStatus(response.state, token)
  } catch (error) {
    if (token !== oauthPollToken || !oauthDialogOpen.value) {
      return
    }
    oauthDialogStatus.value = 'error'
    oauthError.value = errorText(error, '启动 Codex OAuth 失败', 'Failed to start Codex OAuth')
  } finally {
    if (token === oauthPollToken) {
      isStartingOAuth.value = false
    }
  }
}

function openCodexOAuthURL() {
  if (oauthAuthURL.value) {
    window.open(oauthAuthURL.value, '_blank', 'noopener,noreferrer')
  }
}

async function copyCodexOAuthURL() {
  try {
    await copyToClipboard(oauthAuthURL.value)
    message.success(t('授权链接已复制', 'Authorization link copied'))
  } catch (error) {
    message.error(errorText(error, '复制授权链接失败', 'Failed to copy authorization link'))
  }
}

async function submitCodexOAuthCallbackURL() {
  const redirectURL = oauthCallbackURL.value.trim()
  if (!redirectURL || oauthDialogStatus.value !== 'waiting' || isSubmittingOAuthCallback.value) {
    return
  }
  isSubmittingOAuthCallback.value = true
  try {
    await submitCodexKeeperOAuthCallback(redirectURL)
    message.success(t('回调 URL 已提交，正在等待认证结果', 'Callback URL submitted; waiting for authentication'))
  } catch (error) {
    message.error(errorText(error, '提交回调 URL 失败', 'Failed to submit callback URL'))
  } finally {
    isSubmittingOAuthCallback.value = false
  }
}

function triggerAuthFileUpload() {
  if (canManageAccounts.value && !isUploadingAuthFiles.value) {
    authFileInput.value?.click()
  }
}

async function startAccountInspection(): Promise<boolean> {
  if (!canManageAccounts.value || isStartingAccountInspection.value) {
    return false
  }
  isStartingAccountInspection.value = true
  try {
    await runCodexKeeperOnce()
    message.success(t('已开始账号巡检', 'Account inspection started'))
    await loadKeeperStatus()
    void pollKeeperModeUntilIdle('once')
    return true
  } catch (error) {
    message.error(errorText(error, '启动账号巡检失败', 'Failed to start account inspection'))
    return false
  } finally {
    isStartingAccountInspection.value = false
  }
}

async function handleAuthFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (files.length === 0 || isUploadingAuthFiles.value) {
    return
  }
  const validFiles: File[] = []
  const invalidNames: string[] = []
  files.forEach((file) => {
    if (!file.name.toLowerCase().endsWith('.json')) {
      invalidNames.push(`${file.name}: ${t('只能上传 JSON 文件', 'Only JSON files can be uploaded')}`)
      return
    }
    if (file.size > AUTH_FILE_MAX_SIZE) {
      invalidNames.push(`${file.name}: ${t('文件大小不能超过 10 MB', 'File size must not exceed 10 MB')}`)
      return
    }
    validFiles.push(file)
  })
  if (invalidNames.length > 0) {
    message.error(invalidNames.join('; '))
  }
  if (validFiles.length === 0) {
    return
  }
  isUploadingAuthFiles.value = true
  try {
    const result = await uploadCodexKeeperAuthFiles(validFiles)
    if (result.uploaded > 0) {
      message.success(t(`上传成功 ${result.uploaded} 个认证文件`, `Uploaded ${result.uploaded} auth file(s)`))
      await refreshAccounts(result.files, {
        successMessage: t(
          `已开始巡检 ${result.uploaded} 个新账号`,
          `Started inspecting ${result.uploaded} new account(s)`,
        ),
      })
    }
    if (result.failed.length > 0) {
      message.error(result.failed.map((item) => `${item.name}: ${item.error}`).join('; '))
    }
  } catch (error) {
    message.error(errorText(error, '上传认证文件失败', 'Failed to upload auth files'))
  } finally {
    isUploadingAuthFiles.value = false
  }
}

function openBulkDeleteDialog() {
  if (!canBulkDelete.value) {
    return
  }
  bulkDeleteDialog.show = true
}

async function submitBulkDelete() {
  const authNames = selectedAccountNames.value
  if (authNames.length === 0) {
    return
  }
  isBulkDeleting.value = true
  try {
    const result = await bulkDeleteCodexKeeperAccounts({ auth_names: authNames })
    const deletedNames = new Set(result.deleted)
    selectedAccountKeys.value = selectedAccountKeys.value.filter(
      (key) => !deletedNames.has(String(key)),
    )
    if (result.failed.length > 0 && result.deleted.length > 0) {
      message.warning(t(`批量删除完成：成功 ${result.deleted.length} 个，失败 ${result.failed.length} 个`, `Bulk delete complete: ${result.deleted.length} succeeded, ${result.failed.length} failed`))
    } else if (result.failed.length > 0) {
      message.error(t(`批量删除失败：失败 ${result.failed.length} 个`, `Bulk delete failed: ${result.failed.length} failed`))
    } else {
      message.success(t(`已删除 ${result.deleted.length} 个账号`, `Deleted ${result.deleted.length} accounts`))
    }
    bulkDeleteDialog.show = false
    await loadAccounts()
  } catch (error) {
    message.error(errorText(error, '批量删除失败', 'Bulk delete failed'))
  } finally {
    isBulkDeleting.value = false
  }
}

async function toggleSelectedAccounts(action: 'enable' | 'disable') {
  const targets = action === 'enable' ? selectedDisabledAccounts.value : selectedEnabledAccounts.value
  if (targets.length === 0 || isBulkOperationRunning.value) {
    return
  }
  bulkToggleAction.value = action
  try {
    const results = await Promise.allSettled(
      targets.map((account) =>
        action === 'enable'
          ? enableCodexKeeperAccount(account.name)
          : disableCodexKeeperAccount(account.name),
      ),
    )
    const succeededNames = new Set(
      targets
        .filter((_, index) => results[index]?.status === 'fulfilled')
        .map((account) => account.name),
    )
    const failedCount = results.length - succeededNames.size
    selectedAccountKeys.value = selectedAccountKeys.value.filter(
      (key) => !succeededNames.has(String(key)),
    )
    if (succeededNames.size > 0) {
      await loadAccounts()
    }
    const zhAction = action === 'enable' ? '启用' : '禁用'
    const enAction = action === 'enable' ? 'enable' : 'disable'
    if (failedCount > 0 && succeededNames.size > 0) {
      message.warning(t(
        `批量${zhAction}完成：成功 ${succeededNames.size} 个，失败 ${failedCount} 个`,
        `Bulk ${enAction} complete: ${succeededNames.size} succeeded, ${failedCount} failed`,
      ))
    } else if (failedCount > 0) {
      message.error(t(
        `批量${zhAction}失败：失败 ${failedCount} 个`,
        `Bulk ${enAction} failed: ${failedCount} failed`,
      ))
    } else {
      message.success(t(
        `已${zhAction} ${succeededNames.size} 个账号`,
        `${succeededNames.size} accounts ${action === 'enable' ? 'enabled' : 'disabled'}`,
      ))
    }
  } finally {
    bulkToggleAction.value = null
  }
}

function confirmToggleSelectedAccounts(action: 'enable' | 'disable') {
  const count = action === 'enable'
    ? selectedDisabledAccounts.value.length
    : selectedEnabledAccounts.value.length
  if (count === 0) {
    return
  }
  if (action === 'enable') {
    openAccountConfirm(
      t('批量启用账号', 'Bulk Enable Accounts'),
      t(`确认启用已选的 ${count} 个已禁用账号？`, `Enable the ${count} selected disabled accounts?`),
      t('确认启用', 'Confirm Enable'),
      'primary',
      () => toggleSelectedAccounts('enable'),
    )
    return
  }
  openAccountConfirm(
    t('批量禁用账号', 'Bulk Disable Accounts'),
    t(`确认禁用已选的 ${count} 个正常账号？`, `Disable the ${count} selected enabled accounts?`),
    t('确认禁用', 'Confirm Disable'),
    'warning',
    () => toggleSelectedAccounts('disable'),
  )
}

async function loadSelectedAccountNote(accountName: string, requestID: number) {
  try {
    const detail = await getCodexKeeperAuthFile(accountName)
    if (requestID !== selectedAccountNoteRequestID || selectedAccount.value?.name !== accountName) {
      return
    }
    const note = authFileStringField(detail.json?.note).trim()
    selectedAccountNote.value = note || null
  } catch {
    if (requestID === selectedAccountNoteRequestID && selectedAccount.value?.name === accountName) {
      selectedAccountNote.value = null
    }
  } finally {
    if (requestID === selectedAccountNoteRequestID && selectedAccount.value?.name === accountName) {
      isSelectedAccountNoteLoading.value = false
    }
  }
}

function openDetail(account: CodexKeeperAccount) {
  selectedAccount.value = account
  selectedAccountNote.value = null
  isSelectedAccountNoteLoading.value = false
  selectedAccountNoteRequestID += 1
  if (canManageAccounts.value) {
    isSelectedAccountNoteLoading.value = true
    void loadSelectedAccountNote(account.name, selectedAccountNoteRequestID)
  }
  detailOpen.value = true
}

function openPriorityDialog(account: CodexKeeperAccount) {
  priorityDialog.account = account
  const priority = accountPriority(account)
  const mode =
    priority < -1
      ? 'low'
      : priority > 20
        ? 'high'
        : 'default'
  setPriorityDialogMode(defaultPriority(account) === null && mode === 'default' ? 'low' : mode)
  priorityDialog.show = true
}

function setPriorityDialogMode(mode: PriorityMode) {
  priorityDialog.mode = mode
  const account = priorityDialog.account
  if (!account) {
    priorityDialog.value = null
    return
  }
  if (mode === 'low') {
    const priority = accountPriority(account)
    priorityDialog.value = priority < -1 ? priority : -2
    return
  }
  if (mode === 'high') {
    const priority = accountPriority(account)
    priorityDialog.value = priority > 20 ? priority : 21
    return
  }
  priorityDialog.value = defaultPriority(account)
}

async function submitPriorityDialog() {
  if (!priorityDialog.account || !canSubmitPriority.value) {
    return
  }
  const value =
    priorityDialog.mode === 'default'
      ? defaultPriority(priorityDialog.account)
      : priorityDialog.value
  if (value === null) {
    return
  }
  await runAccountAction(
    priorityDialog.account,
    'priority',
    () => updateCodexKeeperPriority(priorityDialog.account!.name, value),
    t('优先级已更新', 'Priority updated'),
  )
  priorityDialog.show = false
}

function openAccountConfirm(
  title: string,
  content: string,
  positiveText: string,
  type: AccountConfirmType,
  action: () => Promise<void>,
) {
  accountConfirmDialog.title = title
  accountConfirmDialog.content = content
  accountConfirmDialog.positiveText = positiveText
  accountConfirmDialog.type = type
  accountConfirmDialog.action = action
  accountConfirmDialog.show = true
}

async function submitAccountConfirm() {
  if (!accountConfirmDialog.action || isAccountConfirmSubmitting.value) {
    return
  }
  isAccountConfirmSubmitting.value = true
  try {
    await accountConfirmDialog.action()
    accountConfirmDialog.show = false
  } finally {
    isAccountConfirmSubmitting.value = false
  }
}

function confirmEnableAccount(account: CodexKeeperAccount) {
  openAccountConfirm(
    t('启用账号', 'Enable Account'),
    t(`启用 ${account.name}？`, `Enable ${account.name}?`),
    t('确认启用', 'Confirm Enable'),
    'primary',
    () => enableAccount(account),
  )
}

function confirmDisableAccount(account: CodexKeeperAccount) {
  openAccountConfirm(
    t('禁用账号', 'Disable Account'),
    t(`禁用 ${account.name}？`, `Disable ${account.name}?`),
    t('确认禁用', 'Confirm Disable'),
    'warning',
    () => disableAccount(account),
  )
}

function confirmDeleteAccount(account: CodexKeeperAccount) {
  openAccountConfirm(
    t('删除账号', 'Delete Account'),
    t(`删除 ${account.name}？此操作会从 CPA 删除认证文件。`, `Delete ${account.name}? This will remove the auth file from CPA.`),
    t('确认删除', 'Confirm Delete'),
    'error',
    () => deleteAccount(account),
  )
}

function enableAccount(account: CodexKeeperAccount) {
  return runAccountAction(
    account,
    'toggle',
    () => enableCodexKeeperAccount(account.name),
    t('账号已启用', 'Account enabled'),
  )
}

function disableAccount(account: CodexKeeperAccount) {
  return runAccountAction(
    account,
    'toggle',
    () => disableCodexKeeperAccount(account.name),
    t('账号已禁用', 'Account disabled'),
  )
}

function deleteAccount(account: CodexKeeperAccount) {
  return runAccountAction(
    account,
    'delete',
    () => deleteCodexKeeperAccount(account.name),
    t('账号已删除', 'Account deleted'),
  )
}

function refreshAccount(account: CodexKeeperAccount, options: { closeDetail?: boolean } = {}) {
  return refreshAccounts([account.name], options)
}

async function refreshSelectedAccounts() {
  await refreshAccounts(selectedAccountNames.value, { clearSelection: true })
}

function uniqueAccountNames(raw: string[]): string[] {
  return [...new Set(raw.map((name) => name.trim()).filter(Boolean))]
}

function sleep(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

async function pollKeeperModeUntilIdle(mode: KeeperPollMode) {
  const token = ++keeperPollTokens[mode]
  for (;;) {
    await sleep(REFRESH_STATUS_POLL_INTERVAL_MS)
    if (token !== keeperPollTokens[mode]) {
      return
    }
    try {
      const status = await getCodexKeeperStatus()
      keeperStatus.value = status
      const runningModes = status.running_modes ?? []
      const modeRunning = runningModes.length > 0 ? runningModes.includes(mode) : status.running
      if (modeRunning) {
        continue
      }
      await loadAccounts()
      return
    } catch {
      continue
    }
  }
}

async function refreshAccounts(
  rawNames: string[],
  options: { closeDetail?: boolean; clearSelection?: boolean; successMessage?: string } = {},
) {
  const authNames = uniqueAccountNames(rawNames)
  if (authNames.length === 0) {
    return
  }
  const refreshKeys = authNames
    .map((name) => accounts.value.find((account) => account.name === name))
    .filter((account): account is CodexKeeperAccount => account !== undefined)
    .map((account) => accountActionKey(account, 'refresh'))
  if (refreshKeys.some((key) => actingActions.value.has(key)) || isBulkOperationRunning.value) {
    return
  }
  const nextActions = new Set(actingActions.value)
  refreshKeys.forEach((key) => nextActions.add(key))
  actingActions.value = nextActions
  if (authNames.length > 1 || options.clearSelection) {
    isBulkRefreshing.value = true
  }
  try {
    await refreshCodexKeeperAccounts({ auth_names: authNames })
    message.success(options.successMessage ?? (authNames.length === 1
      ? t('已开始刷新账号', 'Started refreshing account')
      : t(`已开始刷新 ${authNames.length} 个账号`, `Started refreshing ${authNames.length} accounts`)))
    if (options.closeDetail) {
      detailOpen.value = false
    }
    if (options.clearSelection) {
      selectedAccountKeys.value = []
    }
    void pollKeeperModeUntilIdle('accounts')
  } catch (error) {
    message.error(errorText(error, '刷新账号失败', 'Failed to refresh accounts'))
  } finally {
    const restActions = new Set(actingActions.value)
    refreshKeys.forEach((key) => restActions.delete(key))
    actingActions.value = restActions
    isBulkRefreshing.value = false
  }
}

function accountActionKey(account: CodexKeeperAccount, action: AccountAction): string {
  return `${action}\u0000${account.name}`
}

function isActionLoading(account: CodexKeeperAccount, action: AccountAction): boolean {
  return actingActions.value.has(accountActionKey(account, action))
}

function isRowActing(account: CodexKeeperAccount): boolean {
  return (['toggle', 'priority', 'delete', 'refresh'] as const).some((action) =>
    isActionLoading(account, action),
  )
}

async function runAccountAction(
  account: CodexKeeperAccount,
  actionType: AccountAction,
  action: () => Promise<void>,
  successText: string,
) {
  const key = accountActionKey(account, actionType)
  if (actingActions.value.has(key)) {
    return
  }
  actingActions.value = new Set(actingActions.value).add(key)
  try {
    await action()
    message.success(successText)
    await loadAccounts()
    if (selectedAccount.value?.name === account.name) {
      const freshAccount = accounts.value.find((item) => item.name === account.name) ?? null
      selectedAccount.value = freshAccount
      detailOpen.value = freshAccount !== null
    }
  } catch (error) {
    message.error(errorText(error, '账号操作失败', 'Account operation failed'))
  } finally {
    const nextActions = new Set(actingActions.value)
    nextActions.delete(key)
    actingActions.value = nextActions
  }
}

const baseColumns = computed<DataTableColumns<CodexKeeperAccount>>(() => [
  {
    title: t('账号', 'Account'),
    key: 'identity',
    width: 220,
    render: (row) => renderAccountIdentityCell(row),
  },
  {
    title: t('类型', 'Type'),
    key: 'account_type',
    width: 96,
    render: (row) => renderAccountTypeCell(row),
  },
  {
    title: t('优先级', 'Priority'),
    key: 'priority',
    width: 88,
    render: (row) => renderAccountPriorityCell(row),
  },
  {
    title: t('额度窗口', 'Quota Window'),
    key: 'quota',
    width: 270,
    render: (row) => renderQuotaCell(row),
  },
  {
    title: t('窗口用量', 'Window Usage'),
    key: 'quota_usage',
    width: 250,
    render: (row) => renderQuotaUsageCell(row),
  },
  {
    title: t('窗口预测', 'Window Projection'),
    key: 'quota_prediction',
    width: 90,
    render: (row) => renderQuotaPredictionCell(row),
  },
  {
    title: t('最近巡检', 'Last Inspection'),
    key: 'last_checked_at',
    width: 100,
    render: (row) => renderLastCheckedCell(row),
  },
  {
    title: t('最近操作', 'Latest Action'),
    key: 'latest_action',
    width: 240,
    render: (row) => renderLatestActionCell(row),
  },
])

const manageActionColumn = computed<DataTableColumns<CodexKeeperAccount>[number]>(() => ({
  title: '',
  key: 'actions',
  width: 224,
  fixed: 'right',
  render: (row: CodexKeeperAccount) => {
    return h(
      AppStack,
      { class: 'account-actions', size: 4, wrap: false },
      {
        default: () => [
          h(
            AppButton,
            { size: 'small', quaternary: true, onClick: () => openDetail(row) },
            { default: () => t('详情', 'Details') },
          ),
          h(
            AppButton,
            {
              size: 'small',
              quaternary: true,
              type: row.disabled ? 'primary' : 'warning',
              disabled: isRowActing(row) || isBulkOperationRunning.value,
              loading: isActionLoading(row, 'toggle'),
              onClick: () => row.disabled ? confirmEnableAccount(row) : confirmDisableAccount(row),
            },
            { default: () => row.disabled ? t('启用', 'Enable') : t('禁用', 'Disable') },
          ),
          h(
            AppButton,
            {
              size: 'small',
              quaternary: true,
              type: 'error',
              disabled: isRowActing(row) || isBulkOperationRunning.value,
              loading: isActionLoading(row, 'delete'),
              onClick: () => confirmDeleteAccount(row),
            },
            { default: () => t('删除', 'Delete') },
          ),
          h(
            AppButton,
            {
              size: 'small',
              quaternary: true,
              type: 'primary',
              disabled: isRowActing(row) || isBulkOperationRunning.value,
              loading: isActionLoading(row, 'refresh'),
              onClick: () => refreshAccount(row),
            },
            { default: () => t('刷新', 'Refresh') },
          ),
        ],
      },
    )
  },
}))

const readOnlyActionColumn = computed<DataTableColumns<CodexKeeperAccount>[number]>(() => ({
  title: '',
  key: 'actions',
  width: 72,
  fixed: 'right',
  render: (row: CodexKeeperAccount) =>
    h(
      AppButton,
      { size: 'small', quaternary: true, onClick: () => openDetail(row) },
      { default: () => t('详情', 'Details') },
    ),
}))

const accountColumns = computed<DataTableColumns<CodexKeeperAccount>>(() =>
  canManageAccounts.value
    ? [
        {
          type: 'selection',
          width: 44,
          disabled: (row: CodexKeeperAccount) =>
            isRowActing(row) || isBulkOperationRunning.value,
        },
        ...baseColumns.value,
        manageActionColumn.value,
      ]
    : [...baseColumns.value, readOnlyActionColumn.value],
)

restoreAccountStatusPreferences()

watch(
  [accountDisplaySize, () => accountSort.key, () => accountSort.direction],
  saveAccountStatusPreferences,
)
watch(
  [
    accountDisplaySize,
    () => accountSort.key,
    () => accountSort.direction,
    () => filters.keyword,
    () => filters.accountType,
    () => filters.priority,
    () => filters.status,
  ],
  resetAccountPages,
)
watch(accountListPageCount, clampAccountPages)
watch(visibleListAccounts, pruneSelectedAccountKeys)

onMounted(() => {
  void loadAccounts()
  keeperStatusTimer = window.setInterval(() => {
    relativeTimeNow.value = Date.now()
    void loadKeeperStatus()
  }, KEEPER_STATUS_POLL_INTERVAL_MS)
})

onBeforeUnmount(() => {
  keeperPollTokens.once += 1
  keeperPollTokens.accounts += 1
  oauthPollToken += 1
  if (keeperStatusTimer !== undefined) {
    window.clearInterval(keeperStatusTimer)
  }
})
</script>

<template>
  <section class="page account-status-page">
    <div class="page-toolbar account-page-header">
      <div class="header-actions">
        <AppButton
          v-if="canManageAccounts"
          type="primary"
          @click="openCodexOAuthDialog"
        >
          <template #icon>
            <AppIcon :component="LogIn" />
          </template>
          {{ t('OAuth 登录', 'OAuth Login') }}
        </AppButton>
        <AppButton
          v-if="canManageAccounts"
          type="primary"
          :loading="isUploadingAuthFiles"
          @click="triggerAuthFileUpload"
        >
          <template #icon>
            <AppIcon :component="Upload" />
          </template>
          {{ t('上传文件', 'Upload Files') }}
        </AppButton>
        <AppButton
          v-if="canManageAccounts"
          type="primary"
          :loading="isStartingAccountInspection || isAccountInspectionRunning"
          :disabled="isAccountInspectionBlocked"
          @click="startAccountInspection"
        >
          <template #icon>
            <AppIcon :component="ShieldCheck" />
          </template>
          {{ t('账号巡检', 'Inspect Accounts') }}
        </AppButton>
        <AppButton secondary :loading="isLoading" @click="reloadAccounts">
          <template #icon>
            <AppIcon :component="RefreshCw" />
          </template>
          {{ t('重新加载', 'Reload') }}
        </AppButton>
        <input
          ref="authFileInput"
          class="auth-file-input"
          type="file"
          accept=".json,application/json"
          multiple
          @change="handleAuthFileUpload"
        >
      </div>
    </div>

    <div class="metric-grid account-metrics">
      <div class="metric-card inspection-status-card">
        <div class="metric-icon" aria-hidden="true">
          <Activity :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('运行状态', 'Run Status') }}</div>
        <div class="metric-value inspection-status-value" :title="keeperStatusDetailText">
          <AppBadge class="inspection-status-tag" :type="keeperStateType" size="small" :bordered="false">
            {{ keeperStatusDetailText }}
          </AppBadge>
        </div>
        <div class="metric-footnote">{{ keeperStatusFootnoteText }}</div>
      </div>
      <div class="metric-card">
        <div class="metric-icon" aria-hidden="true">
          <Users :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('账号总数', 'Total Accounts') }}</div>
        <div class="metric-value">{{ formatInteger(accounts.length) }}</div>
        <div class="metric-footnote">{{ t('全部认证文件', 'All auth files') }}</div>
      </div>
      <button
        type="button"
        class="metric-card metric-action is-green"
        :class="{ 'is-active': isStatusFilterActive('enabled') }"
        :aria-pressed="isStatusFilterActive('enabled')"
        @click="toggleStatusFilter('enabled')"
      >
        <div class="metric-icon" aria-hidden="true">
          <ShieldCheck :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('启用中', 'Enabled') }}</div>
        <div class="metric-value">{{ formatInteger(enabledAccountCount) }}</div>
        <div class="metric-footnote">{{ t('可参与调度', 'Available for scheduling') }}</div>
      </button>
      <button
        type="button"
        class="metric-card metric-action is-warning"
        :class="{ 'is-active': isStatusFilterActive('disabled') }"
        :aria-pressed="isStatusFilterActive('disabled')"
        @click="toggleStatusFilter('disabled')"
      >
        <div class="metric-icon" aria-hidden="true">
          <PauseCircle :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('已禁用', 'Disabled') }}</div>
        <div class="metric-value">{{ formatInteger(disabledAccountCount) }}</div>
        <div class="metric-footnote">{{ t('停用账号', 'Inactive accounts') }}</div>
      </button>
      <button
        type="button"
        class="metric-card metric-action is-danger"
        :class="{ 'is-active': isStatusFilterActive('unauthorized') }"
        :aria-pressed="isStatusFilterActive('unauthorized')"
        @click="toggleStatusFilter('unauthorized')"
      >
        <div class="metric-icon" aria-hidden="true">
          <ShieldAlert :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('401报错', '401 Errors') }}</div>
        <div class="metric-value">{{ formatInteger(unauthorizedErrorAccountCount) }}</div>
        <div class="metric-footnote">HTTP 401</div>
      </button>
      <button
        type="button"
        class="metric-card metric-action is-purple"
        :class="{ 'is-active': isStatusFilterActive('quotaExhausted') }"
        :aria-pressed="isStatusFilterActive('quotaExhausted')"
        @click="toggleStatusFilter('quotaExhausted')"
      >
        <div class="metric-icon" aria-hidden="true">
          <Gauge :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('额度耗尽', 'Quota Exhausted') }}</div>
        <div class="metric-value">{{ formatInteger(quotaExhaustedAccountCount) }}</div>
        <div class="metric-footnote">{{ t('临时降级', 'Temporary downgrade') }}</div>
      </button>
    </div>

    <section class="panel account-list-panel">
      <div class="status-toolbar">
        <div class="toolbar-heading">
          <div>
            <h2 class="toolbar-title">{{ t('账号列表', 'Account List') }}</h2>
            <p class="toolbar-subtitle">
              {{ t(`正常 ${filteredNormalAccounts.length} / ${enabledAccountCount} 个账号`, `Normal ${filteredNormalAccounts.length} / ${enabledAccountCount} accounts`) }}
              <template v-if="hasDisabledAccounts">
                {{ t(`，已禁用 ${filteredDisabledAccounts.length} / ${disabledAccountCount} 个账号`, `, disabled ${filteredDisabledAccounts.length} / ${disabledAccountCount} accounts`) }}
              </template>
            </p>
          </div>
          <AppBadge v-if="activeFilterCount > 0" size="small" type="info" :bordered="false">
            {{ t(`已筛选 ${activeFilterCount} 项`, `${activeFilterCount} filters active`) }}
          </AppBadge>
        </div>
        <div class="filter-grid">
          <AppInput v-model:value="filters.keyword" clearable :placeholder="t('搜索账号或邮箱', 'Search account or email')" />
          <AppSelect
            v-model:value="filters.accountType"
            :options="accountTypeOptions"
            clearable
            filterable
            :placeholder="t('账号类型', 'Account Type')"
          />
          <AppSelect
            v-model:value="filters.priority"
            :options="priorityFilterOptions"
          />
        </div>
      </div>

      <div class="account-sections">
        <section class="account-section">
          <div class="account-section-actions-row">
            <div class="sort-control-row" :aria-label="t('账号排序', 'Account Sorting')">
              <span class="sort-control-label">{{ t('排序', 'Sort') }}</span>
              <AppDropdown trigger="click" :options="quotaSortOptions" @select="handleQuotaSortSelect">
                <AppButton
                  secondary
                  size="small"
                  :type="accountSort.key === 'quotaDay' || accountSort.key === 'quotaWeek' ? 'primary' : 'default'"
                >
                  {{ activeQuotaSortLabel ? t(`额度窗口：${activeQuotaSortLabel} ${sortDirectionMark}`, `Quota Window: ${activeQuotaSortLabel} ${sortDirectionMark}`) : t('额度窗口', 'Quota Window') }}
                </AppButton>
              </AppDropdown>
              <AppButton
                secondary
                size="small"
                :type="isAccountSortActive('accountType') ? 'primary' : 'default'"
                @click="toggleAccountSort('accountType')"
              >
                {{ t('类型', 'Type') }} {{ accountSortMark('accountType') }}
              </AppButton>
              <AppButton
                secondary
                size="small"
                :type="isAccountSortActive('status') ? 'primary' : 'default'"
                @click="toggleAccountSort('status')"
              >
                {{ t('状态', 'Status') }} {{ accountSortMark('status') }}
              </AppButton>
              <AppButton
                secondary
                size="small"
                :type="isAccountSortActive('priority') ? 'primary' : 'default'"
                @click="toggleAccountSort('priority')"
              >
                {{ t('优先级', 'Priority') }} {{ accountSortMark('priority') }}
              </AppButton>
              <AppButton
                secondary
                size="small"
                :type="isAccountSortActive('lastCheckedAt') ? 'primary' : 'default'"
                @click="toggleAccountSort('lastCheckedAt')"
              >
                {{ t('最近巡检', 'Last Inspection') }} {{ accountSortMark('lastCheckedAt') }}
              </AppButton>
            </div>
            <div v-if="canManageAccounts" class="account-section-actions">
              <AppButton
                secondary
                type="primary"
                size="small"
                :disabled="!canBulkEnable"
                :loading="bulkToggleAction === 'enable'"
                @click="confirmToggleSelectedAccounts('enable')"
              >
                <template #icon>
                  <AppIcon :component="ShieldCheck" />
                </template>
                {{ t('启用', 'Enable') }}
              </AppButton>
              <AppButton
                secondary
                type="warning"
                size="small"
                :disabled="!canBulkDisable"
                :loading="bulkToggleAction === 'disable'"
                @click="confirmToggleSelectedAccounts('disable')"
              >
                <template #icon>
                  <AppIcon :component="PauseCircle" />
                </template>
                {{ t('禁用', 'Disable') }}
              </AppButton>
              <AppButton
                secondary
                type="primary"
                size="small"
                :disabled="!canRefreshSelected"
                :loading="isBulkRefreshing"
                @click="refreshSelectedAccounts"
              >
                <template #icon>
                  <AppIcon :component="RefreshCw" />
                </template>
                {{ t('刷新', 'Refresh') }}
              </AppButton>
              <AppButton
                secondary
                type="error"
                size="small"
                :disabled="!canBulkDelete"
                :loading="isBulkDeleting"
                @click="openBulkDeleteDialog"
              >
                <template #icon>
                  <AppIcon :component="Trash2" />
                </template>
                {{ t('删除', 'Delete') }}
              </AppButton>
            </div>
          </div>
          <AppDataTable
            class="account-table"
            size="small"
            :loading="tableLoading"
            :columns="accountColumns"
            :data="visibleListAccounts"
            :row-key="accountRowKey"
            :checked-row-keys="selectedAccountKeys"
            :pagination="false"
            table-layout="fixed"
            :scroll-x="accountTableScrollX"
            @update:checked-row-keys="handleAccountSelectionUpdate"
          >
            <template #empty>
              <div class="empty-state">
                {{ showListLoadingState ? t('账号加载中...', 'Loading accounts...') : t('当前筛选下暂无账号', 'No accounts match the current filter') }}
              </div>
            </template>
          </AppDataTable>
        </section>
      </div>

      <div class="account-table-footer">
        <div class="account-range-controls">
          <span class="account-range-text">
            <span>{{ t(`第 ${accountRangeStart} - ${accountRangeEnd} 条`, `Items ${accountRangeStart} - ${accountRangeEnd}`) }}</span>
            <span>{{ t(`共 ${sortedListAccounts.length} 条`, `${sortedListAccounts.length} total`) }}</span>
          </span>
          <div class="page-size-control">
            <span>{{ t('每页显示', 'Show') }}</span>
            <AppSelect
              v-model:value="accountDisplaySize"
              class="display-size-select"
              size="small"
              :options="accountDisplaySizeOptions"
            />
            <span>{{ t('条', 'per page') }}</span>
          </div>
        </div>
        <AppPagination
          v-if="showAccountPagination"
          v-model:page="accountListPage"
          size="small"
          :page-size="accountDisplaySize"
          :item-count="sortedListAccounts.length"
        />
      </div>
    </section>

    <CodexKeeperLogsPanel
      v-if="canManageAccounts"
      :logs="keeperStatus?.logs ?? []"
      @refresh="loadKeeperStatus"
    />

    <AppDrawer v-model:show="detailOpen" placement="right" :width="420">
      <AppDrawerContent>
        <template #header>
          <div class="detail-drawer-header">
            <AppButton quaternary size="small" class="detail-back-button" @click="detailOpen = false">
              <template #icon>
                <AppIcon :component="ArrowLeft" />
              </template>
              {{ t('返回', 'Back') }}
            </AppButton>
            <span class="detail-drawer-title">{{ t('账号详情', 'Account Details') }}</span>
          </div>
        </template>
        <AppDescriptions v-if="selectedAccount" label-placement="left" :column="1" size="small" bordered>
          <AppDescriptionsItem :label="t('账号', 'Account')">{{ selectedAccount.name }}</AppDescriptionsItem>
          <AppDescriptionsItem :label="t('邮箱', 'Email')">{{ selectedAccount.email ?? '-' }}</AppDescriptionsItem>
          <AppDescriptionsItem v-if="canManageAccounts" :label="t('备注', 'Note')">
            {{ isSelectedAccountNoteLoading ? t('加载中...', 'Loading...') : (selectedAccountNote ?? '-') }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('账号类型', 'Account Type')">
            {{ accountTypeLabel(selectedAccount.account_type) }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('启用状态', 'Enabled Status')">
            {{ selectedAccount.disabled ? t('已禁用', 'Disabled') : t('启用中', 'Enabled') }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('当前优先级', 'Current Priority')">
            {{ accountPriority(selectedAccount) }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('类型默认优先级', 'Type Default Priority')">
            {{ defaultPriority(selectedAccount) ?? '-' }}
          </AppDescriptionsItem>
          <AppDescriptionsItem v-if="shouldShowQuotaWindow(selectedAccount)" :label="t('额度窗口', 'Quota Window')">
            {{ quotaText(selectedAccount) }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('状态码', 'Status Code')">
            {{ selectedAccount.last_status_code ?? '-' }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('最近健康', 'Last Healthy')">
            {{ formatDateTime(selectedAccount.last_healthy_at) }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('最近巡检', 'Last Inspection')">
            {{ formatDateTime(selectedAccount.last_checked_at) }}
          </AppDescriptionsItem>
          <AppDescriptionsItem :label="t('最近操作', 'Latest Action')">
            {{ latestActionText(selectedAccount) }}
          </AppDescriptionsItem>
        </AppDescriptions>
        <div v-if="selectedAccount && canManageAccounts" class="detail-action-row">
          <AppStack :size="8" wrap>
            <AppButton
              size="small"
              secondary
              @click="openAuthFileEditor(selectedAccount)"
            >
              <template #icon>
                <AppIcon :component="Pencil" />
              </template>
              {{ t('认证文件详情 / 编辑', 'Auth File Details / Edit') }}
            </AppButton>
            <AppButton
              size="small"
              type="primary"
              secondary
              :disabled="isRowActing(selectedAccount) || isBulkOperationRunning"
              :loading="isActionLoading(selectedAccount, 'refresh')"
              @click="refreshAccount(selectedAccount, { closeDetail: true })"
            >
              {{ t('刷新', 'Refresh') }}
            </AppButton>
            <AppButton
              v-if="selectedAccount.disabled"
              size="small"
              type="primary"
              secondary
              :disabled="isRowActing(selectedAccount) || isBulkOperationRunning"
              :loading="isActionLoading(selectedAccount, 'toggle')"
              @click="confirmEnableAccount(selectedAccount)"
            >
              {{ t('启用', 'Enable') }}
            </AppButton>
            <AppButton
              v-else
              size="small"
              type="warning"
              secondary
              :disabled="isRowActing(selectedAccount) || isBulkOperationRunning"
              :loading="isActionLoading(selectedAccount, 'toggle')"
              @click="confirmDisableAccount(selectedAccount)"
            >
              {{ t('禁用', 'Disable') }}
            </AppButton>
            <AppButton
              size="small"
              secondary
              :disabled="isRowActing(selectedAccount) || isBulkOperationRunning"
              :loading="isActionLoading(selectedAccount, 'priority')"
              @click="openPriorityDialog(selectedAccount)"
            >
              {{ t('修改优先级', 'Change Priority') }}
            </AppButton>
            <AppButton
              size="small"
              type="error"
              secondary
              :disabled="isRowActing(selectedAccount) || isBulkOperationRunning"
              :loading="isActionLoading(selectedAccount, 'delete')"
              @click="confirmDeleteAccount(selectedAccount)"
            >
              {{ t('删除', 'Delete') }}
            </AppButton>
          </AppStack>
        </div>
      </AppDrawerContent>
    </AppDrawer>

    <AppModal
      v-if="canManageAccounts"
      :show="oauthDialogOpen"
      preset="card"
      :style="{ width: 'min(640px, calc(100vw - 32px))' }"
      :title="t('Codex OAuth', 'Codex OAuth')"
      :mask-closable="!isStartingOAuth && !isSubmittingOAuthCallback"
      @update:show="(show) => { if (!show) closeCodexOAuthDialog() }"
    >
      <div class="oauth-dialog">
        <p class="oauth-dialog-description">
          {{ t('通过 Codex OAuth 登录，认证成功后生成的认证文件会自动加入账号管理。', 'Sign in with Codex OAuth. The generated auth file will be added to account management automatically.') }}
        </p>
        <div class="oauth-status-row">
          <span>{{ t('认证状态', 'Authentication status') }}</span>
          <AppBadge :type="oauthStatusType" size="small" :bordered="false">
            {{ oauthStatusText }}
          </AppBadge>
        </div>
        <p
          v-if="oauthError"
          class="oauth-dialog-message"
          :class="oauthDialogStatus === 'error' ? 'is-error' : 'is-warning'"
        >
          {{ oauthError }}
        </p>
        <AppButton
          v-if="oauthDialogStatus === 'idle'"
          type="primary"
          :loading="isStartingOAuth"
          @click="startCodexOAuth"
        >
          {{ t('开始 Codex 登录', 'Start Codex Login') }}
        </AppButton>
        <div v-if="oauthAuthURL" class="oauth-dialog-section">
          <label>{{ t('授权链接', 'Authorization link') }}</label>
          <AppInput
            type="textarea"
            readonly
            :value="oauthAuthURL"
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
          <AppStack>
            <AppButton type="primary" secondary @click="openCodexOAuthURL">
              <template #icon>
                <AppIcon :component="ExternalLink" />
              </template>
              {{ t('打开链接', 'Open Link') }}
            </AppButton>
            <AppButton secondary @click="copyCodexOAuthURL">
              <template #icon>
                <AppIcon :component="Copy" />
              </template>
              {{ t('复制链接', 'Copy Link') }}
            </AppButton>
          </AppStack>
        </div>
        <div v-if="oauthDialogStatus === 'waiting'" class="oauth-dialog-section">
          <label>{{ t('回调 URL', 'Callback URL') }}</label>
          <p class="oauth-dialog-hint">
            {{ t('如果当前浏览器无法访问 localhost 回调地址，请复制浏览器最终跳转后的完整 URL 并粘贴到这里。', 'If this browser cannot reach the localhost callback, paste the complete URL from the browser after its final redirect here.') }}
          </p>
          <AppInput
            v-model:value="oauthCallbackURL"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 5 }"
            :placeholder="t('粘贴完整回调 URL', 'Paste the complete callback URL')"
          />
          <AppButton
            type="primary"
            secondary
            :disabled="!oauthCallbackURL.trim()"
            :loading="isSubmittingOAuthCallback"
            @click="submitCodexOAuthCallbackURL"
          >
            {{ t('提交回调 URL', 'Submit Callback URL') }}
          </AppButton>
        </div>
        <AppButton
          v-if="oauthDialogStatus === 'success' || oauthDialogStatus === 'error'"
          secondary
          :loading="isStartingOAuth"
          @click="startCodexOAuth"
        >
          {{ t('登录另一个账号', 'Sign in to another account') }}
        </AppButton>
      </div>
      <template #footer>
        <AppStack justify="end">
          <AppButton
            :disabled="isStartingOAuth || isSubmittingOAuthCallback"
            @click="closeCodexOAuthDialog"
          >
            {{ t('关闭', 'Close') }}
          </AppButton>
        </AppStack>
      </template>
    </AppModal>

    <AppModal
      v-if="canManageAccounts"
      :show="authFileEditor !== null"
      preset="card"
      :style="{ width: 'min(720px, calc(100vw - 32px))' }"
      :title="authFileEditor ? t(`认证文件详情 / 编辑 - ${authFileEditor.fileName}`, `Auth File Details / Edit - ${authFileEditor.fileName}`) : t('认证文件详情 / 编辑', 'Auth File Details / Edit')"
      :mask-closable="!authFileEditor?.saving"
      @update:show="(show) => { if (!show) closeAuthFileEditor() }"
    >
      <div v-if="authFileEditor" class="auth-file-editor">
        <div v-if="authFileEditor.loading" class="auth-file-editor-loading">
          {{ t('正在加载认证文件...', 'Loading auth file...') }}
        </div>
        <template v-else>
          <div v-if="authFileEditor.error" class="auth-file-editor-error">
            {{ authFileEditor.error }}
          </div>
          <div class="auth-file-editor-json-block">
            <label>{{ t('认证文件信息（info）', 'Auth file info (info)') }}</label>
            <AppInput
              type="textarea"
              readonly
              :value="authFileEditor.fileInfoText"
              :autosize="{ minRows: 6, maxRows: 10 }"
            />
          </div>
          <div class="auth-file-editor-json-block">
            <label>
              {{ authFileEditor.json
                ? t('认证文件 JSON（预览）', 'Auth file JSON (preview)')
                : t('下载内容（已截断）', 'Downloaded content (truncated)') }}
            </label>
            <AppInput
              v-if="authFileEditor.json"
              type="textarea"
              readonly
              :value="authFileEditorUpdatedText"
              :autosize="{ minRows: 8, maxRows: 16 }"
            />
            <pre v-else class="auth-file-editor-invalid-preview">{{ authFileEditor.invalidContentPreview }}</pre>
          </div>
          <div v-if="authFileEditor.json" class="auth-file-editor-fields">
            <div class="auth-file-editor-field">
              <label>{{ t('前缀（prefix）', 'Prefix (prefix)') }}</label>
              <AppInput v-model:value="authFileEditor.prefix" />
            </div>
            <div class="auth-file-editor-field">
              <label>{{ t('代理 URL（proxy_url）', 'Proxy URL (proxy_url)') }}</label>
              <AppInput v-model:value="authFileEditor.proxyUrl" :placeholder="t('socks5://username:password@proxy_ip:port/', 'socks5://username:password@proxy_ip:port/')" />
            </div>
            <div class="auth-file-editor-field">
              <label>{{ t('优先级（priority）', 'Priority (priority)') }}</label>
              <AppInput v-model:value="authFileEditor.priority" :placeholder="t('例如：10 或 -1', 'For example: 10 or -1')" />
              <span class="auth-file-editor-hint">{{ t('仅支持整数；数值越大优先级越高。', 'Integers only; higher values have higher priority.') }}</span>
            </div>
            <div class="auth-file-editor-field auth-file-editor-switch-field">
              <label>{{ t('WebSockets（websockets）', 'WebSockets (websockets)') }}</label>
              <AppSwitch v-model:value="authFileEditor.websockets" @update:value="authFileEditor.websocketsTouched = true" />
            </div>
            <div class="auth-file-editor-field auth-file-editor-wide-field">
              <label>{{ t('自定义请求头（headers）', 'Custom headers (headers)') }}</label>
              <AppInput
                type="textarea"
                :value="authFileEditor.headersText"
                :placeholder="'{\n  &quot;Header-Name&quot;: &quot;value&quot;\n}'"
                :autosize="{ minRows: 4, maxRows: 8 }"
                @update:value="handleAuthFileHeadersChange"
              />
              <span v-if="authFileEditor.headersError" class="auth-file-editor-error">{{ authFileEditor.headersError }}</span>
              <span v-else class="auth-file-editor-hint">{{ t('以 JSON 对象格式输入，每个值都必须是字符串。', 'Enter a JSON object; every value must be a string.') }}</span>
            </div>
            <div class="auth-file-editor-field auth-file-editor-wide-field">
              <label>{{ t('备注（note）', 'Note (note)') }}</label>
              <AppInput
                v-model:value="authFileEditor.note"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 4 }"
                :placeholder="t('输入备注信息，例如：张三的账号', 'Enter a note, for example: account owner')"
                @update:value="authFileEditor.noteTouched = true"
              />
            </div>
          </div>
        </template>
      </div>
      <template #footer>
        <AppStack justify="end">
          <AppButton :disabled="authFileEditor?.saving === true" @click="closeAuthFileEditor">
            {{ authFileEditorDirty ? t('取消', 'Cancel') : t('关闭', 'Close') }}
          </AppButton>
          <AppButton
            secondary
            :disabled="authFileEditor?.saving === true || !authFileEditorUpdatedText"
            @click="copyAuthFileEditorText"
          >
            {{ t('复制', 'Copy') }}
          </AppButton>
          <AppButton
            type="primary"
            :loading="authFileEditor?.saving === true"
            :disabled="!authFileEditorDirty || !!authFileEditor?.headersError || authFileEditor?.loading === true"
            @click="saveAuthFileEditor"
          >
            {{ t('保存', 'Save') }}
          </AppButton>
        </AppStack>
      </template>
    </AppModal>

    <AppModal
      v-if="canManageAccounts"
      v-model:show="accountConfirmDialog.show"
      preset="dialog"
      :title="accountConfirmDialog.title"
      :style="{ width: 'min(420px, calc(100vw - 32px))' }"
    >
      <p class="account-confirm-content">{{ accountConfirmDialog.content }}</p>
      <template #action>
        <AppStack justify="end">
          <AppButton :disabled="isAccountConfirmSubmitting" @click="accountConfirmDialog.show = false">
            {{ t('取消', 'Cancel') }}
          </AppButton>
          <AppButton
            :type="accountConfirmDialog.type"
            :loading="isAccountConfirmSubmitting"
            @click="submitAccountConfirm"
          >
            {{ accountConfirmDialog.positiveText }}
          </AppButton>
        </AppStack>
      </template>
    </AppModal>

    <AppModal
      v-if="canManageAccounts"
      v-model:show="bulkDeleteDialog.show"
      preset="dialog"
      :title="bulkDeleteDialogTitle"
      :style="{ width: 'min(460px, calc(100vw - 32px))' }"
    >
      <div class="bulk-delete-dialog">
        <p class="bulk-delete-warning">
          {{ bulkDeleteWarningText }}
        </p>
        <div v-if="bulkDeletePreviewNames.length > 0" class="bulk-delete-preview">
          <span v-for="name in bulkDeletePreviewNames" :key="name">{{ name }}</span>
          <span v-if="bulkDeletePreviewOverflow > 0">{{ t(`另 ${bulkDeletePreviewOverflow} 个...`, `${bulkDeletePreviewOverflow} more...`) }}</span>
        </div>
      </div>
      <template #action>
        <AppStack justify="end">
          <AppButton :disabled="isBulkDeleting" @click="bulkDeleteDialog.show = false">{{ t('取消', 'Cancel') }}</AppButton>
          <AppButton
            type="error"
            :disabled="selectedAccountCount === 0"
            :loading="isBulkDeleting"
            @click="submitBulkDelete"
          >
            {{ t('确认删除', 'Confirm Delete') }}
          </AppButton>
        </AppStack>
      </template>
    </AppModal>

    <AppModal
      v-if="canManageAccounts"
      v-model:show="priorityDialog.show"
      preset="dialog"
      :title="priorityDialogTitle"
      :style="{ width: 'min(460px, calc(100vw - 32px))' }"
    >
      <div class="priority-dialog">
        <AppSelect
          :value="priorityDialog.mode"
          :options="priorityModeOptions"
          @update:value="(value) => setPriorityDialogMode(value as PriorityMode)"
        />
        <AppNumberInput
          v-if="priorityDialog.mode !== 'default'"
          v-model:value="priorityDialog.value"
          :precision="0"
          v-bind="priorityDialogBounds"
        />
        <p class="priority-hint">{{ priorityDialogHint }}</p>
      </div>
      <template #action>
        <AppStack justify="end">
          <AppButton @click="priorityDialog.show = false">{{ t('取消', 'Cancel') }}</AppButton>
          <AppButton
            type="primary"
            :disabled="!canSubmitPriority"
            :loading="
              priorityDialog.account
                ? isActionLoading(priorityDialog.account, 'priority')
                : false
            "
            @click="submitPriorityDialog"
          >
            {{ t('确认', 'Confirm') }}
          </AppButton>
        </AppStack>
      </template>
    </AppModal>
  </section>
</template>

<style scoped>
.account-status-page,
.account-list-panel,
.account-section,
.account-table {
  min-width: 0;
}

.account-header-copy {
  flex: 1;
  min-width: 0;
}

.header-actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  flex-shrink: 0;
  justify-content: flex-end;
}

.auth-file-input {
  display: none;
}

.account-metrics {
  grid-template-columns: repeat(6, minmax(112px, 1fr));
}

.account-metrics .metric-card {
  min-height: 104px;
  padding: 14px 12px;
}

.account-metrics .metric-action {
  width: 100%;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  appearance: none;
}

.account-metrics .metric-action:hover {
  border-color: color-mix(in srgb, var(--metric-color, var(--cpa-primary)) 45%, var(--cpa-border));
  transform: translateY(-1px);
}

.account-metrics .metric-action:focus-visible {
  outline: 2px solid var(--metric-color, var(--cpa-primary));
  outline-offset: 3px;
}

.account-metrics .metric-action.is-active {
  border-color: color-mix(in srgb, var(--metric-color, var(--cpa-primary)) 65%, var(--cpa-border));
  box-shadow:
    0 0 0 3px color-mix(in srgb, var(--metric-color, var(--cpa-primary)) 16%, transparent),
    var(--cpa-shadow-card),
    var(--cpa-shadow-hairline);
}

.account-metrics .metric-value {
  font-size: 20px;
}

.inspection-status-card {
  min-width: 0;
}

.inspection-status-value {
  display: flex;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}

.inspection-status-tag {
  max-width: 100%;
  min-width: 0;
}

.inspection-status-value :deep(.n-tag) {
  max-width: 100%;
  min-width: 0;
}

.inspection-status-value :deep(.n-tag__content) {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-toolbar {
  display: grid;
  gap: 12px;
  padding: 14px;
  border-bottom: 1px solid var(--cpa-border);
  background: var(--cpa-glass);
  backdrop-filter: blur(14px);
}

.toolbar-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.toolbar-title {
  margin: 0;
  color: var(--cpa-text);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
}

.toolbar-subtitle {
  margin: 3px 0 0;
  color: var(--cpa-text-muted);
  font-size: 13px;
}

.filter-grid {
  display: grid;
  grid-template-columns: minmax(220px, 1.35fr) minmax(150px, 0.8fr) minmax(170px, 0.9fr);
  gap: 8px;
  min-width: 0;
}

.sort-control-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  min-width: 0;
}

.sort-control-label {
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.account-sections {
  display: grid;
  gap: 14px;
  padding: 14px;
}

.account-table-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px;
  border-top: 1px solid var(--cpa-border);
  background: var(--cpa-surface-raised);
}

.account-range-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 14px;
  min-width: 0;
}

.account-range-text,
.page-size-control {
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.account-range-text {
  display: inline-flex;
  align-items: center;
  gap: 12px;
}

.page-size-control {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.display-size-select {
  flex-shrink: 0;
  width: 82px;
}

.account-section {
  display: grid;
  gap: 10px;
}

.account-section-actions-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px 14px;
  min-width: 0;
}

.account-section-actions {
  display: flex;
  flex-wrap: wrap;
  flex-shrink: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.account-table-footer :deep(.n-pagination) {
  flex-wrap: wrap;
  justify-content: flex-end;
}


.detail-action-row {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--cpa-border);
}

.detail-drawer-header {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.detail-back-button {
  flex-shrink: 0;
}

.detail-drawer-title {
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-text-strong);
  font-size: 16px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.oauth-dialog {
  display: grid;
  gap: 14px;
  min-width: 0;
}

.oauth-dialog-description,
.oauth-dialog-message,
.oauth-dialog-hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
}

.oauth-dialog-description,
.oauth-dialog-hint {
  color: var(--cpa-text-muted);
}

.oauth-dialog-message.is-error {
  color: var(--cpa-danger);
}

.oauth-dialog-message.is-warning {
  color: var(--cpa-warning);
}

.oauth-status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  color: var(--cpa-text);
  font-size: 13px;
  background: var(--cpa-surface-muted);
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
}

.oauth-dialog-section {
  display: grid;
  gap: 8px;
  min-width: 0;
}

.oauth-dialog-section > label {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.oauth-dialog-section :deep(.n-input__textarea) {
  font-family: "Cascadia Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
}

.auth-file-editor {
  display: grid;
  gap: 14px;
  min-width: 0;
  max-height: min(72vh, 720px);
  overflow-y: auto;
  padding-right: 2px;
}

.auth-file-editor-loading {
  padding: 24px 0;
  color: var(--cpa-text-muted);
  text-align: center;
}

.auth-file-editor-error {
  color: var(--cpa-danger);
  font-size: 12px;
  line-height: 1.45;
}

.auth-file-editor-json-block,
.auth-file-editor-field {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.auth-file-editor-json-block > label,
.auth-file-editor-field > label {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.auth-file-editor-json-block :deep(.n-input__textarea),
.auth-file-editor-field :deep(.n-input__textarea) {
  font-family: "Cascadia Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
}

.auth-file-editor-invalid-preview {
  min-height: 120px;
  max-height: 260px;
  margin: 0;
  overflow: auto;
  padding: 10px 12px;
  border: 1px solid var(--cpa-border);
  border-radius: 6px;
  background: var(--cpa-surface-muted);
  color: var(--cpa-text);
  font-family: "Cascadia Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.auth-file-editor-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  min-width: 0;
}

.auth-file-editor-wide-field {
  grid-column: 1 / -1;
}

.auth-file-editor-switch-field {
  align-content: start;
}

.auth-file-editor-hint {
  color: var(--cpa-text-muted);
  font-size: 11px;
  line-height: 1.4;
}

.account-table :deep(.n-data-table-th) {
  white-space: nowrap;
}

.account-table :deep(.n-data-table-td) {
  vertical-align: middle;
}

.account-table :deep(.n-data-table-tr.is-refresh-selectable) {
  cursor: pointer;
}

.account-table :deep(.n-data-table-tr.is-refresh-selected .n-data-table-td) {
  background: color-mix(in srgb, var(--cpa-primary) 12%, var(--cpa-surface-raised));
}

.account-table :deep(.n-data-table-tr.is-refresh-selected:hover .n-data-table-td) {
  background: color-mix(in srgb, var(--cpa-primary) 16%, var(--cpa-surface-raised));
}

:global(.quota-window-cell) {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 4px 0;
}

:global(.quota-window-item) {
  display: grid;
  align-content: center;
  gap: 4px;
  min-width: 0;
  min-height: 38px;
}

:global(.quota-window-head) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
  line-height: 1.2;
}

:global(.quota-window-label) {
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.quota-window-meta) {
  display: inline-flex;
  flex-shrink: 0;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

:global(.quota-window-percent) {
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 700;
}

:global(.quota-window-reset) {
  color: var(--cpa-text-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

:global(.quota-window-usage) {
  overflow: hidden;
  color: var(--cpa-text-muted);
  font-size: 11px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.quota-window-usage.is-stale) {
  color: var(--cpa-warning);
}

:global(.quota-usage-cell) {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 4px 0;
}

:global(.quota-usage-item) {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  min-width: 0;
  min-height: 38px;
}

:global(.quota-usage-item.is-projection) {
  grid-template-columns: minmax(0, 1fr);
}

:global(.quota-usage-chip) {
  --usage-accent: var(--cpa-primary);
  display: grid;
  align-content: center;
  gap: 2px;
  min-width: 0;
  min-height: 38px;
  padding: 6px 7px;
  overflow: hidden;
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--usage-accent) 5%, var(--cpa-surface-raised)),
      color-mix(in srgb, var(--cpa-surface-muted) 90%, var(--cpa-surface-raised))
    );
  border: 1px solid color-mix(in srgb, var(--usage-accent) 18%, var(--cpa-border));
  border-radius: var(--cpa-radius-sm);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--cpa-surface-raised) 70%, transparent);
}

:global(.quota-usage-chip:nth-child(2)) {
  --usage-accent: var(--cpa-accent-blue);
}

:global(.quota-usage-chip:nth-child(3)) {
  --usage-accent: var(--cpa-accent-orange);
}

:global(.quota-usage-chip.is-projection) {
  --usage-accent: var(--cpa-accent-purple);
}

:global(.quota-usage-chip-label),
:global(.quota-usage-chip-value) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.quota-usage-chip-label) {
  color: color-mix(in srgb, var(--usage-accent) 62%, var(--cpa-text-muted));
  font-size: 10px;
  font-weight: 700;
  line-height: 1.1;
}

:global(.quota-usage-chip-value) {
  color: var(--cpa-text-strong);
  font-size: 12px;
  font-weight: 800;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

:global(.quota-usage-chip.is-stale) {
  --usage-accent: var(--cpa-warning);
  grid-column: 1 / -1;
  background: color-mix(in srgb, var(--cpa-warning) 9%, var(--cpa-surface-raised));
  border-color: color-mix(in srgb, var(--cpa-warning) 20%, var(--cpa-border));
}

:global(.quota-usage-chip.is-stale .quota-usage-chip-value) {
  color: var(--cpa-warning);
}

:global(.quota-window-track) {
  height: 5px;
  overflow: hidden;
  background: var(--cpa-surface-muted);
  border-radius: 999px;
}

:global(.quota-window-fill) {
  height: 100%;
  min-width: 3px;
  border-radius: inherit;
}

:global(.quota-window-fill.is-healthy) {
  background: var(--cpa-success);
}

:global(.quota-window-fill.is-warning) {
  background: var(--cpa-warning);
}

:global(.quota-window-fill.is-danger) {
  background: var(--cpa-danger);
}

:global(.account-table-identity) {
  display: grid;
  gap: 3px;
  min-width: 0;
  line-height: 1.25;
}

:global(.account-table-email),
:global(.account-table-name) {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.account-table-email) {
  color: var(--cpa-text-strong);
  font-size: 13px;
  font-weight: 650;
}

:global(.account-table-name) {
  color: var(--cpa-text-muted);
  font-size: 12px;
  font-weight: 500;
}

:global(.account-table-meta) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
  padding-top: 1px;
}

:global(.account-table-chip) {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  min-width: 0;
  padding: 1px 6px;
  overflow: hidden;
  font-size: 11px;
  font-weight: 700;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 1px solid transparent;
  border-radius: var(--cpa-radius-sm);
  font-variant-numeric: tabular-nums;
}

:global(.account-table-chip.is-type) {
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 750;
  background: var(--cpa-surface-muted);
  border-color: color-mix(in srgb, var(--cpa-border) 72%, transparent);
}

:global(.account-table-chip.is-success) {
  color: var(--cpa-success);
  background: var(--cpa-success-weak);
  border-color: color-mix(in srgb, var(--cpa-success) 26%, transparent);
}

:global(.account-table-chip.is-warning) {
  color: var(--cpa-warning);
  background: var(--cpa-warning-weak);
  border-color: color-mix(in srgb, var(--cpa-warning) 26%, transparent);
}

:global(.account-table-chip.is-danger) {
  color: var(--cpa-danger);
  background: var(--cpa-danger-weak);
  border-color: color-mix(in srgb, var(--cpa-danger) 26%, transparent);
}

:global(.account-table-chip.is-purple) {
  color: var(--cpa-accent-purple);
  background: var(--cpa-accent-purple-weak);
  border-color: color-mix(in srgb, var(--cpa-accent-purple) 26%, transparent);
}

:global(.account-table-chip.is-priority) {
  color: var(--cpa-primary);
  background: color-mix(in srgb, var(--cpa-primary) 9%, var(--cpa-surface-muted));
  border-color: color-mix(in srgb, var(--cpa-primary) 24%, transparent);
}

:global(.account-table-value-pill) {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  min-width: 0;
  padding: 3px 8px;
  overflow: hidden;
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  font-variant-numeric: tabular-nums;
}

:global(.account-table-value-pill.is-time) {
  color: var(--cpa-primary);
  background: color-mix(in srgb, var(--cpa-primary) 8%, var(--cpa-surface-muted));
  border-color: color-mix(in srgb, var(--cpa-primary) 22%, transparent);
}

:global(.account-table-value-pill.is-action) {
  display: -webkit-box;
  line-height: 1.5;
  white-space: normal;
  overflow-wrap: anywhere;
  background: color-mix(in srgb, var(--cpa-text-muted) 8%, var(--cpa-surface-muted));
  border-color: color-mix(in srgb, var(--cpa-border) 78%, transparent);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  word-break: break-word;
}

:global(.account-table-value-pill.is-action.is-empty) {
  display: inline-flex;
  white-space: nowrap;
}

:global(.account-table-value-pill.is-empty) {
  color: var(--cpa-text-muted);
  font-weight: 700;
}

:global(.account-actions) {
  justify-content: flex-end;
}

.empty-state {
  padding: 42px 0;
  color: var(--cpa-text-muted);
  font-size: 13px;
  text-align: center;
}

.bulk-delete-dialog,
.priority-dialog {
  display: grid;
  gap: 8px;
}

.account-confirm-content {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--cpa-text);
  font-size: 13px;
  line-height: 1.6;
}

.bulk-delete-warning,
.priority-hint {
  margin: 0;
  color: var(--cpa-text-muted);
  font-size: 13px;
}

.bulk-delete-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-height: 116px;
  padding: 8px;
  overflow: auto;
  color: var(--cpa-text);
  font-size: 12px;
  background: var(--cpa-surface-muted);
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
}

.bulk-delete-preview span {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1280px) {
  .account-metrics {
    grid-template-columns: repeat(3, minmax(112px, 1fr));
  }
}

@media (max-width: 980px) {
  .filter-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sort-control-row {
    justify-content: flex-start;
  }
}

@media (max-width: 560px) {
  .account-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .auth-file-editor-fields {
    grid-template-columns: minmax(0, 1fr);
  }

  .toolbar-heading,
  .account-section-actions-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .account-section-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .account-table-footer {
    align-items: flex-start;
    flex-direction: column;
  }

  .sort-control-row {
    width: 100%;
  }

  .filter-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .filter-grid .n-input {
    grid-column: 1 / -1;
  }

}
</style>
