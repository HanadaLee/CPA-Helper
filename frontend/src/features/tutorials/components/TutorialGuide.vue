<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { BookOpenIcon, RefreshCwIcon } from '@lucide/vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { useI18n } from '@/shared/i18n'
import { listTutorials, type Tutorial } from '../api/tutorialsApi'
import type { TutorialContext } from '../tutorialVariables'
import TutorialMarkdown from './TutorialMarkdown.vue'

defineProps<{ context: TutorialContext }>()
const { t, isEnglish, errorText } = useI18n()
const items = ref<Tutorial[]>([])
const loading = ref(true)
const error = ref('')
const selectedID = ref('')
const selected = computed(() => items.value.find((item) => String(item.id) === selectedID.value))
watch(items, (values) => {
  if (!values.some((item) => String(item.id) === selectedID.value)) selectedID.value = values[0] ? String(values[0].id) : ''
})

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
      <CardDescription>{{ t('选择教程查看接入步骤，点击文章中的变量可选择并复制密钥、Endpoint 或可用模型。', 'Choose a tutorial. Click variables to select and copy a key, endpoint or available model.') }}</CardDescription>
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
      <Tabs v-else v-model="selectedID" class="tutorial-tabs">
        <TabsList class="max-w-full flex-wrap h-auto w-fit justify-start gap-1" :aria-label="t('教程', 'Tutorials')">
          <TabsTrigger v-for="item in items" :key="item.id" :value="String(item.id)" :title="isEnglish && item.title_en ? item.title_en : item.title" class="h-8 min-w-0 max-w-full flex-none"><span class="truncate">{{ isEnglish && item.title_en ? item.title_en : item.title }}</span></TabsTrigger>
        </TabsList>
        <TabsContent v-if="selected" :key="selected.id" :value="String(selected.id)" class="min-w-0">
          <article class="py-3">
            <TutorialMarkdown :content="isEnglish && selected.markdown_en ? selected.markdown_en : selected.markdown" :context="context" />
          </article>
        </TabsContent>
      </Tabs>
    </CardContent>
  </Card>
</template>

<style scoped>
.tutorial-tabs { display: flex; flex-direction: column; gap: 1rem; min-width: 0; }
.tutorial-tabs :deep([data-slot="tabs-list"]) { height: auto; }
</style>
