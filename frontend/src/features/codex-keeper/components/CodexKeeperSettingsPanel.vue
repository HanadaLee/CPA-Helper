<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NSwitch,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'

import {
  getCodexKeeperSettings,
  getCodexKeeperStatus,
  previewCodexKeeperSchedule,
  runCodexKeeperOnce,
  startCodexKeeper,
  stopCodexKeeper,
  updateCodexKeeperSettings,
} from '@/features/codex-keeper/api/codexKeeperApi'
import { useI18n } from '@/shared/i18n'
import type {
  CodexKeeperPriorityRule,
  CodexKeeperSettings,
  CodexKeeperSettingsUpdatePayload,
  CodexKeeperStatus,
} from '@/shared/types/api'
import { formatDateTime } from '@/shared/utils/format'

const message = useMessage()
const { errorText, t } = useI18n()
const isLoading = ref(false)
const isActing = ref(false)
const status = ref<CodexKeeperStatus | null>(null)
const priorityRules = ref<CodexKeeperPriorityRule[]>([])
const nextRunTimes = ref<string[]>([])
const schedulePreviewError = ref('')
let statusTimer: number | undefined
let schedulePreviewTimer: number | undefined

const conditionalRefreshIntervalOptions = computed(() => [
  { label: t('关闭', 'Off'), value: 0 },
  { label: t('5 秒', '5 seconds'), value: 5 },
  { label: t('10 秒', '10 seconds'), value: 10 },
  { label: t('30 秒', '30 seconds'), value: 30 },
  { label: t('60 秒', '60 seconds'), value: 60 },
])

const form = reactive({
  schedule_cron: '*/30 * * * *',
  quota_threshold: 100,
  usage_timeout_seconds: 30,
  cpa_timeout_seconds: 30,
  max_retries: 2,
  worker_threads: 8,
  conditional_refresh_interval_seconds: 30,
  account_refresh_cache_minutes: 10,
  dry_run: true,
  enable_credential_websockets: false,
  auto_start_daemon: false,
})

const runningModes = computed(() => new Set(status.value?.running_modes ?? []))
const isDaemonRunning = computed(() => status.value?.daemon_running === true)
const isRunOnceBlocked = computed(
  () => runningModes.value.has('once') || runningModes.value.has('daemon'),
)
const displayedPriorityRules = computed(() =>
  [...priorityRules.value].sort((left, right) => {
    const priorityDiff = Number(right.priority) - Number(left.priority)
    if (priorityDiff !== 0) {
      return priorityDiff
    }
    return left.account_type.localeCompare(right.account_type)
  }),
)

function applySettings(nextSettings: CodexKeeperSettings) {
  form.schedule_cron = nextSettings.schedule_cron
  form.quota_threshold = nextSettings.quota_threshold
  form.usage_timeout_seconds = nextSettings.usage_timeout_seconds
  form.cpa_timeout_seconds = nextSettings.cpa_timeout_seconds
  form.max_retries = nextSettings.max_retries
  form.worker_threads = nextSettings.worker_threads
  form.conditional_refresh_interval_seconds = nextSettings.conditional_refresh_interval_seconds
  form.account_refresh_cache_minutes = nextSettings.account_refresh_cache_minutes
  form.dry_run = nextSettings.dry_run
  form.enable_credential_websockets = nextSettings.enable_credential_websockets
  form.auto_start_daemon = nextSettings.auto_start_daemon
  nextRunTimes.value = nextSettings.next_run_times
  schedulePreviewError.value = ''
  priorityRules.value = nextSettings.priority_rules.map((rule) => ({ ...rule }))
}

async function loadAll() {
  isLoading.value = true
  try {
    const [settings, nextStatus] = await Promise.all([
      getCodexKeeperSettings(),
      getCodexKeeperStatus(),
    ])
    applySettings(settings)
    status.value = nextStatus
  } catch (error) {
    message.error(errorText(error, '加载巡检配置失败', 'Failed to load inspection settings'))
  } finally {
    isLoading.value = false
  }
}

async function loadStatus() {
  try {
    status.value = await getCodexKeeperStatus()
  } catch {
    return
  }
}

function normalizedRules(): CodexKeeperPriorityRule[] {
  const seen = new Set<string>()
  return priorityRules.value
    .map((rule) => ({
      account_type: rule.account_type.trim().toLowerCase(),
      priority: Number(rule.priority),
    }))
    .filter((rule) => {
      if (!rule.account_type || seen.has(rule.account_type)) {
        return false
      }
      seen.add(rule.account_type)
      return rule.priority >= 0 && rule.priority <= 20
    })
}

function settingsPayload(): CodexKeeperSettingsUpdatePayload {
  const rules = normalizedRules()
  if (rules.length !== priorityRules.value.length) {
    throw new Error(t('账号类型不可为空或重复，优先级必须在 0 ~ 20', 'Account types cannot be empty or duplicated, and priorities must be 0-20'))
  }

  return {
    ...form,
    priority_rules: rules,
  }
}

function validateSettings() {
  settingsPayload()
}

async function saveSettings() {
  const saved = await updateCodexKeeperSettings(settingsPayload())
  applySettings(saved)
}

async function loadSchedulePreview() {
  const scheduleCron = form.schedule_cron.trim()
  if (!scheduleCron) {
    nextRunTimes.value = []
    schedulePreviewError.value = t('请填写 Cron 表达式', 'Enter a Cron expression')
    return
  }
  try {
    const preview = await previewCodexKeeperSchedule({ schedule_cron: scheduleCron })
    if (form.schedule_cron.trim() !== scheduleCron) {
      return
    }
    nextRunTimes.value = preview.next_run_times
    schedulePreviewError.value = ''
  } catch (error) {
    if (form.schedule_cron.trim() !== scheduleCron) {
      return
    }
    nextRunTimes.value = []
    schedulePreviewError.value = errorText(
      error,
      'Cron 表达式无效，请使用 5 段格式',
      'Invalid Cron expression. Use the 5-field format',
    )
  }
}

function queueSchedulePreview() {
  if (schedulePreviewTimer !== undefined) {
    window.clearTimeout(schedulePreviewTimer)
  }
  schedulePreviewTimer = window.setTimeout(() => {
    void loadSchedulePreview()
  }, 350)
}

async function runAction(action: () => Promise<void>, successText: string) {
  isActing.value = true
  try {
    await action()
    message.success(successText)
    await loadStatus()
  } catch (error) {
    message.error(errorText(error, '操作失败', 'Operation failed'))
  } finally {
    isActing.value = false
  }
}

function addRule() {
  priorityRules.value.push({ account_type: '', priority: 0 })
}

function removeRule(rule: CodexKeeperPriorityRule) {
  const index = priorityRules.value.indexOf(rule)
  if (index >= 0) {
    priorityRules.value.splice(index, 1)
  }
}

const priorityColumns = computed<DataTableColumns<CodexKeeperPriorityRule>>(() => [
  {
    title: t('账号类型', 'Account Type'),
    key: 'account_type',
    minWidth: 132,
    render: (row) =>
      h(NInput, {
        size: 'small',
        value: row.account_type,
        placeholder: t('例如 pro_20x', 'For example, pro_20x'),
        onUpdateValue: (value: string) => {
          row.account_type = value
        },
      }),
  },
  {
    title: t('优先级', 'Priority'),
    key: 'priority',
    width: 112,
    render: (row) =>
      h(NInputNumber, {
        size: 'small',
        value: row.priority,
        min: 0,
        max: 20,
        onUpdateValue: (value: number | null) => {
          row.priority = value ?? 0
        },
      }),
  },
  {
    title: '',
    key: 'actions',
    width: 58,
    render: (row) =>
      h(
        NButton,
        { size: 'tiny', quaternary: true, type: 'error', onClick: () => removeRule(row) },
        { default: () => t('移除', 'Remove') },
      ),
  },
])

defineExpose({ saveSettings, validateSettings })

onMounted(() => {
  void loadAll()
  statusTimer = window.setInterval(() => {
    void loadStatus()
  }, 3000)
})

watch(() => form.schedule_cron, queueSchedulePreview)

onBeforeUnmount(() => {
  if (statusTimer !== undefined) {
    window.clearInterval(statusTimer)
  }
  if (schedulePreviewTimer !== undefined) {
    window.clearTimeout(schedulePreviewTimer)
  }
})
</script>

<template>
  <div class="grid-two keeper-settings-grid">
    <section class="panel keeper-config-panel">
      <div class="panel-inner config-panel-inner">
        <div class="section-heading">
          <h2 class="section-title">{{ t('巡检配置', 'Inspection Configuration') }}</h2>
          <NSpace class="config-actions" size="small" wrap>
            <NButton size="small" secondary :loading="isLoading" @click="loadAll">
              {{ t('重新加载', 'Reload') }}
            </NButton>
            <NButton
              size="small"
              secondary
              :loading="isActing"
              :disabled="isRunOnceBlocked"
              @click="runAction(runCodexKeeperOnce, t('已开始执行一轮', 'Started one inspection run'))"
            >
              {{ t('执行一轮', 'Run Once') }}
            </NButton>
            <NButton
              size="small"
              type="primary"
              secondary
              :loading="isActing"
              :disabled="isDaemonRunning"
              @click="runAction(startCodexKeeper, t('已开始自动巡检', 'Automatic inspection started'))"
            >
              {{ t('开始自动巡检', 'Start Automatic Inspection') }}
            </NButton>
            <NButton
              size="small"
              secondary
              type="warning"
              :loading="isActing"
              :disabled="!isDaemonRunning"
              @click="runAction(stopCodexKeeper, t('已请求停止', 'Stop requested'))"
            >
              {{ t('停止', 'Stop') }}
            </NButton>
          </NSpace>
        </div>
        <NForm class="config-form" :model="form" label-placement="top" size="small">
          <div class="config-sections">
            <section class="config-block">
              <h3 class="config-block-title">{{ t('调度', 'Schedule') }}</h3>
              <div class="schedule-grid">
                <NFormItem :label="t('Cron 表达式', 'Cron Expression')">
                  <NInput v-model:value="form.schedule_cron" :placeholder="t('例如 */30 * * * *', 'For example, */30 * * * *')" />
                </NFormItem>
                <div class="schedule-preview">
                  <div class="preview-title">{{ t('后续 5 次调用', 'Next 5 Runs') }}</div>
                  <div v-if="schedulePreviewError" class="preview-error">
                    {{ schedulePreviewError }}
                  </div>
                  <div v-else-if="nextRunTimes.length" class="preview-grid">
                    <span v-for="time in nextRunTimes" :key="time" class="preview-time">
                      {{ formatDateTime(time) }}
                    </span>
                  </div>
                  <div v-else class="preview-muted">{{ t('填写 Cron 表达式后显示', 'Enter a Cron expression to preview') }}</div>
                </div>
              </div>
              <div class="conditional-refresh-grid">
                <NFormItem :label="t('按条件扫描间隔', 'Conditional Scan Interval')">
                  <NSelect
                    v-model:value="form.conditional_refresh_interval_seconds"
                    :options="conditionalRefreshIntervalOptions"
                  />
                </NFormItem>
                <NFormItem :label="t('账号刷新缓存（分钟）', 'Account Refresh Cache (minutes)')">
                  <NInputNumber
                    v-model:value="form.account_refresh_cache_minutes"
                    :min="1"
                    :precision="0"
                  />
                </NFormItem>
              </div>
              <div class="conditional-refresh-help">
                <p>
                  <strong>{{ t('按条件扫描间隔：', 'Conditional scan interval:') }}</strong>{{ t('后台自动巡检开启后，每隔多久检查一次是否有账号需要刷新；会查找缓存时间内有实际请求的账号、额度刷新时间已到的账号、检测异常账号，并同步本地记录与 CPA 当前账号列表的差异。', 'How often automatic inspection checks whether accounts need refreshing after it is enabled. It looks for accounts with actual requests during the cache window, expired quota refresh times, inspection errors, and differences between local records and the current CPA account list.') }}
                </p>
                <p>
                  <strong>{{ t('账号刷新缓存：', 'Account refresh cache:') }}</strong>{{ t('控制自动任务的防重复时间；同一账号在缓存时间内不会被自动巡检或按条件扫描重复刷新，手动刷新会绕过缓存但会更新缓存时间。', 'Controls duplicate prevention for automatic tasks. The same account will not be refreshed repeatedly by automatic inspection or conditional scans during the cache window. Manual refresh bypasses the cache but updates the cache time.') }}
                </p>
              </div>
            </section>

            <section class="config-block">
              <h3 class="config-block-title">{{ t('执行参数', 'Execution Parameters') }}</h3>
              <div class="params-grid">
                <NFormItem :label="t('额度阈值（%）', 'Quota Threshold (%)')">
                  <NInputNumber v-model:value="form.quota_threshold" :min="0" :max="100" />
                </NFormItem>
                <NFormItem :label="t('额度检测超时（秒）', 'Quota Check Timeout (seconds)')">
                  <NInputNumber v-model:value="form.usage_timeout_seconds" :min="1" />
                </NFormItem>
                <NFormItem :label="t('账号管理接口超时（秒）', 'Account API Timeout (seconds)')">
                  <NInputNumber v-model:value="form.cpa_timeout_seconds" :min="1" />
                </NFormItem>
                <NFormItem :label="t('失败重试次数', 'Failure Retries')">
                  <NInputNumber v-model:value="form.max_retries" :min="0" :max="5" />
                </NFormItem>
                <NFormItem :label="t('账号处理并发数', 'Account Processing Concurrency')">
                  <NInputNumber v-model:value="form.worker_threads" :min="1" :max="64" />
                </NFormItem>
              </div>
              <div class="switch-row">
                <NFormItem class="switch-form-item">
                  <div class="switch-setting">
                    <div class="switch-copy">
                      <span class="switch-title">{{ t('只检查不修改', 'Check Only') }}</span>
                      <p class="switch-help">{{ t('开启后只模拟处理，不会禁用账号或调整优先级。', 'When enabled, processing is simulated and accounts are not disabled or reprioritized.') }}</p>
                    </div>
                    <NSwitch v-model:value="form.dry_run" class="switch-control" />
                  </div>
                </NFormItem>
                <NFormItem class="switch-form-item">
                  <div class="switch-setting">
                    <div class="switch-copy">
                      <span class="switch-title">{{ t('启用凭证 WebSocket', 'Enable Credential WebSocket') }}</span>
                      <p class="switch-help">
                        {{ t('刷新时为每个 Codex 凭证写入 websockets=true，用于 Responses API 的 WebSocket 传输。', 'During refresh, write websockets=true to each Codex credential for Responses API WebSocket transport.') }}
                      </p>
                    </div>
                    <NSwitch
                      v-model:value="form.enable_credential_websockets"
                      class="switch-control"
                    />
                  </div>
                </NFormItem>
                <NFormItem class="switch-form-item">
                  <div class="switch-setting">
                    <div class="switch-copy">
                      <span class="switch-title">{{ t('启动后自动巡检', 'Auto Inspect on Startup') }}</span>
                      <p class="switch-help">{{ t('每次 CPA-Helper 启动后，自动按上面的计划检查账号。', 'Automatically inspect accounts using the schedule above whenever CPA-Helper starts.') }}</p>
                    </div>
                    <NSwitch v-model:value="form.auto_start_daemon" class="switch-control" />
                  </div>
                </NFormItem>
              </div>
            </section>
          </div>
        </NForm>
        <section class="config-block runtime-block">
          <h3 class="config-block-title">{{ t('运行信息', 'Runtime Information') }}</h3>
          <div class="runtime-info-grid">
            <div class="runtime-stat">
              <span class="runtime-label">CLIProxyAPI</span>
              <strong class="runtime-value">
                {{ status ? t('使用系统设置中的地址和管理密钥', 'Using the system settings URL and admin key') : t('等待加载', 'Waiting to load') }}
              </strong>
            </div>
            <div class="runtime-stat">
              <span class="runtime-label">{{ t('最近开始', 'Last Started') }}</span>
              <strong class="runtime-value">
                {{ formatDateTime(status?.last_started_at ?? null) }}
              </strong>
            </div>
            <div class="runtime-stat">
              <span class="runtime-label">{{ t('最近完成', 'Last Finished') }}</span>
              <strong class="runtime-value">
                {{ formatDateTime(status?.last_finished_at ?? null) }}
              </strong>
            </div>
          </div>
        </section>
      </div>
    </section>

    <section class="panel priority-rules-panel">
      <div class="panel-inner">
        <div class="section-heading">
          <h2 class="section-title">{{ t('账号类型优先级', 'Account Type Priorities') }}</h2>
          <NButton size="small" secondary @click="addRule">{{ t('新增规则', 'Add Rule') }}</NButton>
        </div>
        <p class="section-hint">
          {{ t('账号当前优先级超过 20 时视为手动优先，巡检不会覆盖；0 ~ 20 会按这里的账号类型规则维护。', 'Current account priorities above 20 are treated as manual priority and will not be overwritten. Priorities from 0 to 20 are maintained using the account type rules here.') }}
        </p>
        <NDataTable
          class="priority-table"
          size="small"
          :columns="priorityColumns"
          :data="displayedPriorityRules"
          :pagination="false"
          :scroll-x="320"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.keeper-settings-grid {
  align-items: start;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.section-title {
  margin: 0;
  color: var(--cpa-text);
  font-size: 15px;
}

.section-hint {
  margin: -2px 0 10px;
  color: var(--cpa-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.keeper-config-panel .panel-inner {
  padding: 12px 14px 14px;
}

.config-panel-inner {
  display: grid;
  gap: 10px;
}

.config-actions {
  flex-shrink: 0;
}

.config-form {
  min-width: 0;
}

.config-form :deep(.n-form-item) {
  margin: 0;
}

.config-form :deep(.n-form-item-label) {
  padding: 0 0 4px;
  color: var(--cpa-text);
  font-size: 12px;
}

.config-form :deep(.n-form-item-feedback-wrapper) {
  min-height: 0;
}

.config-form :deep(.n-form-item-blank) {
  min-height: 30px;
}

.switch-form-item :deep(.n-form-item-blank) {
  min-height: 0;
}

.config-form :deep(.n-input-number),
.config-form :deep(.n-input) {
  width: 100%;
}

.config-sections {
  display: grid;
  gap: 10px;
}

.config-block {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--cpa-border);
  border-radius: 6px;
  background: var(--cpa-surface-muted);
}

.config-block-title {
  margin: 0 0 8px;
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.2;
}

.schedule-grid {
  display: grid;
  grid-template-columns: minmax(220px, 0.82fr) minmax(0, 1fr);
  gap: 10px;
  align-items: end;
}

.conditional-refresh-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(180px, 1fr));
  gap: 8px 10px;
  margin-top: 10px;
}

.conditional-refresh-help {
  display: grid;
  gap: 2px;
  margin: 2px 0 0;
  color: var(--cpa-text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.conditional-refresh-help p {
  margin: 0;
}

.params-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(92px, 1fr));
  gap: 8px 10px;
}

.switch-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(220px, 1fr));
  gap: 10px;
  margin-top: 10px;
}

.switch-form-item {
  min-width: 0;
}

.switch-setting {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 72px;
  padding: 10px 12px;
  border: 1px solid var(--cpa-border);
  border-radius: 6px;
  background: var(--cpa-surface);
}

.switch-copy {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.switch-title {
  color: var(--cpa-text);
  font-size: 13px;
  font-weight: 650;
  line-height: 1.25;
}

.switch-control {
  flex: 0 0 auto;
}

.switch-help {
  margin: 0;
  color: var(--cpa-text-muted);
  font-size: 12px;
  line-height: 1.35;
  text-wrap: pretty;
}

.schedule-preview {
  min-width: 0;
  min-height: 54px;
  padding: 7px 10px;
  border: 1px solid var(--cpa-border);
  border-radius: 6px;
  background: var(--cpa-surface);
}

.preview-title {
  margin-bottom: 5px;
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 600;
}

.preview-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 4px;
}

.preview-time {
  min-width: 0;
  padding: 3px 5px;
  overflow: hidden;
  border-radius: 4px;
  background: var(--cpa-surface-muted);
  color: var(--cpa-text);
  font-size: 12px;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-muted,
.preview-error {
  color: var(--cpa-text-muted);
  font-size: 12px;
}

.preview-error {
  color: var(--cpa-danger);
}

.runtime-info-grid {
  display: grid;
  grid-template-columns: minmax(180px, 1.25fr) repeat(2, minmax(120px, 1fr));
  gap: 8px;
}

.runtime-stat {
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--cpa-border);
  border-radius: 6px;
  background: var(--cpa-surface);
}

.runtime-label {
  display: block;
  margin-bottom: 3px;
  color: var(--cpa-text-muted);
  font-size: 12px;
  line-height: 1.2;
}

.runtime-value {
  display: block;
  min-width: 0;
  overflow: hidden;
  color: var(--cpa-text);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.priority-table {
  min-width: 0;
}

.priority-rules-panel {
  align-self: start;
}

.priority-rules-panel .panel-inner {
  padding: 16px 20px 18px;
}

.priority-table :deep(.n-data-table-wrapper),
.priority-table :deep(.n-data-table-base-table),
.priority-table :deep(.n-data-table-base-table-body) {
  min-width: 0;
}

.priority-table :deep(.n-data-table-th),
.priority-table :deep(.n-data-table-td) {
  padding: 6px 10px;
}

.priority-table :deep(.n-input),
.priority-table :deep(.n-input-number) {
  width: 100%;
}

@media (max-width: 760px) {
  .section-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .schedule-grid,
  .conditional-refresh-grid,
  .runtime-info-grid {
    grid-template-columns: 1fr;
  }

  .params-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .preview-grid,
  .switch-row {
    grid-template-columns: 1fr;
  }
}
</style>
