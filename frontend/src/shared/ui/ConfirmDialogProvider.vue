<script setup lang="ts">
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useConfirmDialogState } from '@/shared/ui/confirm-dialog'

const state = useConfirmDialogState()

async function confirm() {
  await state.options?.onPositiveClick?.()
  state.open = false
}

function cancel() {
  state.options?.onNegativeClick?.()
}
</script>

<template>
  <slot />
  <AlertDialog v-model:open="state.open">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ state.options?.title ?? '确认' }}</AlertDialogTitle>
        <AlertDialogDescription>{{ state.options?.content }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel @click="cancel">
          {{ state.options?.negativeText ?? '取消' }}
        </AlertDialogCancel>
        <AlertDialogAction @click="confirm">
          {{ state.options?.positiveText ?? '确认' }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
