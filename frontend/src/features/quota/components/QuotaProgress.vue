<script setup lang="ts">
import { computed } from 'vue'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'
import { useI18n } from '@/shared/i18n'
import { formatUsd } from '@/shared/utils/format'

const props = withDefaults(defineProps<{
  label: string
  remaining: number | null
  total: number | null
  unlimited?: boolean
  inactive?: boolean
  compact?: boolean
}>(), { unlimited: false, inactive: false, compact: false })
const { t } = useI18n()
const percent = computed(() => props.unlimited ? 100 : Math.max(0, Math.min(100,
  props.total && props.total > 0 ? (props.remaining ?? 0) / props.total * 100 : 0,
)))
const amount = computed(() => props.unlimited ? t('无限制', 'Unlimited') :
  `${formatUsd(props.remaining ?? 0)} / ${formatUsd(props.total ?? 0)}`)
const color = computed(() => props.inactive ? 'var(--muted-foreground)' :
  percent.value > 30 ? 'var(--cpa-success)' : percent.value > 10 ? 'var(--cpa-warning)' : 'var(--cpa-danger)')
</script>

<template>
  <div class="quota-progress flex min-w-0 flex-col gap-1.5" :style="{ '--quota-progress-color': color }">
    <div :class="cn('flex items-center justify-between gap-y-1', compact ? 'gap-x-2 text-xs' : 'flex-wrap gap-x-3 text-sm')">
      <span class="shrink-0">{{ label }}</span>
      <span v-if="compact" class="min-w-0 truncate tabular-nums text-muted-foreground" :title="`${t('剩余', 'Remaining')} ${amount}`">{{ amount }}</span>
      <span v-else class="tabular-nums">{{ unlimited ? t('无限制', 'Unlimited') : `${Math.round(percent)}%` }}</span>
    </div>
    <Progress :model-value="percent" :aria-label="label" :aria-valuetext="`${t('剩余', 'Remaining')} ${amount}`" />
    <div v-if="!compact" class="flex flex-wrap justify-between gap-x-3 gap-y-1 text-xs text-muted-foreground">
      <span class="tabular-nums">{{ t('剩余', 'Remaining') }} {{ amount }}</span>
      <slot />
    </div>
  </div>
</template>

<style scoped>
.quota-progress :deep([data-slot='progress-indicator']) {
  background: var(--quota-progress-color);
}
</style>
