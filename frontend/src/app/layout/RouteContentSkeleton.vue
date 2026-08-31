<script setup lang="ts">
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
</script>

<template>
  <section
    class="page route-content-skeleton"
    data-route-loading="true"
    role="status"
    aria-live="polite"
    aria-busy="true"
  >
    <span class="sr-only">正在切换页面 / Loading page</span>

    <div class="page-toolbar" aria-hidden="true">
      <Skeleton class="h-8 w-40" />
      <Skeleton class="h-8 w-24" />
    </div>

    <div class="route-skeleton-metrics" aria-hidden="true">
      <Card v-for="index in 4" :key="index" size="sm">
        <CardHeader class="flex flex-row items-start justify-between gap-3">
          <div class="flex min-w-0 flex-1 flex-col gap-3">
            <Skeleton class="h-3 w-20" />
            <Skeleton class="h-7 w-24" />
          </div>
          <Skeleton class="size-8 shrink-0 rounded-lg" />
        </CardHeader>
        <CardContent>
          <Skeleton class="h-3 w-28 max-w-full" />
        </CardContent>
      </Card>
    </div>

    <Card class="route-skeleton-panel" aria-hidden="true">
      <CardHeader class="flex flex-row items-center justify-between gap-3">
        <Skeleton class="h-5 w-28" />
        <Skeleton class="h-8 w-36" />
      </CardHeader>
      <CardContent class="flex flex-col gap-3">
        <div class="route-skeleton-controls">
          <Skeleton v-for="index in 4" :key="index" class="h-8" />
        </div>
        <Skeleton v-for="index in 7" :key="index" class="h-10 w-full" />
      </CardContent>
    </Card>
  </section>
</template>

<style scoped>
.route-content-skeleton {
  gap: 16px;
}

.route-skeleton-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  min-width: 0;
}

.route-skeleton-panel {
  min-width: 0;
}

.route-skeleton-controls {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

@media (max-width: 980px) {
  .route-skeleton-metrics,
  .route-skeleton-controls {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 620px) {
  .route-skeleton-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .route-skeleton-controls {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .route-content-skeleton :deep([data-slot="skeleton"]) {
    animation: none;
  }
}
</style>
