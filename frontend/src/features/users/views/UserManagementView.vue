<script setup lang="ts">
import type { Component } from 'vue'
import { computed, h, onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import {
  AppAlert,
  AppButton,
  AppDataTable,
  AppForm,
  AppFormItem,
  AppInput,
  AppNumberInput,
  AppModal,
  AppStack,
  AppSwitch,
  AppBadge,
  type DataTableColumns,
} from '@/shared/ui/app-kit'
import { useConfirmDialog } from '@/shared/ui/confirm-dialog'
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
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from '@/components/ui/field'
import {
  CircleDollarSign,
  KeyRound,
  MoreHorizontal,
  Pencil,
  ShieldCheck,
  UserCheck,
  UserRound,
  UserX,
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
const dialog = useConfirmDialog()
const { errorText, t } = useI18n()
const isLoading = ref(false)
const isSavingUser = ref(false)
const users = ref<UserSummary[]>([])
const editorVisible = ref(false)
const editingUserId = ref<number | null>(null)
const userAccount = ref('')
const userPassword = ref('')
const isUserAdmin = ref(false)
const userNickname = ref('')
const quotaUnlimited = ref(true)
const quotaLifetimeUsd = ref(0)
const quotaMonthlyUsd = ref(0)
const quotaWeeklyUsd = ref(0)
const quotaDailyUsd = ref(0)
const isEditingFirstUser = computed(() => editingUserId.value === 1)

interface UserMetricCard {
  key: string
  label: string
  value: string
  footnote: string
  tone: 'primary' | 'blue' | 'purple' | 'green'
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
      tone: 'primary',
      icon: UserRound,
    },
    {
      key: 'admins',
      label: t('管理员', 'Admins'),
      value: formatInteger(adminUsers),
      footnote: t('拥有管理权限', 'Have admin access'),
      tone: 'purple',
      icon: ShieldCheck,
    },
    {
      key: 'keys',
      label: t('绑定 Key', 'Bound keys'),
      value: formatInteger(boundKeys),
      footnote: t('当前用户集合', 'Current users'),
      tone: 'blue',
      icon: KeyRound,
    },
    {
      key: 'cost',
      label: t('今日费用', 'Today cost'),
      value: formatUsd(todayCost),
      footnote: t('按现价估算', 'Estimated at current prices'),
      tone: 'green',
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

function quotaBalanceClass(row: UserSummary): string {
  if (row.quota.paused || !row.quota.can_create_keys) {
    return 'is-error'
  }
  return row.quota.unlimited ? 'is-unlimited' : 'is-normal'
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

function setQuotaLifetimeUsd(value: number | null) {
  quotaLifetimeUsd.value = value ?? 0
}

function setQuotaMonthlyUsd(value: number | null) {
  quotaMonthlyUsd.value = value ?? 0
}

function todayTokenDetail(row: UserSummary): string {
  return t(
    `入 ${formatCompact(row.today_input_tokens)} · 出 ${formatCompact(row.today_output_tokens)} · 缓 ${formatCompact(row.today_cached_tokens)}`,
    `In ${formatCompact(row.today_input_tokens)} · Out ${formatCompact(row.today_output_tokens)} · Cache ${formatCompact(row.today_cached_tokens)}`,
  )
}

function setQuotaWeeklyUsd(value: number | null) {
  quotaWeeklyUsd.value = value ?? 0
}

function setQuotaDailyUsd(value: number | null) {
  quotaDailyUsd.value = value ?? 0
}

function resetEditor() {
  editingUserId.value = null
  userAccount.value = ''
  userPassword.value = ''
  isUserAdmin.value = false
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
  } catch (error) {
    message.error(errorText(error, '加载用户列表失败', 'Failed to load users'))
  } finally {
    isLoading.value = false
  }
}

function isUserDisabled(row: UserSummary): boolean {
  return row.disabled_at !== null
}

async function disableUserRow(row: UserSummary) {
  try {
    await disableUser(row.id)
    message.success(t('用户已禁用', 'User disabled'))
    await refresh()
  } catch (error) {
    message.error(errorText(error, '禁用用户失败', 'Failed to disable user'))
  }
}

async function enableUserRow(row: UserSummary) {
  try {
    await enableUser(row.id)
    message.success(t('用户已启用', 'User enabled'))
    await refresh()
  } catch (error) {
    message.error(errorText(error, '启用用户失败', 'Failed to enable user'))
  }
}

function confirmEnableUser(row: UserSummary) {
  dialog.warning({
    title: t('启用用户', 'Enable user'),
    content: t(
      `启用用户 ${userLabel(row)} 并恢复其 API KEY？`,
      `Enable user ${userLabel(row)} and restore their API keys?`,
    ),
    positiveText: t('启用', 'Enable'),
    negativeText: t('取消', 'Cancel'),
    onPositiveClick: () => enableUserRow(row),
  })
}

function confirmDisableUser(row: UserSummary) {
  dialog.warning({
    title: t('禁用用户', 'Disable user'),
    content: t(
      `禁用用户 ${userLabel(row)} 并从 CPA 移除其 API KEY？`,
      `Disable user ${userLabel(row)} and remove their API keys from CPA?`,
    ),
    positiveText: t('禁用', 'Disable'),
    negativeText: t('取消', 'Cancel'),
    onPositiveClick: () => disableUserRow(row),
  })
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

const columns = computed<DataTableColumns<UserSummary>>(() => [
  {
    title: t('用户', 'User'),
    key: 'nickname',
    width: 145,
    render: (row) =>
      h('div', { class: 'metric-stack' }, [
        h('span', { class: 'metric-primary' }, userLabel(row)),
        h('span', { class: 'metric-muted' }, row.username),
      ]),
  },
  {
    title: t('角色 / 状态', 'Role / status'),
    key: 'is_admin',
    width: 110,
    render: (row) =>
      h('div', { class: 'metric-stack is-compact' }, [
        h('span', { class: 'metric-primary' }, row.is_admin ? t('管理员', 'Admin') : t('普通用户', 'Standard user')),
        h(
          AppBadge,
          {
            size: 'small',
            type: isUserDisabled(row) ? 'warning' : 'success',
            bordered: false,
            class: 'user-status-badge',
          },
          { default: () => (isUserDisabled(row) ? t('已禁用', 'Disabled') : t('启用中', 'Enabled')) },
        ),
      ]),
  },
  {
    title: t('余额', 'Balance'),
    key: 'quota',
    width: 150,
    render: (row) => {
      const detail = quotaDetail(row)
      return h('div', { class: ['metric-stack', 'quota-balance-stack'] }, [
        h('div', { class: ['quota-balance-row', 'is-daily', quotaBalanceClass(row)] }, [
          h('span', { class: 'quota-balance-label' }, t('每日：', 'Daily: ')),
          h('strong', { class: 'quota-balance-value' }, quotaBalanceValue(row, 'daily')),
        ]),
        h('div', { class: ['quota-balance-row', 'is-weekly', quotaBalanceClass(row)] }, [
          h('span', { class: 'quota-balance-label' }, t('每周：', 'Weekly: ')),
          h('strong', { class: 'quota-balance-value' }, quotaBalanceValue(row, 'weekly')),
        ]),
        h('div', { class: ['quota-balance-row', 'is-monthly', quotaBalanceClass(row)] }, [
          h('span', { class: 'quota-balance-label' }, t('每月：', 'Monthly: ')),
          h('strong', { class: 'quota-balance-value' }, quotaBalanceValue(row, 'monthly')),
        ]),
        h('div', { class: ['quota-balance-row', 'is-lifetime', quotaBalanceClass(row)] }, [
          h('span', { class: 'quota-balance-label' }, t('不限时：', 'Lifetime: ')),
          h('strong', { class: 'quota-balance-value' }, quotaBalanceValue(row, 'lifetime')),
        ]),
        ...(detail
          ? [
              h(
                'span',
                { class: ['metric-muted', 'quota-balance-detail', { 'is-error': row.quota.sync_error || row.quota.paused }] },
                detail,
              ),
            ]
          : []),
      ])
    },
  },
  {
    title: t('API KEY 数量', 'API keys'),
    key: 'key_count',
    width: 72,
    render: (row) => t(`${formatInteger(row.key_count)} 个`, `${formatInteger(row.key_count)} keys`),
  },
  {
    title: t('今日请求', 'Today requests'),
    key: 'today_records',
    width: 95,
    render: (row) =>
      h('div', { class: 'metric-stack' }, [
        h('span', { class: 'metric-primary' }, formatInteger(row.today_records)),
        h('span', { class: 'metric-muted' }, todayRequestDetail(row)),
      ]),
  },
  {
    title: t('今日 Token', 'Today tokens'),
    key: 'today_total_tokens',
    width: 145,
    render: (row) =>
      h('div', { class: 'metric-stack' }, [
        h('span', { class: 'metric-primary' }, formatCompact(row.today_total_tokens)),
        h('span', { class: 'metric-muted' }, todayTokenDetail(row)),
      ]),
  },
  {
    title: t('今日费用', 'Today cost'),
    key: 'today_estimated_cost_usd',
    width: 105,
    render: (row) =>
      h('div', { class: 'metric-stack' }, [
        h('span', { class: 'metric-primary' }, formatUsd(row.today_estimated_cost_usd)),
        h(
          'span',
          { class: ['metric-muted', { 'is-error': row.today_unpriced_records > 0 }] },
          todayCostDetail(row),
        ),
      ]),
  },
  {
    title: t('最近使用', 'Last used'),
    key: 'last_seen_at',
    width: 178,
    render: (row) =>
      h('div', { class: 'metric-stack' }, [
        h('span', { class: 'model-value' }, lastModelLabel(row)),
        h('span', { class: 'metric-muted' }, lastProviderLabel(row)),
        h('span', { class: 'metric-muted' }, formatDateTime(row.last_seen_at)),
      ]),
  },
  {
    title: '',
    key: 'actions',
    width: 56,
    align: 'right',
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
                      class: 'row-actions-trigger',
                      'aria-label': t(`打开 ${userLabel(row)} 的操作菜单`, `Open actions for ${userLabel(row)}`),
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
                          { onSelect: () => editUser(row) },
                          { default: () => [h(Pencil), h('span', t('编辑', 'Edit'))] },
                        ),
                    },
                  ),
                  row.id === 1 ? null : h(DropdownMenuSeparator),
                  row.id === 1
                    ? null
                    : isUserDisabled(row)
                      ? h(
                          DropdownMenuItem,
                          { onSelect: () => confirmEnableUser(row) },
                          { default: () => [h(UserCheck), h('span', t('启用', 'Enable'))] },
                        )
                      : h(
                          DropdownMenuItem,
                          { variant: 'destructive', onSelect: () => confirmDisableUser(row) },
                          { default: () => [h(UserX), h('span', t('禁用', 'Disable'))] },
                        ),
                ],
              },
            ),
          ],
        },
      ),
  },
])

onMounted(refresh)
</script>

<template>
  <section class="page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('用户管理', 'User Management') }}</h1>
      <AppStack>
        <AppButton secondary :loading="isLoading" @click="refresh">{{ t('刷新', 'Refresh') }}</AppButton>
        <AppButton type="primary" @click="openCreateUser">{{ t('增加用户', 'Add user') }}</AppButton>
      </AppStack>
    </div>

    <div class="metric-grid user-metrics">
      <div v-for="metric in userMetrics" :key="metric.key" class="metric-card" :class="`is-${metric.tone}`">
        <div class="metric-icon" aria-hidden="true">
          <component :is="metric.icon" :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ metric.label }}</div>
        <div class="metric-value">{{ metric.value }}</div>
        <div class="metric-footnote">{{ metric.footnote }}</div>
      </div>
    </div>

    <section class="panel table-panel">
      <AppDataTable
        size="small"
        :loading="isLoading"
        :columns="columns"
        :data="users"
        :pagination="{ pageSize: 12 }"
        table-layout="fixed"
        :scroll-x="1056"
      />
    </section>

    <AppModal
      v-model:show="editorVisible"
      preset="card"
      :mask-closable="false"
      :closable="false"
      :title="editingUserId ? t('编辑用户', 'Edit user') : t('增加用户', 'Add user')"
      :style="{ width: 'min(640px, calc(100vw - 32px))' }"
    >
      <AppAlert v-if="editingUserId === null" type="warning" :bordered="false" class="user-editor-warning">
        {{ t('账号一旦创建，不允许删除，只允许禁用，请谨慎操作。', 'Accounts cannot be deleted after creation. They can only be disabled, so proceed carefully.') }}
      </AppAlert>

      <AppForm label-placement="top" class="user-editor-form">
        <FieldGroup>
          <FieldGroup class="identity-fields">
            <AppFormItem :label="t('用户昵称', 'User nickname')" required>
              <AppInput
                v-model:value="userNickname"
                :placeholder="t('例如：研发用户', 'Example: Engineering user')"
                @keyup.enter="saveUser"
              />
            </AppFormItem>
            <AppFormItem :label="t('账号', 'Account')" required>
              <AppInput
                v-model:value="userAccount"
                autocomplete="username"
                :disabled="editingUserId !== null"
                :placeholder="t('例如：user001', 'Example: user001')"
                @keyup.enter="saveUser"
              />
            </AppFormItem>
            <AppFormItem class="password-field" :label="t('密码', 'Password')" :required="editingUserId === null">
              <AppInput
                v-model:value="userPassword"
                type="password"
                show-password-on="mousedown"
                autocomplete="new-password"
                :placeholder="editingUserId ? t('留空不修改密码', 'Leave blank to keep the current password') : t('请输入登录密码', 'Enter a sign-in password')"
                @keyup.enter="saveUser"
              />
            </AppFormItem>
          </FieldGroup>

          <Field orientation="horizontal" class="switch-setting" :data-disabled="isEditingFirstUser || undefined">
            <FieldContent>
              <FieldTitle>{{ t('管理员权限', 'Administrator access') }}</FieldTitle>
              <FieldDescription>{{ t('管理员可以管理用户、价格、上游和系统设置。', 'Administrators can manage users, prices, upstreams, and system settings.') }}</FieldDescription>
            </FieldContent>
            <AppSwitch v-model:value="isUserAdmin" :disabled="isEditingFirstUser" />
          </Field>

          <FieldSet class="quota-fieldset">
            <FieldLegend>{{ t('余额设置', 'Balance settings') }}</FieldLegend>
            <FieldDescription>{{ t('扣费顺序为每日、每周、每月、不限时。', 'Charges are deducted from daily, weekly, monthly, then lifetime balance.') }}</FieldDescription>
            <FieldGroup>
              <Field orientation="horizontal" class="switch-setting">
                <FieldContent>
                  <FieldTitle>{{ t('不限制余额', 'Unlimited balance') }}</FieldTitle>
                  <FieldDescription>{{ t('开启后不扣余额，也不会因余额暂停 API Key。', 'Balances are not deducted and API keys are not paused due to balance.') }}</FieldDescription>
                </FieldContent>
                <AppSwitch v-model:value="quotaUnlimited" />
              </Field>
              <FieldGroup class="quota-editor-grid">
                <AppFormItem :label="t('每日余额 USD', 'Daily balance USD')">
                  <AppNumberInput :value="quotaDailyUsd" :disabled="quotaUnlimited" :min="0" :precision="8" placeholder="0" @update:value="setQuotaDailyUsd" />
                </AppFormItem>
                <AppFormItem :label="t('每周余额 USD', 'Weekly balance USD')">
                  <AppNumberInput :value="quotaWeeklyUsd" :disabled="quotaUnlimited" :min="0" :precision="8" placeholder="0" @update:value="setQuotaWeeklyUsd" />
                </AppFormItem>
                <AppFormItem :label="t('每月余额 USD', 'Monthly balance USD')">
                  <AppNumberInput :value="quotaMonthlyUsd" :disabled="quotaUnlimited" :min="0" :precision="8" placeholder="0" @update:value="setQuotaMonthlyUsd" />
                </AppFormItem>
                <AppFormItem :label="t('不限时余额 USD', 'Lifetime balance USD')">
                  <AppNumberInput :value="quotaLifetimeUsd" :disabled="quotaUnlimited" :min="0" :precision="8" placeholder="0" @update:value="setQuotaLifetimeUsd" />
                </AppFormItem>
              </FieldGroup>
            </FieldGroup>
          </FieldSet>

          <div class="user-editor-actions">
            <AppButton secondary @click="editorVisible = false">{{ t('取消', 'Cancel') }}</AppButton>
            <AppButton type="primary" :loading="isSavingUser" @click="saveUser">
              {{ editingUserId ? t('保存', 'Save') : t('创建', 'Create') }}
            </AppButton>
          </div>
        </FieldGroup>
      </AppForm>
    </AppModal>
  </section>
</template>

<style scoped>
.user-metrics {
  grid-template-columns: repeat(4, minmax(150px, 1fr));
}

.user-editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.user-editor-warning {
  margin-bottom: 16px;
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
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface-muted);
}

.quota-fieldset {
  padding: 16px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
}

:global(.row-actions-trigger) {
  margin-left: auto;
  color: var(--cpa-text-muted);
}

:global(.metric-stack) {
  display: grid;
  gap: 2px;
  min-width: 0;
  line-height: 1.28;
}

:global(.quota-balance-stack) {
  gap: 3px;
}

:global(.quota-balance-row) {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  width: fit-content;
  max-width: 100%;
  padding: 2px 7px;
  overflow: hidden;
  border-radius: var(--cpa-radius-sm);
  font-size: 12px;
  line-height: 1.35;
  white-space: nowrap;
}

:global(.quota-balance-row.is-monthly.is-normal) {
  background: var(--cpa-success-weak);
  color: var(--cpa-success);
}

:global(.metric-stack.is-compact) {
  align-items: start;
}

:global(.user-status-badge) {
  width: fit-content;
}

:global(.quota-balance-row.is-weekly.is-normal) {
  background: var(--cpa-accent-blue-weak);
  color: var(--cpa-accent-blue);
}

:global(.quota-balance-row.is-daily.is-normal) {
  background: var(--cpa-accent-orange-weak);
  color: var(--cpa-accent-orange);
}

:global(.quota-balance-row.is-lifetime.is-normal) {
  background: var(--cpa-accent-purple-weak);
  color: var(--cpa-accent-purple);
}

:global(.quota-balance-row.is-unlimited) {
  background: var(--cpa-primary-wash);
  color: var(--cpa-primary);
}

:global(.quota-balance-row.is-error) {
  background: var(--cpa-danger-weak);
  color: var(--cpa-danger);
}

:global(.quota-balance-label),
:global(.quota-balance-value) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

:global(.quota-balance-label) {
  flex: 0 0 auto;
  font-weight: 600;
}

:global(.quota-balance-value) {
  font-weight: 760;
}

:global(.metric-primary) {
  font-weight: 600;
}

:global(.metric-muted) {
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.metric-muted.is-error) {
  color: var(--cpa-danger);
}

:global(.model-value) {
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
  .user-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .quota-editor-grid {
    grid-template-columns: 1fr;
  }

  .identity-fields {
    grid-template-columns: 1fr;
  }

  .password-field {
    grid-column: auto;
  }
}
</style>
