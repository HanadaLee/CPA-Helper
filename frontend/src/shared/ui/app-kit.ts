import type {
  ComputedRef,
  CSSProperties,
  Component,
  HTMLAttributes,
  InjectionKey,
  PropType,
  Slots,
  VNodeChild,
} from 'vue'
import type { DateRange } from 'reka-ui'
import {
  computed,
  defineComponent,
  h,
  inject,
  provide,
  reactive,
  ref,
  watch,
} from 'vue'
import {
  CalendarDaysIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
  InboxIcon,
  LoaderCircleIcon,
  XIcon,
} from '@lucide/vue'
import { CalendarDate } from '@internationalized/date'
import { toast } from 'vue-sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
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
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Empty,
  EmptyContent,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { RangeCalendar } from '@/components/ui/range-calendar'
import { Separator } from '@/components/ui/separator'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/sonner'
import { cn } from '@/lib/utils'
import { localize } from '@/shared/i18n'

export type DataTableRowKey = string | number

export interface SelectOption {
  label: string
  value: string | number | boolean | null
  disabled?: boolean
}

export interface MenuOption {
  type?: 'group' | 'divider'
  label?: string | (() => VNodeChild)
  key: string | number
  icon?: () => VNodeChild
  children?: MenuOption[]
  disabled?: boolean
}

export interface DataTableColumn<T = Record<string, unknown>> {
  type?: 'selection'
  title?: string | (() => VNodeChild)
  key?: string | number
  width?: string | number
  minWidth?: string | number
  maxWidth?: string | number
  fixed?: 'left' | 'right'
  align?: 'left' | 'center' | 'right'
  ellipsis?: boolean | { tooltip?: boolean }
  disabled?: (row: T) => boolean
  render?: (row: T, index: number) => VNodeChild
}

export type DataTableColumns<T = Record<string, unknown>> = DataTableColumn<T>[]

function normalizeSize(size: unknown, iconOnly = false) {
  if (iconOnly) {
    if (size === 'tiny') return 'icon-xs'
    if (size === 'small') return 'icon-sm'
    if (size === 'large') return 'icon-lg'
    return 'icon'
  }
  if (size === 'tiny') return 'xs'
  if (size === 'small') return 'sm'
  if (size === 'large') return 'lg'
  return 'default'
}

function styleSize(value: string | number | undefined) {
  return typeof value === 'number' ? `${value}px` : value
}

function renderSlot(slots: Slots, name = 'default') {
  return slots[name]?.()
}

export const AppButton = defineComponent({
  name: 'AppButton',
  inheritAttrs: false,
  props: {
    type: { type: String, default: 'default' },
    size: { type: String, default: 'medium' },
    loading: Boolean,
    disabled: Boolean,
    secondary: Boolean,
    tertiary: Boolean,
    quaternary: Boolean,
    text: Boolean,
    dashed: Boolean,
    circle: Boolean,
    round: Boolean,
    block: Boolean,
    attrType: { type: String, default: 'button' },
  },
  setup(props, { attrs, slots }) {
    return () => {
      let variant: 'default' | 'outline' | 'secondary' | 'ghost' | 'destructive' | 'link' = 'outline'
      if (props.type === 'primary') variant = 'default'
      else if (props.type === 'error') variant = 'destructive'
      else if (props.text) variant = 'link'
      else if (props.quaternary || props.tertiary) variant = 'ghost'
      else if (props.secondary || props.dashed) variant = 'outline'
      const iconOnly = props.circle
      return h(
        Button,
        {
          ...attrs,
          class: cn(
            'n-button',
            props.round && 'rounded-full',
            props.circle && 'n-button--circle',
            props.block && 'w-full',
            props.dashed && 'border-dashed',
            props.type === 'warning' && 'border-[var(--cpa-warning)] text-[var(--cpa-warning)]',
            attrs.class as HTMLAttributes['class'],
          ),
          variant,
          size: normalizeSize(props.size, iconOnly),
          disabled: props.disabled || props.loading,
          type: props.attrType,
          'aria-busy': props.loading || undefined,
        } as never,
        {
          default: () => [
            props.loading
              ? h(LoaderCircleIcon, { class: 'animate-spin', 'data-icon': 'inline-start', 'aria-hidden': 'true' })
              : renderSlot(slots, 'icon'),
            renderSlot(slots),
          ],
        },
      )
    }
  },
})

export const AppIcon = defineComponent({
  name: 'AppIcon',
  inheritAttrs: false,
  props: {
    component: Object as PropType<Component>,
    size: [String, Number],
    depth: Number,
  },
  setup(props, { attrs, slots }) {
    return () => {
      const child = props.component
        ? h(props.component, { size: props.size, 'aria-hidden': attrs['aria-label'] ? undefined : 'true' })
        : renderSlot(slots)
      return h(
        'span',
        {
          ...attrs,
          class: cn('n-icon inline-flex shrink-0 items-center justify-center leading-none', attrs.class as HTMLAttributes['class']),
          style: [props.size ? { fontSize: styleSize(props.size) } : undefined, attrs.style as CSSProperties],
        },
        child,
      )
    }
  },
})

export const AppStack = defineComponent({
  name: 'AppStack',
  inheritAttrs: false,
  props: {
    size: { type: [String, Number, Array] as PropType<string | number | [number, number]>, default: 'medium' },
    vertical: Boolean,
    wrap: { type: Boolean, default: true },
    align: String,
    justify: String,
  },
  setup(props, { attrs, slots }) {
    return () => {
      const numeric = typeof props.size === 'number' ? props.size : props.size === 'small' ? 8 : props.size === 'large' ? 16 : 12
      const gap = Array.isArray(props.size) ? `${props.size[0]}px ${props.size[1]}px` : `${numeric}px`
      return h(
        'div',
        {
          ...attrs,
          class: cn('n-space flex', props.vertical && 'flex-col', attrs.class as HTMLAttributes['class']),
          style: [
            {
              gap,
              flexWrap: props.wrap ? 'wrap' : 'nowrap',
              alignItems: props.align,
              justifyContent: props.justify === 'space-between' ? 'space-between' : props.justify,
            },
            attrs.style as CSSProperties,
          ],
        },
        renderSlot(slots),
      )
    }
  },
})

export const AppInput = defineComponent({
  name: 'AppInput',
  inheritAttrs: false,
  props: {
    value: [String, Number],
    type: { type: String, default: 'text' },
    disabled: Boolean,
    clearable: Boolean,
    placeholder: String,
    rows: Number,
    autosize: [Boolean, Object],
    maxlength: Number,
    showPasswordOn: String,
  },
  emits: ['update:value', 'clear'],
  setup(props, { attrs, emit, slots }) {
    const revealPassword = ref(false)
    const update = (value: unknown) => emit('update:value', value ?? '')
    return () => {
      const isTextarea = props.type === 'textarea'
      const control = isTextarea
        ? h(Textarea, {
            ...attrs,
            class: cn('n-input__textarea', attrs.class as HTMLAttributes['class']),
            modelValue: String(props.value ?? ''),
            rows: props.rows,
            maxlength: props.maxlength,
            disabled: props.disabled,
            placeholder: props.placeholder,
            'onUpdate:modelValue': update,
          } as never)
        : h(Input, {
            ...attrs,
            class: cn('n-input__input-el', attrs.class as HTMLAttributes['class']),
            modelValue: props.value ?? '',
            type: props.type === 'password' && revealPassword.value ? 'text' : props.type,
            maxlength: props.maxlength,
            disabled: props.disabled,
            placeholder: props.placeholder,
            'onUpdate:modelValue': update,
          } as never)
      return h('div', { class: cn('n-input relative flex min-w-0 items-center', isTextarea && 'items-start') }, [
        renderSlot(slots, 'prefix'),
        control,
        props.type === 'password' && props.showPasswordOn
          ? h(
              'button',
              {
                type: 'button',
                class: 'absolute right-2 text-xs text-muted-foreground',
                onMousedown: () => (revealPassword.value = true),
                onMouseup: () => (revealPassword.value = false),
                onMouseleave: () => (revealPassword.value = false),
              },
              '•••',
            )
          : null,
        props.clearable && props.value !== '' && props.value !== null && props.value !== undefined
          ? h(
              'button',
              {
                type: 'button',
                class: 'absolute right-2 inline-flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-muted',
                onClick: () => {
                  emit('update:value', '')
                  emit('clear')
                },
              },
              h(XIcon, { class: 'size-3.5' }),
            )
          : null,
        renderSlot(slots, 'suffix'),
      ])
    }
  },
})

export const AppNumberInput = defineComponent({
  name: 'AppNumberInput',
  inheritAttrs: false,
  props: {
    value: { type: Number as PropType<number | null>, default: null },
    min: Number,
    max: Number,
    step: Number,
    precision: Number,
    disabled: Boolean,
    clearable: Boolean,
    placeholder: String,
  },
  emits: ['update:value'],
  setup(props, { attrs, emit }) {
    return () =>
      h(Input, {
        ...attrs,
        class: cn('n-input-number n-input', attrs.class as HTMLAttributes['class']),
        type: 'number',
        modelValue: props.value ?? '',
        min: props.min,
        max: props.max,
        step: props.step ?? (props.precision === 0 ? 1 : 'any'),
        disabled: props.disabled,
        placeholder: props.placeholder,
        'onUpdate:modelValue': (value: string | number) => {
          if (value === '') emit('update:value', null)
          else {
            const parsed = Number(value)
            emit('update:value', Number.isFinite(parsed) ? parsed : null)
          }
        },
      } as never)
  },
})

export const AppSelect = defineComponent({
  name: 'AppSelect',
  inheritAttrs: false,
  props: {
    value: [String, Number, Boolean] as PropType<string | number | boolean | null>,
    options: { type: Array as PropType<SelectOption[]>, default: () => [] },
    placeholder: String,
    disabled: Boolean,
    clearable: Boolean,
    filterable: Boolean,
    loading: Boolean,
    consistentMenuWidth: { type: Boolean, default: true },
    size: String,
    icon: Object as PropType<Component>,
    grow: Boolean,
  },
  emits: ['update:value', 'clear'],
  setup(props, { attrs, emit }) {
    const selectedOption = computed(() => props.options.find((option) => Object.is(option.value, props.value)))

    function updateValue(option: SelectOption | string | number | boolean | null | undefined) {
      const value = typeof option === 'object' && option !== null && 'value' in option
        ? option.value
        : option
      emit('update:value', value ?? null)
    }

    function clearValue() {
      emit('update:value', null)
      emit('clear')
    }

    const renderClearButton = () => props.clearable && props.value !== null && props.value !== undefined
      ? h(Button, {
          variant: 'ghost',
          size: 'icon-xs',
          class: 'n-select-clear absolute right-7 top-1/2 z-10 -translate-y-1/2 text-muted-foreground hover:bg-muted hover:text-foreground',
          type: 'button',
          disabled: props.disabled || props.loading,
          'aria-label': 'Clear selection',
          onClick: (event: MouseEvent) => {
            event.stopPropagation()
            clearValue()
          },
        } as never, { default: () => h(XIcon) })
      : null

    return () => {
      const rootClass = cn(
        'n-select n-base-selection relative flex min-w-0 items-center',
        props.grow && 'flex-1 basis-36',
        attrs.class as HTMLAttributes['class'],
      )
      const triggerClass = cn(
        'n-base-selection-label min-w-0 flex-1 justify-between rounded-lg bg-background font-normal',
        props.clearable && props.value !== null && props.value !== undefined && 'pr-14',
        props.size === 'small' && 'h-8 text-[0.8rem]',
        props.size === 'tiny' && 'h-7 text-xs',
      )

      if (props.filterable) {
        return h('div', {
          ...attrs,
          class: rootClass,
          style: [attrs.style as CSSProperties, props.grow ? { flex: '1 0 8rem' } : undefined],
        }, [
          h(Combobox, {
            class: 'w-full min-w-0',
            modelValue: selectedOption.value,
            by: 'value',
            disabled: props.disabled || props.loading,
            'onUpdate:modelValue': updateValue,
          } as never, {
            default: () => [
              h(ComboboxAnchor, { asChild: true } as never, {
                default: () => h(ComboboxTrigger, {
                  asChild: true,
                  'aria-label': selectedOption.value?.label ?? props.placeholder ?? 'Select option',
                } as never, {
                  default: () => h(Button, {
                    variant: 'outline',
                    size: props.size === 'small' ? 'sm' : props.size === 'tiny' ? 'xs' : 'default',
                    class: triggerClass,
                    style: { width: '100%' },
                    disabled: props.disabled || props.loading,
                    role: 'combobox',
                  } as never, {
                    default: () => [
                      props.icon
                        ? h(props.icon, { class: 'shrink-0 text-muted-foreground', 'aria-hidden': true })
                        : null,
                      h('span', { class: cn('truncate', !selectedOption.value && 'text-muted-foreground') }, selectedOption.value?.label ?? props.placeholder ?? ''),
                      props.loading
                        ? h(LoaderCircleIcon, { class: 'ml-auto shrink-0 animate-spin opacity-60', 'data-icon': 'inline-end' })
                        : h(ChevronDownIcon, { class: 'ml-auto shrink-0 opacity-60', 'data-icon': 'inline-end' }),
                    ],
                  }),
                }),
              }),
              h(ComboboxList, { align: 'start' }, {
                default: () => [
                  h(ComboboxInput, { placeholder: props.placeholder ?? 'Search...' }),
                  h(ComboboxEmpty, {}, { default: () => localize('暂无选项', 'No options') }),
                  h(ComboboxGroup, {}, {
                    default: () => props.options.map((option) =>
                      h(ComboboxItem, { value: option, disabled: option.disabled } as never, {
                        default: () => [
                          option.label,
                          h(ComboboxItemIndicator, {}, { default: () => h(CheckIcon) }),
                        ],
                      }),
                    ),
                  }),
                ],
              }),
            ],
          }),
          renderClearButton(),
        ])
      }

      return h('div', {
        ...attrs,
        class: rootClass,
        style: [attrs.style as CSSProperties, props.grow ? { flex: '1 0 8rem' } : undefined],
      }, [
        h(Select, {
          modelValue: props.value ?? undefined,
          disabled: props.disabled || props.loading,
          'onUpdate:modelValue': updateValue,
        } as never, {
          default: () => [
            h(SelectTrigger, {
              class: triggerClass,
              style: { width: '100%' },
              size: props.size === 'small' || props.size === 'tiny' ? 'sm' : 'default',
              'aria-label': selectedOption.value?.label ?? props.placeholder ?? 'Select option',
            }, {
              default: () => [
                props.icon
                  ? h(props.icon, { class: 'shrink-0 text-muted-foreground', 'aria-hidden': true })
                  : null,
                h(SelectValue, { placeholder: props.placeholder }),
              ],
            }),
            h(SelectContent, { position: 'popper' }, {
              default: () => h(SelectGroup, {}, {
                default: () => props.options.map((option) =>
                  h(SelectItem, { value: option.value as never, disabled: option.disabled }, { default: () => option.label }),
                ),
              }),
            }),
          ],
        }),
        renderClearButton(),
      ])
    }
  },
})

export const AppSwitch = defineComponent({
  name: 'AppSwitch',
  inheritAttrs: false,
  props: {
    value: Boolean,
    disabled: Boolean,
    size: String,
  },
  emits: ['update:value'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'span',
        { class: cn('n-switch inline-flex items-center gap-2', attrs.class as HTMLAttributes['class']) },
        [
          h(Switch, {
            ...attrs,
            modelValue: props.value,
            disabled: props.disabled,
            'onUpdate:modelValue': (value: boolean) => emit('update:value', value),
          } as never),
          props.value ? renderSlot(slots, 'checked') : renderSlot(slots, 'unchecked'),
        ],
      )
  },
})

export const AppAlert = defineComponent({
  name: 'AppAlert',
  inheritAttrs: false,
  props: {
    type: { type: String, default: 'default' },
    title: String,
    bordered: { type: Boolean, default: true },
    showIcon: { type: Boolean, default: true },
    closable: Boolean,
  },
  emits: ['close'],
  setup(props, { attrs, slots, emit }) {
    return () =>
      h(
        Alert,
        {
          ...attrs,
          class: cn(
            'n-alert',
            !props.bordered && 'border-transparent',
            props.type === 'error' && 'border-destructive/35 bg-destructive/8 text-destructive',
            props.type === 'warning' && 'border-[var(--cpa-warning)]/35 bg-[var(--cpa-warning-weak)] text-[var(--cpa-warning)]',
            props.type === 'success' && 'border-[var(--cpa-success)]/35 bg-[var(--cpa-success-weak)] text-[var(--cpa-success)]',
            props.type === 'info' && 'border-[var(--cpa-accent-blue)]/35 bg-[var(--cpa-accent-blue-weak)] text-[var(--cpa-accent-blue)]',
            attrs.class as HTMLAttributes['class'],
          ),
        },
        {
          default: () => [
            props.title ? h(AlertTitle, {}, { default: () => props.title }) : null,
            h(AlertDescription, {}, { default: () => renderSlot(slots) }),
            props.closable
              ? h(
                  Button,
                  { variant: 'ghost', size: 'icon-xs', class: 'absolute right-2 top-2', onClick: () => emit('close') },
                  { default: () => h(XIcon) },
                )
              : null,
          ],
        },
      )
  },
})

export const AppBadge = defineComponent({
  name: 'AppBadge',
  inheritAttrs: false,
  props: {
    type: { type: String, default: 'default' },
    size: String,
    bordered: { type: Boolean, default: true },
    round: Boolean,
    closable: Boolean,
    disabled: Boolean,
  },
  emits: ['close'],
  setup(props, { attrs, slots, emit }) {
    return () =>
      h(
        Badge,
        {
          ...attrs,
          variant: props.type === 'error' ? 'destructive' : props.bordered ? 'outline' : 'secondary',
          class: cn(
            'n-tag gap-1',
            props.round && 'rounded-full',
            props.size === 'small' && 'px-1.5 py-0 text-[11px]',
            props.type === 'success' && 'border-[var(--cpa-success)]/30 bg-[var(--cpa-success-weak)] text-[var(--cpa-success)]',
            props.type === 'warning' && 'border-[var(--cpa-warning)]/30 bg-[var(--cpa-warning-weak)] text-[var(--cpa-warning)]',
            props.type === 'info' && 'border-[var(--cpa-accent-blue)]/30 bg-[var(--cpa-accent-blue-weak)] text-[var(--cpa-accent-blue)]',
            props.disabled && 'opacity-50',
            attrs.class as HTMLAttributes['class'],
          ),
        },
        {
          default: () => [
            h('span', { class: 'n-tag__content' }, renderSlot(slots)),
            props.closable
              ? h('button', { type: 'button', class: 'cursor-pointer disabled:cursor-not-allowed', onClick: () => emit('close'), disabled: props.disabled }, h(XIcon, { class: 'size-3' }))
              : null,
          ],
        },
      )
  },
})

export const AppSpinner = defineComponent({
  name: 'AppSpinner',
  inheritAttrs: false,
  props: {
    show: Boolean,
    size: [String, Number],
    description: String,
  },
  setup(props, { attrs, slots }) {
    return () => {
      const spinner = h('div', { class: 'n-spin-content flex flex-col items-center justify-center gap-2 text-primary' }, [
        h(LoaderCircleIcon, { class: cn('animate-spin', props.size === 'large' ? 'size-8' : 'size-5') }),
        props.description ? h('span', { class: 'text-xs text-muted-foreground' }, props.description) : null,
      ])
      if (!slots.default) return h('div', { ...attrs, class: cn('n-spin p-4', attrs.class as HTMLAttributes['class']) }, spinner)
      return h('div', { ...attrs, class: cn('n-spin-container relative min-w-0', attrs.class as HTMLAttributes['class']) }, [
        renderSlot(slots),
        props.show
          ? h('div', { class: 'n-spin-body absolute inset-0 z-20 grid place-items-center bg-background/65 backdrop-blur-[1px]' }, spinner)
          : null,
      ])
    }
  },
})

export const AppEmpty = defineComponent({
  name: 'AppEmpty',
  inheritAttrs: false,
  props: { description: String },
  setup(props, { attrs, slots }) {
    return () =>
      h(Empty, { ...attrs, class: cn('n-empty min-h-36 p-6', attrs.class as HTMLAttributes['class']) }, {
        default: () => [
          h(EmptyHeader, {}, {
            default: () => [
              h(EmptyMedia, { variant: 'icon' }, { default: () => h(InboxIcon) }),
              h(EmptyTitle, {}, { default: () => props.description ?? renderSlot(slots) }),
            ],
          }),
          slots.extra
            ? h(EmptyContent, {}, { default: () => renderSlot(slots, 'extra') })
            : null,
        ],
      })
  },
})

export const AppEllipsis = defineComponent({
  name: 'AppEllipsis',
  inheritAttrs: false,
  props: { tooltip: [Boolean, Object], lineClamp: Number },
  setup(_, { attrs, slots }) {
    return () =>
      h('span', { ...attrs, class: cn('n-ellipsis block truncate', attrs.class as HTMLAttributes['class']), title: String(slots.default?.()[0]?.children ?? '') }, renderSlot(slots))
  },
})

export const AppCard = defineComponent({
  name: 'AppCard',
  inheritAttrs: false,
  props: {
    title: String,
    bordered: { type: Boolean, default: true },
    contentStyle: [String, Object] as PropType<string | CSSProperties>,
    footerStyle: [String, Object] as PropType<string | CSSProperties>,
  },
  setup(props, { attrs, slots }) {
    return () =>
      h(Card, { ...attrs, class: cn('n-card', !props.bordered && 'border-transparent', attrs.class as HTMLAttributes['class']) }, {
        default: () => [
          props.title || slots.header
            ? h(CardHeader, { class: 'n-card-header border-b border-border/70 pb-4' }, { default: () => h(CardTitle, {}, { default: () => slots.header?.() ?? props.title }) })
            : null,
          h(CardContent, { class: 'n-card__content', style: props.contentStyle }, { default: () => renderSlot(slots) }),
          slots.footer ? h(CardFooter, { style: props.footerStyle }, { default: () => renderSlot(slots, 'footer') }) : null,
        ],
      })
  },
})

export const AppForm = defineComponent({
  name: 'AppForm',
  inheritAttrs: false,
  props: {
    model: Object,
    labelPlacement: String,
    size: String,
  },
  setup(_, { attrs, slots }) {
    return () => h('form', { ...attrs, class: cn('n-form grid gap-5', attrs.class as HTMLAttributes['class']) }, renderSlot(slots))
  },
})

export const AppFormItem = defineComponent({
  name: 'AppFormItem',
  inheritAttrs: false,
  props: {
    label: String,
    required: Boolean,
    path: String,
    showLabel: { type: Boolean, default: true },
    feedback: String,
  },
  setup(props, { attrs, slots }) {
    return () =>
      h(Field, { ...attrs, class: cn('n-form-item gap-2', attrs.class as HTMLAttributes['class']) }, {
        default: () => [
          props.showLabel && (props.label || slots.label)
            ? h(FieldLabel, { class: 'n-form-item-label' }, {
                default: () => [
                  slots.label?.() ?? props.label,
                  props.required ? h('span', { class: 'ml-0.5 text-destructive' }, '*') : null,
                ],
              })
            : null,
          h('div', { class: 'n-form-item-blank min-w-0' }, renderSlot(slots)),
          props.feedback
            ? h(FieldDescription, { class: 'n-form-item-feedback-wrapper' }, { default: () => props.feedback })
            : null,
        ],
      })
  },
})

export const AppModal = defineComponent({
  name: 'AppModal',
  inheritAttrs: false,
  props: {
    show: Boolean,
    title: String,
    preset: String,
    maskClosable: { type: Boolean, default: true },
    closable: { type: Boolean, default: true },
    bordered: Boolean,
    contentStyle: [String, Object] as PropType<string | CSSProperties>,
    footerStyle: [String, Object] as PropType<string | CSSProperties>,
  },
  emits: ['update:show', 'mask-click', 'esc'],
  setup(props, { attrs, slots, emit }) {
    return () =>
      h(Dialog, { open: props.show, 'onUpdate:open': (open: boolean) => emit('update:show', open) } as never, {
        default: () =>
          h(
            DialogContent,
            {
              ...attrs,
              class: cn('n-modal max-h-[calc(100dvh-2rem)] gap-6 overflow-y-auto sm:max-w-none', attrs.class as HTMLAttributes['class']),
              showCloseButton: props.closable,
              onPointerDownOutside: (event: Event) => {
                if (!props.maskClosable) event.preventDefault()
                else emit('mask-click')
              },
              onEscapeKeyDown: (event: Event) => {
                if (!props.maskClosable) event.preventDefault()
                else emit('esc')
              },
            } as never,
            {
              default: () => [
                props.title || slots.header
                  ? h(DialogHeader, {}, { default: () => h(DialogTitle, {}, { default: () => slots.header?.() ?? props.title }) })
                  : null,
                h('div', { class: 'n-modal-body min-w-0', style: props.contentStyle }, renderSlot(slots)),
                slots.action || slots.footer
                  ? h(DialogFooter, { style: props.footerStyle }, { default: () => slots.action?.() ?? slots.footer?.() })
                  : null,
              ],
            },
          ),
      })
  },
})

interface DrawerContext {
  placement: ComputedRef<'left' | 'right' | 'top' | 'bottom'>
  width: ComputedRef<string | number | undefined>
  height: ComputedRef<string | number | undefined>
}

const drawerContextKey: InjectionKey<DrawerContext> = Symbol('drawer-context')

export const AppDrawer = defineComponent({
  name: 'AppDrawer',
  inheritAttrs: false,
  props: {
    show: Boolean,
    placement: { type: String as PropType<'left' | 'right' | 'top' | 'bottom'>, default: 'right' },
    width: [String, Number],
    height: [String, Number],
    maskClosable: { type: Boolean, default: true },
  },
  emits: ['update:show'],
  setup(props, { slots, emit }) {
    provide(drawerContextKey, {
      placement: computed(() => props.placement),
      width: computed(() => props.width),
      height: computed(() => props.height),
    })
    return () =>
      h(Sheet, { open: props.show, 'onUpdate:open': (open: boolean) => emit('update:show', open) } as never, {
        default: () => h('div', { class: 'n-drawer-host' }, renderSlot(slots)),
      })
  },
})

export const AppDrawerContent = defineComponent({
  name: 'AppDrawerContent',
  inheritAttrs: false,
  props: {
    title: String,
    closable: { type: Boolean, default: true },
    bodyContentStyle: [String, Object] as PropType<string | CSSProperties>,
    nativeScrollbar: Boolean,
  },
  setup(props, { attrs, slots }) {
    const drawer = inject(drawerContextKey, null)
    return () =>
      h(
        SheetContent,
        {
          ...attrs,
          class: cn('n-drawer n-drawer-content w-[var(--drawer-width,420px)] max-w-[100vw] gap-0 sm:max-w-none', attrs.class as HTMLAttributes['class']),
          side: drawer?.placement.value ?? 'right',
          showCloseButton: props.closable,
          style: [
            drawer?.width.value
              ? {
                  '--drawer-width': styleSize(drawer.width.value),
                  width: styleSize(drawer.width.value),
                  maxWidth: '100vw',
                } as CSSProperties
              : undefined,
            drawer?.height.value ? { height: styleSize(drawer.height.value) } as CSSProperties : undefined,
            attrs.style as CSSProperties,
          ],
        } as never,
        {
          default: () => [
            props.title || slots.header
              ? h(SheetHeader, { class: 'n-drawer-header border-b px-6 py-5' }, { default: () => h(SheetTitle, { class: 'text-lg font-semibold' }, { default: () => slots.header?.() ?? props.title }) })
              : null,
            h('div', { class: 'n-drawer-body min-h-0 flex-1 overflow-auto p-6', style: props.bodyContentStyle }, renderSlot(slots)),
            slots.footer ? h('div', { class: 'n-drawer-footer border-t bg-muted/30 p-5' }, renderSlot(slots, 'footer')) : null,
          ],
        },
      )
  },
})

export const AppTooltip = defineComponent({
  name: 'AppTooltip',
  inheritAttrs: false,
  props: {
    trigger: String,
    placement: String,
    delay: Number,
  },
  setup(props, { attrs, slots }) {
    return () =>
      h(TooltipProvider, { delayDuration: props.delay ?? 250 }, {
        default: () =>
          h(Tooltip, {}, {
            default: () => [
              h(TooltipTrigger, { asChild: true } as never, { default: () => renderSlot(slots, 'trigger') }),
              h(TooltipContent, { ...attrs, side: props.placement as never }, { default: () => renderSlot(slots) }),
            ],
          }),
      })
  },
})

export const AppConfirm = defineComponent({
  name: 'AppConfirm',
  inheritAttrs: false,
  props: {
    positiveText: { type: String, default: '确认' },
    negativeText: { type: String, default: '取消' },
    showIcon: Boolean,
  },
  emits: ['positive-click', 'negative-click'],
  setup(props, { slots, emit }) {
    return () =>
      h(AlertDialog, {}, {
        default: () => [
          h(AlertDialogTrigger, { asChild: true } as never, { default: () => renderSlot(slots, 'trigger') }),
          h(AlertDialogContent, {}, {
            default: () => [
              h(AlertDialogHeader, {}, {
                default: () => [
                  h(AlertDialogTitle, {}, { default: () => props.positiveText }),
                  h(AlertDialogDescription, {}, { default: () => renderSlot(slots) }),
                ],
              }),
              h(AlertDialogFooter, {}, {
                default: () => [
                  h(AlertDialogCancel, { onClick: () => emit('negative-click') }, { default: () => props.negativeText }),
                  h(AlertDialogAction, { onClick: () => emit('positive-click') }, { default: () => props.positiveText }),
                ],
              }),
            ],
          }),
        ],
      })
  },
})

export const AppDropdown = defineComponent({
  name: 'AppDropdown',
  inheritAttrs: false,
  props: {
    options: { type: Array as PropType<MenuOption[]>, default: () => [] },
    trigger: String,
    placement: String,
    disabled: Boolean,
  },
  emits: ['select'],
  setup(props, { slots, emit }) {
    const renderItem = (option: MenuOption) =>
      h(
        DropdownMenuItem,
        { disabled: option.disabled, onSelect: () => emit('select', option.key) },
        { default: () => [option.icon?.(), typeof option.label === 'function' ? option.label() : option.label] },
      )
    const renderOptions = (options: MenuOption[]) => {
      const nodes: VNodeChild[] = []
      let group: MenuOption[] = []
      const flushGroup = () => {
        if (group.length === 0) return
        const current = group
        group = []
        nodes.push(h(DropdownMenuGroup, {}, { default: () => current.map(renderItem) }))
      }
      for (const option of options) {
        if (option.type === 'divider') {
          flushGroup()
          nodes.push(h(DropdownMenuSeparator))
          continue
        }
        group.push(option)
      }
      flushGroup()
      return nodes
    }
    return () =>
      h(DropdownMenu, {}, {
        default: () => [
          h(DropdownMenuTrigger, { asChild: true } as never, { default: () => renderSlot(slots) }),
          h(DropdownMenuContent, { align: 'start' }, { default: () => renderOptions(props.options) }),
        ],
      })
  },
})

export const AppRadioGroup = defineComponent({
  name: 'AppRadioGroup',
  inheritAttrs: false,
  props: { value: [String, Number] },
  emits: ['update:value'],
  setup(props, { attrs, slots, emit }) {
    return () =>
      h(RadioGroup, {
        ...attrs,
        class: cn('n-radio-group flex w-fit flex-wrap gap-1 rounded-lg border border-border/60 bg-muted/60 p-1', attrs.class as HTMLAttributes['class']),
        modelValue: String(props.value ?? ''),
        'onUpdate:modelValue': (value: string) => emit('update:value', value),
      } as never, { default: () => renderSlot(slots) })
  },
})

export const AppRadioButton = defineComponent({
  name: 'AppRadioButton',
  inheritAttrs: false,
  props: { value: [String, Number], disabled: Boolean },
  setup(props, { attrs, slots }) {
    const id = `radio-${Math.random().toString(36).slice(2)}`
    return () =>
      h('label', { class: cn('n-radio-button inline-flex min-h-8 cursor-pointer items-center justify-center gap-1.5 rounded-md border border-transparent px-3 py-1.5 text-sm font-medium text-muted-foreground transition-[color,background-color,box-shadow] hover:bg-background/60 hover:text-foreground has-[[data-state=checked]]:bg-background has-[[data-state=checked]]:text-primary has-[[data-state=checked]]:shadow-xs', props.disabled && 'cursor-not-allowed opacity-50', attrs.class as HTMLAttributes['class']) }, [
        h(RadioGroupItem, { id, value: String(props.value), disabled: props.disabled, class: 'sr-only' } as never),
        renderSlot(slots),
      ])
  },
})

export const AppDescriptions = defineComponent({
  name: 'AppDescriptions',
  inheritAttrs: false,
  props: { column: { type: Number, default: 3 }, bordered: Boolean, labelPlacement: String, size: String },
  setup(props, { attrs, slots }) {
    return () =>
      h('dl', {
        ...attrs,
        class: cn('n-descriptions grid overflow-hidden rounded-lg', props.bordered && 'border border-border', attrs.class as HTMLAttributes['class']),
        style: [{ gridTemplateColumns: `repeat(${props.column}, minmax(0, 1fr))` }, attrs.style as CSSProperties],
      }, renderSlot(slots))
  },
})

export const AppDescriptionsItem = defineComponent({
  name: 'AppDescriptionsItem',
  inheritAttrs: false,
  props: { label: String, span: Number },
  setup(props, { attrs, slots }) {
    return () =>
      h('div', { ...attrs, class: cn('n-descriptions-item min-w-0 border-b border-r border-border p-3 last:border-b-0', attrs.class as HTMLAttributes['class']), style: [{ gridColumn: props.span ? `span ${props.span}` : undefined }, attrs.style as CSSProperties] }, [
        h('dt', { class: 'mb-1 text-xs font-medium text-muted-foreground' }, slots.label?.() ?? props.label),
        h('dd', { class: 'm-0 min-w-0 text-sm text-foreground' }, renderSlot(slots)),
      ])
  },
})

export const AppPagination = defineComponent({
  name: 'AppPagination',
  inheritAttrs: false,
  props: {
    page: { type: Number, default: 1 },
    pageSize: { type: Number, default: 10 },
    itemCount: { type: Number, default: 0 },
    pageSizes: { type: Array as PropType<number[]>, default: () => [10, 20, 50, 100] },
    showSizePicker: Boolean,
    size: String,
  },
  emits: ['update:page', 'update:page-size'],
  setup(props, { attrs, emit }) {
    const pageCount = computed(() => Math.max(1, Math.ceil(props.itemCount / Math.max(1, props.pageSize))))
    const setPage = (page: number) => emit('update:page', Math.min(pageCount.value, Math.max(1, page)))
    return () =>
      h('nav', { ...attrs, class: cn('n-pagination flex flex-wrap items-center gap-1.5', attrs.class as HTMLAttributes['class']), 'aria-label': 'Pagination' }, [
        h(AppButton, { size: 'small', secondary: true, circle: true, disabled: props.page <= 1, 'aria-label': 'First page', onClick: () => setPage(1) }, { default: () => h(ChevronsLeftIcon) }),
        h(AppButton, { size: 'small', secondary: true, circle: true, disabled: props.page <= 1, 'aria-label': 'Previous page', onClick: () => setPage(props.page - 1) }, { default: () => h(ChevronLeftIcon) }),
        h('span', { class: 'inline-flex h-8 min-w-16 items-center justify-center rounded-md border border-border bg-background px-2 text-sm font-medium text-foreground shadow-xs' }, `${props.page} / ${pageCount.value}`),
        h(AppButton, { size: 'small', secondary: true, circle: true, disabled: props.page >= pageCount.value, 'aria-label': 'Next page', onClick: () => setPage(props.page + 1) }, { default: () => h(ChevronRightIcon) }),
        h(AppButton, { size: 'small', secondary: true, circle: true, disabled: props.page >= pageCount.value, 'aria-label': 'Last page', onClick: () => setPage(pageCount.value) }, { default: () => h(ChevronsRightIcon) }),
        props.showSizePicker
          ? h(AppSelect, {
              class: 'ml-1 w-24',
              size: 'small',
              value: props.pageSize,
              options: props.pageSizes.map((size) => ({ label: String(size), value: size })),
              'onUpdate:value': (value: number) => emit('update:page-size', value),
            })
          : null,
      ])
  },
})

function formatDateTimeLocal(timestamp: number) {
  const date = new Date(timestamp)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function parseDateTimeLocal(value: string) {
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? timestamp : null
}

function timestampToCalendarDate(timestamp: number) {
  const date = new Date(timestamp)
  return new CalendarDate(date.getFullYear(), date.getMonth() + 1, date.getDate())
}

function dateValueToInputDate(value: { year: number, month: number, day: number }) {
  return `${String(value.year).padStart(4, '0')}-${String(value.month).padStart(2, '0')}-${String(value.day).padStart(2, '0')}`
}

function formatRangePart(timestamp: number) {
  const formatter = new Intl.DateTimeFormat(localize('zh-CN', 'en-US'), {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  })
  return formatter.format(new Date(timestamp))
}

export const AppDateTimeRange = defineComponent({
  name: 'AppDateTimeRange',
  inheritAttrs: false,
  props: {
    value: Array as unknown as PropType<[number, number] | null>,
    type: String,
    clearable: Boolean,
    disabled: Boolean,
    size: String,
  },
  emits: ['update:value', 'clear'],
  setup(props, { attrs, emit }) {
    const open = ref(false)
    const draftStart = ref('')
    const draftEnd = ref('')
    const calendarRange = ref<DateRange | null>(null)

    const syncDraft = (value: [number, number] | null | undefined) => {
      draftStart.value = value ? formatDateTimeLocal(value[0]) : ''
      draftEnd.value = value ? formatDateTimeLocal(value[1]) : ''
      calendarRange.value = value
        ? { start: timestampToCalendarDate(value[0]), end: timestampToCalendarDate(value[1]) }
        : null
    }

    watch(
      () => props.value,
      syncDraft,
      { immediate: true, deep: true },
    )

    const canApply = computed(() => {
      const start = parseDateTimeLocal(draftStart.value)
      const end = parseDateTimeLocal(draftEnd.value)
      return start !== null && end !== null && start <= end
    })

    const updateCalendarRange = (value: DateRange) => {
      calendarRange.value = value
      if (value.start) {
        const time = draftStart.value.slice(11, 16) || '00:00'
        draftStart.value = `${dateValueToInputDate(value.start)}T${time}`
      }
      if (value.end) {
        const time = draftEnd.value.slice(11, 16) || '23:59'
        draftEnd.value = `${dateValueToInputDate(value.end)}T${time}`
      }
    }

    const updateTime = (target: 'start' | 'end', time: string) => {
      const current = target === 'start' ? draftStart.value : draftEnd.value
      const calendarValue = target === 'start' ? calendarRange.value?.start : calendarRange.value?.end
      const date = current.slice(0, 10) || (calendarValue ? dateValueToInputDate(calendarValue) : '')
      if (!date) return
      if (target === 'start') draftStart.value = `${date}T${time}`
      else draftEnd.value = `${date}T${time}`
    }

    const apply = () => {
      const start = parseDateTimeLocal(draftStart.value)
      const end = parseDateTimeLocal(draftEnd.value)
      if (start === null || end === null || start > end) return
      emit('update:value', [start, end])
      open.value = false
    }

    const clear = () => {
      syncDraft(null)
      emit('update:value', null)
      emit('clear')
      open.value = false
    }

    const cancel = () => {
      syncDraft(props.value)
      open.value = false
    }

    const renderRangeValue = () => {
      if (!props.value) {
        return h(
          'span',
          { class: 'col-span-3 truncate text-left text-muted-foreground' },
          localize('选择日期与时间', 'Select date and time'),
        )
      }
      const start = formatRangePart(props.value[0])
      const end = formatRangePart(props.value[1])
      return [
        h('span', { class: 'n-date-range-start truncate text-left', title: start }, start),
        h('span', { class: 'n-date-range-separator text-center text-muted-foreground' }, '—'),
        h('span', { class: 'n-date-range-end truncate text-right', title: end }, end),
      ]
    }

    return () =>
      h('div', { ...attrs, class: cn('n-date-picker flex min-w-0 items-center gap-1 bg-transparent shadow-none', attrs.class as HTMLAttributes['class']) }, [
        h(Popover, {
          open: open.value,
          'onUpdate:open': (value: boolean) => {
            if (value) syncDraft(props.value)
            open.value = value
          },
        } as never, {
          default: () => [
            h(PopoverTrigger, { asChild: true } as never, {
              default: () => h(Button, {
                variant: 'outline',
                size: props.size === 'small' ? 'sm' : 'default',
                disabled: props.disabled,
                class: 'n-date-range-trigger min-w-0 flex-1 justify-start',
              } as never, {
                default: () => [
                  h(CalendarDaysIcon, { 'data-icon': 'inline-start' }),
                  h(
                    'span',
                    { class: 'n-date-range-value grid min-w-0 flex-1 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 tabular-nums' },
                    renderRangeValue(),
                  ),
                  h(ChevronDownIcon, { class: 'ml-auto text-muted-foreground', 'data-icon': 'inline-end' }),
                ],
              }),
            }),
            h(PopoverContent, { align: 'start', class: 'max-h-[calc(100dvh-2rem)] w-auto max-w-[calc(100vw-2rem)] gap-0 overflow-y-auto p-0' }, {
              default: () => [
                h(RangeCalendar, {
                  modelValue: calendarRange.value,
                  numberOfMonths: 2,
                  weekStartsOn: 1,
                  initialFocus: true,
                  locale: localize('zh-CN', 'en-US'),
                  'onUpdate:modelValue': updateCalendarRange,
                } as never),
                h(Separator),
                h('div', { class: 'grid gap-3 p-3 sm:grid-cols-2' }, [
                  h(Field, {}, {
                    default: () => [
                      h(FieldLabel, {}, { default: () => localize('开始时间', 'Start time') }),
                      h(Input, {
                        type: 'time',
                        modelValue: draftStart.value.slice(11, 16),
                        disabled: !calendarRange.value?.start,
                        'onUpdate:modelValue': (value: string | number) => updateTime('start', String(value)),
                      } as never),
                    ],
                  }),
                  h(Field, {}, {
                    default: () => [
                      h(FieldLabel, {}, { default: () => localize('结束时间', 'End time') }),
                      h(Input, {
                        type: 'time',
                        modelValue: draftEnd.value.slice(11, 16),
                        disabled: !calendarRange.value?.end,
                        'onUpdate:modelValue': (value: string | number) => updateTime('end', String(value)),
                      } as never),
                    ],
                  }),
                ]),
                h('div', { class: 'flex items-center justify-between gap-2 px-3 pb-3' }, [
                  props.clearable
                    ? h(Button, { type: 'button', variant: 'ghost', size: 'sm', disabled: !props.value, onClick: clear } as never, { default: () => localize('清除', 'Clear') })
                    : h('span'),
                  h('div', { class: 'flex items-center gap-2' }, [
                    h(Button, { type: 'button', variant: 'outline', size: 'sm', onClick: cancel } as never, { default: () => localize('取消', 'Cancel') }),
                    h(Button, { type: 'button', size: 'sm', disabled: !canApply.value, onClick: apply } as never, { default: () => localize('应用', 'Apply') }),
                  ]),
                ]),
              ],
            }),
          ],
        }),
      ])
  },
})

type DialogOptions = {
  title?: string
  content?: string | (() => VNodeChild)
  positiveText?: string
  negativeText?: string
  onPositiveClick?: () => void | Promise<void>
  onNegativeClick?: () => void
}

const imperativeDialog = reactive({ open: false, options: null as DialogOptions | null })

function openImperativeDialog(options: DialogOptions) {
  imperativeDialog.options = options
  imperativeDialog.open = true
  return { destroy: () => (imperativeDialog.open = false) }
}

export function useDialog() {
  return {
    warning: openImperativeDialog,
    error: openImperativeDialog,
    info: openImperativeDialog,
    success: openImperativeDialog,
  }
}

export function useMessage() {
  return {
    success: (message: string, options?: object) => toast.success(message, options),
    error: (message: string, options?: object) => toast.error(message, options),
    warning: (message: string, options?: object) => toast.warning(message, options),
    info: (message: string, options?: object) => toast.info(message, options),
    loading: (message: string, options?: object) => toast.loading(message, options),
    destroyAll: () => toast.dismiss(),
  }
}

export const AppMessageProvider = defineComponent({
  name: 'AppMessageProvider',
  setup(_, { slots }) {
    return () => h('div', { class: 'contents' }, [renderSlot(slots), h(Toaster, { richColors: true, position: 'top-center' })])
  },
})

export const AppDialogProvider = defineComponent({
  name: 'AppDialogProvider',
  setup(_, { slots }) {
    const confirm = async () => {
      const options = imperativeDialog.options
      if (!options) return
      await options.onPositiveClick?.()
      imperativeDialog.open = false
    }
    return () =>
      h('div', { class: 'contents' }, [
        renderSlot(slots),
        h(AlertDialog, { open: imperativeDialog.open, 'onUpdate:open': (open: boolean) => (imperativeDialog.open = open) } as never, {
          default: () =>
            h(AlertDialogContent, {}, {
              default: () => [
                h(AlertDialogHeader, {}, {
                  default: () => [
                    h(AlertDialogTitle, {}, { default: () => imperativeDialog.options?.title ?? '确认' }),
                    h(AlertDialogDescription, {}, {
                      default: () => typeof imperativeDialog.options?.content === 'function' ? imperativeDialog.options.content() : imperativeDialog.options?.content,
                    }),
                  ],
                }),
                h(AlertDialogFooter, {}, {
                  default: () => [
                    h(AlertDialogCancel, { onClick: () => imperativeDialog.options?.onNegativeClick?.() }, { default: () => imperativeDialog.options?.negativeText ?? '取消' }),
                    h(AlertDialogAction, { onClick: confirm }, { default: () => imperativeDialog.options?.positiveText ?? '确认' }),
                  ],
                }),
              ],
            }),
        }),
      ])
  },
})

export const AppDataTable = defineComponent({
  name: 'AppDataTable',
  inheritAttrs: false,
  props: {
    columns: { type: Array as PropType<DataTableColumns<any>>, default: () => [] },
    data: { type: Array as PropType<any[]>, default: () => [] },
    loading: Boolean,
    pagination: { type: [Boolean, Object] as PropType<false | Record<string, unknown>>, default: undefined },
    rowKey: [String, Function] as PropType<string | ((row: any) => DataTableRowKey)>,
    checkedRowKeys: { type: Array as PropType<DataTableRowKey[]>, default: () => [] },
    scrollX: [String, Number],
    maxHeight: [String, Number],
    minHeight: [String, Number],
    minRowHeight: Number,
    flexHeight: Boolean,
    tableLayout: String,
    virtualScroll: Boolean,
    remote: Boolean,
    bordered: { type: Boolean, default: true },
    size: String,
    rowProps: Function as PropType<(row: any, index: number) => Record<string, unknown>>,
  },
  emits: ['update:checked-row-keys'],
  setup(props, { attrs, slots, emit }) {
    const localPage = ref(1)
    const pageSize = computed(() => {
      if (typeof props.pagination !== 'object') return props.data.length || 1
      return Number(props.pagination.pageSize ?? 10)
    })
    const currentPage = computed(() => {
      if (typeof props.pagination !== 'object') return 1
      return Number(props.pagination.page ?? localPage.value)
    })
    watch(
      () => props.data.length,
      () => {
        const count = Math.max(1, Math.ceil(props.data.length / pageSize.value))
        if (localPage.value > count) localPage.value = count
      },
    )
    const rows = computed(() => {
      if (typeof props.pagination !== 'object' || props.remote) return props.data
      const start = (currentPage.value - 1) * pageSize.value
      return props.data.slice(start, start + pageSize.value)
    })
    const getRowKey = (row: any, index: number): DataTableRowKey => {
      if (typeof props.rowKey === 'function') return props.rowKey(row)
      if (typeof props.rowKey === 'string' && row && typeof row === 'object') {
        const value = (row as Record<string, unknown>)[props.rowKey]
        if (typeof value === 'string' || typeof value === 'number') return value
      }
      if (row && typeof row === 'object') {
        const record = row as Record<string, unknown>
        const value = record.id ?? record.key
        if (typeof value === 'string' || typeof value === 'number') return value
      }
      return index
    }
    const selectionColumn = computed(() => props.columns.find((column) => column.type === 'selection'))
    const selectableRows = computed(() =>
      rows.value
        .map((row, index) => ({ row, key: getRowKey(row, index) }))
        .filter(({ row }) => !selectionColumn.value?.disabled?.(row)),
    )
    const checkedSet = computed(() => new Set(props.checkedRowKeys))
    const allChecked = computed(() => selectableRows.value.length > 0 && selectableRows.value.every(({ key }) => checkedSet.value.has(key)))
    const someChecked = computed(() => selectableRows.value.some(({ key }) => checkedSet.value.has(key)) && !allChecked.value)
    const updateChecked = (key: DataTableRowKey, checked: boolean) => {
      const next = new Set(props.checkedRowKeys)
      if (checked) next.add(key)
      else next.delete(key)
      emit('update:checked-row-keys', [...next])
    }
    const updateAll = (checked: boolean) => {
      const next = new Set(props.checkedRowKeys)
      selectableRows.value.forEach(({ key }) => checked ? next.add(key) : next.delete(key))
      emit('update:checked-row-keys', [...next])
    }
    const rightOffset = (columnIndex: number) => {
      let offset = 0
      for (let index = columnIndex + 1; index < props.columns.length; index += 1) {
        const column = props.columns[index]
        if (column?.fixed === 'right') offset += Number(column.width ?? 0)
      }
      return offset
    }
    const cellStyle = (column: DataTableColumn<any>, index: number): CSSProperties => ({
      width: styleSize(column.width),
      minWidth: styleSize(column.minWidth ?? column.width),
      maxWidth: styleSize(column.maxWidth ?? column.width),
      textAlign: column.align,
      position: column.fixed ? 'sticky' : undefined,
      left: column.fixed === 'left' ? '0' : undefined,
      right: column.fixed === 'right' ? `${rightOffset(index)}px` : undefined,
      zIndex: column.fixed ? 2 : undefined,
      background: column.fixed ? 'var(--cpa-surface)' : undefined,
    })
    const renderHeader = (column: DataTableColumn<any>) => {
      if (column.type === 'selection') {
        return h(Checkbox, {
          modelValue: allChecked.value ? true : someChecked.value ? 'indeterminate' : false,
          'aria-label': 'Select all rows',
          'onUpdate:modelValue': (value: boolean | 'indeterminate') => updateAll(value === true),
        } as never)
      }
      return typeof column.title === 'function' ? column.title() : column.title ?? String(column.key ?? '')
    }
    const renderCell = (column: DataTableColumn<any>, row: any, rowIndex: number) => {
      if (column.type === 'selection') {
        const key = getRowKey(row, rowIndex)
        return h(Checkbox, {
          modelValue: checkedSet.value.has(key),
          disabled: column.disabled?.(row),
          'aria-label': `Select row ${rowIndex + 1}`,
          'onUpdate:modelValue': (value: boolean | 'indeterminate') => updateChecked(key, value === true),
        } as never)
      }
      const content = column.render
        ? column.render(row, rowIndex)
        : row && typeof row === 'object' && column.key !== undefined
          ? String((row as Record<string, unknown>)[column.key] ?? '')
          : ''
      return column.ellipsis
        ? h('div', { class: 'truncate', title: typeof content === 'string' ? content : undefined }, content as never)
        : content
    }
    const updatePage = (page: number) => {
      localPage.value = page
      if (typeof props.pagination === 'object') {
        const callback = props.pagination.onUpdatePage
        if (typeof callback === 'function') callback(page)
      }
    }
    return () => {
      const wrapperStyle: CSSProperties = {
        height: props.flexHeight ? '100%' : undefined,
        maxHeight: props.flexHeight ? undefined : styleSize(props.maxHeight),
        minHeight: styleSize(props.minHeight),
        overflow: 'auto',
      }
      const tableStyle: CSSProperties = {
        minWidth: styleSize(props.scrollX),
        tableLayout: props.tableLayout as CSSProperties['tableLayout'],
      }
      return h('div', { ...attrs, class: cn('n-data-table relative min-w-0', props.flexHeight && 'flex h-full min-h-0 flex-col', !props.bordered && 'is-borderless', attrs.class as HTMLAttributes['class']) }, [
        h('div', { class: cn('n-data-table-wrapper relative overflow-hidden rounded-xl border border-border/80 bg-card shadow-xs', props.flexHeight && 'flex min-h-0 flex-1 flex-col') }, [
          h('div', { class: cn('n-scrollbar-container', props.flexHeight && 'h-full min-h-0 flex-1'), style: wrapperStyle }, [
            h(Table, { class: 'n-data-table-base-table w-full border-collapse text-sm', style: tableStyle }, {
              default: () => [
                h(TableHeader, { class: 'n-data-table-thead sticky top-0 z-[3]' }, {
                  default: () => h(TableRow, { class: 'n-data-table-tr' }, {
                    default: () => props.columns.map((column, index) =>
                      h(TableHead, {
                        key: String(column.key ?? index),
                        class: 'n-data-table-th h-11 bg-muted/55 px-3 text-left align-middle text-xs font-semibold tracking-[0.01em] text-foreground backdrop-blur-sm',
                        style: {
                          ...cellStyle(column, index),
                          background: column.fixed ? 'var(--cpa-surface-muted)' : undefined,
                          borderTopLeftRadius: index === 0 ? 'calc(var(--cpa-radius) - 1px)' : undefined,
                          borderTopRightRadius: index === props.columns.length - 1 ? 'calc(var(--cpa-radius) - 1px)' : undefined,
                        },
                      }, { default: () => renderHeader(column) as never }),
                    ),
                  }),
                }),
                h(TableBody, { class: 'n-data-table-base-table-body' }, {
                  default: () => props.loading && rows.value.length === 0
                    ? Array.from({ length: 5 }, (_, rowIndex) =>
                        h(TableRow, { key: `skeleton-${rowIndex}` }, {
                          default: () => props.columns.map((column, columnIndex) =>
                            h(TableCell, { class: 'n-data-table-td px-3 py-2.5', style: cellStyle(column, columnIndex) }, { default: () => h(Skeleton, { class: 'h-4 w-full' }) }),
                          ),
                        }),
                      )
                    : rows.value.length === 0
                      ? h(TableRow, {}, {
                          default: () => h(TableCell, { class: 'n-data-table-td p-8 text-center text-muted-foreground', colspan: props.columns.length }, { default: () => slots.empty?.() ?? localize('暂无数据', 'No data') }),
                        })
                      : rows.value.map((row, rowIndex) => {
                          const rowAttrs = props.rowProps?.(row, rowIndex) ?? {}
                          return h(TableRow, {
                            ...rowAttrs,
                            key: getRowKey(row, rowIndex),
                            class: cn('n-data-table-tr transition-colors hover:bg-accent/45', rowAttrs.class as HTMLAttributes['class']),
                          }, {
                            default: () => props.columns.map((column, columnIndex) =>
                              h(TableCell, {
                                key: String(column.key ?? columnIndex),
                                class: 'n-data-table-td px-3 py-3 align-middle text-foreground',
                                style: cellStyle(column, columnIndex),
                              }, { default: () => renderCell(column, row, rowIndex) as never }),
                            ),
                          })
                        }),
                }),
              ],
            }),
          ]),
          props.loading && rows.value.length > 0
            ? h('div', { class: 'absolute inset-0 z-20 grid place-items-center bg-background/55' }, h(LoaderCircleIcon, { class: 'size-6 animate-spin text-primary' }))
            : null,
        ]),
        typeof props.pagination === 'object' && props.data.length > pageSize.value
          ? h(AppPagination, {
              class: 'shrink-0 justify-end pt-3',
              page: currentPage.value,
              pageSize: pageSize.value,
              itemCount: props.data.length,
              'onUpdate:page': updatePage,
            })
          : null,
      ])
    }
  },
})
