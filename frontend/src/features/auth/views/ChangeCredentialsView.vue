<script setup lang="ts">
import { EyeIcon, EyeOffIcon } from '@lucide/vue'
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from '@/components/ui/input-group'
import { Spinner } from '@/components/ui/spinner'
import { changeCredentials, getMe } from '@/features/auth/api/authApi'
import AuthShell from '@/features/auth/components/AuthShell.vue'
import { setCurrentUser } from '@/features/auth/state/currentUser'
import { useI18n } from '@/shared/i18n'

const router = useRouter()
const { errorText, t } = useI18n()
const isLoading = ref(false)
const errorMessage = ref<string | null>(null)
const showNewPassword = ref(false)
const showCurrentPassword = ref(false)
const form = reactive({
  username: '',
  password: '',
  current_password: '',
})

onMounted(async () => {
  try {
    const user = await getMe()
    form.username = user.username
  } catch {
    form.username = ''
  }
})

async function handleSubmit() {
  isLoading.value = true
  errorMessage.value = null
  try {
    const user = await changeCredentials({
      password: form.password,
      current_password: form.current_password || undefined,
    })
    setCurrentUser(user)
    toast.success(t('密码已更新', 'Password updated'))
    await router.push(user.is_admin ? '/admin/usage' : '/account/usage')
  } catch (error) {
    errorMessage.value = errorText(error, '更新失败', 'Update failed')
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <AuthShell :section-label="t('修改密码区域', 'Change password area')">
    <Card class="w-full max-w-[420px]">
      <CardHeader class="text-center">
        <CardTitle>{{ t('修改密码', 'Change password') }}</CardTitle>
        <CardDescription>{{ t('首次登录后需要完成密码更新', 'Update your password after first sign-in') }}</CardDescription>
      </CardHeader>

      <CardContent class="flex flex-col gap-5">
        <Alert v-if="errorMessage" variant="destructive">
          <AlertDescription>{{ errorMessage }}</AlertDescription>
        </Alert>

        <form @submit.prevent="handleSubmit">
          <FieldGroup>
            <Field data-disabled>
              <FieldLabel for="credentials-username">{{ t('账号', 'Account') }}</FieldLabel>
              <Input id="credentials-username" v-model="form.username" autocomplete="username" disabled />
            </Field>

            <Field>
              <FieldLabel for="credentials-new-password">{{ t('新密码', 'New password') }}</FieldLabel>
              <InputGroup>
                <InputGroupInput
                  id="credentials-new-password"
                  v-model="form.password"
                  :type="showNewPassword ? 'text' : 'password'"
                  autocomplete="new-password"
                />
                <InputGroupAddon align="inline-end">
                  <InputGroupButton
                    size="icon-xs"
                    :aria-label="showNewPassword ? t('隐藏密码', 'Hide password') : t('显示密码', 'Show password')"
                    @click="showNewPassword = !showNewPassword"
                  >
                    <EyeOffIcon v-if="showNewPassword" data-icon="inline-start" />
                    <EyeIcon v-else data-icon="inline-start" />
                  </InputGroupButton>
                </InputGroupAddon>
              </InputGroup>
            </Field>

            <Field>
              <FieldLabel for="credentials-current-password">{{ t('当前密码', 'Current password') }}</FieldLabel>
              <InputGroup>
                <InputGroupInput
                  id="credentials-current-password"
                  v-model="form.current_password"
                  :type="showCurrentPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                />
                <InputGroupAddon align="inline-end">
                  <InputGroupButton
                    size="icon-xs"
                    :aria-label="showCurrentPassword ? t('隐藏密码', 'Hide password') : t('显示密码', 'Show password')"
                    @click="showCurrentPassword = !showCurrentPassword"
                  >
                    <EyeOffIcon v-if="showCurrentPassword" data-icon="inline-start" />
                    <EyeIcon v-else data-icon="inline-start" />
                  </InputGroupButton>
                </InputGroupAddon>
              </InputGroup>
            </Field>

            <Button class="w-full" type="submit" :disabled="isLoading" :aria-busy="isLoading">
              <Spinner v-if="isLoading" data-icon="inline-start" />
              {{ t('保存', 'Save') }}
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  </AuthShell>
</template>
