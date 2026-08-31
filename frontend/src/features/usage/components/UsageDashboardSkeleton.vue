<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton'
</script>

<template>
  <section
    class="usage-dashboard-skeleton"
    data-usage-loading="true"
    role="status"
    aria-live="polite"
  >
    <span class="sr-only">正在加载用量数据 / Loading usage data</span>
    <div class="usage-skeleton-visual" aria-hidden="true">
      <div class="usage-skeleton-metrics">
        <div v-for="index in 6" :key="index" class="usage-skeleton-metric">
          <Skeleton class="h-3 w-16" />
          <Skeleton class="mt-3 h-7 w-24 max-w-[70%]" />
          <Skeleton class="mt-5 h-3 w-28 max-w-full" />
          <Skeleton class="usage-skeleton-metric-icon size-9 rounded-lg" />
        </div>
      </div>

      <div class="usage-skeleton-top-grid">
        <div
          v-for="index in 2"
          :key="index"
          class="usage-skeleton-panel"
          :class="{ 'is-token': index === 2 }"
        >
          <div class="usage-skeleton-panel-heading">
            <Skeleton class="h-4 w-24" />
            <Skeleton v-if="index === 1" class="h-7 w-36 rounded-lg" />
          </div>
          <Skeleton class="usage-skeleton-chart h-64 rounded-lg" />
          <div class="usage-skeleton-legend">
            <Skeleton
              v-for="item in index === 2 ? 4 : 3"
              :key="item"
              class="h-3 w-20"
            />
          </div>
        </div>
      </div>

      <div class="usage-skeleton-bottom-grid">
        <div v-for="column in 3" :key="column" class="usage-skeleton-column">
          <div v-for="panel in 2" :key="panel" class="usage-skeleton-panel is-compact">
            <div class="usage-skeleton-panel-heading">
              <Skeleton class="h-4 w-28" />
              <Skeleton v-if="column === 3" class="h-8 w-20 rounded-lg" />
            </div>
            <div class="usage-skeleton-rows">
              <Skeleton v-for="row in 5" :key="row" class="h-8 rounded-md" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.usage-dashboard-skeleton,
.usage-skeleton-visual {
  min-width: 0;
}

.usage-skeleton-visual {
  display: grid;
  gap: 16px;
}

.usage-skeleton-metrics {
  display: grid;
  grid-template-columns: repeat(6, minmax(120px, 1fr));
  gap: 12px;
}

.usage-skeleton-metric,
.usage-skeleton-panel {
  position: relative;
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface);
  box-shadow: var(--cpa-shadow-card);
}

.usage-skeleton-metric {
  min-height: 116px;
  padding: 16px;
}

.usage-skeleton-metric-icon {
  position: absolute;
  top: 16px;
  right: 16px;
}

.usage-skeleton-top-grid,
.usage-skeleton-bottom-grid {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.usage-skeleton-top-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.usage-skeleton-top-grid > .usage-skeleton-panel:first-child {
  grid-column: span 2;
}

.usage-skeleton-bottom-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.usage-skeleton-column {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.usage-skeleton-panel {
  padding: 16px;
}

.usage-skeleton-panel-heading {
  display: flex;
  min-height: 32px;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}

.usage-skeleton-chart {
  width: 100%;
  margin-top: 14px;
}

.usage-skeleton-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 14px;
}

.usage-skeleton-panel.is-token .usage-skeleton-legend {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.usage-skeleton-panel.is-token .usage-skeleton-legend > * {
  width: 100%;
}

.usage-skeleton-panel.is-compact {
  min-height: 300px;
}

.usage-skeleton-rows {
  display: grid;
  gap: 10px;
  margin-top: 16px;
}

@media (max-width: 1320px) {
  .usage-skeleton-metrics {
    grid-template-columns: repeat(3, minmax(128px, 1fr));
  }

  .usage-skeleton-bottom-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .usage-skeleton-top-grid,
  .usage-skeleton-bottom-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .usage-skeleton-top-grid > .usage-skeleton-panel:first-child {
    grid-column: auto;
  }
}

@media (max-width: 720px) {
  .usage-skeleton-visual {
    gap: 10px;
  }

  .usage-skeleton-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .usage-skeleton-metric {
    min-height: 104px;
    padding: 12px;
  }

  .usage-skeleton-metric-icon {
    top: 12px;
    right: 12px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .usage-dashboard-skeleton :deep([data-slot="skeleton"]) {
    animation: none;
  }
}
</style>
