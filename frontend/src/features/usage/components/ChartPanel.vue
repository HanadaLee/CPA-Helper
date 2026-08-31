<script setup lang="ts">
import { ChartNoAxesCombinedIcon } from '@lucide/vue'

import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import { Spinner } from '@/components/ui/spinner'
import { useI18n } from '@/shared/i18n'

const props = defineProps<{
  title: string
  empty: boolean
  loading?: boolean
  compactFooter?: boolean
}>()

const { t } = useI18n()
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
    <div class="chart-loading-container">
      <div class="chart-body">
        <div class="chart-surface" :class="{ 'is-empty': empty }">
          <slot name="chart" />
        </div>
        <div v-if="empty" class="chart-empty">
          <Empty class="border-0 p-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ChartNoAxesCombinedIcon />
              </EmptyMedia>
              <EmptyTitle>{{ t('暂无数据', 'No data') }}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        </div>
      </div>
      <div v-if="$slots.default" class="chart-footer">
        <slot />
      </div>
      <div v-if="loading" class="chart-loading-overlay" role="status" :aria-label="t('加载中', 'Loading')">
        <Spinner class="size-5" />
      </div>
    </div>
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

.chart-loading-container {
  position: relative;
  min-width: 0;
}

.chart-loading-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--background) 68%, transparent);
  backdrop-filter: blur(1px);
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
