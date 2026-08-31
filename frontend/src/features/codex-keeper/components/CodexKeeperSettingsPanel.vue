<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
  Play,
  Plus,
  RefreshCw,
  Square,
  Trash2,
} from '@lucide/vue'

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

const message = toast
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

function numberInput(value: string | number, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function setConditionalRefreshInterval(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    form.conditional_refresh_interval_seconds = value
  }
}

function updateRulePriority(rule: CodexKeeperPriorityRule, value: string | number) {
  rule.priority = numberInput(value, 0)
}

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
  <div class="keeper-settings-grid">
    <Card class="keeper-config-panel">
      <CardHeader class="keeper-card-header">
        <div class="min-w-0">
          <CardTitle>{{ t('巡检配置', 'Inspection Configuration') }}</CardTitle>
          <CardDescription>{{ t('配置巡检调度、执行参数与后台运行方式。', 'Configure inspection scheduling, execution, and background behavior.') }}</CardDescription>
        </div>
        <div class="config-actions">
          <Button size="sm" variant="outline" :disabled="isLoading" @click="loadAll">
            <Spinner v-if="isLoading" data-icon="inline-start" />
            <RefreshCw v-else data-icon="inline-start" />
            {{ t('重新加载', 'Reload') }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="isActing || isRunOnceBlocked"
            @click="runAction(runCodexKeeperOnce, t('已开始执行一轮', 'Started one inspection run'))"
          >
            <Spinner v-if="isActing" data-icon="inline-start" />
            <Play v-else data-icon="inline-start" />
            {{ t('执行一轮', 'Run Once') }}
          </Button>
          <Button
            size="sm"
            :disabled="isActing || isDaemonRunning"
            @click="runAction(startCodexKeeper, t('已开始自动巡检', 'Automatic inspection started'))"
          >
            <Play data-icon="inline-start" />
            {{ t('开始自动巡检', 'Start Automatic Inspection') }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="isActing || !isDaemonRunning"
            @click="runAction(stopCodexKeeper, t('已请求停止', 'Stop requested'))"
          >
            <Square data-icon="inline-start" />
            {{ t('停止', 'Stop') }}
          </Button>
        </div>
      </CardHeader>

      <CardContent class="config-panel-inner">
        <section class="config-block">
          <h3 class="config-block-title">{{ t('调度', 'Schedule') }}</h3>
          <div class="schedule-grid">
            <Field>
              <FieldLabel for="keeper-schedule-cron">{{ t('Cron 表达式', 'Cron Expression') }}</FieldLabel>
              <Input
                id="keeper-schedule-cron"
                v-model="form.schedule_cron"
                :placeholder="t('例如 */30 * * * *', 'For example, */30 * * * *')"
              />
            </Field>
            <div class="schedule-preview">
              <div class="preview-title">{{ t('后续 5 次调用', 'Next 5 Runs') }}</div>
              <div v-if="schedulePreviewError" class="preview-error">{{ schedulePreviewError }}</div>
              <div v-else-if="nextRunTimes.length" class="preview-grid">
                <span v-for="time in nextRunTimes" :key="time" class="preview-time">
                  {{ formatDateTime(time) }}
                </span>
              </div>
              <div v-else class="preview-muted">{{ t('填写 Cron 表达式后显示', 'Enter a Cron expression to preview') }}</div>
            </div>
          </div>

          <FieldGroup class="conditional-refresh-grid">
            <Field>
              <FieldLabel>{{ t('按条件扫描间隔', 'Conditional Scan Interval') }}</FieldLabel>
              <Select
                :model-value="form.conditional_refresh_interval_seconds"
                @update:model-value="setConditionalRefreshInterval"
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem
                      v-for="option in conditionalRefreshIntervalOptions"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel for="keeper-refresh-cache">{{ t('账号刷新缓存（分钟）', 'Account Refresh Cache (minutes)') }}</FieldLabel>
              <Input
                id="keeper-refresh-cache"
                type="number"
                min="1"
                step="1"
                :model-value="form.account_refresh_cache_minutes"
                @update:model-value="form.account_refresh_cache_minutes = numberInput($event, 10)"
              />
            </Field>
          </FieldGroup>

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
          <FieldGroup class="params-grid">
            <Field>
              <FieldLabel for="keeper-quota-threshold">{{ t('额度阈值（%）', 'Quota Threshold (%)') }}</FieldLabel>
              <Input id="keeper-quota-threshold" type="number" min="0" max="100" :model-value="form.quota_threshold" @update:model-value="form.quota_threshold = numberInput($event, 100)" />
            </Field>
            <Field>
              <FieldLabel for="keeper-usage-timeout">{{ t('额度检测超时（秒）', 'Quota Check Timeout (seconds)') }}</FieldLabel>
              <Input id="keeper-usage-timeout" type="number" min="1" :model-value="form.usage_timeout_seconds" @update:model-value="form.usage_timeout_seconds = numberInput($event, 30)" />
            </Field>
            <Field>
              <FieldLabel for="keeper-cpa-timeout">{{ t('账号管理接口超时（秒）', 'Account API Timeout (seconds)') }}</FieldLabel>
              <Input id="keeper-cpa-timeout" type="number" min="1" :model-value="form.cpa_timeout_seconds" @update:model-value="form.cpa_timeout_seconds = numberInput($event, 30)" />
            </Field>
            <Field>
              <FieldLabel for="keeper-max-retries">{{ t('失败重试次数', 'Failure Retries') }}</FieldLabel>
              <Input id="keeper-max-retries" type="number" min="0" max="5" :model-value="form.max_retries" @update:model-value="form.max_retries = numberInput($event, 2)" />
            </Field>
            <Field>
              <FieldLabel for="keeper-workers">{{ t('账号处理并发数', 'Account Processing Concurrency') }}</FieldLabel>
              <Input id="keeper-workers" type="number" min="1" max="64" :model-value="form.worker_threads" @update:model-value="form.worker_threads = numberInput($event, 8)" />
            </Field>
          </FieldGroup>

          <FieldGroup class="switch-row">
            <Field orientation="horizontal" class="switch-setting">
              <FieldContent>
                <FieldTitle>{{ t('只检查不修改', 'Check Only') }}</FieldTitle>
                <FieldDescription>{{ t('开启后只模拟处理，不会禁用账号或调整优先级。', 'When enabled, processing is simulated and accounts are not disabled or reprioritized.') }}</FieldDescription>
              </FieldContent>
              <Switch v-model="form.dry_run" />
            </Field>
            <Field orientation="horizontal" class="switch-setting">
              <FieldContent>
                <FieldTitle>{{ t('启用凭证 WebSocket', 'Enable Credential WebSocket') }}</FieldTitle>
                <FieldDescription>{{ t('刷新时为每个 Codex 凭证写入 websockets=true，用于 Responses API 的 WebSocket 传输。', 'During refresh, write websockets=true to each Codex credential for Responses API WebSocket transport.') }}</FieldDescription>
              </FieldContent>
              <Switch v-model="form.enable_credential_websockets" />
            </Field>
            <Field orientation="horizontal" class="switch-setting">
              <FieldContent>
                <FieldTitle>{{ t('启动后自动巡检', 'Auto Inspect on Startup') }}</FieldTitle>
                <FieldDescription>{{ t('每次 CPA-Helper 启动后，自动按上面的计划检查账号。', 'Automatically inspect accounts using the schedule above whenever CPA-Helper starts.') }}</FieldDescription>
              </FieldContent>
              <Switch v-model="form.auto_start_daemon" />
            </Field>
          </FieldGroup>
        </section>

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
              <strong class="runtime-value">{{ formatDateTime(status?.last_started_at ?? null) }}</strong>
            </div>
            <div class="runtime-stat">
              <span class="runtime-label">{{ t('最近完成', 'Last Finished') }}</span>
              <strong class="runtime-value">{{ formatDateTime(status?.last_finished_at ?? null) }}</strong>
            </div>
          </div>
        </section>
      </CardContent>
    </Card>

    <Card class="priority-rules-panel">
      <CardHeader class="keeper-card-header">
        <div class="min-w-0">
          <CardTitle>{{ t('账号类型优先级', 'Account Type Priorities') }}</CardTitle>
          <CardDescription>
            {{ t('账号当前优先级超过 20 时视为手动优先，巡检不会覆盖；0 ~ 20 会按这里的账号类型规则维护。', 'Current account priorities above 20 are treated as manual priority and will not be overwritten. Priorities from 0 to 20 are maintained using the account type rules here.') }}
          </CardDescription>
        </div>
        <Button size="sm" variant="outline" @click="addRule">
          <Plus data-icon="inline-start" />
          {{ t('新增规则', 'Add Rule') }}
        </Button>
      </CardHeader>
      <CardContent>
        <div class="priority-table-shell">
          <Table class="priority-table">
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('账号类型', 'Account Type') }}</TableHead>
                <TableHead class="w-28">{{ t('优先级', 'Priority') }}</TableHead>
                <TableHead class="w-12"><span class="sr-only">{{ t('操作', 'Actions') }}</span></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableEmpty v-if="displayedPriorityRules.length === 0" :colspan="3">
                {{ t('暂无优先级规则', 'No priority rules') }}
              </TableEmpty>
              <TableRow v-for="rule in displayedPriorityRules" v-else :key="rule.account_type">
                <TableCell>
                  <Input v-model="rule.account_type" :placeholder="t('例如 pro_20x', 'For example, pro_20x')" />
                </TableCell>
                <TableCell>
                  <Input
                    type="number"
                    min="0"
                    max="20"
                    :model-value="rule.priority"
                    @update:model-value="updateRulePriority(rule, $event)"
                  />
                </TableCell>
                <TableCell class="text-right">
                  <Button size="icon-sm" variant="ghost" :aria-label="t('移除', 'Remove')" @click="removeRule(rule)">
                    <Trash2 />
                  </Button>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  </div>
</template>

<style scoped>
.keeper-settings-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(320px, .82fr);
  align-items: start;
  gap: 16px;
  min-width: 0;
}

.keeper-card-header {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.config-panel-inner {
  display: grid;
  gap: 1rem;
  min-width: 0;
}

.config-actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: .5rem;
}

.config-block {
  min-width: 0;
  padding: 1rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--muted) 22%, transparent);
}

.config-block-title {
  margin: 0 0 .875rem;
  color: var(--foreground);
  font-size: .8125rem;
  font-weight: 600;
  line-height: 1.2;
}

.schedule-grid {
  display: grid;
  grid-template-columns: minmax(13.75rem, .82fr) minmax(0, 1fr);
  gap: .75rem;
  align-items: end;
}

.conditional-refresh-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(11.25rem, 1fr));
  gap: .75rem;
  margin-top: 1rem;
}

.conditional-refresh-help {
  display: grid;
  gap: .25rem;
  margin: .75rem 0 0;
  color: var(--muted-foreground);
  font-size: .75rem;
  line-height: 1.45;
}

.conditional-refresh-help p {
  margin: 0;
}

.params-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(5.75rem, 1fr));
  gap: .75rem;
}

.switch-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(13.75rem, 1fr));
  gap: .75rem;
  margin-top: 1rem;
}

.switch-setting {
  width: 100%;
  min-height: 5rem;
  min-width: 0;
  padding: .875rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--card);
}

.schedule-preview {
  min-width: 0;
  min-height: 3.375rem;
  padding: .5rem .625rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--card);
}

.preview-title {
  margin-bottom: .3125rem;
  color: var(--foreground);
  font-size: .75rem;
  font-weight: 600;
}

.preview-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: .25rem;
}

.preview-time {
  min-width: 0;
  padding: .1875rem .3125rem;
  overflow: hidden;
  border-radius: var(--radius-md);
  background: var(--muted);
  color: var(--foreground);
  font-size: .75rem;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-muted,
.preview-error {
  color: var(--muted-foreground);
  font-size: .75rem;
}

.preview-error {
  color: var(--destructive);
}

.runtime-info-grid {
  display: grid;
  grid-template-columns: minmax(11.25rem, 1.25fr) repeat(2, minmax(7.5rem, 1fr));
  gap: .5rem;
}

.runtime-stat {
  min-width: 0;
  padding: .625rem .75rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--card);
}

.runtime-label {
  display: block;
  margin-bottom: .1875rem;
  color: var(--muted-foreground);
  font-size: .75rem;
  line-height: 1.2;
}

.runtime-value {
  display: block;
  min-width: 0;
  overflow: hidden;
  color: var(--foreground);
  font-size: .75rem;
  font-weight: 600;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.priority-table-shell {
  min-width: 0;
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.priority-rules-panel {
  align-self: start;
}

.priority-table {
  min-width: 20rem;
}

@media (max-width: 1180px) {
  .keeper-settings-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .keeper-card-header {
    flex-direction: column;
  }

  .config-actions {
    width: 100%;
    justify-content: flex-start;
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

@media (max-width: 480px) {
  .params-grid {
    grid-template-columns: 1fr;
  }

  .config-actions > button {
    flex: 1 1 auto;
  }
}
</style>
