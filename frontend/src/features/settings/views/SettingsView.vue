<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  AppAlert,
  AppButton,
  AppDescriptions,
  AppDescriptionsItem,
  AppForm,
  AppFormItem,
  AppInput,
  AppNumberInput,
  AppStack,
  AppSwitch,
  AppBadge,
  useMessage,
} from '@/shared/ui/app-kit'
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
import { Activity, Database, Power, Server } from '@lucide/vue'

import CodexKeeperSettingsPanel from '@/features/codex-keeper/components/CodexKeeperSettingsPanel.vue'
import {
  getCollectorStatus,
  getSettings,
  updateSettings,
} from '@/features/settings/api/settingsApi'
import { useI18n } from '@/shared/i18n'
import type { CollectorStatus, ModelRequestExtraEndpoint, SettingsUpdatePayload } from '@/shared/types/api'
import { formatDateTime, formatInteger } from '@/shared/utils/format'

const message = useMessage()
const { errorText, serverText, t } = useI18n()
const isLoading = ref(false)
const isSaving = ref(false)
const collectorStatus = ref<CollectorStatus | null>(null)
const keeperSettingsPanel = ref<InstanceType<typeof CodexKeeperSettingsPanel> | null>(null)

const settingsForm = reactive({
  cliaproxy_url: 'http://127.0.0.1:8317',
  model_request_url: 'http://127.0.0.1:8317',
  model_request_extra_endpoints: [] as ModelRequestExtraEndpoint[],
  cpamc_url: '/management.html',
  management_key: '',
  collector_enabled: false,
  batch_size: 100,
  poll_interval_seconds: 2,
  retry_interval_seconds: 10,
  allow_user_account_status: false,
  allow_user_usage_history: false,
  usage_detail_retention_days: 90,
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
    settingsForm.management_key = settings.management_key
    settingsForm.collector_enabled = settings.collector_enabled
    settingsForm.batch_size = settings.batch_size
    settingsForm.poll_interval_seconds = settings.poll_interval_seconds
    settingsForm.retry_interval_seconds = settings.retry_interval_seconds
    settingsForm.allow_user_account_status = settings.allow_user_account_status
    settingsForm.allow_user_usage_history = settings.allow_user_usage_history
    settingsForm.usage_detail_retention_days = settings.usage_detail_retention_days
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
      management_key: settingsForm.management_key,
      collector_enabled: settingsForm.collector_enabled,
      batch_size: settingsForm.batch_size,
      poll_interval_seconds: settingsForm.poll_interval_seconds,
      retry_interval_seconds: settingsForm.retry_interval_seconds,
      allow_user_account_status: settingsForm.allow_user_account_status,
      allow_user_usage_history: settingsForm.allow_user_usage_history,
      usage_detail_retention_days: settingsForm.usage_detail_retention_days,
    }
    const [saved] = await Promise.all([
      updateSettings(payload),
      keeperPanel.saveSettings(),
    ])
    settingsForm.management_key = saved.management_key
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
      <AppStack>
        <AppButton secondary :loading="isLoading" @click="refresh">{{ t('刷新', 'Refresh') }}</AppButton>
        <AppButton type="primary" :loading="isSaving" @click="saveSettings">{{ t('保存设置', 'Save settings') }}</AppButton>
      </AppStack>
    </div>

    <div class="metric-grid settings-metrics">
      <div class="metric-card" :class="collectorStatus?.enabled ? 'is-green' : 'is-orange'">
        <div class="metric-icon" aria-hidden="true">
          <Power :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('本地采集', 'Local collection') }}</div>
        <div class="metric-value">{{ collectorEnabledText }}</div>
        <div class="metric-footnote">{{ t('系统开关', 'System switch') }}</div>
      </div>
      <div class="metric-card" :class="collectorStatus?.running ? 'is-teal' : 'is-blue'">
        <div class="metric-icon" aria-hidden="true">
          <Activity :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('运行状态', 'Run status') }}</div>
        <div class="metric-value">{{ collectorRunningText }}</div>
        <div class="metric-footnote">{{ t('采集进程', 'Collector process') }}</div>
      </div>
      <div class="metric-card" :class="remoteStatusType === 'success' ? 'is-green' : 'is-purple'">
        <div class="metric-icon" aria-hidden="true">
          <Server :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('远端开关', 'Remote switch') }}</div>
        <div class="metric-value">{{ remoteStatusText }}</div>
        <div class="metric-footnote">CLIProxyAPI</div>
      </div>
      <div class="metric-card is-blue">
        <div class="metric-icon" aria-hidden="true">
          <Database :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('累计写入', 'Records written') }}</div>
        <div class="metric-value">{{ formatInteger(collectorStatus?.records_collected ?? 0) }}</div>
        <div class="metric-footnote">{{ t('本地记录', 'Local records') }}</div>
      </div>
    </div>

    <div class="grid-two">
      <section class="panel">
        <div class="panel-inner">
          <h2 class="section-title">{{ t('系统配置', 'System Settings') }}</h2>
          <AppForm :model="settingsForm" label-placement="top">
            <FieldGroup class="settings-form">
              <FieldSet class="settings-section">
                <FieldLegend>{{ t('连接与入口', 'Connections and endpoints') }}</FieldLegend>
                <FieldDescription>{{ t('配置 CPA 管理接口、模型请求入口与管理页面地址。', 'Configure CPA management APIs, model request endpoints, and the management page.') }}</FieldDescription>
                <FieldGroup class="form-grid">
                  <AppFormItem :label="t('CLIProxyAPI 地址', 'CLIProxyAPI URL')" :feedback="t('用于采集队列、API Key 同步和管理接口。', 'Used for collection queues, API key sync, and management APIs.')">
                    <AppInput v-model:value="settingsForm.cliaproxy_url" />
                  </AppFormItem>
                  <AppFormItem :label="t('模型请求地址（例如：填写 CPA 外网地址）', 'Model request URL (for example, CPA public URL)')" :feedback="t('作为默认 Endpoint 展示，并用于 API 密钥页请求测试。', 'Displayed as the default endpoint and used for API key request tests.')">
                    <AppInput v-model:value="settingsForm.model_request_url" :placeholder="t('例如：http://192.168.26.50:8317', 'Example: http://192.168.26.50:8317')" />
                  </AppFormItem>
                  <Field class="extra-endpoints-field">
                    <div class="extra-endpoints-heading">
                      <FieldContent>
                        <FieldLabel>{{ t('额外 API Endpoint', 'Additional API endpoints') }}</FieldLabel>
                        <FieldDescription>{{ t('仅展示在 API 密钥页，不参与请求测试；每项会生成四类 URL。', 'Shown only on the API keys page and not used for request tests; each item generates four URL types.') }}</FieldDescription>
                      </FieldContent>
                      <AppButton size="small" type="primary" secondary :disabled="settingsForm.model_request_extra_endpoints.length >= 20" @click="addModelRequestExtraEndpoint">
                        {{ t('追加 Endpoint', 'Add endpoint') }}
                      </AppButton>
                    </div>
                    <div v-if="settingsForm.model_request_extra_endpoints.length === 0" class="extra-endpoints-empty">
                      {{ t('暂无额外 Endpoint', 'No additional endpoints') }}
                    </div>
                    <FieldGroup v-else class="extra-endpoints-list">
                      <Field v-for="(endpoint, index) in settingsForm.model_request_extra_endpoints" :key="index" orientation="horizontal" class="extra-endpoint-row">
                        <AppInput v-model:value="endpoint.url" :aria-label="t(`Endpoint ${index + 1} 基础 URL`, `Endpoint ${index + 1} base URL`)" :placeholder="t('基础 URL，例如：https://api.example.com/v1', 'Base URL, for example: https://api.example.com/v1')" />
                        <AppInput v-model:value="endpoint.description" :aria-label="t(`Endpoint ${index + 1} 说明`, `Endpoint ${index + 1} description`)" :placeholder="t('说明，例如：备用线路', 'Description, for example: Backup route')" :maxlength="200" />
                        <AppButton type="error" secondary @click="removeModelRequestExtraEndpoint(index)">{{ t('移除', 'Remove') }}</AppButton>
                      </Field>
                    </FieldGroup>
                  </Field>
                  <AppFormItem :label="t('CPAMC 页面地址', 'CPAMC page URL')" :feedback="t('用于 CPAMC 内嵌页面，支持站内路径或完整 URL。', 'Used by the embedded CPAMC page. Supports site paths or full URLs.')">
                    <AppInput v-model:value="settingsForm.cpamc_url" :placeholder="t('例如：/management.html', 'Example: /management.html')" />
                  </AppFormItem>
                  <AppFormItem :label="t('管理密钥', 'Management key')" :feedback="t('用于访问 CLIProxyAPI 管理接口。', 'Used to access CLIProxyAPI management APIs.')">
                    <AppInput v-model:value="settingsForm.management_key" type="password" show-password-on="mousedown" :placeholder="t('请输入 CLIProxyAPI 管理密钥', 'Enter the CLIProxyAPI management key')" />
                  </AppFormItem>
                </FieldGroup>
              </FieldSet>

              <FieldSet class="settings-section">
                <FieldLegend>{{ t('访问与采集', 'Access and collection') }}</FieldLegend>
                <FieldGroup class="settings-switch-list">
                  <Field orientation="horizontal" class="settings-switch">
                    <FieldContent><FieldTitle>{{ t('开启本地采集', 'Enable local collection') }}</FieldTitle><FieldDescription>{{ t('定时将 CPA 队列写入本地用量数据库。', 'Periodically write CPA queue data to the local usage database.') }}</FieldDescription></FieldContent>
                    <AppSwitch v-model:value="settingsForm.collector_enabled" />
                  </Field>
                  <Field orientation="horizontal" class="settings-switch">
                    <FieldContent><FieldTitle>{{ t('允许普通用户查看账号状态', 'Allow standard users to view account status') }}</FieldTitle><FieldDescription>{{ t('普通用户仅能只读查看账号状态。', 'Standard users receive read-only account status access.') }}</FieldDescription></FieldContent>
                    <AppSwitch v-model:value="settingsForm.allow_user_account_status" />
                  </Field>
                  <Field orientation="horizontal" class="settings-switch">
                    <FieldContent><FieldTitle>{{ t('允许用户查看历史用量', 'Allow users to view usage history') }}</FieldTitle><FieldDescription>{{ t('开放只读历史用量面板，不提供请求明细跳转。', 'Expose the read-only usage history dashboard without record links.') }}</FieldDescription></FieldContent>
                    <AppSwitch v-model:value="settingsForm.allow_user_usage_history" />
                  </Field>
                </FieldGroup>
              </FieldSet>

              <FieldSet class="settings-section">
                <FieldLegend>{{ t('采集与保留参数', 'Collection and retention') }}</FieldLegend>
                <FieldGroup class="form-grid">
                  <AppFormItem :label="t('批量读取数', 'Batch size')"><AppNumberInput v-model:value="settingsForm.batch_size" :min="1" :max="1000" /></AppFormItem>
                  <AppFormItem :label="t('轮询间隔（秒）', 'Poll interval (seconds)')"><AppNumberInput v-model:value="settingsForm.poll_interval_seconds" :min="0.2" /></AppFormItem>
                  <AppFormItem :label="t('重试间隔（秒）', 'Retry interval (seconds)')"><AppNumberInput v-model:value="settingsForm.retry_interval_seconds" :min="1" /></AppFormItem>
                  <AppFormItem :label="t('用量明细保留天数', 'Usage detail retention days')" :feedback="t('最低 31 天；清理明细前会先完成小时聚合。', 'Minimum 31 days; hourly aggregation completes before details are removed.')"><AppNumberInput v-model:value="settingsForm.usage_detail_retention_days" :min="31" :precision="0" /></AppFormItem>
                </FieldGroup>
              </FieldSet>
            </FieldGroup>
          </AppForm>
        </div>
      </section>

      <section class="panel">
        <div class="panel-inner">
          <h2 class="section-title">{{ t('采集状态', 'Collection Status') }}</h2>
          <AppDescriptions label-placement="left" :column="1" size="small" bordered>
            <AppDescriptionsItem :label="t('本地采集', 'Local collection')">
              <AppBadge :type="collectorStatus?.enabled ? 'success' : 'default'" size="small">
                {{ collectorStatus?.enabled ? t('开启', 'On') : t('关闭', 'Off') }}
              </AppBadge>
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('运行状态', 'Run status')">
              <AppBadge :type="collectorStatus?.running ? 'success' : 'default'" size="small">
                {{ collectorStatus?.running ? t('运行中', 'Running') : t('空闲', 'Idle') }}
              </AppBadge>
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('远端开关', 'Remote switch')">
              <AppBadge :type="remoteStatusType" size="small">
                {{ remoteStatusText }}
              </AppBadge>
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('累计写入', 'Records written')">
              {{ formatInteger(collectorStatus?.records_collected ?? 0) }}
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('最后轮询', 'Last poll')">
              {{ formatDateTime(collectorStatus?.last_poll_at ?? null) }}
            </AppDescriptionsItem>
            <AppDescriptionsItem :label="t('最后成功', 'Last success')">
              {{ formatDateTime(collectorStatus?.last_success_at ?? null) }}
            </AppDescriptionsItem>
          </AppDescriptions>
          <AppAlert
            v-if="collectorStatus?.last_error"
            type="warning"
            :bordered="false"
            class="status-alert"
          >
            {{ serverText(collectorStatus.last_error, '采集异常', 'Collector error') }}
          </AppAlert>
        </div>
      </section>
    </div>

    <CodexKeeperSettingsPanel ref="keeperSettingsPanel" />
  </section>
</template>

<style scoped>
.section-title {
  margin: 0 0 12px;
}

.settings-metrics {
  grid-template-columns: repeat(4, minmax(150px, 1fr));
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
  grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr) auto;
  align-items: start;
  gap: 8px;
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

.status-alert {
  margin-top: 10px;
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

  .extra-endpoint-row .n-button {
    justify-self: start;
  }
}

@media (max-width: 560px) {
  .settings-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
