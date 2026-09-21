<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { PencilIcon, PlusIcon, RefreshCwIcon, Trash2Icon } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { useI18n } from '@/shared/i18n'
import { useConfirmDialog } from '@/shared/ui/confirm-dialog'
import TablePaginationFooter from '@/shared/ui/TablePaginationFooter.vue'
import { deleteTutorial, manageTutorials, saveTutorial, type Tutorial, type TutorialInput } from '../api/tutorialsApi'
import { tutorialVariables } from '../tutorialVariables'
import TutorialMarkdown from './TutorialMarkdown.vue'

const { t, errorText, isEnglish } = useI18n()
const confirm = useConfirmDialog()
const items = ref<Tutorial[]>([])
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const deleting = ref(false)
const open = ref(false)
const editingID = ref<number | null>(null)
const editorMode = ref('edit')
const language = ref('zh')
const page = ref(1)
const pageSize = ref(20)
const visibleItems = computed(() => items.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
watch([pageSize, items], () => { page.value = Math.min(page.value, Math.max(1, Math.ceil(items.value.length / pageSize.value))) })
const emptyForm = (): TutorialInput => ({ title: '', title_en: '', client: 'Codex CLI', platform: 'all', markdown: '', markdown_en: '', sort_order: 0, published: false })
const form = reactive(emptyForm())
const body = computed({ get: () => language.value === 'en' ? form.markdown_en : form.markdown, set: (value: string) => { if (language.value === 'en') form.markdown_en = value; else form.markdown = value } })
const platformLabels = computed<Record<string, string>>(() => ({ all: t('通用', 'All platforms'), windows: 'Windows', macos: 'macOS', linux: 'Linux', ios: 'iOS', android: 'Android' }))

async function reload() {
  loading.value = true
  error.value = ''
  try { items.value = await manageTutorials() }
  catch (cause) { error.value = errorText(cause, '加载教程失败', 'Failed to load tutorials') }
  finally { loading.value = false }
}

function edit(item?: Tutorial) {
  editingID.value = item?.id ?? null
  Object.assign(form, item ? {
    title: item.title, title_en: item.title_en, client: item.client, platform: item.platform,
    markdown: item.markdown, markdown_en: item.markdown_en, sort_order: item.sort_order, published: item.published,
  } : emptyForm())
  editorMode.value = 'edit'
  language.value = 'zh'
  open.value = true
}

async function save() {
  if (saving.value) return
  if (!form.title.trim() || !form.client.trim() || !form.markdown.trim()) {
    toast.error(t('请填写标题、客户端和中文正文', 'Enter a title, client and Chinese content.'))
    return
  }
  saving.value = true
  try {
    await saveTutorial(editingID.value, { ...form })
    open.value = false
    toast.success(t('教程已保存', 'Tutorial saved'))
    await reload()
  } catch (cause) { toast.error(errorText(cause, '保存教程失败', 'Failed to save tutorial')) }
  finally { saving.value = false }
}

function remove(item: Tutorial) {
  confirm.warning({
    title: t('删除教程', 'Delete tutorial'),
    content: t(`确定删除「${item.title}」？删除后无法恢复。`, `Delete “${item.title_en || item.title}”? This cannot be undone.`),
    positiveText: t('删除', 'Delete'), negativeText: t('取消', 'Cancel'),
    onPositiveClick: async () => {
      deleting.value = true
      try {
        await deleteTutorial(item.id)
        toast.success(t('教程已删除', 'Tutorial deleted'))
        await reload()
      } catch (cause) {
        toast.error(errorText(cause, '删除教程失败', 'Failed to delete tutorial'))
      } finally { deleting.value = false }
    },
  })
}

async function insertVariable(name: string) {
  const input = document.getElementById('tutorial-body') as HTMLTextAreaElement | null
  const start = input?.selectionStart ?? body.value.length
  const end = input?.selectionEnd ?? start
  const value = `{{${name}}}`
  body.value = body.value.slice(0, start) + value + body.value.slice(end)
  await nextTick()
  input?.focus()
  input?.setSelectionRange(start + value.length, start + value.length)
}

onMounted(reload)
defineExpose({ reload })
</script>

<template>
  <Card data-tutorial-manager>
    <CardHeader>
      <CardTitle>{{ t('教程管理', 'Tutorial management') }}</CardTitle>
      <CardDescription>{{ t('发布的教程展示在 API 密钥页下方。每篇独立保存，支持 Markdown、中英文和动态复制变量。', 'Published tutorials appear below API keys. Each article saves independently and supports Markdown, translations and copy variables.') }}</CardDescription>
      <CardAction><Button @click="edit()"><PlusIcon data-icon="inline-start" />{{ t('新建教程', 'New tutorial') }}</Button></CardAction>
    </CardHeader>
    <CardContent>
      <Skeleton v-if="loading" class="h-48 w-full" />
      <Alert v-else-if="error" variant="destructive"><AlertTitle>{{ t('加载失败', 'Loading failed') }}</AlertTitle><AlertDescription>{{ error }}<Button variant="outline" @click="reload"><RefreshCwIcon />{{ t('重试', 'Retry') }}</Button></AlertDescription></Alert>
      <Empty v-else-if="!items.length"><EmptyHeader><EmptyTitle>{{ t('暂无教程', 'No tutorials yet') }}</EmptyTitle></EmptyHeader></Empty>
      <div v-else class="overflow-hidden rounded-lg border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('标题', 'Title') }}</TableHead><TableHead>{{ t('客户端', 'Client') }}</TableHead><TableHead>{{ t('平台', 'Platform') }}</TableHead><TableHead>{{ t('排序', 'Order') }}</TableHead><TableHead>{{ t('状态', 'Status') }}</TableHead><TableHead class="text-right">{{ t('操作', 'Actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="item in visibleItems" :key="item.id">
              <TableCell class="max-w-64 truncate" :title="item.title">{{ isEnglish && item.title_en ? item.title_en : item.title }}</TableCell>
              <TableCell>{{ item.client }}</TableCell><TableCell>{{ platformLabels[item.platform] }}</TableCell><TableCell>{{ item.sort_order }}</TableCell>
              <TableCell><Badge :variant="item.published ? 'secondary' : 'outline'">{{ item.published ? t('已发布', 'Published') : t('草稿', 'Draft') }}</Badge></TableCell>
              <TableCell>
                <div class="flex justify-end gap-1">
                  <Button variant="ghost" size="sm" @click="edit(item)"><PencilIcon data-icon="inline-start" />{{ t('编辑', 'Edit') }}</Button>
                  <Button variant="ghost" size="icon-sm" :aria-label="t('删除教程', 'Delete tutorial')" :disabled="deleting" @click="remove(item)"><Trash2Icon /></Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
        <TablePaginationFooter v-model:page="page" v-model:page-size="pageSize" :total="items.length" />
      </div>
    </CardContent>
  </Card>
  <Dialog :open="open" @update:open="(value) => { if (!saving) open = value }">
    <DialogContent class="sm:max-w-4xl max-h-[90dvh] overflow-y-auto" @interact-outside.prevent>
      <DialogHeader>
        <DialogTitle>{{ editingID ? t('编辑教程', 'Edit tutorial') : t('新建教程', 'New tutorial') }}</DialogTitle>
        <DialogDescription>{{ t('真实密钥不写入文章；使用变量让读者选择并复制自己的密钥。英文留空时使用中文。', 'Never save real keys in articles. Variables let readers copy their own keys. Empty English fields fall back to Chinese.') }}</DialogDescription>
      </DialogHeader>
      <form class="flex flex-col gap-5" @submit.prevent="save">
        <FieldGroup class="grid gap-4 sm:grid-cols-2">
          <Field><FieldLabel for="tutorial-title">{{ t('标题（中文）', 'Title (Chinese)') }}</FieldLabel><Input id="tutorial-title" v-model="form.title" required maxlength="120" /></Field>
          <Field><FieldLabel for="tutorial-title-en">{{ t('标题（英文，可选）', 'Title (English, optional)') }}</FieldLabel><Input id="tutorial-title-en" v-model="form.title_en" maxlength="120" /></Field>
          <Field><FieldLabel for="tutorial-client">{{ t('客户端', 'Client') }}</FieldLabel><Input id="tutorial-client" v-model="form.client" required maxlength="64" :placeholder="t('例如 Codex CLI、Claude Code', 'e.g. Codex CLI, Claude Code')" /></Field>
          <Field>
            <FieldLabel for="tutorial-platform">{{ t('平台', 'Platform') }}</FieldLabel>
            <Select v-model="form.platform"><SelectTrigger id="tutorial-platform" class="w-full"><SelectValue :placeholder="t('平台', 'Platform')" /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="(label, value) in platformLabels" :key="value" :value="value">{{ label }}</SelectItem></SelectGroup></SelectContent></Select>
          </Field>
          <Field><FieldLabel for="tutorial-order">{{ t('排序（越小越靠前）', 'Order (lowest first)') }}</FieldLabel><Input id="tutorial-order" v-model.number="form.sort_order" type="number" min="0" max="10000" step="1" required /></Field>
          <Field orientation="horizontal" class="self-end rounded-lg border border-border p-3"><FieldLabel for="tutorial-published">{{ t('发布教程', 'Publish tutorial') }}</FieldLabel><Switch id="tutorial-published" v-model="form.published" /></Field>
        </FieldGroup>
        <Tabs v-model="language" class="flex flex-col gap-3">
          <TabsList><TabsTrigger value="zh">{{ t('中文正文', 'Chinese content') }}</TabsTrigger><TabsTrigger value="en">{{ t('英文正文', 'English content') }}</TabsTrigger></TabsList>
          <TabsContent :value="language">
            <Tabs v-model="editorMode" class="flex flex-col gap-3">
              <TabsList variant="line"><TabsTrigger value="edit">Markdown</TabsTrigger><TabsTrigger value="preview">{{ t('预览', 'Preview') }}</TabsTrigger></TabsList>
              <TabsContent value="edit">
                <FieldGroup>
                  <Field>
                    <FieldLabel for="tutorial-body">{{ t('正文', 'Content') }}</FieldLabel>
                    <div class="flex flex-wrap gap-1" role="group" :aria-label="t('插入变量', 'Insert variable')"><Button v-for="name in tutorialVariables" :key="name" type="button" size="sm" variant="outline" @click="insertVariable(name)">{{ name }}</Button></div>
                    <FieldDescription>{{ t('点击上方插入变量。读者点击变量或复制代码时，单个选项直接复制，多个选项弹出选择框。', 'Insert a variable above. Copying selects your key or endpoint; a single option copies immediately.') }}</FieldDescription>
                    <Textarea id="tutorial-body" v-model="body" class="min-h-72 font-mono" />
                  </Field>
                </FieldGroup>
              </TabsContent>
              <TabsContent value="preview"><TutorialMarkdown :content="body" preview /></TabsContent>
            </Tabs>
          </TabsContent>
        </Tabs>
        <DialogFooter><Button type="button" variant="outline" :disabled="saving" @click="open = false">{{ t('取消', 'Cancel') }}</Button><Button type="submit" :disabled="saving"><Spinner v-if="saving" data-icon="inline-start" />{{ t('保存教程', 'Save tutorial') }}</Button></DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
