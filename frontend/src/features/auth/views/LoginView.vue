<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AppAlert, AppButton, AppCard, AppForm, AppFormItem, AppInput, useMessage } from '@/shared/ui/app-kit'

import { getSetupState, login, setupFirstAdmin } from '@/features/auth/api/authApi'
import { setCurrentUser } from '@/features/auth/state/currentUser'
import { useI18n } from '@/shared/i18n'
import { logoUrl } from '@/shared/utils/assets'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { errorText, t } = useI18n()
const isLoading = ref(false)
const isSetupLoading = ref(true)
const setupRequired = ref(false)
const errorMessage = ref<string | null>(null)
const form = reactive({
  username: '',
  password: '',
  nickname: '',
})
const headingTitle = computed(() => (setupRequired.value ? t('创建首个管理员账号', 'Create first admin account') : 'CPA-Helper'))
const headingSubtitle = computed(() =>
  setupRequired.value ? t('首次使用前需要先录入管理员账号', 'Create an admin account before first use') : t('本地 AI 用量管理控制台', 'Local AI usage management console'),
)
const submitText = computed(() => (setupRequired.value ? t('创建并登录', 'Create and sign in') : t('登录', 'Sign in')))

onMounted(async () => {
  try {
    const state = await getSetupState()
    setupRequired.value = state.setup_required
  } catch (error) {
    errorMessage.value = errorText(error, '初始化状态加载失败', 'Failed to load setup state')
  } finally {
    isSetupLoading.value = false
  }
})

async function handleSubmit() {
  if (setupRequired.value && !form.nickname.trim()) {
    message.error(t('用户昵称不能为空', 'User nickname is required'))
    return
  }
  isLoading.value = true
  errorMessage.value = null
  try {
    let homePath = '/account/usage'
    if (setupRequired.value) {
      const user = await setupFirstAdmin({
        username: form.username,
        password: form.password,
        nickname: form.nickname,
      })
      setCurrentUser(user)
      homePath = user.is_admin ? '/admin/usage' : '/account/usage'
      message.success(t('管理员账号已创建', 'Admin account created'))
    } else {
      const user = await login({ username: form.username, password: form.password })
      setCurrentUser(user)
      homePath = user.is_admin ? '/admin/usage' : '/account/usage'
      message.success(t('登录成功', 'Signed in'))
    }
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : homePath
    await router.push(redirect)
  } catch (error) {
    errorMessage.value = errorText(error, '登录失败', 'Sign in failed')
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <main class="auth-screen">
    <section class="auth-brand-panel" aria-hidden="true">
      <div class="brand-stage">
        <span class="brand-word brand-word-cpa">CPA</span>
        <span class="brand-word brand-word-helper">HELPER</span>
      </div>
    </section>

    <section class="auth-content" :aria-label="t('登录区域', 'Sign-in area')">
      <div class="brand-mark">
        <img :src="logoUrl" alt="">
      </div>

      <AppCard class="auth-card" :bordered="true">
        <div class="auth-heading">
          <h1>{{ headingTitle }}</h1>
          <p>{{ headingSubtitle }}</p>
        </div>

        <AppAlert v-if="errorMessage" type="error" :bordered="false" class="auth-alert">
          {{ errorMessage }}
        </AppAlert>

        <AppAlert v-if="setupRequired" type="warning" :bordered="false" class="auth-alert">
          {{ t('账号一旦创建，不允许删除，只允许禁用，请谨慎操作。', 'Accounts cannot be deleted after creation. They can only be disabled, so proceed carefully.') }}
        </AppAlert>

        <AppForm :model="form" label-placement="top" @submit.prevent="handleSubmit">
          <AppFormItem :label="t('账号', 'Account')" path="username">
            <AppInput v-model:value="form.username" autocomplete="username" />
          </AppFormItem>
          <AppFormItem :label="t('密码', 'Password')" path="password">
            <AppInput
              v-model:value="form.password"
              type="password"
              show-password-on="mousedown"
              autocomplete="current-password"
              @keyup.enter="handleSubmit"
            />
          </AppFormItem>
          <AppFormItem v-if="setupRequired" :label="t('用户昵称', 'User nickname')" path="nickname" required>
            <AppInput v-model:value="form.nickname" :placeholder="t('例如：研发用户', 'Example: Engineering user')" />
          </AppFormItem>
          <AppButton type="primary" block attr-type="submit" :loading="isLoading || isSetupLoading">
            {{ submitText }}
          </AppButton>
        </AppForm>
      </AppCard>
    </section>
  </main>
</template>

<style scoped>
.auth-screen {
  display: grid;
  grid-template-columns: minmax(420px, 1fr) minmax(420px, 1fr);
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  overflow: auto;
  background: var(--cpa-shell-bg);
}

.auth-brand-panel {
  position: relative;
  display: grid;
  min-height: 100%;
  align-items: center;
  overflow: hidden;
  padding: 72px 48px;
  background:
    radial-gradient(circle at 18% 18%, rgb(34 193 200 / 26%), transparent 32%),
    radial-gradient(circle at 82% 78%, rgb(0 154 168 / 22%), transparent 38%),
    linear-gradient(145deg, #061c20 0%, #083b3f 52%, #071a1e 100%);
}

.auth-brand-panel::before {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgb(255 255 255 / 4%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(255 255 255 / 4%) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: linear-gradient(135deg, black, transparent 72%);
  content: "";
}

.brand-stage {
  position: relative;
  display: grid;
  z-index: 1;
  width: min(620px, 100%);
  gap: 8px;
  align-content: center;
  justify-self: center;
  font-weight: 820;
  line-height: 0.9;
  text-transform: uppercase;
}

.brand-word {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
  letter-spacing: 0;
}

.brand-word-cpa {
  justify-self: start;
  color: rgb(255 255 255 / 96%);
  font-size: 148px;
}

.brand-word-helper {
  justify-self: end;
  color: #6fe0dc;
  font-size: 116px;
}

.auth-content {
  display: grid;
  min-width: 0;
  align-content: center;
  justify-items: center;
  gap: 24px;
  padding: 48px;
  background:
    radial-gradient(circle at 88% 10%, var(--cpa-primary-weak), transparent 28%),
    var(--cpa-canvas);
}

.auth-card {
  width: min(420px, 100%);
  border: 1px solid var(--cpa-border);
  border-radius: calc(var(--cpa-radius) + 4px);
  overflow: hidden;
  background: var(--cpa-surface-raised);
  box-shadow: var(--cpa-shadow);
}

.auth-card :deep(.n-card__content) {
  padding: 36px 32px 32px;
}

.auth-heading {
  display: grid;
  justify-items: center;
  margin-bottom: 24px;
  text-align: center;
}

.brand-mark {
  display: grid;
  width: 68px;
  height: 68px;
  place-items: center;
  border-radius: 18px;
  overflow: hidden;
  background: var(--cpa-surface-solid);
  border: 1px solid color-mix(in srgb, var(--cpa-primary) 20%, var(--cpa-border));
  box-shadow: 0 12px 28px -18px rgb(0 112 118 / 58%), 0 1px 2px rgb(24 45 53 / 8%);
}

.brand-mark img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

h1 {
  margin: 0;
  color: var(--cpa-text-strong);
  font-size: 24px;
  font-weight: 800;
  line-height: 1.18;
  text-wrap: pretty;
}

p {
  margin: 6px 0 0;
  color: var(--cpa-text-muted);
  text-wrap: pretty;
}

.auth-alert {
  margin-bottom: 12px;
}

.auth-card :deep(.n-form-item-label) {
  font-weight: 650;
}

@media (max-width: 1320px) {
  .brand-word-cpa {
    font-size: 128px;
  }

  .brand-word-helper {
    font-size: 102px;
  }
}

@media (max-width: 1180px) {
  .auth-screen {
    grid-template-columns: minmax(360px, 0.9fr) minmax(390px, 1.1fr);
  }

  .brand-word-cpa {
    font-size: 118px;
  }

  .brand-word-helper {
    font-size: 94px;
  }
}

@media (max-width: 900px) {
  .auth-screen {
    grid-template-columns: 1fr;
  }

  .auth-brand-panel {
    display: none;
  }

  .auth-content {
    min-height: 100%;
    align-content: start;
    gap: 16px;
    padding: max(28px, env(safe-area-inset-top)) 14px 20px;
  }

  .brand-mark {
    width: 56px;
    height: 56px;
    border-radius: 15px;
  }

  .auth-card {
    align-self: center;
  }
}

@media (max-width: 520px) {
  .auth-content {
    gap: 14px;
  }

  .auth-card :deep(.n-card__content) {
    padding: 24px 18px 22px;
  }

  h1 {
    font-size: 22px;
  }
}
</style>
