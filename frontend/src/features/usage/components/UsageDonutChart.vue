<script setup lang="ts">
import type { ChartConfig } from '@/components/ui/chart'
import { VisDonut, VisDonutSelectors, VisSingleContainer } from '@unovis/vue'
import { computed, reactive, watchEffect } from 'vue'

import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/components/ui/chart'

interface DonutItem {
  key: string
  label: string
  value: number
  colorIndex: number
}

type DonutDatum = Record<string, number>

const props = defineProps<{
  items: DonutItem[]
  centerValue: string
  centerLabel: string
}>()

const chartColors = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-4)',
  'var(--cpa-accent-orange)',
  'var(--cpa-success)',
]

const chartConfig = reactive<ChartConfig>({})
watchEffect(() => {
  for (const key of Object.keys(chartConfig)) delete chartConfig[key]
  for (const item of props.items) {
    chartConfig[item.key] = {
      label: item.label,
      color: chartColors[item.colorIndex % chartColors.length],
    }
  }
})

const chartData = computed<DonutDatum[]>(() => props.items.map((item) => ({ [item.key]: item.value })))
const tooltipTemplate = componentToString(chartConfig, ChartTooltipContent, { hideLabel: true })

function datumKey(datum: DonutDatum): string {
  return Object.keys(datum)[0] ?? ''
}

function datumValue(datum: DonutDatum): number {
  return datum[datumKey(datum)] ?? 0
}

function datumColor(datum: DonutDatum): string {
  return chartConfig[datumKey(datum)]?.color ?? chartColors[0]!
}
</script>

<template>
  <ChartContainer
    :config="chartConfig"
    class="mx-auto h-full max-h-[210px] min-h-0 w-full max-w-[300px]"
    :style="{
      '--vis-donut-central-label-font-size': '1.35rem',
      '--vis-donut-central-label-font-weight': '700',
      '--vis-donut-central-label-text-color': 'var(--foreground)',
      '--vis-donut-central-sub-label-text-color': 'var(--muted-foreground)',
    }"
  >
    <VisSingleContainer :data="chartData" :margin="{ top: 12, right: 12, bottom: 12, left: 12 }">
      <VisDonut
        :id="datumKey"
        :value="datumValue"
        :color="datumColor"
        :arc-width="28"
        :pad-angle="0.02"
        :corner-radius="4"
        :central-label="centerValue"
        :central-sub-label="centerLabel"
        :duration="250"
      />
      <ChartTooltip
        v-if="tooltipTemplate"
        :triggers="{
          [VisDonutSelectors.segment]: tooltipTemplate,
        }"
      />
    </VisSingleContainer>
  </ChartContainer>
</template>
