<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { AppAlert, AppButton, AppSpinner } from '@/shared/ui/app-kit'

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
    <div class="page-header">
      <div>
        <h1 class="page-title">CPAMC</h1>
        <p class="page-subtitle">{{ t('CLIProxyAPI 管理中心', 'CLIProxyAPI Management Center') }}</p>
      </div>
      <AppButton secondary :loading="isLoading" @click="loadCPAMC">
        {{ t('刷新', 'Refresh') }}
      </AppButton>
    </div>

    <div v-if="loadError" class="cpamc-error">
      <AppAlert type="error" :title="t('无法打开 CPAMC', 'Unable to open CPAMC')">
        {{ loadError }}
      </AppAlert>
      <AppButton type="primary" @click="loadCPAMC">{{ t('重试', 'Retry') }}</AppButton>
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
        <AppSpinner size="large" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.cpamc-page {
  grid-template-rows: auto minmax(0, 1fr);
  height: calc(100dvh - 60px);
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
  display: grid;
  gap: 14px;
  align-content: start;
}

.cpamc-error :deep(.n-button) {
  justify-self: start;
}

@media (max-width: 860px) {
  .cpamc-page {
    height: calc(100dvh - 78px);
  }
}
</style>
