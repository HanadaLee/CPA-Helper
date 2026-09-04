<script setup lang="ts">
import type { Component, HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { Check, ChevronDown } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  Combobox,
  ComboboxAnchor,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxList,
  ComboboxTrigger,
} from '@/components/ui/combobox'
import { useI18n } from '@/shared/i18n'

export interface FilterComboboxOption {
  label: string
  value: string | number
}

const props = withDefaults(defineProps<{
  modelValue: string | number | null
  options: FilterComboboxOption[]
  placeholder: string
  searchPlaceholder?: string
  emptyText?: string
  clearText?: string
  icon?: Component
  searchable?: boolean
  searchThreshold?: number
  class?: HTMLAttributes['class']
}>(), {
  searchPlaceholder: '',
  emptyText: '',
  clearText: '',
  icon: undefined,
  searchable: undefined,
  searchThreshold: 8,
  class: undefined,
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string | number | null): void
}>()

const { t } = useI18n()
const selectedOption = computed(
  () => props.options.find(option => option.value === props.modelValue) ?? null,
)
const showSearch = computed(
  () => props.searchable ?? props.options.length >= props.searchThreshold,
)
const resolvedSearchPlaceholder = computed(
  () => props.searchPlaceholder || t(`搜索${props.placeholder}`, `Search ${props.placeholder}`),
)
const resolvedEmptyText = computed(
  () => props.emptyText || t('没有匹配选项', 'No matching options'),
)
const resolvedClearText = computed(
  () => props.clearText || t('清除筛选', 'Clear filter'),
)

function updateValue(value: unknown) {
  if (typeof value === 'object' && value !== null && 'value' in value) {
    const optionValue = value.value
    if (typeof optionValue === 'string' || typeof optionValue === 'number') {
      emit('update:modelValue', optionValue)
      return
    }
  }
  emit('update:modelValue', null)
}
</script>

<template>
  <Combobox
    :class="props.class"
    :model-value="selectedOption"
    by="value"
    @update:model-value="updateValue"
  >
    <ComboboxAnchor as-child>
      <ComboboxTrigger as-child>
        <Button
          type="button"
          variant="outline"
          class="filter-combobox-trigger"
          role="combobox"
          aria-haspopup="listbox"
          :aria-label="placeholder"
        >
          <component :is="icon" v-if="icon" data-icon="inline-start" />
          <span class="min-w-0 flex-1 truncate text-left">
            {{ selectedOption?.label ?? placeholder }}
          </span>
          <ChevronDown data-icon="inline-end" class="text-muted-foreground" />
        </Button>
      </ComboboxTrigger>
    </ComboboxAnchor>
    <ComboboxList align="start" side="bottom">
      <ComboboxInput v-if="showSearch" :placeholder="resolvedSearchPlaceholder" />
      <ComboboxEmpty v-if="showSearch">{{ resolvedEmptyText }}</ComboboxEmpty>
      <ComboboxGroup>
        <ComboboxItem :value="null">
          {{ resolvedClearText }}
        </ComboboxItem>
        <ComboboxItem
          v-for="option in options"
          :key="String(option.value)"
          :value="option"
        >
          <span class="truncate">{{ option.label }}</span>
          <ComboboxItemIndicator>
            <Check />
          </ComboboxItemIndicator>
        </ComboboxItem>
      </ComboboxGroup>
    </ComboboxList>
  </Combobox>
</template>

<style scoped>
.filter-combobox-trigger {
  width: 100%;
  justify-content: flex-start;
  border-color: var(--input);
  background: var(--card);
  font-weight: 400;
  box-shadow: var(--cpa-shadow-card);
}

.filter-combobox-trigger[aria-expanded='true'] {
  background: var(--card);
}
</style>
