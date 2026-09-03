<script setup lang="ts">
import { Copy, FileClockIcon, Trash2 } from '@lucide/vue'
import { computed, nextTick, ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { Spinner } from '@/components/ui/spinner'

import { clearCodexKeeperLogs } from '@/features/codex-keeper/api/codexKeeperApi'
import { useI18n } from '@/shared/i18n'
import { copyToClipboard } from '@/shared/utils/clipboard'
import { formatDateTime } from '@/shared/utils/format'

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
const message = toast
const { credentialServerText, errorText, t } = useI18n()
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

function logBadgeVariant(tone: LogTone): 'default' | 'destructive' | 'outline' | 'secondary' {
  if (tone === 'danger') return 'destructive'
  if (tone === 'info') return 'default'
  if (tone === 'debug') return 'secondary'
  return 'outline'
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
    time: formatDateTime(time),
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
  <Card class="log-panel">
    <CardHeader>
      <CardTitle>{{ t('维护日志', 'Maintenance Logs') }}</CardTitle>
      <CardDescription>{{ t('最新日志显示在顶部。', 'Newest entries appear first.') }}</CardDescription>
      <CardAction class="log-actions">
        <Button size="sm" variant="outline" :disabled="!logText" @click="copyLogText">
          <Copy data-icon="inline-start" />
          {{ t('复制日志', 'Copy Logs') }}
        </Button>
        <Button size="sm" variant="destructive" :disabled="isClearing" @click="clearLogs">
          <Spinner v-if="isClearing" data-icon="inline-start" />
          <Trash2 v-else data-icon="inline-start" />
          {{ t('清空日志', 'Clear Logs') }}
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <div
        ref="logBodyRef"
        class="log-view"
        role="log"
        :aria-label="t('维护日志', 'Maintenance Logs')"
        @scroll="handleLogScroll"
      >
        <Empty v-if="displayedLogLines.length === 0" class="log-empty border-0">
          <EmptyHeader>
            <EmptyMedia variant="icon"><FileClockIcon /></EmptyMedia>
            <EmptyTitle>{{ t('暂无日志', 'No logs') }}</EmptyTitle>
          </EmptyHeader>
        </Empty>
        <div v-else class="log-lines">
          <div
            v-for="line in displayedLogLines"
            :key="line.key"
            class="log-line"
            :class="`is-${line.tone}`"
            :title="credentialServerText(line.message, '维护日志', 'Maintenance log')"
          >
            <time class="log-time">{{ line.time }}</time>
            <Badge :variant="logBadgeVariant(line.tone)">{{ line.level }}</Badge>
            <span class="log-component">{{ line.component }}</span>
            <span class="log-message">{{ credentialServerText(line.message, '维护日志', 'Maintenance log') }}</span>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<style scoped>
.log-panel,
.log-view {
  min-height: 0;
}

.log-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-view {
  height: 520px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--muted);
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

.log-message {
  color: var(--cpa-text);
  overflow-wrap: anywhere;
}

.log-empty {
  height: 100%;
  min-height: 180px;
}

@media (max-width: 760px) {
  .log-actions {
    align-items: stretch;
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
