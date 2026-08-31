<script setup lang="ts">
import type { AvatarImageEmits, AvatarImageProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { AvatarImage, useForwardPropsEmits } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<AvatarImageProps & {
  class?: HTMLAttributes['class']
}>()
const emits = defineEmits<AvatarImageEmits>()
const delegatedProps = reactiveOmit(props, 'class')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <AvatarImage
    data-slot="avatar-image"
    v-bind="forwarded"
    :class="cn('size-full object-cover', props.class)"
  />
</template>
