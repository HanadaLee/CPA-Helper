import { currentLanguage } from '@/shared/i18n'

export function formatInteger(value: number): string {
  return new Intl.NumberFormat(currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US', {
    maximumFractionDigits: 0,
  }).format(value)
}

export function formatCompact(value: number): string {
  return new Intl.NumberFormat('en-US', {
    notation: 'compact',
    compactDisplay: 'short',
    maximumFractionDigits: 1,
  }).format(value)
}

export function formatUsd(value: number | null | undefined): string {
  const normalized = typeof value === 'number' && Number.isFinite(value) ? value : 0
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: normalized < 1 ? 6 : 2,
  }).format(normalized)
}

export const BEIJING_TIME_ZONE = 'Asia/Shanghai'
const BEIJING_OFFSET = '+08:00'
const BEIJING_OFFSET_MS = 8 * 60 * 60 * 1000

type DateTimeValue = string | number | Date | null | undefined

function parseDisplayDate(value: Exclude<DateTimeValue, null | undefined>): Date | null {
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? null : value
  }
  if (typeof value === 'number') {
    const parsed = new Date(value)
    return Number.isNaN(parsed.getTime()) ? null : parsed
  }
  const localMatch = value.match(
    /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{1,3})\d*)?)?)?$/,
  )
  if (localMatch) {
    const [, year, month, day, hour = '0', minute = '0', second = '0', millisecond = '0'] =
      localMatch
    return new Date(
      `${year}-${month}-${day}T${hour.padStart(2, '0')}:${minute.padStart(2, '0')}:${second.padStart(2, '0')}.${millisecond.padEnd(3, '0')}${BEIJING_OFFSET}`,
    )
  }
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

export function formatDateTime(
  value: DateTimeValue,
): string {
  if (value === null || value === undefined || value === '') {
    return '-'
  }
  const date = parseDisplayDate(value)
  if (!date) {
    return '-'
  }
  const beijing = new Date(date.getTime() + BEIJING_OFFSET_MS)
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${beijing.getUTCFullYear()}-${pad(beijing.getUTCMonth() + 1)}-${pad(beijing.getUTCDate())} ${pad(beijing.getUTCHours())}:${pad(beijing.getUTCMinutes())}:${pad(beijing.getUTCSeconds())}`
}

export function beijingDayRange(dayOffset = 0, now = Date.now()): [number, number] {
  const beijing = new Date(now + BEIJING_OFFSET_MS)
  const start = Date.UTC(
    beijing.getUTCFullYear(),
    beijing.getUTCMonth(),
    beijing.getUTCDate() + dayOffset,
  ) - BEIJING_OFFSET_MS
  return [start, start + 24 * 60 * 60 * 1000]
}

export function formatRelativeTime(value: string | null, now = Date.now()): string {
  if (!value) {
    return '-'
  }
  const date = parseDisplayDate(value)
  if (!date) {
    return '-'
  }

  const elapsedSeconds = Math.max(0, Math.floor((now - date.getTime()) / 1000))
  let amount: number
  let zhUnit: string
  let enUnit: string
  if (elapsedSeconds < 60) {
    amount = elapsedSeconds
    zhUnit = '秒'
    enUnit = 'second'
  } else if (elapsedSeconds < 60 * 60) {
    amount = Math.floor(elapsedSeconds / 60)
    zhUnit = '分钟'
    enUnit = 'minute'
  } else if (elapsedSeconds < 24 * 60 * 60) {
    amount = Math.floor(elapsedSeconds / (60 * 60))
    zhUnit = '小时'
    enUnit = 'hour'
  } else {
    amount = Math.floor(elapsedSeconds / (24 * 60 * 60))
    zhUnit = '天'
    enUnit = 'day'
  }
  if (currentLanguage.value === 'zh') {
    return `${amount}${zhUnit}前`
  }
  return `${amount} ${enUnit}${amount === 1 ? '' : 's'} ago`
}

export function formatLocalDateTimeParam(value: number): string {
  const date = new Date(value + BEIJING_OFFSET_MS)
  const pad = (part: number) => String(part).padStart(2, '0')
  return [
    `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`,
    `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(date.getUTCSeconds())}${BEIJING_OFFSET}`,
  ].join('T')
}

export function jsonPretty(value: unknown): string {
  return JSON.stringify(value, null, 2)
}
