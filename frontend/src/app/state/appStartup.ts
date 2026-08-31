import { readonly, ref } from 'vue'

const isAppReady = ref(false)

export function markAppReady(): void {
  isAppReady.value = true
}

export function useAppStartup() {
  return {
    isAppReady: readonly(isAppReady),
  }
}
