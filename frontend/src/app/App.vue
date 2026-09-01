<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watchEffect } from 'vue'
import AppStartupSkeleton from '@/app/layout/AppStartupSkeleton.vue'
import { useAppStartup } from '@/app/state/appStartup'
import { Toaster } from '@/components/ui/sonner'
import { getBranding } from '@/features/settings/api/settingsApi'
import { currentLanguage } from '@/shared/i18n'
import type { BrandingResponse } from '@/shared/types/api'
import ConfirmDialogProvider from '@/shared/ui/ConfirmDialogProvider.vue'

const { isAppReady } = useAppStartup()
const branding = reactive<BrandingResponse>({
  brand_name_zh: 'CPA-Helper',
  brand_name_en: 'CPA-Helper',
  brand_subtitle_zh: '边缘网关管理平台',
  brand_subtitle_en: 'Edge Gateway Management Platform',
})
const hasResolvedInitialBranding = ref(false)

function applyBranding(value: Partial<BrandingResponse> | null | undefined) {
  if (!value) return
  for (const key of ['brand_name_zh', 'brand_name_en', 'brand_subtitle_zh', 'brand_subtitle_en'] as const) {
    const nextValue = value[key]
    if (typeof nextValue === 'string' && nextValue.trim()) {
      branding[key] = nextValue.trim()
    }
  }
}

function handleBrandingUpdated(event: Event) {
  applyBranding((event as CustomEvent<Partial<BrandingResponse>>).detail)
  hasResolvedInitialBranding.value = true
}

watchEffect(() => {
  if (!hasResolvedInitialBranding.value) return

  const isChinese = currentLanguage.value === 'zh'
  const name = isChinese ? branding.brand_name_zh : branding.brand_name_en
  const subtitle = isChinese ? branding.brand_subtitle_zh : branding.brand_subtitle_en
  document.title = `${name} - ${subtitle}`
})

onMounted(() => {
  window.addEventListener('cpa:branding-updated', handleBrandingUpdated)
  void getBranding()
    .then(applyBranding)
    .catch(() => {
      // Apply the built-in title only when the public branding endpoint is unavailable.
    })
    .finally(() => {
      hasResolvedInitialBranding.value = true
    })
})

onBeforeUnmount(() => {
  window.removeEventListener('cpa:branding-updated', handleBrandingUpdated)
})
</script>

<template>
  <ConfirmDialogProvider>
    <AppStartupSkeleton v-if="!isAppReady" />
    <RouterView v-else />
    <Toaster rich-colors position="top-center" />
  </ConfirmDialogProvider>
</template>
