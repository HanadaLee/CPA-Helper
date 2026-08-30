<script setup lang="ts">
import {
  AppDialogProvider,
  AppMessageProvider,
} from '@/shared/ui/app-kit'
</script>

<template>
  <AppDialogProvider>
    <AppMessageProvider>
      <RouterView />
    </AppMessageProvider>
  </AppDialogProvider>
</template>
