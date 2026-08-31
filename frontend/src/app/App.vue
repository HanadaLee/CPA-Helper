<script setup lang="ts">
import AppStartupSkeleton from '@/app/layout/AppStartupSkeleton.vue'
import { useAppStartup } from '@/app/state/appStartup'
import { Toaster } from '@/components/ui/sonner'
import ConfirmDialogProvider from '@/shared/ui/ConfirmDialogProvider.vue'

const { isAppReady } = useAppStartup()
</script>

<template>
  <ConfirmDialogProvider>
    <AppStartupSkeleton v-if="!isAppReady" />
    <RouterView v-else />
    <Toaster rich-colors position="top-center" />
  </ConfirmDialogProvider>
</template>
