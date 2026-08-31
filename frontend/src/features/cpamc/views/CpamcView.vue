<script setup lang="ts">
import { RefreshCwIcon } from '@lucide/vue'
import { onMounted, ref } from 'vue'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { getSettings } from '@/features/settings/api/settingsApi'
import { useI18n } from '@/shared/i18n'

const { errorText, t } = useI18n()
const cpamcUrl = ref('')
const loadError = ref('')
const isLoading = ref(false)
const isFrameLoading = ref(false)
const frameKey = ref(0)

async function loadCPAMC() {
  isLoading.value = true
  loadError.value = ''
  try {
    const settings = await getSettings()
    cpamcUrl.value = settings.cpamc_url || '/management.html'
    isFrameLoading.value = true
    frameKey.value += 1
  } catch (error) {
    cpamcUrl.value = ''
    loadError.value = errorText(error, '加载 CPAMC 配置失败', 'Failed to load CPAMC settings')
  } finally {
    isLoading.value = false
  }
}

onMounted(loadCPAMC)
</script>

<template>
  <section class="page cpamc-page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">CPAMC</h1>
      <Button variant="outline" :disabled="isLoading" @click="loadCPAMC">
        <Spinner v-if="isLoading" data-icon="inline-start" />
        <RefreshCwIcon v-else data-icon="inline-start" />
        {{ t('刷新', 'Refresh') }}
      </Button>
    </div>

    <div v-if="loadError" class="cpamc-error">
      <Alert variant="destructive">
        <AlertTitle>{{ t('无法打开 CPAMC', 'Unable to open CPAMC') }}</AlertTitle>
        <AlertDescription>{{ loadError }}</AlertDescription>
      </Alert>
      <Button @click="loadCPAMC">
        <RefreshCwIcon data-icon="inline-start" />
        {{ t('重试', 'Retry') }}
      </Button>
    </div>

    <div v-else class="cpamc-frame-shell">
      <iframe
        v-if="cpamcUrl"
        :key="frameKey"
        class="cpamc-frame"
        :src="cpamcUrl"
        title="CPAMC"
        @load="isFrameLoading = false"
      />
      <div v-if="isLoading || isFrameLoading" class="cpamc-loading">
        <Spinner class="size-5" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.cpamc-page {
  grid-template-rows: auto minmax(0, 1fr);
  height: 100%;
  overflow: hidden;
}

.cpamc-frame-shell {
  position: relative;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--cpa-border);
  border-radius: var(--cpa-radius);
  background: #fff;
  box-shadow: var(--cpa-shadow-card), var(--cpa-shadow-hairline);
}

.cpamc-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: #fff;
}

.cpamc-loading {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: var(--cpa-surface);
}

.cpamc-error {
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-content: start;
}

.cpamc-error [data-slot="button"] {
  justify-self: start;
  align-self: start;
}

</style>
