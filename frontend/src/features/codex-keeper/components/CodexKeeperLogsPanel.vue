<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { AppButton, AppIcon, AppStack, useMessage } from '@/shared/ui/app-kit'
import { Copy, Trash2 } from '@lucide/vue'

import { clearCodexKeeperLogs } from '@/features/codex-keeper/api/codexKeeperApi'
import { useI18n } from '@/shared/i18n'
import { copyToClipboard } from '@/shared/utils/clipboard'

type LogTone = 'danger' | 'debug' | 'default' | 'info' | 'warning'

interface ParsedLogLine {
  component: string
  key: string
  level: string
  message: string
  raw: string
  time: string
  tone: LogTone
}

const props = defineProps<{
  logs: string[]
}>()
const emit = defineEmits<{
  refresh: []
}>()
const message = useMessage()
const { errorText, serverText, t } = useI18n()
const isClearing = ref(false)
const logBodyRef = ref<HTMLElement | null>(null)
const shouldFollowLatestLog = ref(true)

const parsedLogLines = computed(() =>
  props.logs
    .map((line, index) => parseLogLine(line, index))
    .filter((line): line is ParsedLogLine => line !== null),
)
const logText = computed(() => parsedLogLines.value.map((line) => line.raw).join('\n'))
const displayedLogLines = computed(() => [...parsedLogLines.value].reverse())

watch(logText, () => {
  if (shouldFollowLatestLog.value) {
    scrollLogToTop()
  }
})

function logTone(level: string): LogTone {
  const normalizedLevel = level.trim().toUpperCase()
  if (normalizedLevel === 'INFO') {
    return 'info'
  }
  if (normalizedLevel === 'WARNING' || normalizedLevel === 'WARN') {
    return 'warning'
  }
  if (normalizedLevel === 'ERROR' || normalizedLevel === 'CRITICAL' || normalizedLevel === 'FATAL') {
    return 'danger'
  }
  if (normalizedLevel === 'DEBUG') {
    return 'debug'
  }
  return 'default'
}

function parseSlogFields(line: string): Record<string, string> | null {
  const fields: Record<string, string> = {}
  let cursor = 0
  while (cursor < line.length) {
    while (line[cursor] === ' ') {
      cursor += 1
    }
    if (cursor >= line.length) {
      break
    }
    const keyStart = cursor
    while (cursor < line.length && line[cursor] !== '=' && line[cursor] !== ' ') {
      cursor += 1
    }
    if (cursor >= line.length || line[cursor] !== '=') {
      return null
    }
    const key = line.slice(keyStart, cursor)
    cursor += 1

    let value = ''
    if (line[cursor] === '"') {
      cursor += 1
      let escaped = false
      while (cursor < line.length) {
        const char = line[cursor]
        cursor += 1
        if (escaped) {
          value += char
          escaped = false
          continue
        }
        if (char === '\\') {
          escaped = true
          continue
        }
        if (char === '"') {
          break
        }
        value += char
      }
    } else {
      const valueStart = cursor
      while (cursor < line.length && line[cursor] !== ' ') {
        cursor += 1
      }
      value = line.slice(valueStart, cursor)
    }
    fields[key] = value
  }
  return fields.time && fields.level && fields.msg ? fields : null
}

function parseLogLine(line: string, index: number): ParsedLogLine | null {
  const fields = parseSlogFields(line)
  if (!fields) {
    return null
  }
  const time = fields.time
  const level = fields.level
  const messageText = fields.msg
  if (!time || !level || !messageText) {
    return null
  }
  const component = fields.component ?? '-'
  const extraFields = Object.entries(fields)
    .filter(([key]) => !['time', 'level', 'component', 'msg'].includes(key))
    .map(([key, value]) => `${key}=${value}`)
  const logMessage = [messageText, ...extraFields].filter(Boolean).join(' ')
  return {
    component,
    key: `${index}-${line}`,
    level,
    message: logMessage,
    raw: line,
    time,
    tone: logTone(level),
  }
}

function handleLogScroll(event: Event) {
  const target = event.currentTarget
  if (target instanceof HTMLElement) {
    shouldFollowLatestLog.value = target.scrollTop <= 48
  }
}

function scrollLogToTop() {
  void nextTick(() => {
    const logBody = logBodyRef.value
    if (logBody && shouldFollowLatestLog.value) {
      logBody.scrollTop = 0
    }
  })
}

async function copyLogText() {
  if (!logText.value) {
    message.info(t('暂无日志可复制', 'No logs to copy'))
    return
  }
  try {
    await copyToClipboard(logText.value)
    message.success(t('维护日志已复制', 'Maintenance logs copied'))
  } catch (error) {
    message.error(errorText(error, '复制失败', 'Copy failed'))
  }
}

async function clearLogs() {
  isClearing.value = true
  try {
    await clearCodexKeeperLogs()
    message.success(t('日志已清空', 'Logs cleared'))
    emit('refresh')
  } catch (error) {
    message.error(errorText(error, '清空日志失败', 'Failed to clear logs'))
  } finally {
    isClearing.value = false
  }
}
</script>

<template>
  <section class="panel log-panel">
    <div class="panel-inner log-panel-inner">
      <div class="section-heading">
        <h2 class="section-title">{{ t('维护日志', 'Maintenance Logs') }}</h2>
        <AppStack class="log-actions" size="small">
          <AppButton secondary :disabled="!logText" @click="copyLogText">
            <template #icon>
              <AppIcon :component="Copy" />
            </template>
            {{ t('复制日志', 'Copy Logs') }}
          </AppButton>
          <AppButton secondary :loading="isClearing" @click="clearLogs">
            <template #icon>
              <AppIcon :component="Trash2" />
            </template>
            {{ t('清空日志', 'Clear Logs') }}
          </AppButton>
        </AppStack>
      </div>
      <div
        ref="logBodyRef"
        class="log-view"
        role="log"
        :aria-label="t('维护日志', 'Maintenance Logs')"
        @scroll="handleLogScroll"
      >
        <div v-if="displayedLogLines.length === 0" class="log-empty">{{ t('暂无日志', 'No logs') }}</div>
        <div v-else class="log-lines">
          <div
            v-for="line in displayedLogLines"
            :key="line.key"
            class="log-line"
            :class="`is-${line.tone}`"
            :title="serverText(line.message, '维护日志', 'Maintenance log')"
          >
            <time class="log-time">{{ line.time }}</time>
            <span class="log-level">{{ line.level }}</span>
            <span class="log-component">{{ line.component }}</span>
            <span class="log-message">{{ serverText(line.message, '维护日志', 'Maintenance log') }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.log-panel,
.log-panel-inner,
.log-view {
  min-height: 0;
}

.log-panel-inner {
  display: grid;
  gap: 10px;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-title {
  margin: 0;
  color: var(--cpa-text);
  font-size: 15px;
}

.log-actions {
  flex-shrink: 0;
}

.log-view {
  height: 520px;
  overflow: auto;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  background:
    linear-gradient(180deg, rgb(255 255 255 / 54%), rgb(255 255 255 / 18%)),
    var(--cpa-surface-muted);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 78%);
  scrollbar-color: color-mix(in srgb, var(--cpa-text-muted) 44%, transparent) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.log-view::-webkit-scrollbar {
  width: 14px;
  height: 14px;
}

.log-view::-webkit-scrollbar-track,
.log-view::-webkit-scrollbar-corner {
  background: transparent;
}

.log-view::-webkit-scrollbar-thumb {
  min-height: 48px;
  border: 5px solid transparent;
  border-radius: 999px;
  background: color-mix(in srgb, var(--cpa-text-muted) 44%, transparent);
  background-clip: content-box;
}

.log-view::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--cpa-primary) 58%, var(--cpa-text-muted));
  background-clip: content-box;
}

:root.dark .log-view {
  background:
    linear-gradient(180deg, rgb(255 255 255 / 5%), rgb(255 255 255 / 1%)),
    var(--cpa-surface-muted);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 8%);
}

.log-lines {
  display: grid;
  min-width: 860px;
  padding: 8px;
}

.log-line {
  display: grid;
  grid-template-columns: 216px 68px minmax(112px, 148px) minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  min-width: 0;
  padding: 7px 9px;
  border-bottom: 1px solid color-mix(in srgb, var(--cpa-border) 74%, transparent);
  color: var(--cpa-text);
  font-family: "Cascadia Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
}

.log-line:last-child {
  border-bottom: 0;
}

.log-line:hover {
  border-radius: var(--cpa-radius-sm);
  background: var(--cpa-primary-wash);
}

:root.dark .log-line:hover {
  background: color-mix(in srgb, var(--cpa-primary-wash) 70%, transparent);
}

.log-time,
.log-component,
.log-message {
  min-width: 0;
}

.log-time,
.log-component {
  overflow: hidden;
  color: var(--cpa-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-level {
  display: inline-flex;
  justify-content: center;
  width: fit-content;
  min-width: 48px;
  padding: 1px 6px;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius-sm);
  background: var(--cpa-surface);
  color: var(--cpa-text-muted);
  font-size: 11px;
  font-weight: 760;
  line-height: 1.35;
}

.log-message {
  color: var(--cpa-text);
  overflow-wrap: anywhere;
}

.log-line.is-info .log-level {
  border-color: color-mix(in srgb, var(--cpa-primary) 24%, transparent);
  background: var(--cpa-primary-wash);
  color: var(--cpa-primary);
}

.log-line.is-warning .log-level {
  border-color: color-mix(in srgb, var(--cpa-warning) 28%, transparent);
  background: var(--cpa-warning-weak);
  color: var(--cpa-warning);
}

.log-line.is-danger .log-level {
  border-color: color-mix(in srgb, var(--cpa-danger) 28%, transparent);
  background: var(--cpa-danger-weak);
  color: var(--cpa-danger);
}

.log-line.is-debug .log-level {
  border-color: color-mix(in srgb, var(--cpa-accent-blue) 24%, transparent);
  background: var(--cpa-accent-blue-weak);
  color: var(--cpa-accent-blue);
}

.log-empty {
  display: grid;
  height: 100%;
  min-height: 180px;
  place-items: center;
  color: var(--cpa-text-muted);
  font-size: 13px;
}

@media (max-width: 760px) {
  .section-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .log-view {
    height: 420px;
  }

  .log-lines {
    min-width: 0;
  }

  .log-line {
    grid-template-columns: 142px 58px minmax(0, 1fr);
  }

  .log-message {
    grid-column: 1 / -1;
  }
}
</style>
