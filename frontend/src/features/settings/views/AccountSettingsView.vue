<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { AppButton, AppForm, AppFormItem, AppInput, AppStack, useMessage } from '@/shared/ui/app-kit'
import { ShieldCheck, UserRound } from '@lucide/vue'

import { changeCredentials, getMe } from '@/features/auth/api/authApi'
import { setCurrentUser } from '@/features/auth/state/currentUser'
import { useI18n } from '@/shared/i18n'

const message = useMessage()
const { errorText, t } = useI18n()
const isLoading = ref(false)
const isSaving = ref(false)
const isAdmin = ref(false)

const accountForm = reactive({
  username: '',
  password: '',
  current_password: '',
})
const roleText = computed(() => (isAdmin.value ? t('管理员', 'Admin') : t('普通账户', 'Standard account')))

async function refresh() {
  isLoading.value = true
  try {
    const user = await getMe()
    setCurrentUser(user)
    accountForm.username = user.username
    isAdmin.value = user.is_admin
  } catch (error) {
    message.error(errorText(error, '加载账户失败', 'Failed to load account'))
  } finally {
    isLoading.value = false
  }
}

async function saveAccount() {
  isSaving.value = true
  try {
    const user = await changeCredentials({
      password: accountForm.password,
      current_password: accountForm.current_password || undefined,
    })
    setCurrentUser(user)
    window.dispatchEvent(new CustomEvent('cpa:account-updated', { detail: user }))
    accountForm.username = user.username
    accountForm.password = ''
    accountForm.current_password = ''
    message.success(t('账户已更新', 'Account updated'))
  } catch (error) {
    message.error(errorText(error, '账户更新失败', 'Failed to update account'))
  } finally {
    isSaving.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <section class="page">
    <div class="page-toolbar">
      <h1 data-page-title class="page-title">{{ t('账户设置', 'Account Settings') }}</h1>
      <AppStack>
        <AppButton secondary :loading="isLoading" @click="refresh">{{ t('刷新', 'Refresh') }}</AppButton>
        <AppButton type="primary" :loading="isSaving" @click="saveAccount">{{ t('保存账户', 'Save account') }}</AppButton>
      </AppStack>
    </div>

    <div class="metric-grid account-settings-metrics">
      <div class="metric-card is-primary">
        <div class="metric-icon" aria-hidden="true">
          <UserRound :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('当前账号', 'Current account') }}</div>
        <div class="metric-value">{{ accountForm.username || '-' }}</div>
        <div class="metric-footnote">{{ t('登录身份', 'Sign-in identity') }}</div>
      </div>
      <div class="metric-card is-purple">
        <div class="metric-icon" aria-hidden="true">
          <ShieldCheck :size="20" :stroke-width="2.2" />
        </div>
        <div class="metric-label">{{ t('权限', 'Role') }}</div>
        <div class="metric-value">{{ roleText }}</div>
        <div class="metric-footnote">{{ t('当前会话', 'Current session') }}</div>
      </div>
    </div>

    <section class="panel">
      <div class="panel-inner">
        <h2 class="section-title">{{ t('账号与密码', 'Account and Password') }}</h2>
        <AppForm :model="accountForm" label-placement="top">
          <div class="form-grid">
            <AppFormItem class="account-field" :label="t('账号', 'Account')">
              <AppInput v-model:value="accountForm.username" autocomplete="username" disabled />
            </AppFormItem>
            <AppFormItem :label="t('当前密码', 'Current password')">
              <AppInput
                v-model:value="accountForm.current_password"
                type="password"
                show-password-on="mousedown"
                autocomplete="current-password"
              />
            </AppFormItem>
            <AppFormItem :label="t('新密码', 'New password')">
              <AppInput
                v-model:value="accountForm.password"
                type="password"
                show-password-on="mousedown"
                autocomplete="new-password"
                @keyup.enter="saveAccount"
              />
            </AppFormItem>
          </div>
        </AppForm>
      </div>
    </section>
  </section>
</template>

<style scoped>
.account-settings-metrics {
  grid-template-columns: repeat(2, minmax(180px, 1fr));
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 12px;
}

.account-field {
  grid-column: 1 / -1;
}

@media (max-width: 720px) {
  .account-settings-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .form-grid {
    grid-template-columns: 1fr;
  }

  .account-field {
    grid-column: auto;
  }
}
</style>
