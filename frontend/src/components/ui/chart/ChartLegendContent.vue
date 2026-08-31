<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { cn } from '@/lib/utils'
import { useChart } from '.'

const props = withDefaults(defineProps<{
  hideIcon?: boolean
  nameKey?: string
  verticalAlign?: 'bottom' | 'top'
  class?: HTMLAttributes['class']
}>(), {
  verticalAlign: 'bottom',
})

const { config } = useChart()
const payload = computed(() => Object.entries(config.value).map(([key, value]) => ({
  key: props.nameKey || key,
  itemConfig: value,
})))
</script>

<template>
  <div
    :class="cn(
      'flex items-center justify-center gap-4',
      verticalAlign === 'top' ? 'pb-3' : 'pt-3',
      props.class,
    )"
  >
    <div
      v-for="{ key, itemConfig } in payload"
      :key="key"
      class="flex items-center gap-1.5 [&>svg]:size-3 [&>svg]:text-muted-foreground"
    >
      <component :is="itemConfig?.icon" v-if="itemConfig?.icon && !hideIcon" />
      <div
        v-else-if="!hideIcon"
        class="size-2 shrink-0 rounded-xs"
        :style="{ backgroundColor: itemConfig?.color }"
      />
      {{ itemConfig?.label }}
    </div>
  </div>
</template>
