<script setup lang="ts">
import type { ChartConfig } from '@/components/ui/chart'
import { VisArea, VisAxis, VisGroupedBar, VisLine, VisXYContainer } from '@unovis/vue'
import { computed } from 'vue'

import {
  ChartContainer,
  ChartCrosshair,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/components/ui/chart'
import { formatCompact } from '@/shared/utils/format'

interface TrendChartPoint {
  bucket: string
  records: number
  total_tokens: number
  failed_records: number
}

interface ChartDatum {
  x: number
  bucketLabel: string
  requests: number
  tokens: number
  tokenPlot: number
  failed: number
}

const props = defineProps<{
  data: TrendChartPoint[]
  requestsVisible: boolean
  tokensVisible: boolean
  failedVisible: boolean
  requestsLabel: string
  failedLabel: string
  formatBucket: (value: string) => string
}>()

const requestsColor = 'var(--chart-2)'
const tokensColor = 'var(--chart-4)'
const failedColor = 'var(--destructive)'

const requestMaximum = computed(() => Math.max(
  1,
  ...props.data.map((item) => Math.max(item.records, item.failed_records)),
))
const tokenMaximum = computed(() => Math.max(0, ...props.data.map((item) => item.total_tokens)))
const tokenScale = computed(() => tokenMaximum.value > 0 ? requestMaximum.value / tokenMaximum.value : 1)
const chartData = computed<ChartDatum[]>(() => props.data.map((item, index) => ({
  x: index + 1,
  bucketLabel: props.formatBucket(item.bucket),
  requests: item.records,
  tokens: item.total_tokens,
  tokenPlot: item.total_tokens * tokenScale.value,
  failed: item.failed_records,
})))
const visibilityKey = computed(() => [
  props.requestsVisible ? 'requests' : '',
  props.tokensVisible ? 'tokens' : '',
  props.failedVisible ? 'failed' : '',
].filter(Boolean).join('-') || 'empty')

const chartConfig = computed<ChartConfig>(() => ({
  ...(props.requestsVisible ? {
    requests: { label: props.requestsLabel, color: requestsColor },
  } : {}),
  ...(props.tokensVisible ? {
    tokens: { label: 'Token', color: tokensColor },
  } : {}),
  ...(props.failedVisible ? {
    failed: { label: props.failedLabel, color: failedColor },
  } : {}),
}))

const tooltipTemplate = computed(() => componentToString(
  chartConfig.value,
  ChartTooltipContent,
  { indicator: 'dot', labelKey: 'bucketLabel', valueFormatter: formatTooltipValue },
))

function formatTooltipValue(value: unknown, key: string): string {
  if (typeof value !== 'number') return String(value)
  return key === 'tokens' ? formatCompact(value) : value.toLocaleString()
}

function xLabel(value: number): string {
  const item = chartData.value[Math.max(0, Math.round(value) - 1)]
  return item?.bucketLabel ?? ''
}

function tokenAxisLabel(value: number): string {
  return formatCompact(value / tokenScale.value)
}
</script>

<template>
  <ChartContainer :config="chartConfig" cursor class="h-full min-h-0">
    <VisXYContainer
      :key="visibilityKey"
      :data="chartData"
      :margin="{ top: 12, right: 18, bottom: 4, left: 10 }"
      :y-domain="[0, undefined]"
    >
      <VisGroupedBar
        v-if="requestsVisible"
        :x="(d: ChartDatum) => d.x"
        :y="(d: ChartDatum) => d.requests"
        :color="requestsColor"
        :group-max-width="18"
        :rounded-corners="4"
        :group-padding="0.15"
      />
      <VisArea
        v-if="tokensVisible"
        :x="(d: ChartDatum) => d.x"
        :y="(d: ChartDatum) => d.tokenPlot"
        :color="tokensColor"
        :opacity="0.12"
      />
      <VisLine
        v-if="tokensVisible"
        :x="(d: ChartDatum) => d.x"
        :y="(d: ChartDatum) => d.tokenPlot"
        :color="tokensColor"
        :line-width="2.5"
      />
      <VisLine
        v-if="failedVisible"
        :x="(d: ChartDatum) => d.x"
        :y="(d: ChartDatum) => d.failed"
        :color="failedColor"
        :line-width="1.5"
        :line-dash-array="[5, 4]"
      />
      <VisAxis
        type="x"
        :x="(d: ChartDatum) => d.x"
        :num-ticks="Math.min(chartData.length, 6)"
        :tick-format="xLabel"
        :tick-line="false"
        :domain-line="false"
        :grid-line="false"
        :tick-text-hide-overlapping="true"
        :tick-text-width="90"
      />
      <VisAxis
        type="y"
        position="left"
        :num-ticks="4"
        :tick-format="formatCompact"
        :tick-line="false"
        :domain-line="false"
      />
      <VisAxis
        v-if="tokensVisible"
        type="y"
        position="right"
        :y="(d: ChartDatum) => d.tokenPlot"
        :num-ticks="4"
        :tick-format="tokenAxisLabel"
        :tick-line="false"
        :domain-line="false"
        :grid-line="false"
      />
      <ChartTooltip />
      <ChartCrosshair v-if="tooltipTemplate" :template="tooltipTemplate" color="transparent" />
    </VisXYContainer>
  </ChartContainer>
</template>
