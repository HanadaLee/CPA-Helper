<script setup lang="ts">
import {
  Activity,
  Database,
  EyeIcon,
  EyeOffIcon,
  PlusIcon,
  Power,
  RefreshCwIcon,
  SaveIcon,
  Server,
  ShieldCheckIcon,
  Trash2Icon,
  TriangleAlertIcon,
} from '@lucide/vue'
import { computed, onMounted, reactive, ref } from 'vue'
import { toast } from 'vue-sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from '@/components/ui/input-group'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'

import CodexKeeperSettingsPanel from '@/features/codex-keeper/components/CodexKeeperSettingsPanel.vue'
import {
  getCollectorStatus,
  getSettings,
  updateSettings,
} from '@/features/settings/api/settingsApi'
import { useI18n } from '@/shared/i18n'
import type { CollectorStatus, ModelRequestExtraEndpoint, SettingsUpdatePayload } from '@/shared/types/api'
import { formatDateTime, formatInteger } from '@/shared/utils/format'

const message = toast
const { errorText, serverText, t } = useI18n()
const isLoading = ref(false)
const isSaving = ref(false)
const collectorStatus = ref<CollectorStatus | null>(null)
const keeperSettingsPanel = ref<InstanceType<typeof CodexKeeperSettingsPanel> | null>(null)
const showManagementKey = ref(false)

const settingsForm = reactive({
  cliaproxy_url: 'http://127.0.0.1:8317',
  model_request_url: 'http://127.0.0.1:8317',
  model_request_extra_endpoints: [] as ModelRequestExtraEndpoint[],
  cpamc_url: '/management.html',
  brand_name_zh: 'CPA-Helper',
  brand_name_en: 'CPA-Helper',
  brand_subtitle_zh: '边缘网关管理平台',
  brand_subtitle_en: 'Edge Gateway Management Platform',
  management_key: '',
  collector_enabled: false,
  batch_size: 100,
  poll_interval_seconds: 2,
  retry_interval_seconds: 10,
  allow_user_account_status: false,
  allow_user_usage_history: false,
  usage_detail_retention_days: 90,
  cas_enabled: false,
  cas_default_login: false,
  cas_base_url: '',
  cas_validation_url: '',
  cas_validation_host: '',
  cas_public_url: '',
  cas_auto_create_users: true,
})

const remoteStatusType = computed(() => {
  if (collectorStatus.value?.remote_enabled === true) {
    return 'success'
  }
  if (collectorStatus.value?.remote_enabled === false) {
    return 'error'
  }
  return 'warning'
})

const remoteStatusText = computed(() => {
  if (collectorStatus.value?.remote_enabled === true) {
    return t('开启', 'On')
  }
  if (collectorStatus.value?.remote_enabled === false) {
    return t('关闭', 'Off')
  }
  return t('未知', 'Unknown')
})

const collectorEnabledText = computed(() => (collectorStatus.value?.enabled ? t('开启', 'On') : t('关闭', 'Off')))
const collectorRunningText = computed(() => (collectorStatus.value?.running ? t('运行中', 'Running') : t('空闲', 'Idle')))

function addModelRequestExtraEndpoint() {
  settingsForm.model_request_extra_endpoints.push({ url: '', description: '' })
}

function removeModelRequestExtraEndpoint(index: number) {
  settingsForm.model_request_extra_endpoints.splice(index, 1)
}

function updateNumericSetting(
  key: 'batch_size' | 'poll_interval_seconds' | 'retry_interval_seconds' | 'usage_detail_retention_days',
  value: string | number,
) {
  const nextValue = Number(value)
  if (Number.isFinite(nextValue)) {
    settingsForm[key] = nextValue
  }
}

async function refresh() {
  isLoading.value = true
  try {
    const [settings, status] = await Promise.all([
      getSettings(),
      getCollectorStatus(),
    ])
    settingsForm.cliaproxy_url = settings.cliaproxy_url
    settingsForm.model_request_url = settings.model_request_url
    settingsForm.model_request_extra_endpoints = (settings.model_request_extra_endpoints ?? []).map((endpoint) => ({ ...endpoint }))
    settingsForm.cpamc_url = settings.cpamc_url
    settingsForm.brand_name_zh = settings.brand_name_zh
    settingsForm.brand_name_en = settings.brand_name_en
    settingsForm.brand_subtitle_zh = settings.brand_subtitle_zh
    settingsForm.brand_subtitle_en = settings.brand_subtitle_en
    settingsForm.management_key = settings.management_key
    settingsForm.collector_enabled = settings.collector_enabled
    settingsForm.batch_size = settings.batch_size
    settingsForm.poll_interval_seconds = settings.poll_interval_seconds
    settingsForm.retry_interval_seconds = settings.retry_interval_seconds
    settingsForm.allow_user_account_status = settings.allow_user_account_status
    settingsForm.allow_user_usage_history = settings.allow_user_usage_history
    settingsForm.usage_detail_retention_days = settings.usage_detail_retention_days
    settingsForm.cas_enabled = settings.cas_enabled
    settingsForm.cas_default_login = settings.cas_default_login
    settingsForm.cas_base_url = settings.cas_base_url
    settingsForm.cas_validation_url = settings.cas_validation_url
    settingsForm.cas_validation_host = settings.cas_validation_host
    settingsForm.cas_public_url = settings.cas_public_url
    settingsForm.cas_auto_create_users = settings.cas_auto_create_users
    collectorStatus.value = status
  } catch (error) {
    message.error(errorText(error, '加载设置失败', 'Failed to load settings'))
  } finally {
    isLoading.value = false
  }
}

async function saveSettings() {
  isSaving.value = true
  try {
    const keeperPanel = keeperSettingsPanel.value
    if (!keeperPanel) {
      throw new Error(t('巡检配置尚未加载', 'Inspection settings have not loaded yet'))
    }
    keeperPanel.validateSettings()

    const payload: SettingsUpdatePayload = {
      cliaproxy_url: settingsForm.cliaproxy_url,
      model_request_url: settingsForm.model_request_url,
      model_request_extra_endpoints: settingsForm.model_request_extra_endpoints.map((endpoint) => ({ ...endpoint })),
      cpamc_url: settingsForm.cpamc_url,
      brand_name_zh: settingsForm.brand_name_zh,
      brand_name_en: settingsForm.brand_name_en,
      brand_subtitle_zh: settingsForm.brand_subtitle_zh,
      brand_subtitle_en: settingsForm.brand_subtitle_en,
      management_key: settingsForm.management_key,
      collector_enabled: settingsForm.collector_enabled,
      batch_size: settingsForm.batch_size,
      poll_interval_seconds: settingsForm.poll_interval_seconds,
      retry_interval_seconds: settingsForm.retry_interval_seconds,
      allow_user_account_status: settingsForm.allow_user_account_status,
      allow_user_usage_history: settingsForm.allow_user_usage_history,
      usage_detail_retention_days: settingsForm.usage_detail_retention_days,
      cas_enabled: settingsForm.cas_enabled,
      cas_default_login: settingsForm.cas_default_login,
      cas_base_url: settingsForm.cas_base_url,
      cas_validation_url: settingsForm.cas_validation_url,
      cas_validation_host: settingsForm.cas_validation_host,
      cas_public_url: settingsForm.cas_public_url,
      cas_auto_create_users: settingsForm.cas_auto_create_users,
    }
    const [saved] = await Promise.all([
      updateSettings(payload),
      keeperPanel.saveSettings(),
    ])
    settingsForm.management_key = saved.management_key
    window.dispatchEvent(new CustomEvent('cpa:branding-updated', {
      detail: {
        brand_name_zh: saved.brand_name_zh,
        brand_name_en: saved.brand_name_en,
        brand_subtitle_zh: saved.brand_subtitle_zh,
        brand_subtitle_en: saved.brand_subtitle_en,
      },
    }))
    message.success(t('设置已保存', 'Settings saved'))
    await refresh()
  } catch (error) {
    message.error(errorText(error, '保存设置失败', 'Failed to save settings'))
  } finally {
    isSaving.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <section class="page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('系统设置', 'System Settings') }}</h1>
      <div class="flex items-center gap-2">
        <Button variant="outline" :disabled="isLoading" @click="refresh">
          <Spinner v-if="isLoading" data-icon="inline-start" />
          <RefreshCwIcon v-else data-icon="inline-start" />
          {{ t('刷新', 'Refresh') }}
        </Button>
        <Button :disabled="isSaving" @click="saveSettings">
          <Spinner v-if="isSaving" data-icon="inline-start" />
          <SaveIcon v-else data-icon="inline-start" />
          {{ t('保存设置', 'Save settings') }}
        </Button>
      </div>
    </div>

    <div class="metric-grid settings-metrics">
      <Card size="sm" class="settings-metric-card">
        <CardHeader>
          <CardDescription>{{ t('本地采集', 'Local collection') }}</CardDescription>
          <CardAction><Power /></CardAction>
        </CardHeader>
        <CardContent>
          <CardTitle class="text-2xl">{{ collectorEnabledText }}</CardTitle>
          <p class="settings-metric-footnote">{{ t('系统开关', 'System switch') }}</p>
        </CardContent>
      </Card>
      <Card size="sm" class="settings-metric-card">
        <CardHeader>
          <CardDescription>{{ t('运行状态', 'Run status') }}</CardDescription>
          <CardAction><Activity /></CardAction>
        </CardHeader>
        <CardContent>
          <CardTitle class="text-2xl">{{ collectorRunningText }}</CardTitle>
          <p class="settings-metric-footnote">{{ t('采集进程', 'Collector process') }}</p>
        </CardContent>
      </Card>
      <Card size="sm" class="settings-metric-card">
        <CardHeader>
          <CardDescription>{{ t('远端开关', 'Remote switch') }}</CardDescription>
          <CardAction><Server /></CardAction>
        </CardHeader>
        <CardContent>
          <CardTitle class="text-2xl">{{ remoteStatusText }}</CardTitle>
          <p class="settings-metric-footnote">CLIProxyAPI</p>
        </CardContent>
      </Card>
      <Card size="sm" class="settings-metric-card">
        <CardHeader>
          <CardDescription>{{ t('累计写入', 'Records written') }}</CardDescription>
          <CardAction><Database /></CardAction>
        </CardHeader>
        <CardContent>
          <CardTitle class="text-2xl">{{ formatInteger(collectorStatus?.records_collected ?? 0) }}</CardTitle>
          <p class="settings-metric-footnote">{{ t('本地记录', 'Local records') }}</p>
        </CardContent>
      </Card>
    </div>

    <div class="grid-two">
      <Card>
        <CardHeader>
          <CardTitle>{{ t('通用配置', 'General Settings') }}</CardTitle>
          <CardDescription>{{ t('管理界面品牌、连接入口、访问权限以及采集保留参数。', 'Manage interface branding, endpoints, access, and collection retention settings.') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <form @submit.prevent="saveSettings">
            <FieldGroup class="settings-form">
              <FieldSet class="settings-section">
                <FieldLegend>{{ t('界面品牌', 'Interface branding') }}</FieldLegend>
                <FieldDescription>{{ t('分别配置中文和英文界面左上角显示的名称与小标题；浏览器标题使用“名称 - 小标题”格式。', 'Configure the name and subtitle shown at the top left for each language. The browser title uses the “Name - Subtitle” format.') }}</FieldDescription>
                <FieldGroup class="form-grid">
                  <Field>
                    <FieldLabel for="brand-name-zh">{{ t('名称（中文）', 'Name (Chinese)') }}</FieldLabel>
                    <Input id="brand-name-zh" v-model="settingsForm.brand_name_zh" :maxlength="80" />
                  </Field>
                  <Field>
                    <FieldLabel for="brand-name-en">{{ t('名称（英文）', 'Name (English)') }}</FieldLabel>
                    <Input id="brand-name-en" v-model="settingsForm.brand_name_en" :maxlength="80" />
                  </Field>
                  <Field>
                    <FieldLabel for="brand-subtitle-zh">{{ t('小标题（中文）', 'Subtitle (Chinese)') }}</FieldLabel>
                    <Input id="brand-subtitle-zh" v-model="settingsForm.brand_subtitle_zh" :maxlength="120" />
                  </Field>
                  <Field>
                    <FieldLabel for="brand-subtitle-en">{{ t('小标题（英文）', 'Subtitle (English)') }}</FieldLabel>
                    <Input id="brand-subtitle-en" v-model="settingsForm.brand_subtitle_en" :maxlength="120" />
                  </Field>
                </FieldGroup>
              </FieldSet>

              <FieldSet class="settings-section">
                <FieldLegend>{{ t('连接与入口', 'Connections and endpoints') }}</FieldLegend>
                <FieldDescription>{{ t('配置 CPA 管理接口、模型请求入口与管理页面地址。', 'Configure CPA management APIs, model request endpoints, and the management page.') }}</FieldDescription>
                <FieldGroup class="form-grid">
                  <Field>
                    <FieldLabel for="cliaproxy-url">{{ t('CLIProxyAPI 地址', 'CLIProxyAPI URL') }}</FieldLabel>
                    <Input id="cliaproxy-url" v-model="settingsForm.cliaproxy_url" />
                    <FieldDescription>{{ t('用于采集队列、API Key 同步和管理接口。', 'Used for collection queues, API key sync, and management APIs.') }}</FieldDescription>
                  </Field>
                  <Field>
                    <FieldLabel for="model-request-url">{{ t('模型请求地址（例如：填写 CPA 外网地址）', 'Model request URL (for example, CPA public URL)') }}</FieldLabel>
                    <Input id="model-request-url" v-model="settingsForm.model_request_url" :placeholder="t('例如：http://192.168.26.50:8317', 'Example: http://192.168.26.50:8317')" />
                    <FieldDescription>{{ t('作为默认 Endpoint 展示，并用于 API 密钥页请求测试。', 'Displayed as the default endpoint and used for API key request tests.') }}</FieldDescription>
                  </Field>
                  <Field class="extra-endpoints-field">
                    <div class="extra-endpoints-heading">
                      <FieldContent>
                        <FieldLabel>{{ t('额外 API Endpoint', 'Additional API endpoints') }}</FieldLabel>
                        <FieldDescription>{{ t('仅展示在 API 密钥页，不参与请求测试；每项会生成四类 URL。', 'Shown only on the API keys page and not used for request tests; each item generates four URL types.') }}</FieldDescription>
                      </FieldContent>
                      <Button size="sm" variant="outline" :disabled="settingsForm.model_request_extra_endpoints.length >= 20" @click="addModelRequestExtraEndpoint">
                        <PlusIcon data-icon="inline-start" />
                        {{ t('追加 Endpoint', 'Add endpoint') }}
                      </Button>
                    </div>
                    <div v-if="settingsForm.model_request_extra_endpoints.length === 0" class="extra-endpoints-empty">
                      {{ t('暂无额外 Endpoint', 'No additional endpoints') }}
                    </div>
                    <FieldGroup v-else class="extra-endpoints-list">
                      <div v-for="(endpoint, index) in settingsForm.model_request_extra_endpoints" :key="index" class="extra-endpoint-row">
                        <InputGroup>
                          <InputGroupInput
                            v-model="endpoint.url"
                            :aria-label="t(`Endpoint ${index + 1} 基础 URL`, `Endpoint ${index + 1} base URL`)"
                            :placeholder="t('基础 URL，例如：https://api.example.com/v1', 'Base URL, for example: https://api.example.com/v1')"
                          />
                        </InputGroup>
                        <InputGroup>
                          <InputGroupInput
                            v-model="endpoint.description"
                            :aria-label="t(`Endpoint ${index + 1} 说明`, `Endpoint ${index + 1} description`)"
                            :placeholder="t('说明，例如：备用线路', 'Description, for example: Backup route')"
                            :maxlength="200"
                          />
                          <InputGroupAddon align="inline-end">
                            <InputGroupButton
                              size="icon-xs"
                              :aria-label="t('移除', 'Remove')"
                              @click="removeModelRequestExtraEndpoint(index)"
                            >
                              <Trash2Icon data-icon="inline-start" />
                            </InputGroupButton>
                          </InputGroupAddon>
                        </InputGroup>
                      </div>
                    </FieldGroup>
                  </Field>
                  <Field>
                    <FieldLabel for="cpamc-url">{{ t('CPAMC 页面地址', 'CPAMC page URL') }}</FieldLabel>
                    <Input id="cpamc-url" v-model="settingsForm.cpamc_url" :placeholder="t('例如：/management.html', 'Example: /management.html')" />
                    <FieldDescription>{{ t('用于 CPAMC 内嵌页面，支持站内路径或完整 URL。', 'Used by the embedded CPAMC page. Supports site paths or full URLs.') }}</FieldDescription>
                  </Field>
                  <Field>
                    <FieldLabel for="management-key">{{ t('管理密钥', 'Management key') }}</FieldLabel>
                    <InputGroup>
                      <InputGroupInput
                        id="management-key"
                        v-model="settingsForm.management_key"
                        :type="showManagementKey ? 'text' : 'password'"
                        :placeholder="t('请输入 CLIProxyAPI 管理密钥', 'Enter the CLIProxyAPI management key')"
                      />
                      <InputGroupAddon align="inline-end">
                        <InputGroupButton
                          size="icon-xs"
                          :aria-label="showManagementKey ? t('隐藏管理密钥', 'Hide management key') : t('显示管理密钥', 'Show management key')"
                          @click="showManagementKey = !showManagementKey"
                        >
                          <EyeOffIcon v-if="showManagementKey" data-icon="inline-start" />
                          <EyeIcon v-else data-icon="inline-start" />
                        </InputGroupButton>
                      </InputGroupAddon>
                    </InputGroup>
                    <FieldDescription>{{ t('用于访问 CLIProxyAPI 管理接口。', 'Used to access CLIProxyAPI management APIs.') }}</FieldDescription>
                  </Field>
                </FieldGroup>
              </FieldSet>

              <FieldSet class="settings-section">
                <FieldLegend>{{ t('访问配置', 'Access control') }}</FieldLegend>
                <FieldGroup class="settings-switch-list">
                  <Field orientation="horizontal" class="settings-switch">
                    <FieldContent><FieldTitle>{{ t('允许普通用户查看账号状态', 'Allow standard users to view account status') }}</FieldTitle><FieldDescription>{{ t('普通用户仅能只读查看账号状态。', 'Standard users receive read-only account status access.') }}</FieldDescription></FieldContent>
                    <Switch v-model="settingsForm.allow_user_account_status" />
                  </Field>
                  <Field orientation="horizontal" class="settings-switch">
                    <FieldContent><FieldTitle>{{ t('允许用户查看历史用量', 'Allow users to view usage history') }}</FieldTitle><FieldDescription>{{ t('开放只读历史用量面板，不提供请求明细跳转。', 'Expose the read-only usage history dashboard without record links.') }}</FieldDescription></FieldContent>
                    <Switch v-model="settingsForm.allow_user_usage_history" />
                  </Field>
                </FieldGroup>
              </FieldSet>

              <FieldSet class="settings-section">
                <FieldLegend>{{ t('采集与保留参数', 'Collection and retention') }}</FieldLegend>
                <FieldGroup class="settings-switch-list">
                  <Field orientation="horizontal" class="settings-switch">
                    <FieldContent><FieldTitle>{{ t('开启本地采集', 'Enable local collection') }}</FieldTitle><FieldDescription>{{ t('定时将 CPA 队列写入本地用量数据库。', 'Periodically write CPA queue data to the local usage database.') }}</FieldDescription></FieldContent>
                    <Switch v-model="settingsForm.collector_enabled" />
                  </Field>
                </FieldGroup>
                <FieldGroup class="form-grid">
                  <Field>
                    <FieldLabel for="collector-batch-size">{{ t('批量读取数', 'Batch size') }}</FieldLabel>
                    <Input id="collector-batch-size" type="number" :model-value="settingsForm.batch_size" :min="1" :max="1000" step="1" @update:model-value="updateNumericSetting('batch_size', $event)" />
                  </Field>
                  <Field>
                    <FieldLabel for="collector-poll-interval">{{ t('轮询间隔（秒）', 'Poll interval (seconds)') }}</FieldLabel>
                    <Input id="collector-poll-interval" type="number" :model-value="settingsForm.poll_interval_seconds" :min="0.2" step="0.1" @update:model-value="updateNumericSetting('poll_interval_seconds', $event)" />
                  </Field>
                  <Field>
                    <FieldLabel for="collector-retry-interval">{{ t('重试间隔（秒）', 'Retry interval (seconds)') }}</FieldLabel>
                    <Input id="collector-retry-interval" type="number" :model-value="settingsForm.retry_interval_seconds" :min="1" step="1" @update:model-value="updateNumericSetting('retry_interval_seconds', $event)" />
                  </Field>
                  <Field>
                    <FieldLabel for="usage-retention-days">{{ t('用量明细保留天数', 'Usage detail retention days') }}</FieldLabel>
                    <Input id="usage-retention-days" type="number" :model-value="settingsForm.usage_detail_retention_days" :min="31" step="1" @update:model-value="updateNumericSetting('usage_detail_retention_days', $event)" />
                    <FieldDescription>{{ t('最低 31 天；清理明细前会先完成小时聚合。', 'Minimum 31 days; hourly aggregation completes before details are removed.') }}</FieldDescription>
                  </Field>
                </FieldGroup>
              </FieldSet>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{{ t('采集状态', 'Collection Status') }}</CardTitle>
          <CardDescription>{{ t('当前采集器状态与最近一次执行结果。', 'Current collector state and latest execution result.') }}</CardDescription>
        </CardHeader>
        <CardContent class="status-content">
          <div class="status-list">
            <div class="status-row">
              <span>{{ t('本地采集', 'Local collection') }}</span>
              <Badge :variant="collectorStatus?.enabled ? 'default' : 'secondary'">
                {{ collectorStatus?.enabled ? t('开启', 'On') : t('关闭', 'Off') }}
              </Badge>
            </div>
            <div class="status-row">
              <span>{{ t('运行状态', 'Run status') }}</span>
              <Badge :variant="collectorStatus?.running ? 'default' : 'secondary'">
                {{ collectorStatus?.running ? t('运行中', 'Running') : t('空闲', 'Idle') }}
              </Badge>
            </div>
            <div class="status-row">
              <span>{{ t('远端开关', 'Remote switch') }}</span>
              <Badge :variant="remoteStatusType === 'error' ? 'destructive' : remoteStatusType === 'success' ? 'default' : 'outline'">
                {{ remoteStatusText }}
              </Badge>
            </div>
            <div class="status-row">
              <span>{{ t('累计写入', 'Records written') }}</span>
              <strong>{{ formatInteger(collectorStatus?.records_collected ?? 0) }}</strong>
            </div>
            <div class="status-row">
              <span>{{ t('最后轮询', 'Last poll') }}</span>
              <strong>{{ formatDateTime(collectorStatus?.last_poll_at ?? null) }}</strong>
            </div>
            <div class="status-row">
              <span>{{ t('最后成功', 'Last success') }}</span>
              <strong>{{ formatDateTime(collectorStatus?.last_success_at ?? null) }}</strong>
            </div>
          </div>

          <Alert v-if="collectorStatus?.last_error">
            <TriangleAlertIcon />
            <AlertTitle>{{ t('采集异常', 'Collector error') }}</AlertTitle>
            <AlertDescription>{{ serverText(collectorStatus.last_error, '采集异常', 'Collector error') }}</AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2">
          <ShieldCheckIcon class="size-4 text-muted-foreground" />
          {{ t('CAS 单点登录', 'CAS single sign-on') }}
        </CardTitle>
        <CardDescription>{{ t('由 CPA-Helper 直接完成 CAS 登录、用户映射和本地会话签发；默认关闭。', 'Let CPA-Helper handle CAS login, user mapping, and local sessions directly. Disabled by default.') }}</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup class="settings-form">
          <Field orientation="horizontal" class="settings-switch">
            <FieldContent>
              <FieldTitle>{{ t('启用 CAS 登录', 'Enable CAS login') }}</FieldTitle>
              <FieldDescription>{{ t('启用后登录页会提供 CAS 入口，退出登录时也会同步退出 CAS。', 'Adds CAS to the sign-in page and signs out from CAS when the local session ends.') }}</FieldDescription>
            </FieldContent>
            <Switch v-model="settingsForm.cas_enabled" />
          </Field>

          <Field
            orientation="horizontal"
            class="settings-switch"
            :data-disabled="!settingsForm.cas_enabled || undefined"
          >
            <FieldContent>
              <FieldTitle>{{ t('默认使用 CAS 登录', 'Use CAS login by default') }}</FieldTitle>
              <FieldDescription>{{ t('启用后访问登录页会自动进入 CAS；使用 /login?skipsso=true 可进入本地登录。', 'Automatically redirects the sign-in page to CAS. Use /login?skipsso=true to access local sign-in.') }}</FieldDescription>
            </FieldContent>
            <Switch v-model="settingsForm.cas_default_login" :disabled="!settingsForm.cas_enabled" />
          </Field>

          <FieldGroup class="form-grid">
            <Field>
              <FieldLabel for="cas-base-url">{{ t('CAS 服务地址', 'CAS server URL') }}</FieldLabel>
              <Input id="cas-base-url" v-model="settingsForm.cas_base_url" placeholder="https://cas.example.com/cas/app" />
              <FieldDescription>{{ t('CPA-Helper 会在此地址下调用 login、logout 和 serviceValidate。', 'CPA-Helper calls login, logout, and serviceValidate under this URL.') }}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel for="cas-public-url">{{ t('CPA-Helper 公网地址', 'CPA-Helper public URL') }}</FieldLabel>
              <Input id="cas-public-url" v-model="settingsForm.cas_public_url" placeholder="https://gateway.example.com" />
              <FieldDescription>{{ t('用于生成 CAS 回调 service，必须与浏览器实际访问地址一致。', 'Used to build the CAS callback service and must match the browser-facing URL.') }}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel for="cas-validation-url">{{ t('CAS 验证地址（可选）', 'CAS validation URL (optional)') }}</FieldLabel>
              <Input id="cas-validation-url" v-model="settingsForm.cas_validation_url" placeholder="http://127.0.0.1:8080/cas/app" />
              <FieldDescription>{{ t('仅服务端验证 Ticket 时使用；留空则使用 CAS 服务地址。', 'Used only for server-side ticket validation. Leave blank to use the CAS server URL.') }}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel for="cas-validation-host">{{ t('CAS 验证 Host（可选）', 'CAS validation Host (optional)') }}</FieldLabel>
              <Input id="cas-validation-host" v-model="settingsForm.cas_validation_host" placeholder="cas.example.com" />
              <FieldDescription>{{ t('内网验证地址需要指定原始 Host 时填写。', 'Set this when an internal validation URL requires the original Host header.') }}</FieldDescription>
            </Field>
          </FieldGroup>

          <Field orientation="horizontal" class="settings-switch">
            <FieldContent>
              <FieldTitle>{{ t('自动创建普通用户', 'Automatically create standard users') }}</FieldTitle>
              <FieldDescription>{{ t('CAS 用户首次登录时自动创建只含本地会话、没有密码的普通用户；已有用户继续沿用原角色。', 'Create passwordless standard users on first CAS sign-in; existing users keep their current roles.') }}</FieldDescription>
            </FieldContent>
            <Switch v-model="settingsForm.cas_auto_create_users" />
          </Field>
        </FieldGroup>
      </CardContent>
    </Card>

    <CodexKeeperSettingsPanel ref="keeperSettingsPanel" />
  </section>
</template>

<style scoped>
.settings-metrics {
  grid-template-columns: repeat(4, minmax(150px, 1fr));
}

.settings-metric-card {
  min-width: 0;
}

.settings-metric-card :deep([data-slot="card-action"]) {
  color: var(--muted-foreground);
}

.settings-metric-footnote {
  margin: 6px 0 0;
  color: var(--muted-foreground);
  font-size: 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 16px;
}

.settings-form {
  gap: 18px;
}

.settings-section {
  padding: 16px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface-raised);
}

.extra-endpoints-field {
  grid-column: 1 / -1;
}

.extra-endpoints-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.extra-endpoints-list {
  gap: 8px;
}

.extra-endpoint-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 16px;
}

.extra-endpoints-empty {
  padding: 12px;
  border: 1px dashed var(--cpa-border);
  border-radius: var(--cpa-radius);
  color: var(--cpa-text-muted);
  font-size: 13px;
  text-align: center;
}

.settings-switch-list {
  gap: 10px;
}

.settings-switch {
  gap: 18px;
  padding: 14px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface-muted);
}

.status-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.status-list {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.status-row {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}

.status-row:last-child {
  border-bottom: 0;
}

.status-row > span {
  color: var(--muted-foreground);
}

.status-row > strong {
  min-width: 0;
  overflow-wrap: anywhere;
  text-align: right;
}

@media (max-width: 900px) {
  .settings-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .extra-endpoint-row {
    grid-template-columns: 1fr;
  }

}

@media (max-width: 560px) {
  .settings-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
