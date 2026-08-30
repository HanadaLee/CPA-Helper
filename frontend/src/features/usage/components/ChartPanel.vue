<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { AppEmpty, AppSpinner } from '@/shared/ui/app-kit'
import * as echarts from 'echarts/core'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import {
  AxisPointerComponent,
  DatasetComponent,
  GridComponent,
  GridSimpleComponent,
  LegendComponent,
  TooltipComponent,
  type GridComponentOption,
  type LegendComponentOption,
  type TooltipComponentOption,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { BarSeriesOption, LineSeriesOption, PieSeriesOption } from 'echarts/charts'
import type { ComposeOption, ECharts } from 'echarts/core'

import { useThemePreference } from '@/shared/composables/useThemePreference'
import { useI18n } from '@/shared/i18n'

echarts.use([
  BarChart,
  LineChart,
  PieChart,
  AxisPointerComponent,
  DatasetComponent,
  GridComponent,
  GridSimpleComponent,
  LegendComponent,
  TooltipComponent,
  CanvasRenderer,
])

export type ChartOption = ComposeOption<
  | BarSeriesOption
  | LineSeriesOption
  | PieSeriesOption
  | GridComponentOption
  | LegendComponentOption
  | TooltipComponentOption
>

const props = defineProps<{
  title: string
  option: ChartOption
  empty: boolean
  loading?: boolean
  compactFooter?: boolean
}>()

const chartEl = ref<HTMLDivElement | null>(null)
const chart = ref<ECharts | null>(null)
const { isDark } = useThemePreference()
const { t } = useI18n()

let chartThemeFrame: number | undefined

function getChartTextColor(): string {
  return (
    getComputedStyle(document.documentElement).getPropertyValue('--cpa-text').trim() ||
    (isDark.value ? '#dbe7f7' : '#334155')
  )
}

function getChartMutedColor(): string {
  return (
    getComputedStyle(document.documentElement).getPropertyValue('--cpa-text-muted').trim() ||
    (isDark.value ? '#94a3b8' : '#64748b')
  )
}

function buildCurrentOption(): ChartOption {
  const option: ChartOption = {
    backgroundColor: 'transparent',
    textStyle: {
      fontFamily: 'Aptos, Segoe UI, Microsoft YaHei UI, sans-serif',
      color: getChartTextColor(),
    },
    tooltip: {
      backgroundColor: isDark.value ? 'rgba(17, 27, 45, 0.97)' : 'rgba(255, 255, 255, 0.97)',
      borderColor: isDark.value ? 'rgba(148, 163, 184, 0.24)' : 'rgba(15, 23, 42, 0.14)',
      textStyle: {
        color: getChartTextColor(),
      },
      extraCssText: 'box-shadow: 0 18px 40px rgba(15, 23, 42, 0.16); border-radius: 10px; padding: 10px 12px;',
    },
    ...props.option,
  }
  const configuredLegend = props.option.legend
  if (Array.isArray(configuredLegend)) {
    return option
  }
  option.legend = {
    ...configuredLegend,
    textStyle: {
      ...configuredLegend?.textStyle,
      color: getChartMutedColor(),
    },
  }
  return option
}

function initializeChart() {
  if (!chartEl.value) {
    return
  }
  chart.value = echarts.init(chartEl.value, isDark.value ? 'dark' : undefined)
  chart.value.setOption(buildCurrentOption())
}

function resize() {
  chart.value?.resize()
}

onMounted(() => {
  initializeChart()
  window.addEventListener('resize', resize)
})

watch(
  () => props.option,
  () => {
    chart.value?.setOption(buildCurrentOption(), true)
  },
  { deep: true },
)

watch(isDark, () => {
  if (chartThemeFrame !== undefined) {
    window.cancelAnimationFrame(chartThemeFrame)
  }
  chartThemeFrame = window.requestAnimationFrame(() => {
    if (!chartEl.value) {
      chartThemeFrame = undefined
      return
    }
    chart.value?.dispose()
    initializeChart()
    chartThemeFrame = undefined
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  if (chartThemeFrame !== undefined) {
    window.cancelAnimationFrame(chartThemeFrame)
  }
  chart.value?.dispose()
})
</script>

<template>
  <section
    class="panel chart-panel"
    :class="{ 'has-chart-footer': $slots.default, 'has-compact-footer': props.compactFooter }"
  >
    <div class="chart-heading">
      <h2>{{ title }}</h2>
      <div v-if="$slots.actions" class="chart-actions">
        <slot name="actions" />
      </div>
    </div>
    <AppSpinner :show="loading ?? false">
      <div class="chart-body">
        <div ref="chartEl" class="chart-surface" :class="{ 'is-empty': empty }" />
        <div v-if="empty" class="chart-empty">
          <AppEmpty :description="t('暂无数据', 'No data')" />
        </div>
      </div>
      <div v-if="$slots.default" class="chart-footer">
        <slot />
      </div>
    </AppSpinner>
  </section>
</template>

<style scoped>
.chart-panel {
  min-height: 270px;
}

.chart-panel.has-chart-footer {
  min-height: 318px;
}

.chart-panel.has-chart-footer.has-compact-footer {
  min-height: 278px;
}

.chart-heading {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  box-sizing: border-box;
  min-height: 56px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--cpa-border);
}

.chart-actions {
  min-width: 0;
}

h2 {
  margin: 0;
  color: var(--cpa-text-strong);
  font-size: 15px;
  font-weight: 760;
  letter-spacing: -0.01em;
}

.chart-body,
.chart-surface,
.chart-empty {
  width: 100%;
  height: 222px;
}

.chart-panel.has-chart-footer .chart-body,
.chart-panel.has-chart-footer .chart-surface,
.chart-panel.has-chart-footer .chart-empty {
  height: 160px;
}

.chart-body {
  position: relative;
  background: transparent;
}

.chart-surface.is-empty {
  visibility: hidden;
}

.chart-empty {
  display: grid;
  position: absolute;
  inset: 0;
  place-items: center;
}

.chart-footer {
  padding: 0 18px 18px;
}
</style>
