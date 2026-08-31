<script setup lang="ts">
import type { Component } from 'vue'
import { computed, onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
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
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
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
import { Switch } from '@/components/ui/switch'
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
  AlertTriangle,
  CircleDollarSign,
  KeyRound,
  Pencil,
  Plus,
  RefreshCw,
  ShieldCheck,
  UserRound,
} from '@lucide/vue'

import {
  createUser,
  disableUser,
  enableUser,
  listUsers,
  updateUser,
  updateUserQuota,
} from '@/features/users/api/usersApi'
import { useI18n } from '@/shared/i18n'
import type { UserSummary } from '@/shared/types/api'
import { formatCompact, formatDateTime, formatInteger, formatUsd } from '@/shared/utils/format'

const message = toast
const { errorText, t } = useI18n()
const isLoading = ref(false)
const isSavingUser = ref(false)
const users = ref<UserSummary[]>([])
const editorVisible = ref(false)
const editingUserId = ref<number | null>(null)
const userAccount = ref('')
const userPassword = ref('')
const isUserAdmin = ref(false)
const userEnabled = ref(true)
const originalUserEnabled = ref(true)
const userNickname = ref('')
const quotaUnlimited = ref(true)
const quotaLifetimeUsd = ref(0)
const quotaMonthlyUsd = ref(0)
const quotaWeeklyUsd = ref(0)
const quotaDailyUsd = ref(0)
const page = ref(1)
const pageSize = 12
const isEditingFirstUser = computed(() => editingUserId.value === 1)

interface UserMetricCard {
  key: string
  label: string
  value: string
  footnote: string
  icon: Component
}

const userMetrics = computed<UserMetricCard[]>(() => {
  const activeUsers = users.value.filter((user) => user.disabled_at === null).length
  const adminUsers = users.value.filter((user) => user.is_admin).length
  const boundKeys = users.value.reduce((total, user) => total + user.key_count, 0)
  const todayCost = users.value.reduce((total, user) => total + user.today_estimated_cost_usd, 0)
  return [
    {
      key: 'active',
      label: t('启用用户', 'Active users'),
      value: formatInteger(activeUsers),
      footnote: t(`共 ${formatInteger(users.value.length)} 个账号`, `${formatInteger(users.value.length)} accounts total`),
      icon: UserRound,
    },
    {
      key: 'admins',
      label: t('管理员', 'Admins'),
      value: formatInteger(adminUsers),
      footnote: t('拥有管理权限', 'Have admin access'),
      icon: ShieldCheck,
    },
    {
      key: 'keys',
      label: t('绑定 Key', 'Bound keys'),
      value: formatInteger(boundKeys),
      footnote: t('当前用户集合', 'Current users'),
      icon: KeyRound,
    },
    {
      key: 'cost',
      label: t('今日费用', 'Today cost'),
      value: formatUsd(todayCost),
      footnote: t('按现价估算', 'Estimated at current prices'),
      icon: CircleDollarSign,
    },
  ]
})

function userLabel(row: UserSummary): string {
  return row.nickname.trim() || row.username.trim() || t('未知用户', 'Unknown user')
}

function quotaBalanceValue(row: UserSummary, bucket: 'monthly' | 'weekly' | 'daily' | 'lifetime'): string {
  if (row.quota.unlimited) {
    return t('无限制', 'Unlimited')
  }

  const values = {
    monthly: row.quota.monthly_remaining_usd,
    weekly: row.quota.weekly_remaining_usd,
    daily: row.quota.daily_remaining_usd,
    lifetime: row.quota.lifetime_remaining_usd,
  }
  const value = values[bucket]
  return formatUsd(value)
}

function quotaBadgeVariant(row: UserSummary): 'destructive' | 'secondary' | 'outline' {
  if (row.quota.paused || !row.quota.can_create_keys) {
    return 'destructive'
  }
  return row.quota.unlimited ? 'secondary' : 'outline'
}

function quotaDetail(row: UserSummary): string | null {
  if (row.quota.paused) {
    return t('Key 已因余额暂停', 'Keys paused due to balance')
  }
  if (row.quota.sync_error) {
    return t('同步异常', 'Sync error')
  }
  return null
}

function todayRequestDetail(row: UserSummary): string {
  if (!row.today_records) {
    return t(`累计 ${formatInteger(row.records)}`, `${formatInteger(row.records)} total`)
  }
  const failed = row.today_failed_records
    ? t(`失败 ${formatInteger(row.today_failed_records)}`, `${formatInteger(row.today_failed_records)} failed`)
    : t('无失败', 'No failures')
  const rate = Math.round((row.today_success_records / row.today_records) * 100)
  return `${rate}% · ${failed}`
}

function todayCostDetail(row: UserSummary): string {
  if (!row.today_records) {
    return t('今日无请求', 'No requests today')
  }
  if (row.today_unpriced_records > 0) {
    return t(`未计价 ${formatInteger(row.today_unpriced_records)}`, `${formatInteger(row.today_unpriced_records)} unpriced`)
  }
  return t('已计价', 'Priced')
}

function lastModelLabel(row: UserSummary): string {
  return row.last_model ?? '-'
}

function lastProviderLabel(row: UserSummary): string {
  return row.last_provider ?? t('未知服务商', 'Unknown provider')
}

function normalizeQuotaInput(value: string | number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0
}

function setQuotaLifetimeUsd(value: string | number) {
  quotaLifetimeUsd.value = normalizeQuotaInput(value)
}

function setQuotaMonthlyUsd(value: string | number) {
  quotaMonthlyUsd.value = normalizeQuotaInput(value)
}

function todayTokenDetail(row: UserSummary): string {
  return t(
    `入 ${formatCompact(row.today_input_tokens)} · 出 ${formatCompact(row.today_output_tokens)} · 缓 ${formatCompact(row.today_cached_tokens)}`,
    `In ${formatCompact(row.today_input_tokens)} · Out ${formatCompact(row.today_output_tokens)} · Cache ${formatCompact(row.today_cached_tokens)}`,
  )
}

function setQuotaWeeklyUsd(value: string | number) {
  quotaWeeklyUsd.value = normalizeQuotaInput(value)
}

function setQuotaDailyUsd(value: string | number) {
  quotaDailyUsd.value = normalizeQuotaInput(value)
}

const pagedUsers = computed(() => {
  const start = (page.value - 1) * pageSize
  return users.value.slice(start, start + pageSize)
})

function handlePageChange(value: number) {
  page.value = value
}

function resetEditor() {
  editingUserId.value = null
  userAccount.value = ''
  userPassword.value = ''
  isUserAdmin.value = false
  userEnabled.value = true
  originalUserEnabled.value = true
  userNickname.value = ''
  quotaUnlimited.value = true
  quotaLifetimeUsd.value = 0
  quotaMonthlyUsd.value = 0
  quotaWeeklyUsd.value = 0
  quotaDailyUsd.value = 0
}

function openCreateUser() {
  resetEditor()
  userPassword.value = 'password'
  editorVisible.value = true
}

function editUser(row: UserSummary) {
  editingUserId.value = row.id
  userAccount.value = row.username
  userPassword.value = ''
  isUserAdmin.value = row.id === 1 ? true : row.is_admin
  userEnabled.value = !isUserDisabled(row)
  originalUserEnabled.value = userEnabled.value
  userNickname.value = row.nickname
  quotaUnlimited.value = row.quota.unlimited
  quotaLifetimeUsd.value = row.quota.lifetime_quota_usd ?? 0
  quotaMonthlyUsd.value = row.quota.monthly_quota_usd ?? 0
  quotaWeeklyUsd.value = row.quota.weekly_quota_usd ?? 0
  quotaDailyUsd.value = row.quota.daily_quota_usd ?? 0
  editorVisible.value = true
}

async function refresh() {
  isLoading.value = true
  try {
    users.value = await listUsers()
    const lastPage = Math.max(1, Math.ceil(users.value.length / pageSize))
    page.value = Math.min(page.value, lastPage)
  } catch (error) {
    message.error(errorText(error, '加载用户列表失败', 'Failed to load users'))
  } finally {
    isLoading.value = false
  }
}

function isUserDisabled(row: UserSummary): boolean {
  return row.disabled_at !== null
}

async function saveUser() {
  const nickname = userNickname.value.trim()
  if (!nickname) {
    message.error(t('用户昵称不能为空', 'User nickname is required'))
    return
  }
  const username = userAccount.value.trim()
  if (!username) {
    message.error(t('账号不能为空', 'Account is required'))
    return
  }
  const isEditing = editingUserId.value !== null
  const password = userPassword.value.trim()
  if (!isEditing && !password) {
    message.error(t('密码不能为空', 'Password is required'))
    return
  }
  isSavingUser.value = true
  try {
    const payload = {
      username,
      password: password || undefined,
      is_admin: isEditingFirstUser.value ? true : isUserAdmin.value,
      nickname,
    }
    const saved =
      editingUserId.value !== null
        ? await updateUser(editingUserId.value, payload)
        : await createUser(payload)
    await updateUserQuota(saved.id, {
      lifetime_quota_usd: quotaUnlimited.value ? null : quotaLifetimeUsd.value,
      monthly_quota_usd: quotaUnlimited.value ? null : quotaMonthlyUsd.value,
      weekly_quota_usd: quotaUnlimited.value ? null : quotaWeeklyUsd.value,
      daily_quota_usd: quotaUnlimited.value ? null : quotaDailyUsd.value,
    })
    if (isEditing && !isEditingFirstUser.value && userEnabled.value !== originalUserEnabled.value) {
      if (userEnabled.value) await enableUser(saved.id)
      else await disableUser(saved.id)
    }
    message.success(isEditing ? t('用户已保存', 'User saved') : t('用户已创建', 'User created'))
    editorVisible.value = false
    resetEditor()
    await refresh()
  } catch (error) {
    message.error(errorText(error, '保存用户失败', 'Failed to save user'))
  } finally {
    isSavingUser.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <section class="page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('用户管理', 'User Management') }}</h1>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <Button variant="outline" :disabled="isLoading" @click="refresh">
          <Spinner v-if="isLoading" data-icon="inline-start" />
          <RefreshCw v-else data-icon="inline-start" />
          {{ t('刷新', 'Refresh') }}
        </Button>
        <Button @click="openCreateUser">
          <Plus data-icon="inline-start" />
          {{ t('增加用户', 'Add user') }}
        </Button>
      </div>
    </div>

    <div class="metric-grid user-metrics">
      <Card v-for="metric in userMetrics" :key="metric.key" class="user-metric-card">
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

    <section class="panel table-panel user-table-panel">
      <div class="user-table">
        <Table class="min-w-[1056px] table-fixed">
          <TableHeader>
            <TableRow>
              <TableHead class="w-[145px]">{{ t('用户', 'User') }}</TableHead>
              <TableHead class="w-[110px]">{{ t('角色 / 状态', 'Role / status') }}</TableHead>
              <TableHead class="w-[150px]">{{ t('余额', 'Balance') }}</TableHead>
              <TableHead class="w-[72px]">{{ t('API KEY 数量', 'API keys') }}</TableHead>
              <TableHead class="w-[95px]">{{ t('今日请求', 'Today requests') }}</TableHead>
              <TableHead class="w-[145px]">{{ t('今日 Token', 'Today tokens') }}</TableHead>
              <TableHead class="w-[105px]">{{ t('今日费用', 'Today cost') }}</TableHead>
              <TableHead class="w-[178px]">{{ t('最近使用', 'Last used') }}</TableHead>
              <TableHead class="w-[56px]">
                <span class="sr-only">{{ t('操作', 'Actions') }}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="isLoading && users.length === 0">
              <TableRow v-for="rowIndex in 8" :key="`user-skeleton-${rowIndex}`">
                <TableCell v-for="columnIndex in 9" :key="columnIndex">
                  <Skeleton class="h-4 w-full" />
                </TableCell>
              </TableRow>
            </template>

            <TableEmpty v-else-if="users.length === 0" :colspan="9">
              {{ t('暂无用户', 'No users') }}
            </TableEmpty>

            <TableRow v-for="row in pagedUsers" v-else :key="row.id">
              <TableCell>
                <div class="metric-stack">
                  <span class="metric-primary">{{ userLabel(row) }}</span>
                  <span class="metric-muted" :title="row.username">{{ row.username }}</span>
                </div>
              </TableCell>
              <TableCell>
                <div class="metric-stack is-compact">
                  <Badge :variant="row.is_admin ? 'default' : 'secondary'">
                    {{ row.is_admin ? t('管理员', 'Admin') : t('普通用户', 'Standard user') }}
                  </Badge>
                  <Badge :variant="isUserDisabled(row) ? 'destructive' : 'outline'">
                    {{ isUserDisabled(row) ? t('已禁用', 'Disabled') : t('启用中', 'Enabled') }}
                  </Badge>
                </div>
              </TableCell>
              <TableCell>
                <div class="metric-stack quota-balance-stack">
                  <Badge :variant="quotaBadgeVariant(row)" class="quota-balance-row">
                    <span>{{ t('每日', 'Daily') }}</span>
                    <strong>{{ quotaBalanceValue(row, 'daily') }}</strong>
                  </Badge>
                  <Badge :variant="quotaBadgeVariant(row)" class="quota-balance-row">
                    <span>{{ t('每周', 'Weekly') }}</span>
                    <strong>{{ quotaBalanceValue(row, 'weekly') }}</strong>
                  </Badge>
                  <Badge :variant="quotaBadgeVariant(row)" class="quota-balance-row">
                    <span>{{ t('每月', 'Monthly') }}</span>
                    <strong>{{ quotaBalanceValue(row, 'monthly') }}</strong>
                  </Badge>
                  <Badge :variant="quotaBadgeVariant(row)" class="quota-balance-row">
                    <span>{{ t('不限时', 'Lifetime') }}</span>
                    <strong>{{ quotaBalanceValue(row, 'lifetime') }}</strong>
                  </Badge>
                  <span
                    v-if="quotaDetail(row)"
                    class="metric-muted quota-balance-detail"
                    :class="{ 'is-error': row.quota.sync_error || row.quota.paused }"
                  >
                    {{ quotaDetail(row) }}
                  </span>
                </div>
              </TableCell>
              <TableCell>{{ t(`${formatInteger(row.key_count)} 个`, `${formatInteger(row.key_count)} keys`) }}</TableCell>
              <TableCell>
                <div class="metric-stack">
                  <span class="metric-primary">{{ formatInteger(row.today_records) }}</span>
                  <span class="metric-muted">{{ todayRequestDetail(row) }}</span>
                </div>
              </TableCell>
              <TableCell>
                <div class="metric-stack">
                  <span class="metric-primary">{{ formatCompact(row.today_total_tokens) }}</span>
                  <span class="metric-muted" :title="todayTokenDetail(row)">{{ todayTokenDetail(row) }}</span>
                </div>
              </TableCell>
              <TableCell>
                <div class="metric-stack">
                  <span class="metric-primary">{{ formatUsd(row.today_estimated_cost_usd) }}</span>
                  <span class="metric-muted" :class="{ 'is-error': row.today_unpriced_records > 0 }">
                    {{ todayCostDetail(row) }}
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <div class="metric-stack">
                  <span class="model-value" :title="lastModelLabel(row)">{{ lastModelLabel(row) }}</span>
                  <span class="metric-muted">{{ lastProviderLabel(row) }}</span>
                  <span class="metric-muted">{{ formatDateTime(row.last_seen_at) }}</span>
                </div>
              </TableCell>
              <TableCell class="text-right">
                <Button
                  variant="ghost"
                  size="icon-sm"
                  class="row-actions-trigger"
                  :aria-label="t(`编辑 ${userLabel(row)}`, `Edit ${userLabel(row)}`)"
                  :title="t('编辑', 'Edit')"
                  @click="editUser(row)"
                >
                  <Pencil />
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>

      <div v-if="users.length > pageSize" class="user-pagination">
        <span>{{ t(`共 ${formatInteger(users.length)} 个用户`, `${formatInteger(users.length)} users total`) }}</span>
        <Pagination
          :page="page"
          :items-per-page="pageSize"
          :total="users.length"
          :sibling-count="1"
          @update:page="handlePageChange"
        >
          <PaginationContent v-slot="{ items }">
            <PaginationPrevious />
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
            <PaginationNext />
          </PaginationContent>
        </Pagination>
      </div>
    </section>

    <Dialog v-model:open="editorVisible">
      <DialogContent
        :show-close-button="false"
        class="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-2xl"
      >
        <form class="flex flex-col gap-5" @submit.prevent="saveUser">
          <DialogHeader>
            <DialogTitle>
              {{ editingUserId ? t('编辑用户', 'Edit user') : t('增加用户', 'Add user') }}
            </DialogTitle>
            <DialogDescription>
              {{ t('配置登录身份、管理权限和分层余额。', 'Configure sign-in identity, administrator access, and balance buckets.') }}
            </DialogDescription>
          </DialogHeader>

          <Alert v-if="editingUserId === null" class="user-editor-warning">
            <AlertTriangle />
            <AlertDescription>
              {{ t('账号一旦创建，不允许删除，只允许禁用，请谨慎操作。', 'Accounts cannot be deleted after creation. They can only be disabled, so proceed carefully.') }}
            </AlertDescription>
          </Alert>

          <FieldGroup class="user-editor-form">
            <FieldGroup class="identity-fields">
              <Field>
                <FieldLabel for="user-nickname">{{ t('用户昵称', 'User nickname') }}</FieldLabel>
                <Input
                  id="user-nickname"
                  v-model="userNickname"
                  required
                  :placeholder="t('例如：研发用户', 'Example: Engineering user')"
                />
              </Field>
              <Field :data-disabled="editingUserId !== null || undefined">
                <FieldLabel for="user-account">{{ t('账号', 'Account') }}</FieldLabel>
                <Input
                  id="user-account"
                  v-model="userAccount"
                  required
                  autocomplete="username"
                  :disabled="editingUserId !== null"
                  :placeholder="t('例如：user001', 'Example: user001')"
                />
              </Field>
              <Field class="password-field">
                <FieldLabel for="user-password">{{ t('密码', 'Password') }}</FieldLabel>
                <Input
                  id="user-password"
                  v-model="userPassword"
                  type="password"
                  :required="editingUserId === null"
                  autocomplete="new-password"
                  :placeholder="editingUserId ? t('留空不修改密码', 'Leave blank to keep the current password') : t('请输入登录密码', 'Enter a sign-in password')"
                />
                <FieldDescription v-if="editingUserId !== null">
                  {{ t('留空时保留当前密码。', 'Leave blank to keep the current password.') }}
                </FieldDescription>
              </Field>
            </FieldGroup>

            <Field orientation="horizontal" class="switch-setting" :data-disabled="isEditingFirstUser || undefined">
              <FieldContent>
                <FieldLabel for="user-admin">{{ t('管理员权限', 'Administrator access') }}</FieldLabel>
                <FieldDescription>
                  {{ t('管理员可以管理用户、价格、上游和系统设置。', 'Administrators can manage users, prices, upstreams, and system settings.') }}
                </FieldDescription>
              </FieldContent>
              <Switch id="user-admin" v-model="isUserAdmin" :disabled="isEditingFirstUser" />
            </Field>

            <Field v-if="editingUserId !== null && !isEditingFirstUser" orientation="horizontal" class="switch-setting">
              <FieldContent>
                <FieldLabel for="user-enabled">{{ t('启用账号', 'Account enabled') }}</FieldLabel>
                <FieldDescription>
                  {{ t('关闭并保存后将禁用账号，并从 CPA 移除其 API KEY。', 'Turning this off disables the account and removes its API keys from CPA when saved.') }}
                </FieldDescription>
              </FieldContent>
              <Switch id="user-enabled" v-model="userEnabled" />
            </Field>

            <FieldSet class="quota-fieldset">
              <FieldLegend>{{ t('余额设置', 'Balance settings') }}</FieldLegend>
              <FieldDescription>
                {{ t('扣费顺序为每日、每周、每月、不限时。', 'Charges are deducted from daily, weekly, monthly, then lifetime balance.') }}
              </FieldDescription>
              <FieldGroup>
                <Field orientation="horizontal" class="switch-setting">
                  <FieldContent>
                    <FieldLabel for="quota-unlimited">{{ t('不限制余额', 'Unlimited balance') }}</FieldLabel>
                    <FieldDescription>
                      {{ t('开启后不扣余额，也不会因余额暂停 API Key。', 'Balances are not deducted and API keys are not paused due to balance.') }}
                    </FieldDescription>
                  </FieldContent>
                  <Switch id="quota-unlimited" v-model="quotaUnlimited" />
                </Field>
                <FieldGroup class="quota-editor-grid">
                  <Field>
                    <FieldLabel for="quota-daily">{{ t('每日余额 USD', 'Daily balance USD') }}</FieldLabel>
                    <Input
                      id="quota-daily"
                      type="number"
                      min="0"
                      step="0.00000001"
                      :model-value="quotaDailyUsd"
                      :disabled="quotaUnlimited"
                      @update:model-value="setQuotaDailyUsd"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="quota-weekly">{{ t('每周余额 USD', 'Weekly balance USD') }}</FieldLabel>
                    <Input
                      id="quota-weekly"
                      type="number"
                      min="0"
                      step="0.00000001"
                      :model-value="quotaWeeklyUsd"
                      :disabled="quotaUnlimited"
                      @update:model-value="setQuotaWeeklyUsd"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="quota-monthly">{{ t('每月余额 USD', 'Monthly balance USD') }}</FieldLabel>
                    <Input
                      id="quota-monthly"
                      type="number"
                      min="0"
                      step="0.00000001"
                      :model-value="quotaMonthlyUsd"
                      :disabled="quotaUnlimited"
                      @update:model-value="setQuotaMonthlyUsd"
                    />
                  </Field>
                  <Field>
                    <FieldLabel for="quota-lifetime">{{ t('不限时余额 USD', 'Lifetime balance USD') }}</FieldLabel>
                    <Input
                      id="quota-lifetime"
                      type="number"
                      min="0"
                      step="0.00000001"
                      :model-value="quotaLifetimeUsd"
                      :disabled="quotaUnlimited"
                      @update:model-value="setQuotaLifetimeUsd"
                    />
                  </Field>
                </FieldGroup>
              </FieldGroup>
            </FieldSet>
          </FieldGroup>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSavingUser" @click="editorVisible = false">
              {{ t('取消', 'Cancel') }}
            </Button>
            <Button type="submit" :disabled="isSavingUser">
              <Spinner v-if="isSavingUser" data-icon="inline-start" />
              {{ editingUserId ? t('保存', 'Save') : t('创建', 'Create') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </section>
</template>

<style scoped>
.user-metrics {
  grid-template-columns: repeat(4, minmax(150px, 1fr));
}

.user-metric-card {
  min-width: 0;
}

.user-metric-card :deep([data-slot="card-header"]) {
  padding-bottom: 10px;
}

.user-metric-card :deep([data-slot="card-content"]) {
  padding-top: 0;
}

.user-table-panel {
  min-width: 0;
  overflow: hidden;
}

.user-table {
  min-width: 0;
  overflow: auto;
}

.user-pagination {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  color: var(--muted-foreground);
  font-size: 12px;
}

.identity-fields,
.quota-editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.password-field {
  grid-column: 1 / -1;
}

.switch-setting {
  gap: 20px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: color-mix(in oklch, var(--muted) 45%, transparent);
}

.quota-fieldset {
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.row-actions-trigger {
  margin-left: auto;
  color: var(--muted-foreground);
}

.metric-stack {
  display: grid;
  gap: 3px;
  min-width: 0;
  line-height: 1.28;
}

.metric-stack.is-compact {
  align-items: start;
}

.quota-balance-stack {
  gap: 4px;
}

.quota-balance-row {
  width: 100%;
  min-width: 0;
  justify-content: space-between;
}

.metric-primary {
  font-weight: 600;
}

.metric-muted {
  min-width: 0;
  overflow: hidden;
  color: var(--muted-foreground);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.metric-muted.is-error {
  color: var(--destructive);
}

.model-value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 900px) {
  .user-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .quota-editor-grid,
  .identity-fields {
    grid-template-columns: 1fr;
  }

  .password-field {
    grid-column: auto;
  }

  .user-pagination {
    justify-content: flex-start;
    overflow-x: auto;
  }
}
</style>
