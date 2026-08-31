<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton'
import { useThemePreference } from '@/shared/composables/useThemePreference'

useThemePreference()

const navigationWidths = ['68%', '76%', '61%', '72%', '64%', '78%', '58%', '70%']
</script>

<template>
  <div
    class="startup-shell"
    data-app-loading="true"
    role="status"
    aria-live="polite"
    aria-busy="true"
  >
    <span class="sr-only">正在加载应用 / Loading application</span>

    <div class="startup-visual" aria-hidden="true">
      <aside class="startup-sidebar">
        <div class="startup-brand">
          <Skeleton class="size-8 shrink-0 rounded-lg" />
          <div class="startup-brand-copy">
            <Skeleton class="h-4 w-28" />
            <Skeleton class="h-3 w-36" />
          </div>
        </div>

        <div class="startup-navigation">
          <Skeleton class="mb-1 h-3 w-20" />
          <div
            v-for="(width, index) in navigationWidths"
            :key="index"
            class="startup-navigation-row"
          >
            <Skeleton class="size-4 shrink-0 rounded-sm" />
            <Skeleton class="h-4" :style="{ width }" />
          </div>
        </div>

        <div class="startup-account">
          <Skeleton class="size-8 shrink-0 rounded-full" />
          <div class="startup-account-copy">
            <Skeleton class="h-4 w-24" />
            <Skeleton class="h-3 w-14" />
          </div>
          <Skeleton class="ml-auto size-4 shrink-0 rounded-sm" />
        </div>
      </aside>

      <section class="startup-main">
        <header class="startup-header">
          <Skeleton class="size-8 rounded-md" />
          <span class="startup-header-divider" />
          <Skeleton class="h-4 w-24" />
          <div class="startup-header-actions">
            <Skeleton class="size-8 rounded-md" />
            <Skeleton class="size-8 rounded-md" />
          </div>
        </header>

        <main class="startup-content">
          <div class="startup-page">
            <div class="startup-page-heading">
              <Skeleton class="h-8 w-40" />
              <Skeleton class="h-9 w-28 rounded-lg" />
            </div>

            <div class="startup-filter-panel">
              <div class="startup-filter-heading">
                <div>
                  <Skeleton class="h-4 w-16" />
                  <Skeleton class="mt-2 h-3 w-44" />
                </div>
                <Skeleton class="h-8 w-16 rounded-lg" />
              </div>
              <div class="startup-filter-controls">
                <Skeleton v-for="index in 5" :key="index" class="h-9 rounded-lg" />
              </div>
            </div>

            <div class="startup-metrics">
              <div v-for="index in 6" :key="index" class="startup-metric-card">
                <Skeleton class="h-3 w-16" />
                <Skeleton class="mt-3 h-7 w-24" />
                <Skeleton class="mt-5 h-3 w-28 max-w-full" />
                <Skeleton class="startup-metric-icon size-9 rounded-lg" />
              </div>
            </div>

            <div class="startup-panels">
              <div class="startup-panel is-wide">
                <div class="startup-panel-heading">
                  <Skeleton class="h-4 w-24" />
                  <Skeleton class="h-7 w-36 rounded-lg" />
                </div>
                <Skeleton class="startup-chart h-56 rounded-lg" />
              </div>
              <div class="startup-panel">
                <div class="startup-panel-heading">
                  <Skeleton class="h-4 w-24" />
                </div>
                <Skeleton class="startup-chart h-56 rounded-lg" />
              </div>
            </div>
          </div>
        </main>
      </section>
    </div>
  </div>
</template>

<style scoped>
.startup-shell,
.startup-visual {
  width: 100%;
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  overflow: hidden;
}

.startup-visual {
  display: grid;
  grid-template-columns: 16rem minmax(0, 1fr);
  background: var(--cpa-shell-bg);
}

.startup-sidebar {
  display: flex;
  min-height: 0;
  flex-direction: column;
  border-right: 1px solid var(--sidebar-border);
  background: var(--sidebar);
}

.startup-brand,
.startup-account {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px 16px;
}

.startup-brand {
  min-height: 64px;
}

.startup-brand-copy,
.startup-account-copy {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.startup-navigation {
  display: grid;
  flex: 1 1 auto;
  align-content: start;
  gap: 6px;
  min-height: 0;
  padding: 16px 14px;
}

.startup-navigation-row {
  display: flex;
  height: 42px;
  gap: 12px;
  align-items: center;
  padding: 0 10px;
  border-radius: var(--radius-md);
}

.startup-account {
  min-height: 66px;
  margin: 8px;
  padding: 10px 8px;
}

.startup-main {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  background: var(--cpa-canvas);
}

.startup-header {
  display: flex;
  height: 56px;
  flex: 0 0 56px;
  gap: 12px;
  align-items: center;
  padding: 0 20px;
  border-bottom: 1px solid var(--cpa-border);
  background: color-mix(in srgb, var(--cpa-canvas) 94%, transparent);
}

.startup-header-divider {
  width: 1px;
  height: 16px;
  background: var(--cpa-border);
}

.startup-header-actions {
  display: flex;
  gap: 4px;
  align-items: center;
  margin-left: auto;
}

.startup-content {
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
  padding: 28px 32px 36px;
  background: var(--cpa-canvas);
}

.startup-page {
  display: grid;
  width: 100%;
  max-width: 1920px;
  gap: 16px;
  margin-inline: auto;
}

.startup-page-heading,
.startup-filter-heading,
.startup-panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.startup-filter-panel,
.startup-panel,
.startup-metric-card {
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: var(--cpa-surface);
  box-shadow: var(--cpa-shadow-card);
}

.startup-filter-panel {
  padding: 16px 18px;
}

.startup-filter-controls {
  display: grid;
  grid-template-columns: repeat(5, minmax(112px, 1fr));
  gap: 10px;
  margin-top: 16px;
}

.startup-metrics {
  display: grid;
  grid-template-columns: repeat(6, minmax(120px, 1fr));
  gap: 12px;
}

.startup-metric-card {
  position: relative;
  min-width: 0;
  min-height: 116px;
  padding: 16px;
}

.startup-metric-icon {
  position: absolute;
  top: 16px;
  right: 16px;
}

.startup-panels {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(300px, 5fr);
  gap: 16px;
}

.startup-panel {
  min-width: 0;
  padding: 16px;
}

.startup-chart {
  width: 100%;
  margin-top: 16px;
}

@media (max-width: 1180px) {
  .startup-metrics {
    grid-template-columns: repeat(3, minmax(128px, 1fr));
  }

  .startup-panels {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .startup-visual {
    grid-template-columns: minmax(0, 1fr);
  }

  .startup-sidebar {
    display: none;
  }

  .startup-header {
    height: 58px;
    flex-basis: 58px;
    padding-inline: 10px;
  }

  .startup-header > :nth-child(3) {
    width: 34%;
    margin-inline: auto;
  }

  .startup-content {
    padding: 16px 12px 24px;
  }

  .startup-filter-controls {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .startup-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .startup-metric-card {
    min-height: 104px;
    padding: 12px;
  }

  .startup-metric-icon {
    top: 12px;
    right: 12px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .startup-shell :deep([data-slot="skeleton"]) {
    animation: none;
  }
}
</style>
