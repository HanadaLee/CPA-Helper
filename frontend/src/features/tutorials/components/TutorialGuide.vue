<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { BookOpenIcon, RefreshCwIcon } from '@lucide/vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { useI18n } from '@/shared/i18n'
import { listTutorials, type Tutorial } from '../api/tutorialsApi'
import type { TutorialContext } from '../tutorialVariables'
import TutorialMarkdown from './TutorialMarkdown.vue'

defineProps<{ context: TutorialContext }>()
const { t, isEnglish, errorText } = useI18n()
const items = ref<Tutorial[]>([])
const loading = ref(true)
const error = ref('')
const platform = ref(/Mac/i.test(navigator.platform) ? 'macos' : /Linux/i.test(navigator.platform) ? 'linux' : 'windows')
const client = ref('')
const platformNames = computed(() => ({ windows: 'Windows', macos: 'macOS', linux: 'Linux', ios: 'iOS', android: 'Android', all: t('通用', 'General') }))
const platforms = computed(() => Object.entries(platformNames.value).filter(([key]) => items.value.some((item) => item.platform === key)))
const platformItems = computed(() => items.value.filter((item) => item.platform === platform.value || item.platform === 'all'))
const clients = computed(() => [...new Set(platformItems.value.map((item) => item.client))])
const selected = computed(() => platformItems.value.filter((item) => item.client === client.value))
watch(platforms, (values) => {
  if (!values.some(([key]) => key === platform.value)) platform.value = values[0]?.[0] ?? 'all'
})
watch(clients, (values) => { if (!values.includes(client.value)) client.value = values[0] ?? '' })

async function reload() {
  loading.value = true
  error.value = ''
  try { items.value = await listTutorials() }
  catch (cause) { error.value = errorText(cause, '加载教程失败', 'Failed to load tutorials') }
  finally { loading.value = false }
}
onMounted(reload)
defineExpose({ reload })
</script>

<template>
  <Card data-tutorial-guide>
    <CardHeader>
      <CardTitle class="flex items-center gap-2"><BookOpenIcon class="size-4" />{{ t('接入教程', 'Setup tutorials') }}</CardTitle>
      <CardDescription>{{ t('按平台和客户端查看接入步骤，点击文章中的变量可选择并复制自己的密钥或 Endpoint。', 'Choose a platform and client. Click variables to select and copy your own key or endpoint.') }}</CardDescription>
    </CardHeader>
    <CardContent class="min-w-0">
      <div v-if="loading" class="flex flex-col gap-4" data-tutorial-loading>
        <Skeleton class="h-9 w-64 max-w-full" /><Skeleton class="h-6 w-48" /><Skeleton class="h-24 w-full" /><Skeleton class="h-40 w-full" />
      </div>
      <Alert v-else-if="error" variant="destructive">
        <AlertTitle>{{ t('教程暂时不可用', 'Tutorials unavailable') }}</AlertTitle>
        <AlertDescription>{{ error }}<Button variant="outline" size="sm" @click="reload"><RefreshCwIcon data-icon="inline-start" />{{ t('重试', 'Retry') }}</Button></AlertDescription>
      </Alert>
      <Empty v-else-if="!items.length"><EmptyHeader><EmptyTitle>{{ t('暂无教程', 'No tutorials yet') }}</EmptyTitle><EmptyDescription>{{ t('管理员发布后将在此展示。', 'Published tutorials will appear here.') }}</EmptyDescription></EmptyHeader></Empty>
      <Tabs v-else v-model="platform" class="tutorial-tabs">
        <TabsList class="max-w-full flex-wrap h-auto w-fit" :aria-label="t('教程平台', 'Tutorial platform')">
          <TabsTrigger v-for="[value, label] in platforms" :key="value" :value="value">{{ label }}</TabsTrigger>
        </TabsList>
        <TabsContent :value="platform" class="min-w-0">
          <Tabs v-model="client" class="tutorial-tabs">
            <TabsList variant="line" class="max-w-full flex-wrap h-auto w-fit" :aria-label="t('教程客户端', 'Tutorial client')">
              <TabsTrigger v-for="name in clients" :key="name" :value="name">{{ name }}</TabsTrigger>
            </TabsList>
            <TabsContent :value="client" class="min-w-0">
              <article v-for="(item, index) in selected" :key="item.id" class="py-3">
                <Separator v-if="index" class="mb-6" />
                <h3 class="mb-5 text-lg font-semibold">{{ isEnglish && item.title_en ? item.title_en : item.title }}</h3>
                <TutorialMarkdown :content="isEnglish && item.markdown_en ? item.markdown_en : item.markdown" :context="context" />
              </article>
            </TabsContent>
          </Tabs>
        </TabsContent>
      </Tabs>
    </CardContent>
  </Card>
</template>

<style scoped>
.tutorial-tabs { display: flex; flex-direction: column; gap: 1rem; min-width: 0; }
</style>
