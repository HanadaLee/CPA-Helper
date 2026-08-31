<script setup lang="ts">
import AppStartupSkeleton from '@/app/layout/AppStartupSkeleton.vue'
import { useAppStartup } from '@/app/state/appStartup'
import {
  AppDialogProvider,
  AppMessageProvider,
} from '@/shared/ui/app-kit'

const { isAppReady } = useAppStartup()
</script>

<template>
  <AppDialogProvider>
    <AppMessageProvider>
      <AppStartupSkeleton v-if="!isAppReady" />
      <RouterView v-else />
    </AppMessageProvider>
  </AppDialogProvider>
</template>
