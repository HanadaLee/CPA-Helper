import { reactive } from 'vue'

export interface ConfirmDialogOptions {
  title?: string
  content?: string
  positiveText?: string
  negativeText?: string
  onPositiveClick?: () => void | Promise<void>
  onNegativeClick?: () => void
}

const confirmDialogState = reactive({
  open: false,
  options: null as ConfirmDialogOptions | null,
})

function openConfirmDialog(options: ConfirmDialogOptions) {
  confirmDialogState.options = options
  confirmDialogState.open = true
  return {
    destroy: () => {
      confirmDialogState.open = false
    },
  }
}

export function useConfirmDialog() {
  return {
    warning: openConfirmDialog,
    error: openConfirmDialog,
    info: openConfirmDialog,
    success: openConfirmDialog,
  }
}

export function useConfirmDialogState() {
  return confirmDialogState
}
